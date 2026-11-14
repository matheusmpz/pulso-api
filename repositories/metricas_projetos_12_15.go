package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"pulso-api/models"
	"pulso-api/utils"
)

// Quantos itens os cards mostram.
const (
	limiteTopEstouros      = 5  // proj-12
	limiteSprintsHistorico = 6  // proj-13
	LimiteTopHoras         = 10 // proj-14 (o resto vira "Outros" no calculador)
)

func (r *metricasProjetosRepository) Estouros(ctx context.Context, f models.Filtros) (models.Estouros, error) {
	c := &ConsultaSQL{}
	pAte := c.Arg(f.Ate.String())
	// Horas agregadas por atividade antes de juntar com a atividade
	query := "WITH " + CTEsEscopo(c, f) + `,
	reais AS (
		SELECT h.atividade_id, SUM(h.horas) AS horas
		FROM h WHERE h.trabalhado_em <= ` + pAte + `::date
		GROUP BY h.atividade_id
	),
	estouradas AS (
		SELECT a.titulo, pr.nome AS projeto, a.estimativa_horas, r.horas
		FROM a
		JOIN reais r ON r.atividade_id = a.id
		JOIN projetos pr ON pr.id = a.projeto_id
		WHERE r.horas > a.estimativa_horas
	)
	SELECT titulo, projeto, estimativa_horas::float8, horas::float8, COUNT(*) OVER ()::int
	FROM estouradas
	ORDER BY horas - estimativa_horas DESC, titulo
	LIMIT ` + fmt.Sprint(limiteTopEstouros)

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return models.Estouros{}, err
	}
	defer rows.Close()

	e := models.Estouros{Top: []models.AtividadeEstourada{}}
	for rows.Next() {
		var a models.AtividadeEstourada
		if err := rows.Scan(&a.Titulo, &a.Projeto, &a.Estimativa, &a.Reais, &e.Total); err != nil {
			return models.Estouros{}, err
		}
		e.Top = append(e.Top, a)
	}
	return e, rows.Err()
}

func (r *metricasProjetosRepository) HistoricoSprints(ctx context.Context, f models.Filtros) ([]models.SprintHistorico, error) {
	c := &ConsultaSQL{}
	_, fimAte := utils.Intervalo(f.De, f.Ate)
	pAte, pFim, pTZ := c.Arg(f.Ate.String()), c.Arg(fimAte), c.Arg(utils.FusoApp().String())
	var id interface{}
	if f.ID != nil {
		id = *f.ID
	}
	pID := c.Arg(id)
	pEscopo := c.Arg(f.Escopo)

	// Sprints candidatas e atividades consideradas, por escopo (trechos fixos)
	var candidatas, atividades string
	switch f.Escopo {
	case models.EscopoSprint:
		// Sprints do mesmo time da sprint id, terminando nela. Em
		// escopo=sprint, A só tem as atividades daquela sprint; o
		// histórico usa as atividades das sprints listadas, em projetos
		// que passam nos filtros de tipo e status.
		candidatas = `SELECT s.* FROM sprints s
			WHERE s.time_id = (SELECT time_id FROM sprints WHERE id = ` + pID + `::bigint)
			  AND s.inicio <= (SELECT inicio FROM sprints WHERE id = ` + pID + `::bigint)`
		atividades = `SELECT at.id, at.sprint_id, at.estimativa_horas FROM atividades at
			WHERE at.projeto_id IN (SELECT id FROM pf) AND at.criado_em < ` + pFim + `::timestamptz`
	case models.EscopoTime:
		// Sprints do time (de todos os times, se vazio); atividades de A
		candidatas = `SELECT s.* FROM sprints s
			WHERE (` + pID + `::bigint IS NULL OR s.time_id = ` + pID + `::bigint)`
		atividades = `SELECT a.id, a.sprint_id, a.estimativa_horas FROM a`
	default:
		// Sprints com atividades de A (do projeto, ou de todos os de P)
		candidatas = `SELECT s.* FROM sprints s WHERE s.id IN (SELECT sprint_id FROM a)`
		atividades = `SELECT a.id, a.sprint_id, a.estimativa_horas FROM a`
	}

	fimAvaliacao := "LEAST(" + fimDoDiaSQL("s.fim", pTZ) + ", " + pFim + "::timestamptz)"
	query := "WITH " + CTEsEscopo(c, f) + `, ` + CTEProjetosFiltrados(c, f) + `,
	candidatas AS (` + candidatas + `),
	ultimas AS (
		SELECT * FROM candidatas WHERE inicio <= ` + pAte + `::date
		ORDER BY inicio DESC, id DESC LIMIT ` + fmt.Sprint(limiteSprintsHistorico) + `
	),
	ativ AS (` + atividades + `)
	SELECT s.id, s.time_id, t.nome, s.numero, s.inicio, s.fim,
	       (SELECT COALESCE(SUM(x.estimativa_horas), 0)::float8 FROM ativ x WHERE x.sprint_id = s.id),
	       (SELECT COALESCE(SUM(x.estimativa_horas), 0)::float8 FROM ativ x
	        WHERE x.sprint_id = s.id AND ` + StatusAtividadeAte("x.id", fimAvaliacao) + ` = 'concluida'),
	       COALESCE(` + pEscopo + `::text = 'sprint' AND s.id = ` + pID + `::bigint, false)
	FROM ultimas s JOIN times t ON t.id = s.time_id
	ORDER BY s.inicio, s.id`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	historico := []models.SprintHistorico{}
	for rows.Next() {
		var h models.SprintHistorico
		var inicio, fim time.Time
		if err := rows.Scan(&h.Sprint.ID, &h.Sprint.TimeID, &h.Sprint.TimeNome, &h.Sprint.Numero, &inicio, &fim,
			&h.Planejado, &h.Concluido, &h.Destaque); err != nil {
			return nil, err
		}
		h.Sprint.Inicio, h.Sprint.Fim = utils.DataDoBanco(inicio), utils.DataDoBanco(fim)
		historico = append(historico, h)
	}
	return historico, rows.Err()
}

func (r *metricasProjetosRepository) DistribuicaoHoras(ctx context.Context, f models.Filtros) ([]models.GrupoHoras, error) {
	c := &ConsultaSQL{}
	pDe, pAte := c.Arg(f.De.String()), c.Arg(f.Ate.String())
	periodo := `h.trabalhado_em BETWEEN ` + pDe + `::date AND ` + pAte + `::date`

	// escopo=projeto → por desenvolvedor; time ou sprint → por projeto
	agrupar := `SELECT d.nome AS rotulo, SUM(h.horas) AS horas
		FROM h JOIN desenvolvedores d ON d.id = h.desenvolvedor_id
		WHERE ` + periodo + `
		GROUP BY d.id, d.nome`
	if f.Escopo == models.EscopoTime || f.Escopo == models.EscopoSprint {
		agrupar = `SELECT pr.nome AS rotulo, SUM(h.horas) AS horas
			FROM h JOIN projetos pr ON pr.id = h.projeto_id
			WHERE ` + periodo + `
			GROUP BY pr.id, pr.nome`
	}

	query := "WITH " + CTEsEscopo(c, f) + `, grupos AS (` + agrupar + `)
	SELECT rotulo, horas::float8 FROM grupos ORDER BY horas DESC, rotulo`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grupos := []models.GrupoHoras{}
	for rows.Next() {
		var g models.GrupoHoras
		if err := rows.Scan(&g.Rotulo, &g.Horas); err != nil {
			return nil, err
		}
		grupos = append(grupos, g)
	}
	return grupos, rows.Err()
}

func (r *metricasIARepository) ConsumoNoTempo(ctx context.Context, f models.Filtros) (models.ConsumoNoTempo, error) {
	c := &ConsultaSQL{}
	_, fimAte := utils.Intervalo(f.De, f.Ate)
	pAte, pFim, pTZ := c.Arg(f.Ate.String()), c.Arg(fimAte), c.Arg(utils.FusoApp().String())
	query := "WITH " + CTEsEscopo(c, f) + `,
	por_dia AS (
		SELECT (t.ocorrido_em AT TIME ZONE ` + pTZ + `::text)::date AS dia,
		       SUM(t.tokens_entrada::bigint + t.tokens_saida) AS tokens
		FROM t WHERE t.ocorrido_em < ` + pFim + `::timestamptz
		GROUP BY 1
	),
	dias AS (
		SELECT g::date AS dia
		FROM generate_series((SELECT MIN(dia) FROM por_dia), ` + pAte + `::date, INTERVAL '1 day') g
	)
	SELECT d.dia,
	       (SUM(COALESCE(p.tokens, 0)) OVER (ORDER BY d.dia))::bigint,
	       (SELECT valor FROM quota)
	FROM dias d LEFT JOIN por_dia p ON p.dia = d.dia
	ORDER BY d.dia`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return models.ConsumoNoTempo{}, err
	}
	defer rows.Close()

	res := models.ConsumoNoTempo{Dias: []models.DiaConsumo{}}
	for rows.Next() {
		var dia time.Time
		var d models.DiaConsumo
		if err := rows.Scan(&dia, &d.TokensAcumulados, &res.Quota); err != nil {
			return models.ConsumoNoTempo{}, err
		}
		d.Dia = utils.DataDoBanco(dia)
		res.Dias = append(res.Dias, d)
	}
	if err := rows.Err(); err != nil {
		return models.ConsumoNoTempo{}, err
	}

	// Sem consumo, a quota ainda vale para a linha de referência
	if len(res.Dias) == 0 {
		c2 := &ConsultaSQL{}
		q2 := "WITH " + CTEsEscopo(c2, f) + ` SELECT valor FROM quota`
		if err := r.db.QueryRowContext(ctx, q2, c2.Args()...).Scan(&res.Quota); err != nil && err != sql.ErrNoRows {
			return models.ConsumoNoTempo{}, err
		}
	}
	return res, nil
}
