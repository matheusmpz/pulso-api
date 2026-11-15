package calculos

import (
	"context"
	"testing"

	"pulso-api/models"
)

func (r repoProjetosFalso) TempoMedio(_ context.Context, _ models.Filtros, agrupamento string) ([]models.GrupoTempoMedio, error) {
	return r.tempoMedio[agrupamento], r.err
}

func TestProj10TempoMedioComplexidade(t *testing.T) {
	repo := repoProjetosFalso{tempoMedio: map[string][]models.GrupoTempoMedio{
		"complexidade": {
			{Grupo: "grande", N: 2, MediaReal: 55.555, MediaEstimada: 44},
			{Grupo: "pequena", N: 3, MediaReal: 8, MediaEstimada: 7.333},
		},
	}}
	// Sempre 3 itens, na ordem Pequena, Média, Grande; Média sem atividades = null
	esperado := `{"series":[{"chave":"real","nome":"Real"},{"chave":"estimado","nome":"Estimado"}],"itens":[` +
		`{"estimado":7.33,"n":3,"real":8,"rotulo":"Pequena"},` +
		`{"estimado":null,"n":0,"real":null,"rotulo":"Média"},` +
		`{"estimado":44,"n":2,"real":55.56,"rotulo":"Grande"}],"unidade":"h"}`
	if got := jsonDados(t, NovoTempoMedioComplexidade(repo)); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}

	vazio := `{"series":[{"chave":"real","nome":"Real"},{"chave":"estimado","nome":"Estimado"}],"itens":[` +
		`{"estimado":null,"n":0,"real":null,"rotulo":"Pequena"},` +
		`{"estimado":null,"n":0,"real":null,"rotulo":"Média"},` +
		`{"estimado":null,"n":0,"real":null,"rotulo":"Grande"}],"unidade":"h"}`
	if got := jsonDados(t, NovoTempoMedioComplexidade(repoProjetosFalso{})); got != vazio {
		t.Errorf("sem atividades:\nesperado %s\nobtido   %s", vazio, got)
	}
}

func TestProj11TempoMedioTipo(t *testing.T) {
	repo := repoProjetosFalso{tempoMedio: map[string][]models.GrupoTempoMedio{
		"tipo": {
			{Grupo: "correcao", N: 2, MediaReal: 15, MediaEstimada: 10},
			{Grupo: "analise", N: 1, MediaReal: 14, MediaEstimada: 14},
		},
	}}
	// Só os tipos com atividades, na ordem do enum (análise antes de correção)
	esperado := `{"series":[{"chave":"real","nome":"Real"},{"chave":"estimado","nome":"Estimado"}],"itens":[` +
		`{"estimado":14,"n":1,"real":14,"rotulo":"Análise"},` +
		`{"estimado":10,"n":2,"real":15,"rotulo":"Correção"}],"unidade":"h"}`
	if got := jsonDados(t, NovoTempoMedioTipo(repo)); got != esperado {
		t.Errorf("\nesperado %s\nobtido   %s", esperado, got)
	}
	if got := jsonDados(t, NovoTempoMedioTipo(repoProjetosFalso{})); got != `{"series":[{"chave":"real","nome":"Real"},{"chave":"estimado","nome":"Estimado"}],"itens":[],"unidade":"h"}` {
		t.Errorf("sem atividades deveria ter itens vazio: %s", got)
	}
}
