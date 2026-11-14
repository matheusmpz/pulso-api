package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"

	"pulso-api/models"
	"pulso-api/utils"
)

func (r *metricasIARepository) IntensidadeDesempenho(ctx context.Context, f models.Filtros) ([]models.FaixaIntensidade, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim, pAte := c.Arg(inicio), c.Arg(fimAte), c.Arg(f.Ate.String())
	pModelos := c.Arg(pq.Array(f.Modelo))
	// Base: atividades de A concluídas no período com tokens > 0. Tokens e
	// horas são agregados por atividade antes de juntar.
	query := "WITH " + CTEsEscopo(c, f) + `,
	concluidas AS (
		SELECT a.id, a.complexidade, a.estimativa_horas FROM a
		WHERE ` + StatusAtividadeAte("a.id", pFim) + ` = 'concluida'
		  AND (SELECT MAX(e.ocorrido_em) FROM atividade_eventos e
		       WHERE e.atividade_id = a.id AND e.status = 'concluida' AND e.ocorrido_em < ` + pFim + `::timestamptz)
		      >= ` + pIni + `::timestamptz
	),
	tokens AS (
		SELECT ct.atividade_id, SUM(ct.tokens_entrada::bigint + ct.tokens_saida) AS tokens
		FROM consumo_tokens ct
		WHERE ct.atividade_id IN (SELECT id FROM concluidas) AND ct.ocorrido_em < ` + pFim + `::timestamptz
		  AND (` + pModelos + `::text[] IS NULL OR ct.modelo = ANY(` + pModelos + `::text[]))
		GROUP BY ct.atividade_id
	),
	horas AS (
		SELECT rh.atividade_id, SUM(rh.horas) AS horas
		FROM registros_horas rh
		WHERE rh.atividade_id IN (SELECT id FROM concluidas) AND rh.trabalhado_em <= ` + pAte + `::date
		GROUP BY rh.atividade_id
	),
	base AS (
		SELECT c.id, c.complexidade,
		       tk.tokens / c.estimativa_horas AS intensidade,
		       COALESCE(h.horas, 0) / c.estimativa_horas * 100 AS desvio
		FROM concluidas c
		JOIN tokens tk ON tk.atividade_id = c.id AND tk.tokens > 0
		LEFT JOIN horas h ON h.atividade_id = c.id
	),
	faixas AS (
		SELECT *, NTILE(3) OVER (PARTITION BY complexidade ORDER BY intensidade, id) AS faixa FROM base
	)
	SELECT complexidade::text, faixa, AVG(desvio)::float8, COUNT(*)
	FROM faixas
	GROUP BY complexidade, faixa`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	faixas := []models.FaixaIntensidade{}
	for rows.Next() {
		var x models.FaixaIntensidade
		if err := rows.Scan(&x.Complexidade, &x.Faixa, &x.DesvioMedio, &x.N); err != nil {
			return nil, err
		}
		faixas = append(faixas, x)
	}
	return faixas, rows.Err()
}

func (r *metricasIARepository) PesoIA(ctx context.Context, f models.Filtros) ([]models.PesoIAProjeto, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim := c.Arg(inicio), c.Arg(fimAte)
	pDe, pAte := c.Arg(f.De.String()), c.Arg(f.Ate.String())
	// Custo das horas (H) e de IA (T) no período, agregados por projeto
	// separadamente e só então juntados
	query := "WITH " + CTEsEscopo(c, f) + `,
	custo_horas AS (
		SELECT h.projeto_id, SUM(h.horas * h.custo_hora) AS custo
		FROM h WHERE h.trabalhado_em BETWEEN ` + pDe + `::date AND ` + pAte + `::date
		GROUP BY h.projeto_id
	),
	custo_ia AS (
		SELECT t.projeto_id, SUM(t.custo) AS custo
		FROM t WHERE t.ocorrido_em >= ` + pIni + `::timestamptz AND t.ocorrido_em < ` + pFim + `::timestamptz
		GROUP BY t.projeto_id
	)
	SELECT pr.nome, COALESCE(ch.custo, 0)::float8, COALESCE(ci.custo, 0)::float8
	FROM projetos pr
	LEFT JOIN custo_horas ch ON ch.projeto_id = pr.id
	LEFT JOIN custo_ia ci ON ci.projeto_id = pr.id
	WHERE pr.id IN (SELECT id FROM p) AND (ch.custo IS NOT NULL OR ci.custo IS NOT NULL)
	ORDER BY COALESCE(ch.custo, 0) + COALESCE(ci.custo, 0) DESC, pr.nome`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projetos := []models.PesoIAProjeto{}
	for rows.Next() {
		var x models.PesoIAProjeto
		if err := rows.Scan(&x.Projeto, &x.CustoHoras, &x.CustoIA); err != nil {
			return nil, err
		}
		projetos = append(projetos, x)
	}
	return projetos, rows.Err()
}

func (r *metricasIARepository) AlertasQuota(ctx context.Context, f models.Filtros, limiar float64) ([]models.AlertaQuota, error) {
	c := &ConsultaSQL{}
	_, fimAte := utils.Intervalo(f.De, f.Ate)
	pFim, pTZ, pLimiar := c.Arg(fimAte), c.Arg(utils.FusoApp().String()), c.Arg(limiar)
	// Por projeto de P: consumo de T acumulado por dia até ate; alerta se o
	// total passar do limiar, com o primeiro dia em que passou
	query := "WITH " + CTEsEscopo(c, f) + `,
	por_dia AS (
		SELECT t.projeto_id, (t.ocorrido_em AT TIME ZONE ` + pTZ + `::text)::date AS dia,
		       SUM(t.tokens_entrada::bigint + t.tokens_saida) AS tokens
		FROM t WHERE t.ocorrido_em < ` + pFim + `::timestamptz
		GROUP BY 1, 2
	),
	acumulado AS (
		SELECT projeto_id, dia, SUM(tokens) OVER (PARTITION BY projeto_id ORDER BY dia) AS acum FROM por_dia
	)
	SELECT pr.nome,
	       (MAX(ac.acum) * 100.0 / pr.limite_tokens)::float8,
	       MIN(ac.dia) FILTER (WHERE ac.acum * 100.0 / pr.limite_tokens >= ` + pLimiar + `::float8)
	FROM acumulado ac JOIN projetos pr ON pr.id = ac.projeto_id
	WHERE pr.id IN (SELECT id FROM p)
	GROUP BY pr.id, pr.nome, pr.limite_tokens
	HAVING MAX(ac.acum) * 100.0 / pr.limite_tokens >= ` + pLimiar + `::float8`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alertas := []models.AlertaQuota{}
	for rows.Next() {
		var a models.AlertaQuota
		var quando time.Time
		if err := rows.Scan(&a.Projeto, &a.Percentual, &quando); err != nil {
			return nil, err
		}
		a.Quando = utils.DataDoBanco(quando)
		alertas = append(alertas, a)
	}
	return alertas, rows.Err()
}

func (r *metricasIARepository) AlertasPico(ctx context.Context, f models.Filtros, fator float64, janela int) ([]models.AlertaPico, error) {
	c := &ConsultaSQL{}
	_, fimAte := utils.Intervalo(f.De, f.Ate)
	desde := utils.SomarDias(f.De, -janela)
	pDesde, pDe, pAte := c.Arg(desde.String()), c.Arg(f.De.String()), c.Arg(f.Ate.String())
	pFim, pTZ, pFator, pJanela := c.Arg(fimAte), c.Arg(utils.FusoApp().String()), c.Arg(fator), c.Arg(janela)
	// Grade projeto × dia (com zeros) desde "janela" dias antes de de, para
	// a média dos dias anteriores valer já no primeiro dia do período
	query := "WITH " + CTEsEscopo(c, f) + `,
	dias AS (SELECT g::date AS dia FROM generate_series(` + pDesde + `::date, ` + pAte + `::date, INTERVAL '1 day') g),
	por_dia AS (
		SELECT t.projeto_id, (t.ocorrido_em AT TIME ZONE ` + pTZ + `::text)::date AS dia,
		       SUM(t.tokens_entrada::bigint + t.tokens_saida) AS tokens
		FROM t
		WHERE t.ocorrido_em >= (` + pDesde + `::date::timestamp AT TIME ZONE ` + pTZ + `::text)
		  AND t.ocorrido_em < ` + pFim + `::timestamptz
		GROUP BY 1, 2
	),
	grade AS (
		SELECT p.id AS projeto_id, d.dia, COALESCE(pd.tokens, 0) AS tokens
		FROM p CROSS JOIN dias d
		LEFT JOIN por_dia pd ON pd.projeto_id = p.id AND pd.dia = d.dia
	),
	medias AS (
		-- soma dos "janela" dias anteriores (o frame não aceita parâmetro;
		-- janela é um int do código, nunca entrada do usuário)
		SELECT projeto_id, dia, tokens,
		       COALESCE(SUM(tokens) OVER (PARTITION BY projeto_id ORDER BY dia
		                ROWS BETWEEN ` + fmt.Sprint(janela) + ` PRECEDING AND 1 PRECEDING), 0) AS soma_janela
		FROM grade
	)
	SELECT pr.nome, m.dia, (m.tokens / (m.soma_janela / ` + pJanela + `::float8))::float8
	FROM medias m JOIN projetos pr ON pr.id = m.projeto_id
	WHERE m.dia >= ` + pDe + `::date
	  AND m.soma_janela > 0
	  AND m.tokens > ` + pFator + `::float8 * (m.soma_janela / ` + pJanela + `::float8)`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	picos := []models.AlertaPico{}
	for rows.Next() {
		var p models.AlertaPico
		var dia time.Time
		if err := rows.Scan(&p.Projeto, &dia, &p.Razao); err != nil {
			return nil, err
		}
		p.Dia = utils.DataDoBanco(dia)
		picos = append(picos, p)
	}
	return picos, rows.Err()
}
