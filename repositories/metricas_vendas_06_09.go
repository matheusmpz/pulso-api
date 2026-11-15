package repositories

import (
	"context"
	"database/sql"

	"pulso-api/models"
	"pulso-api/utils"
)

func (r *metricasVendasRepository) Pipeline(ctx context.Context, f models.Filtros) ([]models.EtapaPipeline, error) {
	c := &ConsultaSQL{}
	_, fimAte := utils.Intervalo(f.De, f.Ate)
	pFim := c.Arg(fimAte)
	// Oportunidades abertas na data ate (estado não final; sem evento = lead),
	// com o orçamento de maior versão emitido até ate cujo status atual não
	// seja recusada. orc já aplica tipo_projeto.
	query := "WITH " + CTEsVendas(c, f) + `,
	abertas AS (
		SELECT op.id, COALESCE(u.etapa, 'lead'::etapa_funil) AS etapa
		FROM op
		LEFT JOIN LATERAL ` + UltimoEventoOportunidade("op.id", pFim) + ` u ON true
		WHERE op.criado_em < ` + pFim + `::timestamptz
		  AND COALESCE(u.etapa, 'lead'::etapa_funil) NOT IN ('ganha', 'perdida')
	),
	ultima_versao AS (
		SELECT DISTINCT ON (orc.oportunidade_id) orc.id, orc.oportunidade_id, orc.valor
		FROM orc
		WHERE orc.emitido_em < ` + pFim + `::timestamptz
		ORDER BY orc.oportunidade_id, orc.versao DESC
	),
	vivas AS (
		SELECT a.etapa, v.valor
		FROM abertas a
		JOIN ultima_versao v ON v.oportunidade_id = a.id
		WHERE COALESCE((
			SELECT e.status FROM orcamento_eventos e
			WHERE e.orcamento_id = v.id AND e.ocorrido_em < ` + pFim + `::timestamptz
			ORDER BY e.ocorrido_em DESC, e.id DESC LIMIT 1), 'pendente'::status_orcamento) <> 'recusada'
	)
	SELECT v.etapa::text, SUM(v.valor)::float8, SUM(v.valor * fe.probabilidade)::float8
	FROM vivas v JOIN funil_etapas fe ON fe.etapa = v.etapa
	GROUP BY v.etapa, fe.ordem
	ORDER BY fe.ordem`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	etapas := []models.EtapaPipeline{}
	for rows.Next() {
		var e models.EtapaPipeline
		if err := rows.Scan(&e.Etapa, &e.Bruto, &e.Ponderado); err != nil {
			return nil, err
		}
		etapas = append(etapas, e)
	}
	return etapas, rows.Err()
}

func (r *metricasVendasRepository) MotivosPerda(ctx context.Context, f models.Filtros) ([]models.GrupoContagem, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim := c.Arg(inicio), c.Arg(fimAte)
	query := "WITH " + CTEsVendas(c, f) + `
	SELECT e.motivo_perda, COUNT(DISTINCT e.oportunidade_id)
	FROM oportunidade_eventos e
	WHERE e.etapa = 'perdida' AND e.oportunidade_id IN (SELECT id FROM op)
	  AND e.ocorrido_em >= ` + pIni + `::timestamptz AND e.ocorrido_em < ` + pFim + `::timestamptz
	GROUP BY e.motivo_perda
	ORDER BY 2 DESC, 1`
	return r.contagens(ctx, query, c.Args())
}

func (r *metricasVendasRepository) contagens(ctx context.Context, query string, args []interface{}) ([]models.GrupoContagem, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grupos := []models.GrupoContagem{}
	for rows.Next() {
		var g models.GrupoContagem
		if err := rows.Scan(&g.Rotulo, &g.N); err != nil {
			return nil, err
		}
		grupos = append(grupos, g)
	}
	return grupos, rows.Err()
}

func (r *metricasVendasRepository) OrigemLeads(ctx context.Context, f models.Filtros) ([]models.OrigemLeads, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim := c.Arg(inicio), c.Arg(fimAte)
	// Leads criados no período; desfecho = estado final até ate
	query := "WITH " + CTEsVendas(c, f) + `
	SELECT op.origem, COUNT(*),
	       COUNT(*) FILTER (WHERE u.etapa = 'ganha'),
	       COUNT(*) FILTER (WHERE u.etapa = 'perdida')
	FROM op
	LEFT JOIN LATERAL ` + UltimoEventoOportunidade("op.id", pFim) + ` u ON true
	WHERE op.criado_em >= ` + pIni + `::timestamptz AND op.criado_em < ` + pFim + `::timestamptz
	GROUP BY op.origem
	ORDER BY 2 DESC, 1`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	origens := []models.OrigemLeads{}
	for rows.Next() {
		var o models.OrigemLeads
		if err := rows.Scan(&o.Origem, &o.Leads, &o.Ganhas, &o.Perdidas); err != nil {
			return nil, err
		}
		origens = append(origens, o)
	}
	return origens, rows.Err()
}

func (r *metricasVendasRepository) ReunioesAteFechar(ctx context.Context, f models.Filtros) ([]models.ReunioesDesfecho, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim := c.Arg(inicio), c.Arg(fimAte)
	// Reuniões contadas por oportunidade antes de juntar (1 linha por oportunidade)
	query := "WITH " + CTEsVendas(c, f) + `,
	encerradas AS (
		SELECT op.id, u.etapa
		FROM op
		CROSS JOIN LATERAL ` + UltimoEventoOportunidade("op.id", pFim) + ` u
		WHERE u.etapa IN ('ganha', 'perdida') AND u.ocorrido_em >= ` + pIni + `::timestamptz
	),
	por_oportunidade AS (
		SELECT re.oportunidade_id, COUNT(*) AS n
		FROM reunioes re
		WHERE re.oportunidade_id IN (SELECT id FROM encerradas) AND re.realizada_em < ` + pFim + `::timestamptz
		GROUP BY re.oportunidade_id
	)
	SELECT d.desfecho,
	       (SELECT AVG(COALESCE(p.n, 0))::float8 FROM encerradas e LEFT JOIN por_oportunidade p ON p.oportunidade_id = e.id WHERE e.etapa::text = d.desfecho),
	       (SELECT COUNT(*) FROM encerradas e WHERE e.etapa::text = d.desfecho)
	FROM (VALUES ('ganha', 1), ('perdida', 2)) d(desfecho, ordem)
	ORDER BY d.ordem`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grupos := []models.ReunioesDesfecho{}
	for rows.Next() {
		var g models.ReunioesDesfecho
		var media sql.NullFloat64
		if err := rows.Scan(&g.Desfecho, &media, &g.N); err != nil {
			return nil, err
		}
		if media.Valid {
			g.Media = &media.Float64
		}
		grupos = append(grupos, g)
	}
	return grupos, rows.Err()
}
