package calculos

import (
	"context"
	"testing"

	"pulso-api/models"
)

// HorasEtapa do repoProjetosFalso: devolve o valor do mapa para a etapa
// pedida e registra a etapa recebida.
func (r repoProjetosFalso) HorasEtapa(_ context.Context, _ models.Filtros, etapa string) (models.HorasEtapa, error) {
	if r.etapaPedida != nil {
		*r.etapaPedida = etapa
	}
	return r.horasEtapa[etapa], r.err
}

func TestProj07a09HorasPorEtapa(t *testing.T) {
	var pedida string
	repo := repoProjetosFalso{etapaPedida: &pedida, horasEtapa: map[string]models.HorasEtapa{
		"analise":         {Estimado: 10, Real: 12.004},
		"desenvolvimento": {Estimado: 80, Real: 60},
		"testes":          {Estimado: 0, Real: 0},
	}}

	casos := []struct {
		codigo, etapa, esperado string
	}{
		{"proj-07", "analise", `{"series":[{"chave":"estimado","nome":"Estimado"},{"chave":"real","nome":"Real"}],"itens":[{"estimado":10,"percentual":120,"real":12,"rotulo":"Análise"}],"unidade":"h"}`},
		{"proj-08", "desenvolvimento", `{"series":[{"chave":"estimado","nome":"Estimado"},{"chave":"real","nome":"Real"}],"itens":[{"estimado":80,"percentual":75,"real":60,"rotulo":"Desenvolvimento"}],"unidade":"h"}`},
		{"proj-09", "testes", `{"series":[{"chave":"estimado","nome":"Estimado"},{"chave":"real","nome":"Real"}],"itens":[{"estimado":0,"percentual":null,"real":0,"rotulo":"Testes"}],"unidade":"h"}`},
	}

	reg := NovoRegistro(nil)
	for _, c := range casos {
		t.Run(c.codigo, func(t *testing.T) {
			if _, ok := reg[c.codigo].(horasEtapa); !ok {
				t.Fatalf("%s não está registrado com o cálculo real", c.codigo)
			}
			calc := NovoHorasEtapa(c.codigo, c.etapa, repo)
			if got := jsonDados(t, calc); got != c.esperado {
				t.Errorf("\nesperado %s\nobtido   %s", c.esperado, got)
			}
			if pedida != c.etapa {
				t.Errorf("o repository deveria receber a etapa %q, recebeu %q", c.etapa, pedida)
			}
		})
	}
}
