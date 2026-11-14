package calculos

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"pulso-api/models"
)

// repoProjetosFalso devolve números fixos.
type repoProjetosFalso struct {
	progresso models.ResumoProgresso
	horas     models.ResumoHoras
	burnUp    models.BurnUp
	status    models.ContagemStatus
	burndown  models.Burndown
	// proj-07 a 09
	horasEtapa  map[string]models.HorasEtapa
	etapaPedida *string
	// proj-10 e 11 (agrupamento → grupos)
	tempoMedio map[string][]models.GrupoTempoMedio
	err        error
}

func (r repoProjetosFalso) Progresso(context.Context, models.Filtros) (models.ResumoProgresso, error) {
	return r.progresso, r.err
}

func (r repoProjetosFalso) Horas(context.Context, models.Filtros) (models.ResumoHoras, error) {
	return r.horas, r.err
}

type repoIAFalso struct {
	consumo models.ResumoConsumoIA
	err     error
}

func (r repoIAFalso) ConsumoAcumulado(context.Context, models.Filtros) (models.ResumoConsumoIA, error) {
	return r.consumo, r.err
}

// jsonDados devolve o JSON de "dados", para comparar o formato exato.
func jsonDados(t *testing.T, c Calculador) string {
	t.Helper()
	r, err := c.Calcular(context.Background(), models.Filtros{})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	b, err := json.Marshal(r.Dados)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestProj01Progresso(t *testing.T) {
	casos := []struct {
		nome     string
		resumo   models.ResumoProgresso
		esperado string
	}{
		{
			"ponderado pela estimativa: 1 de 3 atividades, mas 29 de 40 horas",
			models.ResumoProgresso{EstimativaConcluida: 29, EstimativaTotal: 40, Concluidas: 1, Total: 3},
			`{"valor":72.5,"unidade":"%","variacao":null,"detalhe":[{"rotulo":"Concluídas","valor":1,"unidade":"atividades"},{"rotulo":"Total","valor":3,"unidade":"atividades"}]}`,
		},
		{
			"sem atividades: valor null, contagens zero",
			models.ResumoProgresso{},
			`{"valor":null,"unidade":"%","variacao":null,"detalhe":[{"rotulo":"Concluídas","valor":0,"unidade":"atividades"},{"rotulo":"Total","valor":0,"unidade":"atividades"}]}`,
		},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := jsonDados(t, NovoProgresso(repoProjetosFalso{progresso: c.resumo})); got != c.esperado {
				t.Errorf("\nesperado %s\nobtido   %s", c.esperado, got)
			}
		})
	}
}

func TestProj02Horas(t *testing.T) {
	casos := []struct {
		nome     string
		resumo   models.ResumoHoras
		esperado string
	}{
		{
			"estouro passa de 100%",
			models.ResumoHoras{Reais: 52.484, Estimadas: 42},
			`{"valor":125,"unidade":"%","variacao":null,"detalhe":[{"rotulo":"Horas reais","valor":52.48,"unidade":"h"},{"rotulo":"Horas estimadas","valor":42,"unidade":"h"}]}`,
		},
		{
			"sem estimativa: valor null",
			models.ResumoHoras{Reais: 3},
			`{"valor":null,"unidade":"%","variacao":null,"detalhe":[{"rotulo":"Horas reais","valor":3,"unidade":"h"},{"rotulo":"Horas estimadas","valor":0,"unidade":"h"}]}`,
		},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := jsonDados(t, NovoHoras(repoProjetosFalso{horas: c.resumo})); got != c.esperado {
				t.Errorf("\nesperado %s\nobtido   %s", c.esperado, got)
			}
		})
	}
}

func TestProj03CustoIA(t *testing.T) {
	casos := []struct {
		nome     string
		resumo   models.ResumoConsumoIA
		esperado string
	}{
		{
			"quota passada (crítica)",
			models.ResumoConsumoIA{TokensAcumulados: 2593551, Quota: 1800000, CustoPeriodo: 41.23456},
			`{"valor":144.1,"unidade":"%","variacao":null,"detalhe":[{"rotulo":"Custo no período","valor":41.23,"unidade":"R$"},{"rotulo":"Tokens consumidos","valor":2593551,"unidade":"tokens"}]}`,
		},
		{
			"sem projetos no escopo: quota 0 → valor null",
			models.ResumoConsumoIA{},
			`{"valor":null,"unidade":"%","variacao":null,"detalhe":[{"rotulo":"Custo no período","valor":0,"unidade":"R$"},{"rotulo":"Tokens consumidos","valor":0,"unidade":"tokens"}]}`,
		},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := jsonDados(t, NovoCustoIAProjeto(repoIAFalso{consumo: c.resumo})); got != c.esperado {
				t.Errorf("\nesperado %s\nobtido   %s", c.esperado, got)
			}
		})
	}
}

func TestProj01a03PropagamErro(t *testing.T) {
	falha := errors.New("sql ruim")
	for _, c := range []Calculador{
		NovoProgresso(repoProjetosFalso{err: falha}),
		NovoHoras(repoProjetosFalso{err: falha}),
		NovoCustoIAProjeto(repoIAFalso{err: falha}),
	} {
		if _, err := c.Calcular(context.Background(), models.Filtros{}); !errors.Is(err, falha) {
			t.Errorf("%s deveria devolver o erro do repository, obtido %v", c.Codigo(), err)
		}
	}
}

func TestProj01a03Registrados(t *testing.T) {
	reg := NovoRegistro(nil)
	for _, codigo := range []string{"proj-01", "proj-02", "proj-03"} {
		if _, pendente := reg[codigo].(pendente); pendente {
			t.Errorf("%s ainda está pendente no registro", codigo)
		}
		if len(reg[codigo].Gatilhos()) == 0 {
			t.Errorf("%s sem gatilhos", codigo)
		}
	}
}
