package calculos

import (
	"context"
	"testing"

	"pulso-api/models"
)

func (r repoVendasFalso) Pipeline(_ context.Context, f models.Filtros) ([]models.EtapaPipeline, error) {
	r.anota(f)
	return r.pipeline, r.err
}

func (r repoVendasFalso) MotivosPerda(_ context.Context, f models.Filtros) ([]models.GrupoContagem, error) {
	r.anota(f)
	return r.motivos, r.err
}

func (r repoVendasFalso) OrigemLeads(_ context.Context, f models.Filtros) ([]models.OrigemLeads, error) {
	r.anota(f)
	return r.origens, r.err
}

func (r repoVendasFalso) ReunioesAteFechar(_ context.Context, f models.Filtros) ([]models.ReunioesDesfecho, error) {
	r.anota(f)
	return r.reunioes, r.err
}

func TestVen06Pipeline(t *testing.T) {
	var recebidos []models.Filtros
	repo := repoVendasFalso{recebidos: &recebidos, pipeline: []models.EtapaPipeline{
		{Etapa: "proposta_enviada", Bruto: 45000, Ponderado: 13500},
		{Etapa: "negociacao", Bruto: 168360, Ponderado: 101016.004},
	}}
	esperado := `{"series":[{"chave":"bruto","nome":"Bruto"},{"chave":"ponderado","nome":"Ponderado"}],"itens":[` +
		`{"bruto":45000,"ponderado":13500,"rotulo":"Proposta enviada"},{"bruto":168360,"ponderado":101016,"rotulo":"Negociação"}],"unidade":"R$"}`
	if got := calcularComFiltros(t, NovoPipeline(repo), filtrosVendas()); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
	if recebidos[0].TipoProjeto == nil {
		t.Error("ven-06 usa orçamentos: tipo_projeto deve valer")
	}
}

func TestVen07MotivosPerda(t *testing.T) {
	var recebidos []models.Filtros
	repo := repoVendasFalso{recebidos: &recebidos, motivos: []models.GrupoContagem{{Rotulo: "Preço acima do orçamento", N: 2}, {Rotulo: "Prazo de entrega", N: 1}}}
	esperado := `{"fatias":[{"rotulo":"Preço acima do orçamento","chave":"Preço acima do orçamento","valor":2},{"rotulo":"Prazo de entrega","chave":"Prazo de entrega","valor":1}],"total":3}`
	if got := calcularComFiltros(t, NovoMotivosPerda(repo), filtrosVendas()); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
	if recebidos[0].TipoProjeto != nil {
		t.Error("ven-07 é de funil: tipo_projeto não pode valer")
	}
	if got := calcularComFiltros(t, NovoMotivosPerda(repoVendasFalso{}), filtrosVendas()); got != `{"fatias":[],"total":0}` {
		t.Errorf("sem perdas: %s", got)
	}
}

func TestVen08OrigemLeads(t *testing.T) {
	repo := repoVendasFalso{origens: []models.OrigemLeads{
		{Origem: "site", Leads: 5, Ganhas: 1, Perdidas: 2},
		{Origem: "evento", Leads: 2}, // nenhum encerrado → extra null
	}}
	esperado := `{"itens":[{"rotulo":"site","valor":5,"extra":"33,3%"},{"rotulo":"evento","valor":2,"extra":null}],"unidade":"leads","horizontal":false,"total":null}`
	if got := calcularComFiltros(t, NovoOrigemLeads(repo), filtrosVendas()); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
}

func TestVen09ReunioesAteFechar(t *testing.T) {
	repo := repoVendasFalso{reunioes: []models.ReunioesDesfecho{
		{Desfecho: "ganha", Media: Ptr(2.333), N: 3},
		{Desfecho: "perdida", N: 0},
	}}
	esperado := `{"itens":[{"rotulo":"Ganhas","valor":2.3,"extra":{"n":3}},{"rotulo":"Perdidas","valor":null,"extra":{"n":0}}],"unidade":"reuniões","horizontal":false,"total":null}`
	if got := calcularComFiltros(t, NovoReunioesAteFechar(repo), filtrosVendas()); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
}
