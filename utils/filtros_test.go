package utils

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"pulso-api/models"
)

func data(t *testing.T, s string) models.Data {
	t.Helper()
	d, err := ParseData(s)
	if err != nil {
		t.Fatalf("data inválida no teste: %s", s)
	}
	return d
}

func ptr[T any](v T) *T { return &v }

// sprintFalsa simula o banco: só a sprint 4 existe; a 99 dá erro de banco.
func sprintFalsa(_ context.Context, id int64) (models.Sprint, error) {
	switch id {
	case 4:
		inicio, _ := ParseData("2026-11-03")
		fim, _ := ParseData("2026-11-14")
		return models.Sprint{ID: 4, Inicio: inicio, Fim: fim}, nil
	case 99:
		return models.Sprint{}, errors.New("conexão caiu")
	default:
		return models.Sprint{}, sql.ErrNoRows
	}
}

func TestEntradaDeQuery(t *testing.T) {
	q, _ := url.ParseQuery("escopo=time&id=7&de=2026-11-01&tipo_projeto=web,mobile&tipo_projeto=api&modelo=&origem_lead=%20site%20")
	e, err := EntradaDeQuery(q)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if *e.Escopo != "time" || *e.ID != 7 || *e.De != "2026-11-01" || e.Ate != nil {
		t.Errorf("campos simples inesperados: %+v", e)
	}
	if !reflect.DeepEqual(e.TipoProjeto, []string{"web", "mobile", "api"}) {
		t.Errorf("tipo_projeto inesperado: %v", e.TipoProjeto)
	}

	for _, invalido := range []string{"id=abc", "id=1.5"} {
		q, _ := url.ParseQuery(invalido)
		if _, err := EntradaDeQuery(q); err == nil {
			t.Errorf("%s deveria dar erro", invalido)
		}
	}
}

func TestNormalizarFiltros(t *testing.T) {
	hoje := data(t, "2026-11-14")

	casos := []struct {
		nome      string
		entrada   models.FiltrosEntrada
		dashboard string
		esperado  models.Filtros
		erro      string // trecho da mensagem; "" = sem erro
	}{
		{
			nome:      "padrões em projetos",
			dashboard: "projetos",
			esperado:  models.Filtros{Dashboard: "projetos", Escopo: "projeto", De: data(t, "2026-10-16"), Ate: hoje},
		},
		{
			nome:      "vendas ignora escopo, id, status e modelo (mesmo inválidos)",
			dashboard: "vendas",
			entrada: models.FiltrosEntrada{
				Escopo: ptr("xyz"), ID: ptr(int64(-1)), StatusProjeto: []string{"???"}, Modelo: []string{"m"},
				OrigemLead: []string{"site", " indicacao ", "site"},
			},
			esperado: models.Filtros{Dashboard: "vendas", De: data(t, "2026-10-16"), Ate: hoje, OrigemLead: []string{"indicacao", "site"}},
		},
		{
			nome:      "projetos ignora origem e modelo",
			dashboard: "projetos",
			entrada:   models.FiltrosEntrada{OrigemLead: []string{"site"}, Modelo: []string{"m"}},
			esperado:  models.Filtros{Dashboard: "projetos", Escopo: "projeto", De: data(t, "2026-10-16"), Ate: hoje},
		},
		{
			nome:      "ia com listas: sem repetição, ordenadas e sem vazios",
			dashboard: "ia",
			entrada: models.FiltrosEntrada{
				Escopo: ptr("time"), ID: ptr(int64(2)),
				TipoProjeto: []string{"web", "api", "web", ""}, StatusProjeto: []string{"pausado", "em_andamento"},
				Modelo: []string{"modelo-b", "modelo-a"},
			},
			esperado: models.Filtros{
				Dashboard: "ia", Escopo: "time", ID: ptr(int64(2)), De: data(t, "2026-10-16"), Ate: hoje,
				TipoProjeto: []string{"api", "web"}, StatusProjeto: []string{"em_andamento", "pausado"},
				Modelo: []string{"modelo-a", "modelo-b"},
			},
		},
		{
			nome:      "lista só com vazios vira nil (todos)",
			dashboard: "projetos",
			entrada:   models.FiltrosEntrada{TipoProjeto: []string{"", " "}},
			esperado:  models.Filtros{Dashboard: "projetos", Escopo: "projeto", De: data(t, "2026-10-16"), Ate: hoje},
		},
		{
			nome:      "só de: vai até hoje",
			dashboard: "vendas",
			entrada:   models.FiltrosEntrada{De: ptr("2026-01-01")},
			esperado:  models.Filtros{Dashboard: "vendas", De: data(t, "2026-01-01"), Ate: hoje},
		},
		{
			nome:      "só ate: 30 dias terminados em ate",
			dashboard: "vendas",
			entrada:   models.FiltrosEntrada{Ate: ptr("2026-03-31")},
			esperado:  models.Filtros{Dashboard: "vendas", De: data(t, "2026-03-02"), Ate: data(t, "2026-03-31")},
		},
		{
			nome:      "de igual a ate",
			dashboard: "vendas",
			entrada:   models.FiltrosEntrada{De: ptr("2026-05-05"), Ate: ptr("2026-05-05")},
			esperado:  models.Filtros{Dashboard: "vendas", De: data(t, "2026-05-05"), Ate: data(t, "2026-05-05")},
		},
		{
			nome:      "exatamente 5 anos é aceito",
			dashboard: "vendas",
			entrada:   models.FiltrosEntrada{De: ptr("2021-11-14"), Ate: ptr("2026-11-14")},
			esperado:  models.Filtros{Dashboard: "vendas", De: data(t, "2021-11-14"), Ate: hoje},
		},
		{
			nome:      "escopo=sprint usa as datas da sprint e ignora as enviadas",
			dashboard: "projetos",
			entrada:   models.FiltrosEntrada{Escopo: ptr("sprint"), ID: ptr(int64(4)), De: ptr("data-ruim"), Ate: ptr("2020-01-01")},
			esperado:  models.Filtros{Dashboard: "projetos", Escopo: "sprint", ID: ptr(int64(4)), De: data(t, "2026-11-03"), Ate: data(t, "2026-11-14")},
		},
		{nome: "escopo inválido", dashboard: "projetos", entrada: models.FiltrosEntrada{Escopo: ptr("cliente")}, erro: "escopo inválido"},
		{nome: "id zero", dashboard: "ia", entrada: models.FiltrosEntrada{ID: ptr(int64(0))}, erro: "id inválido"},
		{nome: "escopo=sprint sem id", dashboard: "projetos", entrada: models.FiltrosEntrada{Escopo: ptr("sprint")}, erro: "exige o id"},
		{nome: "sprint inexistente", dashboard: "projetos", entrada: models.FiltrosEntrada{Escopo: ptr("sprint"), ID: ptr(int64(5))}, erro: "Sprint 5 não encontrada"},
		{nome: "data inválida em de", dashboard: "vendas", entrada: models.FiltrosEntrada{De: ptr("14/11/2026")}, erro: "data inválida em 'de'"},
		{nome: "dia inexistente em ate", dashboard: "vendas", entrada: models.FiltrosEntrada{Ate: ptr("2026-02-30")}, erro: "data inválida em 'ate'"},
		{nome: "de maior que ate", dashboard: "vendas", entrada: models.FiltrosEntrada{De: ptr("2026-11-10"), Ate: ptr("2026-11-01")}, erro: "não pode ser maior"},
		{nome: "mais de 5 anos", dashboard: "vendas", entrada: models.FiltrosEntrada{De: ptr("2021-11-13"), Ate: ptr("2026-11-14")}, erro: "5 anos"},
		{nome: "tipo_projeto inválido", dashboard: "vendas", entrada: models.FiltrosEntrada{TipoProjeto: []string{"desktop"}}, erro: "tipo_projeto inválido"},
		{nome: "status_projeto inválido", dashboard: "ia", entrada: models.FiltrosEntrada{StatusProjeto: []string{"ativo"}}, erro: "status_projeto inválido"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			obtido, err := NormalizarFiltros(context.Background(), c.entrada, c.dashboard, hoje, sprintFalsa)
			if c.erro != "" {
				var ef *ErroFiltro
				if !errors.As(err, &ef) {
					t.Fatalf("esperado ErroFiltro contendo %q, obtido %v", c.erro, err)
				}
				if !strings.Contains(ef.Mensagem, c.erro) {
					t.Errorf("mensagem %q não contém %q", ef.Mensagem, c.erro)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if obtido.Chave() != c.esperado.Chave() {
				t.Errorf("\nesperado %s\nobtido   %s", c.esperado.Chave(), obtido.Chave())
			}
		})
	}
}

func TestNormalizarFiltrosErroDoBanco(t *testing.T) {
	e := models.FiltrosEntrada{Escopo: ptr("sprint"), ID: ptr(int64(99))}
	_, err := NormalizarFiltros(context.Background(), e, "projetos", data(t, "2026-11-14"), sprintFalsa)
	var ef *ErroFiltro
	if err == nil || errors.As(err, &ef) {
		t.Fatalf("erro de banco não pode virar ErroFiltro (400): %v", err)
	}
}

func TestNormalizarFiltrosLimites(t *testing.T) {
	hoje := data(t, "2026-11-14")
	longo := string(make([]rune, 101))
	muitos := make([]string, 51)
	for i := range muitos {
		muitos[i] = string(rune('a'+i%26)) + string(rune('a'+i/26))
	}
	for nome, e := range map[string]models.FiltrosEntrada{
		"texto longo":  {OrigemLead: []string{longo}},
		"itens demais": {OrigemLead: muitos},
	} {
		if _, err := NormalizarFiltros(context.Background(), e, "vendas", hoje, sprintFalsa); err == nil {
			t.Errorf("%s deveria dar erro", nome)
		}
	}
}

func TestChaveFiltrosEquivalentes(t *testing.T) {
	hoje := data(t, "2026-11-14")
	a, _ := NormalizarFiltros(context.Background(), models.FiltrosEntrada{TipoProjeto: []string{"web", "api"}}, "projetos", hoje, sprintFalsa)
	b, _ := NormalizarFiltros(context.Background(), models.FiltrosEntrada{TipoProjeto: []string{"api", "web", "api"}, Escopo: ptr("projeto")}, "projetos", hoje, sprintFalsa)
	if a.Chave() != b.Chave() {
		t.Errorf("filtros equivalentes devem ter a mesma chave:\n%s\n%s", a.Chave(), b.Chave())
	}
	c, _ := NormalizarFiltros(context.Background(), models.FiltrosEntrada{TipoProjeto: []string{"web", "api"}}, "ia", hoje, sprintFalsa)
	if a.Chave() == c.Chave() {
		t.Error("dashboards diferentes devem ter chaves diferentes")
	}
}
