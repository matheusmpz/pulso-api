package repositories

import (
	"context"
	"time"

	"github.com/lib/pq"

	"pulso-api/models"
	"pulso-api/utils"
)

// JanelaMediaDias é a janela da média móvel e da média diária (seção 12).
const JanelaMediaDias = 7

func (r *metricasIARepository) TokensPeriodo(ctx context.Context, f models.Filtros) (models.ResumoTokensPeriodo, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim := c.Arg(inicio), c.Arg(fimAte)
	query := "WITH " + CTEsEscopo(c, f) + `
	SELECT COALESCE(SUM(t.custo), 0)::float8,
	       COALESCE(SUM(t.tokens_entrada), 0)::bigint,
	       COALESCE(SUM(t.tokens_saida), 0)::bigint
	FROM t
	WHERE t.ocorrido_em >= ` + pIni + `::timestamptz AND t.ocorrido_em < ` + pFim + `::timestamptz`

	var res models.ResumoTokensPeriodo
	err := r.db.QueryRowContext(ctx, query, c.Args()...).Scan(&res.Custo, &res.Entrada, &res.Saida)
	return res, err
}

func (r *metricasIARepository) CustoAtividadesConcluidas(ctx context.Context, f models.Filtros) (models.ResumoCustoAtividades, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim := c.Arg(inicio), c.Arg(fimAte)
	pModelos := c.Arg(pq.Array(f.Modelo))
	// Atividades de A concluídas no período (status atual concluida e
	// último evento concluida no período). Custo: todo o consumo ligado a
	// elas até ate (de qualquer desenvolvedor), respeitando o filtro de modelo.
	query := "WITH " + CTEsEscopo(c, f) + `,
	concluidas AS (
		SELECT a.id FROM a
		WHERE ` + StatusAtividadeAte("a.id", pFim) + ` = 'concluida'
		  AND (SELECT MAX(e.ocorrido_em) FROM atividade_eventos e
		       WHERE e.atividade_id = a.id AND e.status = 'concluida' AND e.ocorrido_em < ` + pFim + `::timestamptz)
		      >= ` + pIni + `::timestamptz
	)
	SELECT (SELECT COUNT(*) FROM concluidas),
	       (SELECT COALESCE(SUM(ct.custo), 0)::float8 FROM consumo_tokens ct
	        WHERE ct.atividade_id IN (SELECT id FROM concluidas)
	          AND ct.ocorrido_em < ` + pFim + `::timestamptz
	          AND (` + pModelos + `::text[] IS NULL OR ct.modelo = ANY(` + pModelos + `::text[])))`

	var res models.ResumoCustoAtividades
	err := r.db.QueryRowContext(ctx, query, c.Args()...).Scan(&res.Atividades, &res.Custo)
	return res, err
}

func (r *metricasIARepository) ConsumoDiario(ctx context.Context, f models.Filtros) (models.ConsumoDiario, error) {
	c := &ConsultaSQL{}
	_, fimAte := utils.Intervalo(f.De, f.Ate)
	// Começa JanelaMediaDias − 1 dias antes de de, para a média móvel do
	// primeiro dia do período usar dias anteriores
	desde := utils.SomarDias(f.De, -(JanelaMediaDias - 1))
	pDesde, pAte, pFim, pTZ := c.Arg(desde.String()), c.Arg(f.Ate.String()), c.Arg(fimAte), c.Arg(utils.FusoApp().String())
	query := "WITH " + CTEsEscopo(c, f) + `,
	por_dia AS (
		SELECT (t.ocorrido_em AT TIME ZONE ` + pTZ + `::text)::date AS dia,
		       SUM(t.tokens_entrada::bigint + t.tokens_saida) AS tokens
		FROM t
		WHERE t.ocorrido_em >= (` + pDesde + `::date::timestamp AT TIME ZONE ` + pTZ + `::text)
		  AND t.ocorrido_em < ` + pFim + `::timestamptz
		GROUP BY 1
	),
	dias AS (
		SELECT g::date AS dia FROM generate_series(` + pDesde + `::date, ` + pAte + `::date, INTERVAL '1 day') g
	)
	SELECT d.dia, COALESCE(p.tokens, 0)::bigint,
	       (SELECT COALESCE(SUM(t.tokens_entrada::bigint + t.tokens_saida), 0)::bigint FROM t WHERE t.ocorrido_em < ` + pFim + `::timestamptz),
	       (SELECT valor FROM quota)
	FROM dias d LEFT JOIN por_dia p ON p.dia = d.dia
	ORDER BY d.dia`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return models.ConsumoDiario{}, err
	}
	defer rows.Close()

	res := models.ConsumoDiario{Dias: []models.DiaTokens{}}
	for rows.Next() {
		var dia time.Time
		var d models.DiaTokens
		if err := rows.Scan(&dia, &d.Tokens, &res.TokensAcumulados, &res.Quota); err != nil {
			return models.ConsumoDiario{}, err
		}
		d.Dia = utils.DataDoBanco(dia)
		res.Dias = append(res.Dias, d)
	}
	return res, rows.Err()
}
