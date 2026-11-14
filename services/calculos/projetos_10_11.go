package calculos

import (
	"context"

	"pulso-api/models"
	"pulso-api/repositories"
)

// ---------------------------------------------------------------------
// proj-10 · Tempo médio por complexidade · barras_agrupadas · Período
// proj-11 · Tempo médio por tipo de atividade · barras_agrupadas · Período
//
// Base: atividades de A cujo status atual é concluida e cuja data de
// conclusão (último evento concluida) está no período. Séries "real"
// (média das horas lançadas totais da atividade) e "estimado" (média de
// estimativa_horas); null quando o grupo não tem atividades; "n" =
// quantidade (chave extra). Unidade h.
//   proj-10: sempre 3 itens (Pequena, Média, Grande).
//   proj-11: só os tipos com atividades, na ordem do enum.
// ---------------------------------------------------------------------

type tempoMedio struct {
	codigo      string
	agrupamento string // "complexidade" ou "tipo"
	repo        repositories.MetricasProjetosRepository
}

// NovoTempoMedioComplexidade cria o calculador do proj-10.
func NovoTempoMedioComplexidade(repo repositories.MetricasProjetosRepository) Calculador {
	return tempoMedio{codigo: "proj-10", agrupamento: "complexidade", repo: repo}
}

// NovoTempoMedioTipo cria o calculador do proj-11.
func NovoTempoMedioTipo(repo repositories.MetricasProjetosRepository) Calculador {
	return tempoMedio{codigo: "proj-11", agrupamento: "tipo", repo: repo}
}

func (c tempoMedio) Codigo() string     { return c.codigo }
func (c tempoMedio) Gatilhos() []string { return gatilhosPorCard[c.codigo] }

func (c tempoMedio) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	grupos, err := c.repo.TempoMedio(ctx, f, c.agrupamento)
	if err != nil {
		return models.CardResultado{}, err
	}
	if c.agrupamento == "complexidade" {
		return models.CardResultado{Dados: MontarTempoMedio(grupos, models.Complexidades, true)}, nil
	}
	return models.CardResultado{Dados: MontarTempoMedio(grupos, models.TiposAtividade, false)}, nil
}

// MontarTempoMedio monta os itens na ordem de opcoes. Com todos = true,
// todo valor de opcoes vira item (vazio = null); senão, só os que têm
// atividades.
func MontarTempoMedio(grupos []models.GrupoTempoMedio, opcoes []models.OpcaoEnum, todos bool) models.DadosBarrasAgrupadas {
	porGrupo := map[string]models.GrupoTempoMedio{}
	for _, g := range grupos {
		porGrupo[g.Grupo] = g
	}

	itens := []map[string]interface{}{}
	for _, o := range opcoes {
		g, ok := porGrupo[o.Valor]
		if !ok && !todos {
			continue
		}
		item := map[string]interface{}{"rotulo": o.Rotulo, "real": nil, "estimado": nil, "n": 0}
		if ok && g.N > 0 {
			item["real"] = Arredondar(g.MediaReal, CasasValor)
			item["estimado"] = Arredondar(g.MediaEstimada, CasasValor)
			item["n"] = g.N
		}
		itens = append(itens, item)
	}

	return models.DadosBarrasAgrupadas{
		Series: []models.Serie{
			{Chave: "real", Nome: "Real"},
			{Chave: "estimado", Nome: "Estimado"},
		},
		Itens:   itens,
		Unidade: "h",
	}
}
