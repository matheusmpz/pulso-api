package calculos

import (
	"context"
	"fmt"
	"testing"

	"pulso-api/models"
)

func (r repoIAFalso) IntensidadeDesempenho(context.Context, models.Filtros) ([]models.FaixaIntensidade, error) {
	return r.faixas, r.err
}

func (r repoIAFalso) PesoIA(context.Context, models.Filtros) ([]models.PesoIAProjeto, error) {
	return r.peso, r.err
}

func (r repoIAFalso) AlertasQuota(_ context.Context, _ models.Filtros, limiar float64) ([]models.AlertaQuota, error) {
	if limiar != LimiarQuotaAtencao {
		return nil, fmt.Errorf("limiar inesperado: %v", limiar)
	}
	return r.quotas, r.err
}

func (r repoIAFalso) AlertasPico(_ context.Context, _ models.Filtros, fator float64, janela int) ([]models.AlertaPico, error) {
	if fator != FatorPico || janela != JanelaMediaDias {
		return nil, fmt.Errorf("fator/janela inesperados: %v %v", fator, janela)
	}
	return r.picos, r.err
}

func TestIA10Intensidade(t *testing.T) {
	repo := repoIAFalso{faixas: []models.FaixaIntensidade{
		{Complexidade: "media", Faixa: 1, DesvioMedio: 95.04, N: 3},
		{Complexidade: "media", Faixa: 3, DesvioMedio: 120, N: 3},
		{Complexidade: "grande", Faixa: 2, DesvioMedio: 130, N: 1},
	}}
	esperado := `{"series":[{"chave":"pequena","nome":"Pequena"},{"chave":"media","nome":"Média"},{"chave":"grande","nome":"Grande"}],"itens":[` +
		`{"grande":null,"media":95,"n_grande":0,"n_media":3,"n_pequena":0,"pequena":null,"rotulo":"Baixa"},` +
		`{"grande":130,"media":null,"n_grande":1,"n_media":0,"n_pequena":0,"pequena":null,"rotulo":"Média"},` +
		`{"grande":null,"media":120,"n_grande":0,"n_media":3,"n_pequena":0,"pequena":null,"rotulo":"Alta"}],` +
		`"unidade":"%","referencia":100,"amostra_minima":5}`
	if got := calcularComFiltros(t, NovoIntensidadeDesempenho(repo), filtrosIA()); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
}

func TestIA11PesoIA(t *testing.T) {
	repo := repoIAFalso{peso: []models.PesoIAProjeto{
		{Projeto: "Portal", CustoHoras: 9000, CustoIA: 1000},
		{Projeto: "Sem horas", CustoIA: 5},
	}}
	esperado := `{"series":[{"chave":"custo_horas","nome":"Custo das horas","pilha":"custo"},{"chave":"custo_ia","nome":"Custo de IA","pilha":"custo"}],"itens":[` +
		`{"custo_horas":9000,"custo_ia":1000,"percentual_ia":10,"rotulo":"Portal"},` +
		`{"custo_horas":0,"custo_ia":5,"percentual_ia":100,"rotulo":"Sem horas"}],"unidade":"R$"}`
	if got := calcularComFiltros(t, NovoPesoIA(repo), filtrosIA()); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
}

func TestIA12Alertas(t *testing.T) {
	quotas := []models.AlertaQuota{
		{Projeto: "App", Percentual: 83.6, Quando: d("2026-11-05")},
		{Projeto: "API", Percentual: 144.06, Quando: d("2026-10-30")},
	}
	picos := []models.AlertaPico{
		{Projeto: "API", Dia: d("2026-11-11"), Razao: 9.14},
		{Projeto: "App", Dia: d("2026-10-30"), Razao: 33.18},
	}
	esperado := `{"colunas":["Projeto","Alerta","Valor","Quando"],"linhas":[` +
		`["API","Quota crítica","144,1%","2026-10-30"],` +
		`["App","Quota em atenção","83,6%","2026-11-05"],` +
		`["App","Pico","33,2x","2026-10-30"],` +
		`["API","Pico","9,1x","2026-11-11"]],"total":4}`
	repo := repoIAFalso{quotas: quotas, picos: picos}
	if got := calcularComFiltros(t, NovoAlertasConsumo(repo), filtrosIA()); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}

	// Limite de 20 linhas com total antes do limite
	var muitos []models.AlertaPico
	for i := 0; i < 25; i++ {
		muitos = append(muitos, models.AlertaPico{Projeto: "P", Dia: d("2026-11-01"), Razao: float64(30 - i)})
	}
	dados := MontarAlertas(nil, muitos)
	if len(dados.Linhas) != 20 || dados.Total != 25 || dados.Linhas[0][2] != "30,0x" {
		t.Errorf("limite: %d linhas, total %d, primeira %v", len(dados.Linhas), dados.Total, dados.Linhas[0])
	}

	if got := calcularComFiltros(t, NovoAlertasConsumo(repoIAFalso{}), filtrosIA()); got != `{"colunas":["Projeto","Alerta","Valor","Quando"],"linhas":[],"total":0}` {
		t.Errorf("sem alertas: %s", got)
	}
}

func TestOs41CardsImplementados(t *testing.T) {
	reg := NovoRegistro(nil)
	if len(reg) != 41 {
		t.Fatalf("esperados 41 cards, obtidos %d", len(reg))
	}
	for codigo, calc := range reg {
		if _, pendenteAinda := calc.(pendente); pendenteAinda {
			t.Errorf("%s ainda está pendente", codigo)
		}
		if calc.Codigo() != codigo {
			t.Errorf("chave %s aponta para %s", codigo, calc.Codigo())
		}
	}
}
