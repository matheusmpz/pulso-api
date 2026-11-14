package calculos

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"pulso-api/models"
)

// Métodos novos do repoProjetosFalso (definido em projetos_01_03_test.go).

func (r repoProjetosFalso) BurnUp(context.Context, models.Filtros) (models.BurnUp, error) {
	return r.burnUp, r.err
}

func (r repoProjetosFalso) ContagemStatus(context.Context, models.Filtros) (models.ContagemStatus, error) {
	return r.status, r.err
}

func (r repoProjetosFalso) Burndown(context.Context, models.Filtros, models.Data) (models.Burndown, error) {
	return r.burndown, r.err
}

func d(s string) models.Data {
	t, _ := time.Parse(models.FormatoData, s)
	return models.Data{Time: t}
}

func pontosJSON(t *testing.T, p []map[string]interface{}) string {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestMontarBurnUp(t *testing.T) {
	casos := []struct {
		nome     string
		b        models.BurnUp
		esperado string
	}{
		{
			"dias normais + ponto final até a estimativa",
			models.BurnUp{EstimativaTotal: 40, Dias: []models.DiaBurnUp{
				{Dia: d("2026-11-01"), HorasAcumuladas: 4, EstimativaConcluida: 0},
				{Dia: d("2026-11-02"), HorasAcumuladas: 10.004, EstimativaConcluida: 8},
				{Dia: d("2026-11-03"), HorasAcumuladas: 22, EstimativaConcluida: 20},
			}},
			`[{"data":"2026-11-01","ideal":10,"real":0,"x":4},` +
				`{"data":"2026-11-02","ideal":25,"real":20,"x":10},` +
				`{"data":"2026-11-03","ideal":55,"real":50,"x":22},` +
				`{"ideal":100,"real":null,"x":40}]`,
		},
		{
			"estouro: x passa da estimativa, ideal para em 100 e não há ponto final",
			models.BurnUp{EstimativaTotal: 10, Dias: []models.DiaBurnUp{
				{Dia: d("2026-11-01"), HorasAcumuladas: 6, EstimativaConcluida: 0},
				{Dia: d("2026-11-02"), HorasAcumuladas: 13, EstimativaConcluida: 10},
			}},
			`[{"data":"2026-11-01","ideal":60,"real":0,"x":6},{"data":"2026-11-02","ideal":100,"real":100,"x":13}]`,
		},
		{
			"sem horas (só eventos)",
			models.BurnUp{EstimativaTotal: 12, Dias: []models.DiaBurnUp{{Dia: d("2026-11-01")}}},
			`[{"ideal":0,"real":0,"x":0},{"ideal":100,"real":null,"x":12}]`,
		},
		{
			"sem nenhum dia",
			models.BurnUp{EstimativaTotal: 12},
			`[{"ideal":0,"real":0,"x":0},{"ideal":100,"real":null,"x":12}]`,
		},
		{"sem atividades", models.BurnUp{}, `[]`},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			dados := MontarBurnUp(c.b)
			if got := pontosJSON(t, dados.Pontos); got != c.esperado {
				t.Errorf("\nesperado %s\nobtido   %s", c.esperado, got)
			}
			if len(dados.Series) != 2 || dados.EixoX != "Horas gastas (acumuladas)" || dados.Unidade != "%" {
				t.Errorf("metadados inesperados: %+v", dados)
			}
		})
	}
}

func TestMontarBurndown(t *testing.T) {
	sprint := &models.Sprint{ID: 9, TimeID: 2, Numero: 5, Inicio: d("2026-11-10"), Fim: d("2026-11-14")}
	b := models.Burndown{Sprint: sprint, Total: 20, Dias: []models.DiaBurndown{
		{Dia: d("2026-11-10"), Restante: 20}, {Dia: d("2026-11-11"), Restante: 20},
		{Dia: d("2026-11-12"), Restante: 12}, {Dia: d("2026-11-13"), Restante: 8},
		{Dia: d("2026-11-14"), Restante: 8},
	}}

	dados := MontarBurndown(b, d("2026-11-12"))
	esperado := `[{"ideal":20,"restante":20,"x":"2026-11-10"},{"ideal":15,"restante":20,"x":"2026-11-11"},` +
		`{"ideal":10,"restante":12,"x":"2026-11-12"},{"ideal":5,"restante":null,"x":"2026-11-13"},` +
		`{"ideal":0,"restante":null,"x":"2026-11-14"}]`
	if got := pontosJSON(t, dados.Pontos); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
	extra, _ := json.Marshal(dados.Extra)
	if string(extra) != `{"sprint":{"fim":"2026-11-14","id":9,"inicio":"2026-11-10","numero":5,"time_id":2}}` {
		t.Errorf("extra inesperado: %s", extra)
	}

	vazio := MontarBurndown(models.Burndown{Dias: []models.DiaBurndown{}}, d("2026-11-12"))
	extra, _ = json.Marshal(vazio.Extra)
	if len(vazio.Pontos) != 0 || string(extra) != `{"sprint":null}` || vazio.Unidade != "h" {
		t.Errorf("sem sprint deveria vir vazio: %+v %s", vazio, extra)
	}

	um := MontarBurndown(models.Burndown{Sprint: sprint, Total: 5, Dias: []models.DiaBurndown{{Dia: d("2026-11-10"), Restante: 5}}}, d("2026-11-12"))
	if got := pontosJSON(t, um.Pontos); got != `[{"ideal":0,"restante":5,"x":"2026-11-10"}]` {
		t.Errorf("sprint de um dia: %s", got)
	}
}

func TestProj05IssuesPorStatus(t *testing.T) {
	got := jsonDados(t, NovoIssuesPorStatus(repoProjetosFalso{status: models.ContagemStatus{Abertas: 6, EmAndamento: 5, Concluidas: 26}}))
	esperado := `{"fatias":[{"rotulo":"Abertas","chave":"aberta","valor":6},{"rotulo":"Em andamento","chave":"em_andamento","valor":5},{"rotulo":"Concluídas","chave":"concluida","valor":26}],"total":37}`
	if got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
	vazio := jsonDados(t, NovoIssuesPorStatus(repoProjetosFalso{}))
	if vazio != `{"fatias":[{"rotulo":"Abertas","chave":"aberta","valor":0},{"rotulo":"Em andamento","chave":"em_andamento","valor":0},{"rotulo":"Concluídas","chave":"concluida","valor":0}],"total":0}` {
		t.Errorf("vazio deveria ter as 3 fatias zeradas: %s", vazio)
	}
}

func TestProj06LimiteNoAteOuHoje(t *testing.T) {
	sprint := &models.Sprint{ID: 1, Inicio: d("2026-11-10"), Fim: d("2026-11-12")}
	repo := repoProjetosFalso{burndown: models.Burndown{Sprint: sprint, Total: 3, Dias: []models.DiaBurndown{
		{Dia: d("2026-11-10"), Restante: 3}, {Dia: d("2026-11-11"), Restante: 2}, {Dia: d("2026-11-12"), Restante: 1},
	}}}
	calc := burndown{repo: repo, hoje: func() models.Data { return d("2026-11-12") }}

	// ate anterior a hoje: o limite é ate
	r, _ := calc.Calcular(context.Background(), models.Filtros{Ate: d("2026-11-10")})
	pontos := r.Dados.(models.DadosSerieTemporal).Pontos
	if pontos[0]["restante"] != 3.0 || pontos[1]["restante"] != nil {
		t.Errorf("com ate=10/11, só o primeiro dia tem valor: %v", pontos)
	}
	// ate igual a hoje: todos os dias até hoje têm valor
	r, _ = calc.Calcular(context.Background(), models.Filtros{Ate: d("2026-11-12")})
	pontos = r.Dados.(models.DadosSerieTemporal).Pontos
	if pontos[2]["restante"] != 1.0 {
		t.Errorf("com ate=hoje, o último dia tem valor: %v", pontos)
	}
}
