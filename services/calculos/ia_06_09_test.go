package calculos

import (
	"context"
	"fmt"
	"testing"

	"pulso-api/models"
)

func (r repoIAFalso) CustoPorProjeto(context.Context, models.Filtros) ([]models.GrupoCusto, error) {
	return r.porProjeto, r.err
}

func (r repoIAFalso) CustoPorSprint(context.Context, models.Filtros) ([]models.GrupoCusto, error) {
	return r.porSprint, r.err
}

func (r repoIAFalso) CustoPorModelo(context.Context, models.Filtros) ([]models.GrupoCusto, error) {
	return r.porModelo, r.err
}

func (r repoIAFalso) CustoPorEtapa(context.Context, models.Filtros) ([]models.GrupoCusto, error) {
	return r.porEtapa, r.err
}

func (r repoIAFalso) CustoTipoComplexidade(context.Context, models.Filtros) ([]models.CustoTipoComplexidade, error) {
	return r.tipoCx, r.err
}

func TestIA06CustoPorProjetoOuSprint(t *testing.T) {
	var projetos []models.GrupoCusto
	for i := 12; i >= 1; i-- {
		projetos = append(projetos, models.GrupoCusto{Rotulo: fmt.Sprintf("P%02d", i), Custo: float64(i)})
	}
	repo := repoIAFalso{
		porProjeto: projetos,
		porSprint:  []models.GrupoCusto{{Rotulo: "Sprint 4 · Time Beta", Custo: 30.123}, {Rotulo: "Sem sprint", Custo: 5}},
	}

	// Sem id: por projeto, top 10
	r, _ := NovoCustoPorProjeto(repo).Calcular(context.Background(), filtrosIA())
	itens := r.Dados.(models.DadosBarras).Itens
	if len(itens) != 10 || itens[0].Rotulo != "P12" || itens[9].Rotulo != "P03" {
		t.Errorf("esperado top 10 projetos (P12..P03): %d itens, %v .. %v", len(itens), itens[0].Rotulo, itens[len(itens)-1].Rotulo)
	}

	// escopo=projeto com id: por sprint
	f := filtrosIA()
	id := int64(4)
	f.ID = &id
	if got := calcularComFiltros(t, NovoCustoPorProjeto(repo), f); got != `{"itens":[{"rotulo":"Sprint 4 · Time Beta","valor":30.12,"extra":null},{"rotulo":"Sem sprint","valor":5,"extra":null}],"unidade":"R$","horizontal":true,"total":null}` {
		t.Errorf("por sprint: %s", got)
	}

	// escopo=time com id: continua por projeto
	f.Escopo = "time"
	r, _ = NovoCustoPorProjeto(repo).Calcular(context.Background(), f)
	if r.Dados.(models.DadosBarras).Itens[0].Rotulo != "P12" {
		t.Error("em escopo=time deve agrupar por projeto")
	}
}

func TestIA07CustoPorModelo(t *testing.T) {
	repo := repoIAFalso{porModelo: []models.GrupoCusto{
		{Chave: "claude-sonnet", Rotulo: "claude-sonnet", Custo: 70.004, Tokens: 2000000},
		{Chave: "gpt-4o-mini", Rotulo: "gpt-4o-mini", Custo: 2.5, Tokens: 900000},
	}}
	esperado := `{"fatias":[{"rotulo":"claude-sonnet","chave":"claude-sonnet","valor":70,"tokens":2000000},{"rotulo":"gpt-4o-mini","chave":"gpt-4o-mini","valor":2.5,"tokens":900000}],"total":72.5}`
	if got := calcularComFiltros(t, NovoCustoPorModelo(repo), filtrosIA()); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
}

func TestIA08CustoPorEtapa(t *testing.T) {
	repo := repoIAFalso{porEtapa: []models.GrupoCusto{{Chave: "desenvolvimento", Custo: 50}, {Chave: "analise", Custo: 20.555}}}
	esperado := `{"itens":[{"rotulo":"Análise","valor":20.56,"extra":null},{"rotulo":"Desenvolvimento","valor":50,"extra":null},{"rotulo":"Testes","valor":0,"extra":null}],"unidade":"R$","horizontal":false,"total":null}`
	if got := calcularComFiltros(t, NovoCustoPorEtapa(repo), filtrosIA()); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
}

func TestIA09CustoTipoComplexidade(t *testing.T) {
	repo := repoIAFalso{tipoCx: []models.CustoTipoComplexidade{
		{Tipo: "correcao", Complexidade: "pequena", Custo: 6, Atividades: 2},
		{Tipo: "nova_funcionalidade", Complexidade: "grande", Custo: 30, Atividades: 3},
		{Tipo: "nova_funcionalidade", Complexidade: "media", Custo: 10, Atividades: 4},
	}}
	esperado := `{"series":[{"chave":"pequena","nome":"Pequena"},{"chave":"media","nome":"Média"},{"chave":"grande","nome":"Grande"}],"itens":[` +
		`{"grande":10,"media":2.5,"pequena":null,"rotulo":"Nova funcionalidade"},` +
		`{"grande":null,"media":null,"pequena":3,"rotulo":"Correção"}],"unidade":"R$"}`
	if got := calcularComFiltros(t, NovoCustoTipoComplexidade(repo), filtrosIA()); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
}
