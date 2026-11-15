package repositories

import (
	"context"
	"database/sql"

	"pulso-api/models"
	"pulso-api/utils"
)

// MetricasVendasRepository calcula os números dos cards do dashboard
// vendas (só leitura). Base: CTEsVendas (op, orc, vd).
type MetricasVendasRepository interface {
	// Receita: soma e quantidade das vendas com fechada_em entre de e ate
	// (ven-01, ven-03). Use outros de/ate em f para o período anterior.
	Receita(ctx context.Context, f models.Filtros) (models.ResumoReceita, error)
	// Conversao: oportunidades cujo estado final (ganha ou perdida, o
	// último evento até ate) ocorreu no período (ven-02).
	Conversao(ctx context.Context, f models.Filtros) (models.ResumoConversao, error)
	// Ciclo: média de dias entre a criação e o evento ganha, nas
	// oportunidades ganhas no período (ven-04).
	Ciclo(ctx context.Context, f models.Filtros) (models.ResumoCiclo, error)
	// Funil: para cada etapa de lead a ganha (ordem de funil_etapas),
	// oportunidades distintas com evento daquela etapa no período (ven-05).
	Funil(ctx context.Context, f models.Filtros) ([]models.EtapaFunil, error)
	// Pipeline: oportunidades abertas em ate, com o orçamento de maior
	// versão não recusado, por etapa atual: bruto e ponderado (ven-06).
	Pipeline(ctx context.Context, f models.Filtros) ([]models.EtapaPipeline, error)
	// MotivosPerda: oportunidades perdidas no período, por motivo (ven-07).
	MotivosPerda(ctx context.Context, f models.Filtros) ([]models.GrupoContagem, error)
	// OrigemLeads: leads criados no período por origem, com ganhas e
	// perdidas entre eles (ven-08).
	OrigemLeads(ctx context.Context, f models.Filtros) ([]models.OrigemLeads, error)
	// ReunioesAteFechar: média de reuniões por oportunidade encerrada no
	// período, para ganhas e perdidas (ven-09).
	ReunioesAteFechar(ctx context.Context, f models.Filtros) ([]models.ReunioesDesfecho, error)
	// ReceitaMensal: receita por mês de fechamento, em meses inteiros de
	// mes(de) a mes(ate), com a meta do mês (ven-10).
	ReceitaMensal(ctx context.Context, f models.Filtros) ([]models.MesReceita, error)
	// StatusPropostas: orçamentos emitidos no período por status atual (ven-11).
	StatusPropostas(ctx context.Context, f models.Filtros) (models.StatusPropostas, error)
	// MargemPorComplexidade: orçamentos emitidos no período por
	// complexidade: margem média, valor e horas (ven-12).
	MargemPorComplexidade(ctx context.Context, f models.Filtros) ([]models.MargemComplexidade, error)
	// OrcadoReal: orçamentos com venda no período, por tipo de projeto,
	// com o realizado nos projetos ligados até ate (ven-13, ven-14).
	OrcadoReal(ctx context.Context, f models.Filtros) ([]models.OrcadoRealTipo, error)
}

type metricasVendasRepository struct {
	db *sql.DB
}

// NewMetricasVendasRepository cria o repository de métricas de vendas.
func NewMetricasVendasRepository(db *sql.DB) MetricasVendasRepository {
	return &metricasVendasRepository{db}
}

func (r *metricasVendasRepository) Receita(ctx context.Context, f models.Filtros) (models.ResumoReceita, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim := c.Arg(inicio), c.Arg(fimAte)
	query := "WITH " + CTEsVendas(c, f) + `
	SELECT COALESCE(SUM(vd.valor), 0)::float8, COUNT(*)
	FROM vd
	WHERE vd.fechada_em >= ` + pIni + `::timestamptz AND vd.fechada_em < ` + pFim + `::timestamptz`

	var res models.ResumoReceita
	err := r.db.QueryRowContext(ctx, query, c.Args()...).Scan(&res.Valor, &res.Vendas)
	return res, err
}

func (r *metricasVendasRepository) Conversao(ctx context.Context, f models.Filtros) (models.ResumoConversao, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim := c.Arg(inicio), c.Arg(fimAte)
	query := "WITH " + CTEsVendas(c, f) + `
	SELECT COUNT(*) FILTER (WHERE u.etapa = 'ganha'),
	       COUNT(*) FILTER (WHERE u.etapa = 'perdida')
	FROM op
	CROSS JOIN LATERAL ` + UltimoEventoOportunidade("op.id", pFim) + ` u
	WHERE u.etapa IN ('ganha', 'perdida') AND u.ocorrido_em >= ` + pIni + `::timestamptz`

	var res models.ResumoConversao
	err := r.db.QueryRowContext(ctx, query, c.Args()...).Scan(&res.Ganhas, &res.Perdidas)
	return res, err
}

func (r *metricasVendasRepository) Ciclo(ctx context.Context, f models.Filtros) (models.ResumoCiclo, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim := c.Arg(inicio), c.Arg(fimAte)
	query := "WITH " + CTEsVendas(c, f) + `
	SELECT AVG(EXTRACT(EPOCH FROM (u.ocorrido_em - op.criado_em)) / 86400.0)::float8, COUNT(*)
	FROM op
	CROSS JOIN LATERAL ` + UltimoEventoOportunidade("op.id", pFim) + ` u
	WHERE u.etapa = 'ganha' AND u.ocorrido_em >= ` + pIni + `::timestamptz`

	var res models.ResumoCiclo
	var media sql.NullFloat64
	if err := r.db.QueryRowContext(ctx, query, c.Args()...).Scan(&media, &res.Ganhas); err != nil {
		return res, err
	}
	if media.Valid {
		res.MediaDias = &media.Float64
	}
	return res, nil
}

func (r *metricasVendasRepository) Funil(ctx context.Context, f models.Filtros) ([]models.EtapaFunil, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim := c.Arg(inicio), c.Arg(fimAte)
	query := "WITH " + CTEsVendas(c, f) + `
	SELECT fe.etapa::text,
	       (SELECT COUNT(DISTINCT e.oportunidade_id) FROM oportunidade_eventos e
	        WHERE e.etapa = fe.etapa AND e.oportunidade_id IN (SELECT id FROM op)
	          AND e.ocorrido_em >= ` + pIni + `::timestamptz AND e.ocorrido_em < ` + pFim + `::timestamptz)
	FROM funil_etapas fe
	WHERE fe.etapa <> 'perdida'
	ORDER BY fe.ordem`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	etapas := []models.EtapaFunil{}
	for rows.Next() {
		var e models.EtapaFunil
		if err := rows.Scan(&e.Etapa, &e.Oportunidades); err != nil {
			return nil, err
		}
		etapas = append(etapas, e)
	}
	return etapas, rows.Err()
}
