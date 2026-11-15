package repositories

import (
	"fmt"

	"github.com/lib/pq"

	"pulso-api/models"
)

// CTEsVendas monta as CTEs do dashboard vendas, para usar depois de um WITH:
//
//	op   oportunidades filtradas por origem_lead (o funil usa só esta)
//	orc  orçamentos das oportunidades de op, filtrados por tipo_projeto
//	vd   vendas dos orçamentos de orc
//
// tipo_projeto não vale para os cards de funil (ven-02, 04, 05, 07, 08 e
// 09), porque lead ainda não tem tipo: eles usam só op. escopo, id e
// status_projeto não valem em vendas.
func CTEsVendas(c *ConsultaSQL, f models.Filtros) string {
	pOrigens := c.Arg(pq.Array(f.OrigemLead))
	pTipos := c.Arg(pq.Array(f.TipoProjeto))
	return fmt.Sprintf(`
	op AS (
		SELECT o.* FROM oportunidades o
		WHERE %[1]s::text[] IS NULL OR o.origem = ANY(%[1]s::text[])
	),
	orc AS (
		SELECT r.* FROM orcamentos r
		WHERE r.oportunidade_id IN (SELECT id FROM op)
		  AND (%[2]s::tipo_projeto[] IS NULL OR r.tipo_projeto = ANY(%[2]s::tipo_projeto[]))
	),
	vd AS (
		SELECT v.* FROM vendas v
		WHERE v.orcamento_id IN (SELECT id FROM orc)
	)`, pOrigens, pTipos)
}

// UltimoEventoOportunidade devolve a subconsulta (LATERAL) do último evento
// de uma oportunidade antes de pFim: colunas etapa e ocorrido_em.
func UltimoEventoOportunidade(colunaID, pFim string) string {
	return fmt.Sprintf(`(
		SELECT e.etapa, e.ocorrido_em FROM oportunidade_eventos e
		WHERE e.oportunidade_id = %s AND e.ocorrido_em < %s::timestamptz
		ORDER BY e.ocorrido_em DESC, e.id DESC
		LIMIT 1)`, colunaID, pFim)
}
