package repositories

import (
	"fmt"

	"github.com/lib/pq"

	"pulso-api/models"
	"pulso-api/utils"
)

// ConsultaSQL acumula os parâmetros de uma consulta e numera os
// placeholders ($1, $2...). Toda entrada do usuário entra como parâmetro;
// o texto SQL só é montado a partir de trechos fixos deste pacote.
type ConsultaSQL struct {
	args []interface{}
}

// Arg registra um valor e devolve o placeholder correspondente.
func (c *ConsultaSQL) Arg(v interface{}) string {
	c.args = append(c.args, v)
	return fmt.Sprintf("$%d", len(c.args))
}

// Args devolve os valores na ordem dos placeholders.
func (c *ConsultaSQL) Args() []interface{} {
	return c.args
}

// StatusAtividadeAte devolve a expressão SQL do status atual de uma
// atividade até um instante (seção 8): o status do último evento com
// ocorrido_em antes de pFim (maior ocorrido_em; empate, maior id). Sem
// evento, 'aberta'. colunaID é a coluna com o id da atividade (ex.: a.id).
func StatusAtividadeAte(colunaID, pFim string) string {
	return fmt.Sprintf(`COALESCE((
		SELECT e.status FROM atividade_eventos e
		WHERE e.atividade_id = %s AND e.ocorrido_em < %s::timestamptz
		ORDER BY e.ocorrido_em DESC, e.id DESC
		LIMIT 1), 'aberta'::status_atividade)`, colunaID, pFim)
}

// CTEProjetosFiltrados monta "pf AS (...)": projetos filtrados só por
// tipo_projeto e status_projeto, sem o recorte do escopo. Usado quando o
// card precisa olhar além de A (ex.: o histórico de sprints do proj-13).
func CTEProjetosFiltrados(c *ConsultaSQL, f models.Filtros) string {
	pTipos := c.Arg(pq.Array(f.TipoProjeto))
	pStatus := c.Arg(pq.Array(f.StatusProjeto))
	return fmt.Sprintf(`
	pf AS (
		SELECT pr.id FROM projetos pr
		WHERE (%[1]s::tipo_projeto[] IS NULL OR pr.tipo = ANY(%[1]s::tipo_projeto[]))
		  AND (%[2]s::status_projeto[] IS NULL OR pr.status = ANY(%[2]s::status_projeto[]))
	)`, pTipos, pStatus)
}

// CTEsEscopo monta as CTEs dos conjuntos do escopo (seção 7.2), para
// usar depois de um WITH:
//
//	p      projetos do escopo (id)
//	a      atividades do escopo, criadas até o fim do dia "ate"
//	h      registros_horas das atividades de a
//	t      consumo_tokens do escopo (filtrado por modelo, se houver)
//	quota  soma de limite_tokens dos projetos de p (coluna "valor")
//
// As CTEs não aplicam a regra de período (7.3): cada card filtra h e t
// pelo período ou pelo acumulado que usa.
//
// Exemplo de uso:
//
//	c := &ConsultaSQL{}
//	sql := "WITH " + CTEsEscopo(c, f) + " SELECT COUNT(*) FROM a"
//	db.QueryRowContext(ctx, sql, c.Args()...)
func CTEsEscopo(c *ConsultaSQL, f models.Filtros) string {
	var id interface{}
	if f.ID != nil {
		id = *f.ID
	}
	pID := c.Arg(id)
	pTipos := c.Arg(pq.Array(f.TipoProjeto))
	pStatus := c.Arg(pq.Array(f.StatusProjeto))
	pModelos := c.Arg(pq.Array(f.Modelo))
	_, fimAte := utils.Intervalo(f.De, f.Ate)
	pFimAte := c.Arg(fimAte)

	// Trechos fixos por escopo (whitelist: o escopo já foi validado, e
	// qualquer outro valor cai no padrão "projeto").
	var condP, condA, condT string
	switch f.Escopo {
	case models.EscopoTime:
		// Projetos com ao menos uma atividade do time (de qualquer time, se vazio)
		condP = fmt.Sprintf(`EXISTS (SELECT 1 FROM atividades x
			WHERE x.projeto_id = pr.id AND (%[1]s::bigint IS NULL OR x.time_id = %[1]s::bigint))`, pID)
		condA = fmt.Sprintf(`(%[1]s::bigint IS NULL OR at.time_id = %[1]s::bigint)`, pID)
		// Consumo de desenvolvedores do time, em projetos de p
		condT = fmt.Sprintf(`ct.projeto_id IN (SELECT id FROM p)
			AND ct.desenvolvedor_id IN (SELECT d.id FROM desenvolvedores d
				WHERE %[1]s::bigint IS NULL OR d.time_id = %[1]s::bigint)`, pID)
	case models.EscopoSprint:
		// Projetos com ao menos uma atividade da sprint
		condP = fmt.Sprintf(`EXISTS (SELECT 1 FROM atividades x
			WHERE x.projeto_id = pr.id AND x.sprint_id = %[1]s::bigint)`, pID)
		condA = fmt.Sprintf(`at.sprint_id = %[1]s::bigint`, pID)
		// Consumo ligado às atividades da sprint
		condT = `ct.atividade_id IN (SELECT id FROM a)`
	default: // models.EscopoProjeto
		condP = fmt.Sprintf(`(%[1]s::bigint IS NULL OR pr.id = %[1]s::bigint)`, pID)
		condA = `true`
		// Todo o consumo dos projetos de p (inclusive sem atividade)
		condT = `ct.projeto_id IN (SELECT id FROM p)`
	}

	return fmt.Sprintf(`
	p AS (
		SELECT pr.id
		FROM projetos pr
		WHERE (%[1]s::tipo_projeto[] IS NULL OR pr.tipo = ANY(%[1]s::tipo_projeto[]))
		  AND (%[2]s::status_projeto[] IS NULL OR pr.status = ANY(%[2]s::status_projeto[]))
		  AND %[3]s
	),
	a AS (
		SELECT at.id, at.projeto_id, at.time_id, at.sprint_id, at.titulo, at.tipo,
		       at.complexidade, at.estimativa_analise, at.estimativa_desenvolvimento,
		       at.estimativa_testes, at.estimativa_horas, at.criado_em
		FROM atividades at
		WHERE at.projeto_id IN (SELECT id FROM p)
		  AND at.criado_em < %[4]s::timestamptz
		  AND %[5]s
	),
	h AS (
		SELECT rh.*
		FROM registros_horas rh
		WHERE rh.atividade_id IN (SELECT id FROM a)
	),
	t AS (
		SELECT ct.*
		FROM consumo_tokens ct
		WHERE %[6]s
		  AND (%[7]s::text[] IS NULL OR ct.modelo = ANY(%[7]s::text[]))
	),
	quota AS (
		SELECT COALESCE(SUM(pr.limite_tokens), 0)::bigint AS valor
		FROM projetos pr
		WHERE pr.id IN (SELECT id FROM p)
	)`, pTipos, pStatus, condP, pFimAte, condA, condT, pModelos)
}
