package repositories

import (
	"context"

	"pulso-api/models"
	"pulso-api/utils"
)

// periodoT devolve o filtro de período de T e registra os parâmetros.
func periodoT(c *ConsultaSQL, f models.Filtros) string {
	inicio, fimAte := utils.Intervalo(f.De, f.Ate)
	return `t.ocorrido_em >= ` + c.Arg(inicio) + `::timestamptz AND t.ocorrido_em < ` + c.Arg(fimAte) + `::timestamptz`
}

func (r *metricasIARepository) grupos(ctx context.Context, query string, args []interface{}) ([]models.GrupoCusto, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grupos := []models.GrupoCusto{}
	for rows.Next() {
		var g models.GrupoCusto
		if err := rows.Scan(&g.Chave, &g.Rotulo, &g.Custo, &g.Tokens); err != nil {
			return nil, err
		}
		grupos = append(grupos, g)
	}
	return grupos, rows.Err()
}

func (r *metricasIARepository) CustoPorProjeto(ctx context.Context, f models.Filtros) ([]models.GrupoCusto, error) {
	c := &ConsultaSQL{}
	periodo := periodoT(c, f)
	query := "WITH " + CTEsEscopo(c, f) + `
	SELECT pr.id::text, pr.nome, SUM(t.custo)::float8, SUM(t.tokens_entrada::bigint + t.tokens_saida)::bigint
	FROM t JOIN projetos pr ON pr.id = t.projeto_id
	WHERE ` + periodo + `
	GROUP BY pr.id, pr.nome
	ORDER BY 3 DESC, 2`
	return r.grupos(ctx, query, c.Args())
}

func (r *metricasIARepository) CustoPorSprint(ctx context.Context, f models.Filtros) ([]models.GrupoCusto, error) {
	c := &ConsultaSQL{}
	periodo := periodoT(c, f)
	// Consumo com atividade em sprint vai para a sprint; sem atividade ou
	// com atividade de backlog vai para "Sem sprint" (chave vazia)
	query := "WITH " + CTEsEscopo(c, f) + `
	SELECT COALESCE(s.id::text, ''),
	       COALESCE('Sprint ' || s.numero || ' · ' || tm.nome, 'Sem sprint'),
	       SUM(t.custo)::float8, SUM(t.tokens_entrada::bigint + t.tokens_saida)::bigint
	FROM t
	LEFT JOIN atividades at ON at.id = t.atividade_id
	LEFT JOIN sprints s ON s.id = at.sprint_id
	LEFT JOIN times tm ON tm.id = s.time_id
	WHERE ` + periodo + `
	GROUP BY s.id, s.numero, tm.nome
	ORDER BY 3 DESC, 2`
	return r.grupos(ctx, query, c.Args())
}

func (r *metricasIARepository) CustoPorModelo(ctx context.Context, f models.Filtros) ([]models.GrupoCusto, error) {
	c := &ConsultaSQL{}
	periodo := periodoT(c, f)
	query := "WITH " + CTEsEscopo(c, f) + `
	SELECT t.modelo, t.modelo, SUM(t.custo)::float8, SUM(t.tokens_entrada::bigint + t.tokens_saida)::bigint
	FROM t
	WHERE ` + periodo + `
	GROUP BY t.modelo
	ORDER BY 3 DESC, 1`
	return r.grupos(ctx, query, c.Args())
}

func (r *metricasIARepository) CustoPorEtapa(ctx context.Context, f models.Filtros) ([]models.GrupoCusto, error) {
	c := &ConsultaSQL{}
	periodo := periodoT(c, f)
	query := "WITH " + CTEsEscopo(c, f) + `
	SELECT t.etapa::text, t.etapa::text, SUM(t.custo)::float8, SUM(t.tokens_entrada::bigint + t.tokens_saida)::bigint
	FROM t
	WHERE ` + periodo + `
	GROUP BY t.etapa`
	return r.grupos(ctx, query, c.Args())
}

func (r *metricasIARepository) CustoTipoComplexidade(ctx context.Context, f models.Filtros) ([]models.CustoTipoComplexidade, error) {
	c := &ConsultaSQL{}
	periodo := periodoT(c, f)
	// Base: atividades de A com algum consumo de T no período
	query := "WITH " + CTEsEscopo(c, f) + `,
	consumo AS (
		SELECT t.atividade_id, SUM(t.custo) AS custo
		FROM t WHERE ` + periodo + ` AND t.atividade_id IN (SELECT id FROM a)
		GROUP BY t.atividade_id
	)
	SELECT a.tipo::text, a.complexidade::text, SUM(co.custo)::float8, COUNT(DISTINCT a.id)
	FROM consumo co JOIN a ON a.id = co.atividade_id
	GROUP BY a.tipo, a.complexidade`

	rows, err := r.db.QueryContext(ctx, query, c.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grupos := []models.CustoTipoComplexidade{}
	for rows.Next() {
		var g models.CustoTipoComplexidade
		if err := rows.Scan(&g.Tipo, &g.Complexidade, &g.Custo, &g.Atividades); err != nil {
			return nil, err
		}
		grupos = append(grupos, g)
	}
	return grupos, rows.Err()
}
