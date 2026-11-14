package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"pulso-api/models"
	"pulso-api/utils"
)

// MetricasProjetosRepository calcula os números dos cards do dashboard
// projetos (só leitura). Cada tabela é agregada separadamente; nada de
// JOIN entre tabelas de evento para somar.
type MetricasProjetosRepository interface {
	// Progresso: estimativa das atividades de A concluídas até o fim do
	// dia ate, sobre a estimativa total de A (proj-01).
	Progresso(ctx context.Context, f models.Filtros) (models.ResumoProgresso, error)
	// Horas: horas de H lançadas até ate e estimativa total de A (proj-02).
	Horas(ctx context.Context, f models.Filtros) (models.ResumoHoras, error)
	// BurnUp: por dia, do primeiro dia com horas ou eventos de A até ate,
	// as horas acumuladas e a estimativa concluída acumulada (proj-04).
	BurnUp(ctx context.Context, f models.Filtros) (models.BurnUp, error)
	// ContagemStatus: atividades de A por status atual ao fim de ate (proj-05).
	ContagemStatus(ctx context.Context, f models.Filtros) (models.ContagemStatus, error)
	// Burndown: sprint alvo e, por dia dela, a estimativa ainda não
	// concluída (proj-06). hoje decide a sprint ativa.
	Burndown(ctx context.Context, f models.Filtros, hoje models.Data) (models.Burndown, error)
	// HorasEtapa: estimativa da etapa em A e horas de H na etapa até ate
	// (proj-07 análise, proj-08 desenvolvimento, proj-09 testes).
	HorasEtapa(ctx context.Context, f models.Filtros, etapa string) (models.HorasEtapa, error)
	// TempoMedio: atividades de A concluídas no período (status atual
	// concluida e último evento concluida entre de e ate), agrupadas por
	// "complexidade" (proj-10) ou "tipo" (proj-11).
	TempoMedio(ctx context.Context, f models.Filtros, agrupamento string) ([]models.GrupoTempoMedio, error)
}

type metricasProjetosRepository struct {
	db *sql.DB
}

// NewMetricasProjetosRepository cria o repository de métricas de projetos.
func NewMetricasProjetosRepository(db *sql.DB) MetricasProjetosRepository {
	return &metricasProjetosRepository{db}
}

func (r *metricasProjetosRepository) Progresso(ctx context.Context, f models.Filtros) (models.ResumoProgresso, error) {
	c := &ConsultaSQL{}
	_, fimAte := utils.Intervalo(f.De, f.Ate)
	pFim := c.Arg(fimAte)
	query := "WITH " + CTEsEscopo(c, f) + `,
	situacao AS (
		SELECT a.estimativa_horas, ` + StatusAtividadeAte("a.id", pFim) + ` AS status
		FROM a
	)
	SELECT COALESCE(SUM(estimativa_horas) FILTER (WHERE status = 'concluida'), 0)::float8,
	       COALESCE(SUM(estimativa_horas), 0)::float8,
	       COUNT(*) FILTER (WHERE status = 'concluida'),
	       COUNT(*)
	FROM situacao`

	var p models.ResumoProgresso
	err := r.db.QueryRowContext(ctx, query, c.Args()...).Scan(&p.EstimativaConcluida, &p.EstimativaTotal, &p.Concluidas, &p.Total)
	return p, err
}

func (r *metricasProjetosRepository) Horas(ctx context.Context, f models.Filtros) (models.ResumoHoras, error) {
	c := &ConsultaSQL{}
	pAte := c.Arg(f.Ate.String())
	// Dois agregados independentes (h e a), cada um na sua subconsulta
	query := "WITH " + CTEsEscopo(c, f) + `
	SELECT (SELECT COALESCE(SUM(h.horas), 0)::float8 FROM h WHERE h.trabalhado_em <= ` + pAte + `::date),
	       (SELECT COALESCE(SUM(a.estimativa_horas), 0)::float8 FROM a)`

	var h models.ResumoHoras
	err := r.db.QueryRowContext(ctx, query, c.Args()...).Scan(&h.Reais, &h.Estimadas)
	return h, err
}

// fimDoDiaSQL devolve a expressão do início do dia seguinte a uma coluna
// DATE, no fuso da aplicação (pTZ é o placeholder do nome do fuso).
func fimDoDiaSQL(colunaDia, pTZ string) string {
	return fmt.Sprintf("((%s + 1)::timestamp AT TIME ZONE %s::text)", colunaDia, pTZ)
}

func (r *metricasProjetosRepository) BurnUp(ctx context.Context, f models.Filtros) (models.BurnUp, error) {
	c := &ConsultaSQL{}
	_, fimAte := utils.Intervalo(f.De, f.Ate)
	pAte, pFim, pTZ := c.Arg(f.Ate.String()), c.Arg(fimAte), c.Arg(utils.FusoApp().String())
	query := "WITH " + CTEsEscopo(c, f) + `,
	primeiro AS (
		SELECT LEAST(
			(SELECT MIN(h.trabalhado_em) FROM h WHERE h.trabalhado_em <= ` + pAte + `::date),
			(SELECT MIN((e.ocorrido_em AT TIME ZONE ` + pTZ + `::text)::date)
			 FROM atividade_eventos e
			 WHERE e.atividade_id IN (SELECT id FROM a) AND e.ocorrido_em < ` + pFim + `::timestamptz)
		) AS dia
	),
	dias AS (
		SELECT g::date AS dia
		FROM generate_series((SELECT dia FROM primeiro), ` + pAte + `::date, INTERVAL '1 day') g
	)
	SELECT d.dia,
	       (SELECT COALESCE(SUM(h.horas), 0)::float8 FROM h WHERE h.trabalhado_em <= d.dia),
	       (SELECT COALESCE(SUM(a.estimativa_horas), 0)::float8 FROM a
	        WHERE ` + StatusAtividadeAte("a.id", fimDoDiaSQL("d.dia", pTZ)) + ` = 'concluida'),
	       (SELECT COALESCE(SUM(a.estimativa_horas), 0)::float8 FROM a)
	FROM dias d
	ORDER BY d.dia`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return models.BurnUp{}, err
	}
	defer rows.Close()

	b := models.BurnUp{Dias: []models.DiaBurnUp{}}
	for rows.Next() {
		var dia time.Time
		var d models.DiaBurnUp
		if err := rows.Scan(&dia, &d.HorasAcumuladas, &d.EstimativaConcluida, &b.EstimativaTotal); err != nil {
			return models.BurnUp{}, err
		}
		d.Dia = utils.DataDoBanco(dia)
		b.Dias = append(b.Dias, d)
	}
	if err := rows.Err(); err != nil {
		return models.BurnUp{}, err
	}

	// Sem nenhum dia (sem horas nem eventos), a estimativa total ainda conta
	if len(b.Dias) == 0 {
		c2 := &ConsultaSQL{}
		q2 := "WITH " + CTEsEscopo(c2, f) + ` SELECT COALESCE(SUM(estimativa_horas), 0)::float8 FROM a`
		if err := r.db.QueryRowContext(ctx, q2, c2.Args()...).Scan(&b.EstimativaTotal); err != nil {
			return models.BurnUp{}, err
		}
	}
	return b, nil
}

func (r *metricasProjetosRepository) ContagemStatus(ctx context.Context, f models.Filtros) (models.ContagemStatus, error) {
	c := &ConsultaSQL{}
	_, fimAte := utils.Intervalo(f.De, f.Ate)
	pFim := c.Arg(fimAte)
	query := "WITH " + CTEsEscopo(c, f) + `,
	situacao AS (SELECT ` + StatusAtividadeAte("a.id", pFim) + ` AS status FROM a)
	SELECT COUNT(*) FILTER (WHERE status = 'aberta'),
	       COUNT(*) FILTER (WHERE status = 'em_andamento'),
	       COUNT(*) FILTER (WHERE status = 'concluida')
	FROM situacao`

	var s models.ContagemStatus
	err := r.db.QueryRowContext(ctx, query, c.Args()...).Scan(&s.Abertas, &s.EmAndamento, &s.Concluidas)
	return s, err
}

func (r *metricasProjetosRepository) Burndown(ctx context.Context, f models.Filtros, hoje models.Data) (models.Burndown, error) {
	b := models.Burndown{Dias: []models.DiaBurndown{}}

	// 1. Sprint alvo
	c := &ConsultaSQL{}
	var alvo string
	if f.Escopo == models.EscopoSprint && f.ID != nil {
		alvo = `SELECT s.id FROM sprints s WHERE s.id = ` + c.Arg(*f.ID)
	} else {
		// Ativa hoje com mais atividades de A; senão, a mais recente já
		// iniciada que tenha atividades de A
		pHoje := c.Arg(hoje.String())
		alvo = `SELECT s.id
			FROM sprints s
			JOIN (SELECT sprint_id, COUNT(*) AS n FROM a WHERE sprint_id IS NOT NULL GROUP BY sprint_id) x
			  ON x.sprint_id = s.id
			WHERE s.inicio <= ` + pHoje + `::date
			ORDER BY (` + pHoje + `::date BETWEEN s.inicio AND s.fim) DESC, x.n DESC, s.inicio DESC, s.id
			LIMIT 1`
	}
	query := "WITH " + CTEsEscopo(c, f) + `, alvo AS (` + alvo + `)
	SELECT s.id, s.time_id, t.nome, s.numero, s.inicio, s.fim
	FROM sprints s JOIN times t ON t.id = s.time_id
	WHERE s.id = (SELECT id FROM alvo)`

	sprint, err := escanearSprint(r.db.QueryRowContext(ctx, query, c.Args()...))
	if errors.Is(err, sql.ErrNoRows) {
		return b, nil // nenhuma sprint alvo: dados vazios
	}
	if err != nil {
		return b, err
	}
	b.Sprint = &sprint

	// 2. Estimativa restante por dia da sprint
	c = &ConsultaSQL{}
	pSprint, pTZ := c.Arg(sprint.ID), c.Arg(utils.FusoApp().String())
	pIni, pFimS := c.Arg(sprint.Inicio.String()), c.Arg(sprint.Fim.String())
	query = "WITH " + CTEsEscopo(c, f) + `,
	da_sprint AS (SELECT a.id, a.estimativa_horas FROM a WHERE a.sprint_id = ` + pSprint + `::bigint),
	dias AS (SELECT g::date AS dia FROM generate_series(` + pIni + `::date, ` + pFimS + `::date, INTERVAL '1 day') g)
	SELECT d.dia,
	       (SELECT COALESCE(SUM(s.estimativa_horas), 0)::float8 FROM da_sprint s
	        WHERE ` + StatusAtividadeAte("s.id", fimDoDiaSQL("d.dia", pTZ)) + ` <> 'concluida'),
	       (SELECT COALESCE(SUM(estimativa_horas), 0)::float8 FROM da_sprint)
	FROM dias d
	ORDER BY d.dia`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return b, err
	}
	defer rows.Close()
	for rows.Next() {
		var dia time.Time
		var d models.DiaBurndown
		if err := rows.Scan(&dia, &d.Restante, &b.Total); err != nil {
			return b, err
		}
		d.Dia = utils.DataDoBanco(dia)
		b.Dias = append(b.Dias, d)
	}
	return b, rows.Err()
}

// colunaEstimativaPorEtapa é a whitelist das colunas de estimativa por
// etapa (o nome da coluna nunca vem do usuário).
var colunaEstimativaPorEtapa = map[string]string{
	"analise":         "estimativa_analise",
	"desenvolvimento": "estimativa_desenvolvimento",
	"testes":          "estimativa_testes",
}

func (r *metricasProjetosRepository) HorasEtapa(ctx context.Context, f models.Filtros, etapa string) (models.HorasEtapa, error) {
	coluna, ok := colunaEstimativaPorEtapa[etapa]
	if !ok {
		return models.HorasEtapa{}, fmt.Errorf("etapa desconhecida: %q", etapa)
	}
	c := &ConsultaSQL{}
	pEtapa, pAte := c.Arg(etapa), c.Arg(f.Ate.String())
	query := "WITH " + CTEsEscopo(c, f) + `
	SELECT (SELECT COALESCE(SUM(a.` + coluna + `), 0)::float8 FROM a),
	       (SELECT COALESCE(SUM(h.horas), 0)::float8 FROM h
	        WHERE h.etapa = ` + pEtapa + `::etapa_trabalho AND h.trabalhado_em <= ` + pAte + `::date)`

	var h models.HorasEtapa
	err := r.db.QueryRowContext(ctx, query, c.Args()...).Scan(&h.Estimado, &h.Real)
	return h, err
}

// colunaAgrupamentoTempo é a whitelist dos agrupamentos do tempo médio.
var colunaAgrupamentoTempo = map[string]string{
	"complexidade": "complexidade",
	"tipo":         "tipo",
}

func (r *metricasProjetosRepository) TempoMedio(ctx context.Context, f models.Filtros, agrupamento string) ([]models.GrupoTempoMedio, error) {
	coluna, ok := colunaAgrupamentoTempo[agrupamento]
	if !ok {
		return nil, fmt.Errorf("agrupamento desconhecido: %q", agrupamento)
	}
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim, pAte := c.Arg(inicio), c.Arg(fimAte), c.Arg(f.Ate.String())
	query := "WITH " + CTEsEscopo(c, f) + `,
	base AS (
		SELECT a.id, a.` + coluna + `::text AS grupo, a.estimativa_horas,
		       ` + StatusAtividadeAte("a.id", pFim) + ` AS status,
		       (SELECT MAX(e.ocorrido_em) FROM atividade_eventos e
		        WHERE e.atividade_id = a.id AND e.status = 'concluida'
		          AND e.ocorrido_em < ` + pFim + `::timestamptz) AS concluida_em
		FROM a
	),
	concluidas AS (
		SELECT * FROM base
		WHERE status = 'concluida'
		  AND concluida_em >= ` + pIni + `::timestamptz AND concluida_em < ` + pFim + `::timestamptz
	),
	-- horas agregadas por atividade antes de juntar (1 linha por atividade)
	horas AS (
		SELECT h.atividade_id, SUM(h.horas) AS total
		FROM h
		WHERE h.atividade_id IN (SELECT id FROM concluidas) AND h.trabalhado_em <= ` + pAte + `::date
		GROUP BY h.atividade_id
	)
	SELECT c.grupo, COUNT(*), AVG(COALESCE(h.total, 0))::float8, AVG(c.estimativa_horas)::float8
	FROM concluidas c
	LEFT JOIN horas h ON h.atividade_id = c.id
	GROUP BY c.grupo`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grupos := []models.GrupoTempoMedio{}
	for rows.Next() {
		var g models.GrupoTempoMedio
		if err := rows.Scan(&g.Grupo, &g.N, &g.MediaReal, &g.MediaEstimada); err != nil {
			return nil, err
		}
		grupos = append(grupos, g)
	}
	return grupos, rows.Err()
}
