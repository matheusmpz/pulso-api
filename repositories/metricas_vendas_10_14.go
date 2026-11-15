package repositories

import (
	"context"
	"database/sql"

	"pulso-api/models"
	"pulso-api/utils"
)

func (r *metricasVendasRepository) ReceitaMensal(ctx context.Context, f models.Filtros) ([]models.MesReceita, error) {
	c := &ConsultaSQL{}
	pDe, pAte, pTZ := c.Arg(f.De.String()), c.Arg(f.Ate.String()), c.Arg(utils.FusoApp().String())
	// Meses inteiros de mes(de) a mes(ate); vd já aplica tipo e origem
	query := "WITH " + CTEsVendas(c, f) + `,
	meses AS (
		SELECT g::date AS mes
		FROM generate_series(date_trunc('month', ` + pDe + `::date), date_trunc('month', ` + pAte + `::date), INTERVAL '1 month') g
	),
	por_mes AS (
		SELECT date_trunc('month', (vd.fechada_em AT TIME ZONE ` + pTZ + `::text))::date AS mes, SUM(vd.valor) AS receita
		FROM vd GROUP BY 1
	)
	SELECT to_char(m.mes, 'YYYY-MM'), COALESCE(p.receita, 0)::float8, mr.valor::float8
	FROM meses m
	LEFT JOIN por_mes p ON p.mes = m.mes
	LEFT JOIN metas_receita mr ON mr.mes = m.mes
	ORDER BY m.mes`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	meses := []models.MesReceita{}
	for rows.Next() {
		var m models.MesReceita
		var meta sql.NullFloat64
		if err := rows.Scan(&m.Mes, &m.Receita, &meta); err != nil {
			return nil, err
		}
		if meta.Valid {
			m.Meta = &meta.Float64
		}
		meses = append(meses, m)
	}
	return meses, rows.Err()
}

func (r *metricasVendasRepository) StatusPropostas(ctx context.Context, f models.Filtros) (models.StatusPropostas, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim := c.Arg(inicio), c.Arg(fimAte)
	query := "WITH " + CTEsVendas(c, f) + `,
	situacao AS (
		SELECT COALESCE((
			SELECT e.status FROM orcamento_eventos e
			WHERE e.orcamento_id = orc.id AND e.ocorrido_em < ` + pFim + `::timestamptz
			ORDER BY e.ocorrido_em DESC, e.id DESC LIMIT 1), 'pendente'::status_orcamento) AS status
		FROM orc
		WHERE orc.emitido_em >= ` + pIni + `::timestamptz AND orc.emitido_em < ` + pFim + `::timestamptz
	)
	SELECT COUNT(*) FILTER (WHERE status = 'pendente'), COUNT(*) FILTER (WHERE status = 'aprovada'),
	       COUNT(*) FILTER (WHERE status = 'recusada'), COUNT(*) FILTER (WHERE status = 'em_revisao')
	FROM situacao`

	var s models.StatusPropostas
	err := r.db.QueryRowContext(ctx, query, c.Args()...).Scan(&s.Pendentes, &s.Aprovadas, &s.Recusadas, &s.EmRevisao)
	return s, err
}

func (r *metricasVendasRepository) MargemPorComplexidade(ctx context.Context, f models.Filtros) ([]models.MargemComplexidade, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim := c.Arg(inicio), c.Arg(fimAte)
	query := "WITH " + CTEsVendas(c, f) + `
	SELECT orc.complexidade::text, COUNT(*), AVG(orc.margem_prevista)::float8,
	       SUM(orc.valor)::float8, SUM(orc.horas_estimadas)::float8
	FROM orc
	WHERE orc.emitido_em >= ` + pIni + `::timestamptz AND orc.emitido_em < ` + pFim + `::timestamptz
	GROUP BY orc.complexidade`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grupos := []models.MargemComplexidade{}
	for rows.Next() {
		var g models.MargemComplexidade
		if err := rows.Scan(&g.Complexidade, &g.N, &g.MargemMedia, &g.Valor, &g.Horas); err != nil {
			return nil, err
		}
		grupos = append(grupos, g)
	}
	return grupos, rows.Err()
}

func (r *metricasVendasRepository) OrcadoReal(ctx context.Context, f models.Filtros) ([]models.OrcadoRealTipo, error) {
	c := &ConsultaSQL{}
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	pIni, pFim, pAte := c.Arg(inicio), c.Arg(fimAte), c.Arg(f.Ate.String())
	// Base: orçamentos com venda fechada no período. Horas, custo de horas e
	// custo de IA são agregados por projeto ANTES de juntar (sem JOIN entre
	// tabelas de evento para somar).
	query := "WITH " + CTEsVendas(c, f) + `,
	base AS (
		SELECT orc.id AS orcamento_id, orc.tipo_projeto, orc.horas_estimadas,
		       orc.valor * orc.margem_prevista / 100 AS margem_prevista,
		       vd.valor AS vendido, pr.id AS projeto_id
		FROM vd
		JOIN orc ON orc.id = vd.orcamento_id
		LEFT JOIN projetos pr ON pr.orcamento_id = orc.id
		WHERE vd.fechada_em >= ` + pIni + `::timestamptz AND vd.fechada_em < ` + pFim + `::timestamptz
	),
	horas AS (
		SELECT rh.projeto_id, SUM(rh.horas) AS horas, SUM(rh.horas * rh.custo_hora) AS custo
		FROM registros_horas rh
		WHERE rh.projeto_id IN (SELECT projeto_id FROM base) AND rh.trabalhado_em <= ` + pAte + `::date
		GROUP BY rh.projeto_id
	),
	ia AS (
		SELECT ct.projeto_id, SUM(ct.custo) AS custo
		FROM consumo_tokens ct
		WHERE ct.projeto_id IN (SELECT projeto_id FROM base) AND ct.ocorrido_em < ` + pFim + `::timestamptz
		GROUP BY ct.projeto_id
	)
	SELECT b.tipo_projeto::text,
	       SUM(b.horas_estimadas)::float8, COALESCE(SUM(h.horas), 0)::float8,
	       SUM(b.margem_prevista)::float8, SUM(b.vendido)::float8,
	       COALESCE(SUM(h.custo), 0)::float8, COALESCE(SUM(i.custo), 0)::float8
	FROM base b
	LEFT JOIN horas h ON h.projeto_id = b.projeto_id
	LEFT JOIN ia i ON i.projeto_id = b.projeto_id
	GROUP BY b.tipo_projeto`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tipos := []models.OrcadoRealTipo{}
	for rows.Next() {
		var t models.OrcadoRealTipo
		if err := rows.Scan(&t.Tipo, &t.HorasEstimadas, &t.HorasReais, &t.MargemPrevista, &t.ValorVendido, &t.CustoHoras, &t.CustoIA); err != nil {
			return nil, err
		}
		tipos = append(tipos, t)
	}
	return tipos, rows.Err()
}
