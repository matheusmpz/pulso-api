package calculos

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"pulso-api/models"
)

func (r repoProjetosFalso) Estouros(context.Context, models.Filtros) (models.Estouros, error) {
	return r.estouros, r.err
}

func (r repoProjetosFalso) HistoricoSprints(context.Context, models.Filtros) ([]models.SprintHistorico, error) {
	return r.historico, r.err
}

func (r repoProjetosFalso) DistribuicaoHoras(context.Context, models.Filtros) ([]models.GrupoHoras, error) {
	return r.distribuicao, r.err
}

func (r repoIAFalso) ConsumoNoTempo(context.Context, models.Filtros) (models.ConsumoNoTempo, error) {
	return r.noTempo, r.err
}

func TestProj12Estouros(t *testing.T) {
	repo := repoProjetosFalso{estouros: models.Estouros{Total: 6, Top: []models.AtividadeEstourada{
		{Titulo: "Catálogo de produtos", Projeto: "Portal", Estimativa: 48, Reais: 67.214},
		{Titulo: "Correção de fuso", Projeto: "App", Estimativa: 8, Reais: 14.4},
	}}}
	esperado := `{"itens":[{"rotulo":"Catálogo de produtos","valor":19.21,"extra":{"desvio":"140%","projeto":"Portal"}},` +
		`{"rotulo":"Correção de fuso","valor":6.4,"extra":{"desvio":"180%","projeto":"App"}}],"unidade":"h","horizontal":true,"total":6}`
	if got := jsonDados(t, NovoEstouros(repo)); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
	if got := jsonDados(t, NovoEstouros(repoProjetosFalso{})); got != `{"itens":[],"unidade":"h","horizontal":true,"total":0}` {
		t.Errorf("sem estouros: %s", got)
	}
}

func TestProj13HistoricoSprints(t *testing.T) {
	um := []models.SprintHistorico{
		{Sprint: models.Sprint{ID: 3, TimeID: 1, TimeNome: "Alfa", Numero: 3}, Planejado: 52, Concluido: 40.004},
		{Sprint: models.Sprint{ID: 5, TimeID: 1, TimeNome: "Alfa", Numero: 4}, Planejado: 86, Concluido: 56, Destaque: true},
	}
	esperado := `{"series":[{"chave":"planejado","nome":"Planejado"},{"chave":"concluido","nome":"Concluído"}],"itens":[` +
		`{"concluido":40,"destaque":false,"planejado":52,"rotulo":"Sprint 3","sprint_id":3},` +
		`{"concluido":56,"destaque":true,"planejado":86,"rotulo":"Sprint 4","sprint_id":5}],"unidade":"h"}`
	if got := jsonDados(t, NovoHistoricoSprints(repoProjetosFalso{historico: um})); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}

	// Dois times na lista: o rótulo leva o nome do time
	dois := append(um, models.SprintHistorico{Sprint: models.Sprint{ID: 6, TimeID: 2, TimeNome: "Beta", Numero: 4}})
	dados := MontarHistoricoSprints(dois)
	if dados.Itens[0]["rotulo"] != "Sprint 3 · Alfa" || dados.Itens[2]["rotulo"] != "Sprint 4 · Beta" {
		t.Errorf("rótulos com dois times: %v %v", dados.Itens[0]["rotulo"], dados.Itens[2]["rotulo"])
	}
}

func TestProj14TopComOutros(t *testing.T) {
	var grupos []models.GrupoHoras
	for i := 12; i >= 1; i-- {
		grupos = append(grupos, models.GrupoHoras{Rotulo: fmt.Sprintf("Dev %02d", i), Horas: float64(i)})
	}
	dados := MontarTopComOutros(grupos, 10, "h")
	if len(dados.Itens) != 11 || dados.Itens[10].Rotulo != "Outros" || *dados.Itens[10].Valor != 3 {
		t.Errorf("esperado top 10 + Outros (2+1=3): %+v", dados.Itens[10])
	}
	if !dados.Horizontal || dados.Total != nil || dados.Unidade != "h" {
		t.Errorf("metadados inesperados: %+v", dados)
	}

	poucos := MontarTopComOutros(grupos[:3], 10, "h")
	if len(poucos.Itens) != 3 {
		t.Errorf("sem passar do limite não há Outros: %d itens", len(poucos.Itens))
	}
	exatos := MontarTopComOutros(grupos[:10], 10, "h")
	if len(exatos.Itens) != 10 {
		t.Errorf("exatamente 10 grupos não geram Outros: %d itens", len(exatos.Itens))
	}
	if b, _ := json.Marshal(MontarTopComOutros(nil, 10, "h").Itens); string(b) != "[]" {
		t.Errorf("sem horas: %s", b)
	}
}

func TestProj15ConsumoNoTempo(t *testing.T) {
	repo := repoIAFalso{noTempo: models.ConsumoNoTempo{Quota: 1800000, Dias: []models.DiaConsumo{
		{Dia: d("2026-11-01"), TokensAcumulados: 1000}, {Dia: d("2026-11-02"), TokensAcumulados: 1000},
		{Dia: d("2026-11-03"), TokensAcumulados: 5500},
	}}}
	esperado := `{"series":[{"chave":"consumo","nome":"Consumo acumulado"}],"pontos":[` +
		`{"consumo":1000,"x":"2026-11-01"},{"consumo":1000,"x":"2026-11-02"},{"consumo":5500,"x":"2026-11-03"}],` +
		`"eixo_x":"Dia","eixo_y":"Tokens acumulados","unidade":"tokens","referencia":{"nome":"Limite de tokens","valor":1800000},"extra":null}`
	if got := jsonDados(t, NovoConsumoNoTempo(repo)); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
	vazio := jsonDados(t, NovoConsumoNoTempo(repoIAFalso{}))
	if vazio != `{"series":[{"chave":"consumo","nome":"Consumo acumulado"}],"pontos":[],"eixo_x":"Dia","eixo_y":"Tokens acumulados","unidade":"tokens","referencia":null,"extra":null}` {
		t.Errorf("sem consumo e sem quota: %s", vazio)
	}
}

func TestDashboardProjetosCompleto(t *testing.T) {
	reg := NovoRegistro(nil)
	for i := 1; i <= 15; i++ {
		codigo := fmt.Sprintf("proj-%02d", i)
		if _, pendenteAinda := reg[codigo].(pendente); pendenteAinda {
			t.Errorf("%s ainda está pendente", codigo)
		}
	}
}
