package calculos

import (
	"context"

	"pulso-api/models"
	"pulso-api/repositories"
)

// limiteTopProjetosIA: quantos projetos o ia-06 mostra.
const limiteTopProjetosIA = 10

// ---------------------------------------------------------------------
// ia-06 · Custo por projeto · barras (horizontal) · Período
// Top 10 projetos de P por custo de IA em T (R$). Com escopo=projeto e
// id informado, agrupa por sprint (sem atividade ou de backlog = "Sem sprint").
// ---------------------------------------------------------------------

type custoPorProjeto struct {
	repo repositories.MetricasIARepository
}

// NovoCustoPorProjeto cria o calculador do ia-06.
func NovoCustoPorProjeto(repo repositories.MetricasIARepository) Calculador {
	return custoPorProjeto{repo}
}

func (custoPorProjeto) Codigo() string     { return "ia-06" }
func (custoPorProjeto) Gatilhos() []string { return gatilhosPorCard["ia-06"] }

func (c custoPorProjeto) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	var grupos []models.GrupoCusto
	var err error
	if f.Escopo == models.EscopoProjeto && f.ID != nil {
		grupos, err = c.repo.CustoPorSprint(ctx, f)
	} else {
		grupos, err = c.repo.CustoPorProjeto(ctx, f)
	}
	if err != nil {
		return models.CardResultado{}, err
	}
	itens := []models.ItemBarra{}
	for i, g := range grupos {
		if i >= limiteTopProjetosIA {
			break
		}
		itens = append(itens, models.ItemBarra{Rotulo: g.Rotulo, Valor: PtrArredondado(g.Custo, CasasValor)})
	}
	return models.CardResultado{Dados: models.DadosBarras{Itens: itens, Unidade: "R$", Horizontal: true}}, nil
}

// ---------------------------------------------------------------------
// ia-07 · Custo por modelo · donut · Período
// Fatias = modelo; valor = custo (R$); chave extra tokens; total = custo total.
// ---------------------------------------------------------------------

type custoPorModelo struct {
	repo repositories.MetricasIARepository
}

// NovoCustoPorModelo cria o calculador do ia-07.
func NovoCustoPorModelo(repo repositories.MetricasIARepository) Calculador {
	return custoPorModelo{repo}
}

func (custoPorModelo) Codigo() string     { return "ia-07" }
func (custoPorModelo) Gatilhos() []string { return gatilhosPorCard["ia-07"] }

func (c custoPorModelo) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	grupos, err := c.repo.CustoPorModelo(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	fatias := []models.Fatia{}
	total := 0.0
	for _, g := range grupos {
		tokens := float64(g.Tokens)
		fatias = append(fatias, models.Fatia{Rotulo: g.Rotulo, Chave: g.Chave, Valor: Arredondar(g.Custo, CasasValor), Tokens: &tokens})
		total += g.Custo
	}
	return models.CardResultado{Dados: models.DadosDonut{Fatias: fatias, Total: Arredondar(total, CasasValor)}}, nil
}

// ---------------------------------------------------------------------
// ia-08 · Custo por etapa · barras · Período
// Sempre 3 itens (Análise, Desenvolvimento, Testes): custo de T por etapa.
// ---------------------------------------------------------------------

type custoPorEtapa struct {
	repo repositories.MetricasIARepository
}

// NovoCustoPorEtapa cria o calculador do ia-08.
func NovoCustoPorEtapa(repo repositories.MetricasIARepository) Calculador {
	return custoPorEtapa{repo}
}

func (custoPorEtapa) Codigo() string     { return "ia-08" }
func (custoPorEtapa) Gatilhos() []string { return gatilhosPorCard["ia-08"] }

func (c custoPorEtapa) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	grupos, err := c.repo.CustoPorEtapa(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	porEtapa := map[string]float64{}
	for _, g := range grupos {
		porEtapa[g.Chave] = g.Custo
	}
	itens := []models.ItemBarra{}
	for _, e := range models.EtapasTrabalho {
		itens = append(itens, models.ItemBarra{Rotulo: e.Rotulo, Valor: PtrArredondado(porEtapa[e.Valor], CasasValor)})
	}
	return models.CardResultado{Dados: models.DadosBarras{Itens: itens, Unidade: "R$"}}, nil
}

// ---------------------------------------------------------------------
// ia-09 · Custo por tipo e complexidade · barras_agrupadas · Período
// Base: atividades de A com consumo no período. Categoria = tipo (só tipos
// com dados); séries pequena, media, grande; valor = custo no período ÷
// atividades distintas do grupo (null se vazio).
// ---------------------------------------------------------------------

type custoTipoComplexidade struct {
	repo repositories.MetricasIARepository
}

// NovoCustoTipoComplexidade cria o calculador do ia-09.
func NovoCustoTipoComplexidade(repo repositories.MetricasIARepository) Calculador {
	return custoTipoComplexidade{repo}
}

func (custoTipoComplexidade) Codigo() string     { return "ia-09" }
func (custoTipoComplexidade) Gatilhos() []string { return gatilhosPorCard["ia-09"] }

func (c custoTipoComplexidade) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	grupos, err := c.repo.CustoTipoComplexidade(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: MontarCustoTipoComplexidade(grupos)}, nil
}

// MontarCustoTipoComplexidade monta um item por tipo (na ordem do enum, só
// os que têm dados) com uma chave por complexidade.
func MontarCustoTipoComplexidade(grupos []models.CustoTipoComplexidade) models.DadosBarrasAgrupadas {
	porTipo := map[string]map[string]models.CustoTipoComplexidade{}
	for _, g := range grupos {
		if porTipo[g.Tipo] == nil {
			porTipo[g.Tipo] = map[string]models.CustoTipoComplexidade{}
		}
		porTipo[g.Tipo][g.Complexidade] = g
	}

	itens := []map[string]interface{}{}
	for _, tipo := range models.TiposAtividade {
		cx, ok := porTipo[tipo.Valor]
		if !ok {
			continue
		}
		item := map[string]interface{}{"rotulo": tipo.Rotulo}
		for _, comp := range models.Complexidades {
			var valor interface{}
			if g, ok := cx[comp.Valor]; ok && g.Atividades > 0 {
				valor = Arredondar(g.Custo/float64(g.Atividades), CasasValor)
			}
			item[comp.Valor] = valor
		}
		itens = append(itens, item)
	}

	series := []models.Serie{}
	for _, comp := range models.Complexidades {
		series = append(series, models.Serie{Chave: comp.Valor, Nome: comp.Rotulo})
	}
	return models.DadosBarrasAgrupadas{Series: series, Itens: itens, Unidade: "R$"}
}
