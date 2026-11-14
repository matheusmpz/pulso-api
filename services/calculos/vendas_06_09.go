package calculos

import (
	"context"

	"pulso-api/models"
	"pulso-api/repositories"
)

// ---------------------------------------------------------------------
// ven-06 · Pipeline em aberto · barras_agrupadas · Acumulado (foto até ate)
// Por etapa atual: bruto (soma do orçamento de maior versão não recusado)
// e ponderado (× probabilidade fixa da etapa). Unidade R$.
// ---------------------------------------------------------------------

type pipeline struct {
	repo repositories.MetricasVendasRepository
}

// NovoPipeline cria o calculador do ven-06.
func NovoPipeline(repo repositories.MetricasVendasRepository) Calculador {
	return pipeline{repo}
}

func (pipeline) Codigo() string     { return "ven-06" }
func (pipeline) Gatilhos() []string { return gatilhosPorCard["ven-06"] }

func (c pipeline) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	etapas, err := c.repo.Pipeline(ctx, f) // usa orçamentos: tipo_projeto vale
	if err != nil {
		return models.CardResultado{}, err
	}
	itens := []map[string]interface{}{}
	for _, e := range etapas {
		itens = append(itens, map[string]interface{}{
			"rotulo":    models.RotuloEnum(models.EtapasFunil, e.Etapa),
			"bruto":     Arredondar(e.Bruto, CasasValor),
			"ponderado": Arredondar(e.Ponderado, CasasValor),
		})
	}
	return models.CardResultado{Dados: models.DadosBarrasAgrupadas{
		Series: []models.Serie{
			{Chave: "bruto", Nome: "Bruto"},
			{Chave: "ponderado", Nome: "Ponderado"},
		},
		Itens:   itens,
		Unidade: "R$",
	}}, nil
}

// ---------------------------------------------------------------------
// ven-07 · Motivos de perda · donut · Período
// Oportunidades com evento perdida no período, por motivo_perda.
// ---------------------------------------------------------------------

type motivosPerda struct {
	repo repositories.MetricasVendasRepository
}

// NovoMotivosPerda cria o calculador do ven-07.
func NovoMotivosPerda(repo repositories.MetricasVendasRepository) Calculador {
	return motivosPerda{repo}
}

func (motivosPerda) Codigo() string     { return "ven-07" }
func (motivosPerda) Gatilhos() []string { return gatilhosPorCard["ven-07"] }

func (c motivosPerda) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	grupos, err := c.repo.MotivosPerda(ctx, semTipo(f))
	if err != nil {
		return models.CardResultado{}, err
	}
	fatias := []models.Fatia{}
	total := 0
	for _, g := range grupos {
		fatias = append(fatias, models.Fatia{Rotulo: g.Rotulo, Chave: g.Rotulo, Valor: float64(g.N)})
		total += g.N
	}
	return models.CardResultado{Dados: models.DadosDonut{Fatias: fatias, Total: float64(total)}}, nil
}

// ---------------------------------------------------------------------
// ven-08 · Origem dos leads · barras · Período
// Leads criados no período por origem (ordem decrescente); extra = taxa
// de conversão da origem entre os encerrados ("40,0%"; null se nenhum).
// ---------------------------------------------------------------------

type origemLeads struct {
	repo repositories.MetricasVendasRepository
}

// NovoOrigemLeads cria o calculador do ven-08.
func NovoOrigemLeads(repo repositories.MetricasVendasRepository) Calculador {
	return origemLeads{repo}
}

func (origemLeads) Codigo() string     { return "ven-08" }
func (origemLeads) Gatilhos() []string { return gatilhosPorCard["ven-08"] }

func (c origemLeads) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	origens, err := c.repo.OrigemLeads(ctx, semTipo(f))
	if err != nil {
		return models.CardResultado{}, err
	}
	itens := []models.ItemBarra{}
	for _, o := range origens {
		var extra interface{}
		if p := Percentual(float64(o.Ganhas), float64(o.Ganhas+o.Perdidas)); p != nil {
			extra = FormatarPercentual(*p)
		}
		itens = append(itens, models.ItemBarra{Rotulo: o.Origem, Valor: Ptr(float64(o.Leads)), Extra: extra})
	}
	return models.CardResultado{Dados: models.DadosBarras{Itens: itens, Unidade: "leads"}}, nil
}

// ---------------------------------------------------------------------
// ven-09 · Reuniões até fechar · barras · Período
// Oportunidades encerradas no período; dois itens, Ganhas e Perdidas:
// média de reuniões por oportunidade (1 casa; null se não houver).
// ---------------------------------------------------------------------

type reunioesAteFechar struct {
	repo repositories.MetricasVendasRepository
}

// NovoReunioesAteFechar cria o calculador do ven-09.
func NovoReunioesAteFechar(repo repositories.MetricasVendasRepository) Calculador {
	return reunioesAteFechar{repo}
}

func (reunioesAteFechar) Codigo() string     { return "ven-09" }
func (reunioesAteFechar) Gatilhos() []string { return gatilhosPorCard["ven-09"] }

func (c reunioesAteFechar) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	grupos, err := c.repo.ReunioesAteFechar(ctx, semTipo(f))
	if err != nil {
		return models.CardResultado{}, err
	}
	porDesfecho := map[string]models.ReunioesDesfecho{}
	for _, g := range grupos {
		porDesfecho[g.Desfecho] = g
	}
	itens := []models.ItemBarra{}
	for _, d := range []struct{ chave, rotulo string }{{"ganha", "Ganhas"}, {"perdida", "Perdidas"}} {
		g := porDesfecho[d.chave]
		var valor *float64
		if g.Media != nil && g.N > 0 {
			valor = PtrArredondado(*g.Media, 1)
		}
		itens = append(itens, models.ItemBarra{Rotulo: d.rotulo, Valor: valor, Extra: map[string]interface{}{"n": g.N}})
	}
	return models.CardResultado{Dados: models.DadosBarras{Itens: itens, Unidade: "reuniões"}}, nil
}
