package calculos

import (
	"context"
	"fmt"
	"testing"

	"pulso-api/models"
)

func (r repoVendasFalso) ReceitaMensal(_ context.Context, f models.Filtros) ([]models.MesReceita, error) {
	r.anota(f)
	return r.mensal, r.err
}

func (r repoVendasFalso) StatusPropostas(_ context.Context, f models.Filtros) (models.StatusPropostas, error) {
	r.anota(f)
	return r.propostas, r.err
}

func (r repoVendasFalso) MargemPorComplexidade(_ context.Context, f models.Filtros) ([]models.MargemComplexidade, error) {
	r.anota(f)
	return r.margens, r.err
}

func (r repoVendasFalso) OrcadoReal(_ context.Context, f models.Filtros) ([]models.OrcadoRealTipo, error) {
	r.anota(f)
	return r.orcadoReal, r.err
}

func TestVen10ReceitaMensal(t *testing.T) {
	repo := repoVendasFalso{mensal: []models.MesReceita{
		{Mes: "2026-09", Receita: 0},
		{Mes: "2026-10", Receita: 66240, Meta: Ptr(90000)},
	}}
	esperado := `{"series":[{"chave":"receita","nome":"Receita"},{"chave":"meta","nome":"Meta"}],"pontos":[` +
		`{"meta":null,"receita":0,"x":"2026-09"},{"meta":90000,"receita":66240,"x":"2026-10"}],` +
		`"eixo_x":"Mês","eixo_y":"Receita (R$)","unidade":"R$","referencia":null,"extra":null}`
	if got := calcularComFiltros(t, NovoReceitaMensal(repo), filtrosVendas()); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
}

func TestVen11StatusPropostas(t *testing.T) {
	repo := repoVendasFalso{propostas: models.StatusPropostas{Pendentes: 2, Aprovadas: 1, Recusadas: 3, EmRevisao: 1}}
	esperado := `{"fatias":[{"rotulo":"Pendentes","chave":"pendente","valor":2},{"rotulo":"Aprovadas","chave":"aprovada","valor":1},` +
		`{"rotulo":"Recusadas","chave":"recusada","valor":3},{"rotulo":"Em revisão","chave":"em_revisao","valor":1}],"total":7}`
	if got := calcularComFiltros(t, NovoStatusPropostas(repo), filtrosVendas()); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
}

func TestVen12MargemComplexidade(t *testing.T) {
	repo := repoVendasFalso{margens: []models.MargemComplexidade{
		{Complexidade: "media", N: 2, MargemMedia: 36.25, Valor: 130000, Horas: 400},
		{Complexidade: "grande", N: 1, MargemMedia: 30, Valor: 115000, Horas: 350},
	}}
	esperado := `{"itens":[{"rotulo":"Pequena","valor":null,"extra":null},{"rotulo":"Média","valor":36.3,"extra":"R$ 325,00/h"},` +
		`{"rotulo":"Grande","valor":30,"extra":"R$ 328,57/h"}],"unidade":"%","horizontal":false,"total":null}`
	if got := calcularComFiltros(t, NovoMargemComplexidade(repo), filtrosVendas()); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
}

func TestVen13e14OrcadoReal(t *testing.T) {
	tipos := []models.OrcadoRealTipo{
		// fora da ordem do enum de propósito: o calculador ordena
		{Tipo: "integracao", HorasEstimadas: 160, HorasReais: 28.6, MargemPrevista: 12000, ValorVendido: 48000, CustoHoras: 3300, CustoIA: 5.5},
		{Tipo: "web", HorasEstimadas: 420, HorasReais: 380.25, MargemPrevista: 41000, ValorVendido: 118320, CustoHoras: 125000, CustoIA: 60},
	}
	repo := repoVendasFalso{orcadoReal: tipos}

	esperado13 := `{"series":[{"chave":"estimado","nome":"Orçado"},{"chave":"real","nome":"Real"}],"itens":[` +
		`{"estimado":420,"real":380.25,"rotulo":"Web"},{"estimado":160,"real":28.6,"rotulo":"Integração"}],"unidade":"h"}`
	if got := calcularComFiltros(t, NovoOrcadoReal(repo), filtrosVendas()); got != esperado13 {
		t.Errorf("ven-13:\nesperado %s\nobtido   %s", esperado13, got)
	}

	// web: margem real = 118320 − 125000 − 60 = −6740 (negativa é permitido)
	esperado14 := `{"series":[{"chave":"previsto","nome":"Margem prevista","pilha":"previsto"},` +
		`{"chave":"margem_real","nome":"Margem real","pilha":"real"},{"chave":"custo_ia","nome":"Custo de IA","pilha":"real"}],"itens":[` +
		`{"custo_ia":60,"margem_real":-6740,"previsto":41000,"rotulo":"Web"},` +
		`{"custo_ia":5.5,"margem_real":44694.5,"previsto":12000,"rotulo":"Integração"}],"unidade":"R$"}`
	if got := calcularComFiltros(t, NovoMargemPrevistaReal(repo), filtrosVendas()); got != esperado14 {
		t.Errorf("ven-14:\nesperado %s\nobtido   %s", esperado14, got)
	}
}

func TestDashboardVendasCompleto(t *testing.T) {
	reg := NovoRegistro(nil)
	for i := 1; i <= 14; i++ {
		codigo := fmt.Sprintf("ven-%02d", i)
		if _, pendenteAinda := reg[codigo].(pendente); pendenteAinda {
			t.Errorf("%s ainda está pendente", codigo)
		}
	}
}
