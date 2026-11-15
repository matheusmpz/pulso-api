package calculos

import (
	"context"
	"testing"

	"pulso-api/models"
)

// repoVendasFalso devolve números fixos e registra os filtros recebidos.
type repoVendasFalso struct {
	receitas  map[string]models.ResumoReceita // chave: "de|ate"
	conversao models.ResumoConversao
	ciclo     models.ResumoCiclo
	funil     []models.EtapaFunil
	recebidos *[]models.Filtros
	err       error
}

func (r repoVendasFalso) anota(f models.Filtros) {
	if r.recebidos != nil {
		*r.recebidos = append(*r.recebidos, f)
	}
}

func (r repoVendasFalso) Receita(_ context.Context, f models.Filtros) (models.ResumoReceita, error) {
	r.anota(f)
	return r.receitas[f.De.String()+"|"+f.Ate.String()], r.err
}

func (r repoVendasFalso) Conversao(_ context.Context, f models.Filtros) (models.ResumoConversao, error) {
	r.anota(f)
	return r.conversao, r.err
}

func (r repoVendasFalso) Ciclo(_ context.Context, f models.Filtros) (models.ResumoCiclo, error) {
	r.anota(f)
	return r.ciclo, r.err
}

func (r repoVendasFalso) Funil(_ context.Context, f models.Filtros) ([]models.EtapaFunil, error) {
	r.anota(f)
	return r.funil, r.err
}

func filtrosVendas() models.Filtros {
	return models.Filtros{Dashboard: "vendas", De: d("2026-11-11"), Ate: d("2026-11-20"), TipoProjeto: []string{"web"}}
}

func calcularComFiltros(t *testing.T, c Calculador, f models.Filtros) string {
	t.Helper()
	r, err := c.Calcular(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	return jsonDe(t, r.Dados)
}

func TestVen01ReceitaComVariacao(t *testing.T) {
	var recebidos []models.Filtros
	repo := repoVendasFalso{recebidos: &recebidos, receitas: map[string]models.ResumoReceita{
		"2026-11-11|2026-11-20": {Valor: 110000, Vendas: 3},
		"2026-11-01|2026-11-10": {Valor: 100000, Vendas: 2}, // período anterior (10 dias)
	}}
	got := calcularComFiltros(t, NovoReceitaFechada(repo), filtrosVendas())
	esperado := `{"valor":110000,"unidade":"R$","variacao":{"valor":10,"unidade":"%"},"detalhe":[{"rotulo":"Vendas","valor":3,"unidade":"vendas"}]}`
	if got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
	if len(recebidos) != 2 || recebidos[1].De.String() != "2026-11-01" || recebidos[1].Ate.String() != "2026-11-10" {
		t.Errorf("deveria consultar o período anterior 01/11 a 10/11: %+v", recebidos)
	}
	if len(recebidos[0].TipoProjeto) != 1 {
		t.Error("ven-01 usa vendas: tipo_projeto deve valer")
	}

	// Anterior zero → variação null
	got = calcularComFiltros(t, NovoReceitaFechada(repoVendasFalso{receitas: map[string]models.ResumoReceita{"2026-11-11|2026-11-20": {Valor: 5, Vendas: 1}}}), filtrosVendas())
	if got != `{"valor":5,"unidade":"R$","variacao":{"valor":null,"unidade":"%"},"detalhe":[{"rotulo":"Vendas","valor":1,"unidade":"vendas"}]}` {
		t.Errorf("variação com anterior 0: %s", got)
	}
}

func TestVen02a04(t *testing.T) {
	var recebidos []models.Filtros
	repo := repoVendasFalso{
		recebidos: &recebidos,
		receitas:  map[string]models.ResumoReceita{"2026-11-11|2026-11-20": {Valor: 100000, Vendas: 3}},
		conversao: models.ResumoConversao{Ganhas: 1, Perdidas: 2},
		ciclo:     models.ResumoCiclo{MediaDias: Ptr(19.96), Ganhas: 1},
	}
	casos := []struct {
		calc     Calculador
		esperado string
		semTipo  bool
	}{
		{NovoTaxaConversao(repo), `{"valor":33.3,"unidade":"%","variacao":null,"detalhe":[{"rotulo":"Ganhas","valor":1,"unidade":"oportunidades"},{"rotulo":"Perdidas","valor":2,"unidade":"oportunidades"}]}`, true},
		{NovoTicketMedio(repo), `{"valor":33333.33,"unidade":"R$","variacao":null,"detalhe":[{"rotulo":"Vendas","valor":3,"unidade":"vendas"}]}`, false},
		{NovoCicloVenda(repo), `{"valor":20,"unidade":"dias","variacao":null,"detalhe":[{"rotulo":"Ganhas","valor":1,"unidade":"oportunidades"}]}`, true},
	}
	for _, c := range casos {
		recebidos = nil
		if got := calcularComFiltros(t, c.calc, filtrosVendas()); got != c.esperado {
			t.Errorf("%s:\nesperado %s\nobtido   %s", c.calc.Codigo(), c.esperado, got)
		}
		if c.semTipo && recebidos[0].TipoProjeto != nil {
			t.Errorf("%s é de funil: tipo_projeto não pode valer", c.calc.Codigo())
		}
	}

	vazio := repoVendasFalso{}
	for _, c := range []struct {
		calc     Calculador
		esperado string
	}{
		{NovoTaxaConversao(vazio), `{"valor":null,"unidade":"%","variacao":null,"detalhe":[{"rotulo":"Ganhas","valor":0,"unidade":"oportunidades"},{"rotulo":"Perdidas","valor":0,"unidade":"oportunidades"}]}`},
		{NovoTicketMedio(vazio), `{"valor":null,"unidade":"R$","variacao":null,"detalhe":[{"rotulo":"Vendas","valor":0,"unidade":"vendas"}]}`},
		{NovoCicloVenda(vazio), `{"valor":null,"unidade":"dias","variacao":null,"detalhe":[{"rotulo":"Ganhas","valor":0,"unidade":"oportunidades"}]}`},
	} {
		if got := calcularComFiltros(t, c.calc, filtrosVendas()); got != c.esperado {
			t.Errorf("%s vazio:\nesperado %s\nobtido   %s", c.calc.Codigo(), c.esperado, got)
		}
	}
}

func TestVen05Funil(t *testing.T) {
	etapas := []models.EtapaFunil{
		{Etapa: "lead", Oportunidades: 9}, {Etapa: "qualificacao", Oportunidades: 6},
		{Etapa: "reuniao", Oportunidades: 4}, {Etapa: "proposta_enviada", Oportunidades: 0},
		{Etapa: "negociacao", Oportunidades: 1}, {Etapa: "ganha", Oportunidades: 1},
	}
	esperado := `{"itens":[{"rotulo":"Lead","valor":9,"extra":null},{"rotulo":"Qualificação","valor":6,"extra":"66,7%"},` +
		`{"rotulo":"Reunião","valor":4,"extra":"66,7%"},{"rotulo":"Proposta enviada","valor":0,"extra":"0,0%"},` +
		`{"rotulo":"Negociação","valor":1,"extra":null},{"rotulo":"Ganha","valor":1,"extra":"100,0%"}],"unidade":"oportunidades","horizontal":true,"total":null}`
	if got := calcularComFiltros(t, NovoFunilVendas(repoVendasFalso{funil: etapas}), filtrosVendas()); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
}
