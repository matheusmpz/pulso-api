package repositories

import (
	"strings"
	"testing"
	"time"

	"pulso-api/models"
)

func filtrosTeste(escopo string, id *int64) models.Filtros {
	de := models.Data{Time: time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)}
	ate := models.Data{Time: time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)}
	return models.Filtros{
		Dashboard: "ia", Escopo: escopo, ID: id, De: de, Ate: ate,
		TipoProjeto: []string{"web' OR 1=1 --"}, Modelo: []string{"modelo'); DROP TABLE x; --"},
	}
}

func TestConsultaSQLNumeraPlaceholders(t *testing.T) {
	c := &ConsultaSQL{}
	if p := c.Arg("a"); p != "$1" {
		t.Errorf("esperado $1, obtido %s", p)
	}
	if p := c.Arg(2); p != "$2" {
		t.Errorf("esperado $2, obtido %s", p)
	}
	if len(c.Args()) != 2 {
		t.Errorf("esperados 2 args, obtidos %d", len(c.Args()))
	}
}

func TestCTEsEscopoNaoConcatenaEntrada(t *testing.T) {
	id := int64(7)
	for _, escopo := range []string{"projeto", "time", "sprint", "invalido"} {
		t.Run(escopo, func(t *testing.T) {
			c := &ConsultaSQL{}
			// Começa com um parâmetro do card, para conferir a numeração seguinte
			c.Arg("parametro do card")
			sql := CTEsEscopo(c, filtrosTeste(escopo, &id))

			if strings.Contains(sql, "OR 1=1") || strings.Contains(sql, "DROP TABLE") {
				t.Fatal("texto do usuário apareceu no SQL")
			}
			if len(c.Args()) != 6 {
				t.Errorf("esperados 6 args (1 do card + 5 do escopo), obtidos %d", len(c.Args()))
			}
			if !strings.Contains(sql, "$2::bigint") || strings.Contains(sql, "$1::") {
				t.Error("os placeholders do escopo deveriam começar em $2")
			}
			for _, cte := range []string{"p AS (", "a AS (", "h AS (", "t AS (", "quota AS ("} {
				if !strings.Contains(sql, cte) {
					t.Errorf("CTE %q ausente", cte)
				}
			}
		})
	}
}
