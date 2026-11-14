package calculos

import (
	"context"

	"pulso-api/models"
	"pulso-api/repositories"
)

// ---------------------------------------------------------------------
// ven-10 · Receita ao longo do tempo · area · Período em meses inteiros
// x = "AAAA-MM"; séries receita (soma das vendas do mês) e meta
// (metas_receita do mês; null se não houver). Unidade R$.
// ---------------------------------------------------------------------

type receitaMensal struct {
	repo repositories.MetricasVendasRepository
}

// NovoReceitaMensal cria o calculador do ven-10.
func NovoReceitaMensal(repo repositories.MetricasVendasRepository) Calculador {
	return receitaMensal{repo}
}

func (receitaMensal) Codigo() string     { return "ven-10" }
func (receitaMensal) Gatilhos() []string { return gatilhosPorCard["ven-10"] }

func (c receitaMensal) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	meses, err := c.repo.ReceitaMensal(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	pontos := []map[string]interface{}{}
	for _, m := range meses {
		var meta interface{}
		if m.Meta != nil {
			meta = Arredondar(*m.Meta, CasasValor)
		}
		pontos = append(pontos, map[string]interface{}{"x": m.Mes, "receita": Arredondar(m.Receita, CasasValor), "meta": meta})
	}
	return models.CardResultado{Dados: models.DadosSerieTemporal{
		Series: []models.Serie{
			{Chave: "receita", Nome: "Receita"},
			{Chave: "meta", Nome: "Meta"},
		},
		Pontos:  pontos,
		EixoX:   "Mês",
		EixoY:   "Receita (R$)",
		Unidade: "R$",
	}}, nil
}

// ---------------------------------------------------------------------
// ven-11 · Propostas por status · donut
// Orçamentos com emitido_em no período, pelo status atual até ate (sem
// evento = pendente). Sempre 4 fatias.
// ---------------------------------------------------------------------

type statusPropostas struct {
	repo repositories.MetricasVendasRepository
}

// NovoStatusPropostas cria o calculador do ven-11.
func NovoStatusPropostas(repo repositories.MetricasVendasRepository) Calculador {
	return statusPropostas{repo}
}

func (statusPropostas) Codigo() string     { return "ven-11" }
func (statusPropostas) Gatilhos() []string { return gatilhosPorCard["ven-11"] }

func (c statusPropostas) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	s, err := c.repo.StatusPropostas(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: models.DadosDonut{
		Fatias: []models.Fatia{
			{Rotulo: "Pendentes", Chave: "pendente", Valor: float64(s.Pendentes)},
			{Rotulo: "Aprovadas", Chave: "aprovada", Valor: float64(s.Aprovadas)},
			{Rotulo: "Recusadas", Chave: "recusada", Valor: float64(s.Recusadas)},
			{Rotulo: "Em revisão", Chave: "em_revisao", Valor: float64(s.EmRevisao)},
		},
		Total: float64(s.Pendentes + s.Aprovadas + s.Recusadas + s.EmRevisao),
	}}, nil
}

// ---------------------------------------------------------------------
// ven-12 · Margem prevista por complexidade · barras · Período (emitido_em)
// Sempre 3 itens: valor = média de margem_prevista (%); extra = valor
// cobrado por hora ("R$ 300,00/h"). null quando o grupo está vazio.
// ---------------------------------------------------------------------

type margemComplexidade struct {
	repo repositories.MetricasVendasRepository
}

// NovoMargemComplexidade cria o calculador do ven-12.
func NovoMargemComplexidade(repo repositories.MetricasVendasRepository) Calculador {
	return margemComplexidade{repo}
}

func (margemComplexidade) Codigo() string     { return "ven-12" }
func (margemComplexidade) Gatilhos() []string { return gatilhosPorCard["ven-12"] }

func (c margemComplexidade) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	grupos, err := c.repo.MargemPorComplexidade(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	porComplexidade := map[string]models.MargemComplexidade{}
	for _, g := range grupos {
		porComplexidade[g.Complexidade] = g
	}
	itens := []models.ItemBarra{}
	for _, o := range models.Complexidades {
		item := models.ItemBarra{Rotulo: o.Rotulo}
		if g, ok := porComplexidade[o.Valor]; ok && g.N > 0 {
			item.Valor = PtrArredondado(g.MargemMedia, CasasPercentual)
			if g.Horas > 0 {
				item.Extra = FormatarMoeda(g.Valor/g.Horas) + "/h"
			}
		}
		itens = append(itens, item)
	}
	return models.CardResultado{Dados: models.DadosBarras{Itens: itens, Unidade: "%"}}, nil
}

// ---------------------------------------------------------------------
// ven-13 · Orçado x real (horas) · barras_agrupadas · Período
// ven-14 · Margem prevista x real · barras_agrupadas com pilha · Período
// Base: orçamentos com venda fechada no período, por tipo_projeto do
// orçamento (só tipos com dados, na ordem do enum).
// ---------------------------------------------------------------------

type orcadoReal struct {
	repo repositories.MetricasVendasRepository
}

// NovoOrcadoReal cria o calculador do ven-13.
func NovoOrcadoReal(repo repositories.MetricasVendasRepository) Calculador {
	return orcadoReal{repo}
}

func (orcadoReal) Codigo() string     { return "ven-13" }
func (orcadoReal) Gatilhos() []string { return gatilhosPorCard["ven-13"] }

func (c orcadoReal) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	tipos, err := c.repo.OrcadoReal(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	itens := []map[string]interface{}{}
	for _, t := range ordenarPorTipo(tipos) {
		itens = append(itens, map[string]interface{}{
			"rotulo":   models.RotuloEnum(models.TiposProjeto, t.Tipo),
			"estimado": Arredondar(t.HorasEstimadas, CasasValor),
			"real":     Arredondar(t.HorasReais, CasasValor),
		})
	}
	return models.CardResultado{Dados: models.DadosBarrasAgrupadas{
		Series: []models.Serie{
			{Chave: "estimado", Nome: "Orçado"},
			{Chave: "real", Nome: "Real"},
		},
		Itens:   itens,
		Unidade: "h",
	}}, nil
}

type margemPrevistaReal struct {
	repo repositories.MetricasVendasRepository
}

// NovoMargemPrevistaReal cria o calculador do ven-14.
func NovoMargemPrevistaReal(repo repositories.MetricasVendasRepository) Calculador {
	return margemPrevistaReal{repo}
}

func (margemPrevistaReal) Codigo() string     { return "ven-14" }
func (margemPrevistaReal) Gatilhos() []string { return gatilhosPorCard["ven-14"] }

func (c margemPrevistaReal) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	tipos, err := c.repo.OrcadoReal(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: MontarMargemPrevistaReal(tipos)}, nil
}

// MontarMargemPrevistaReal: previsto (pilha "previsto") = valor × margem
// prevista; margem_real e custo_ia (pilha "real"), onde margem_real =
// vendido − custo das horas − custo de IA. A pilha "real" soma a margem
// real sem o custo de IA. margem_real pode ser negativa.
func MontarMargemPrevistaReal(tipos []models.OrcadoRealTipo) models.DadosBarrasAgrupadas {
	previsto, real := "previsto", "real"
	itens := []map[string]interface{}{}
	for _, t := range ordenarPorTipo(tipos) {
		itens = append(itens, map[string]interface{}{
			"rotulo":      models.RotuloEnum(models.TiposProjeto, t.Tipo),
			"previsto":    Arredondar(t.MargemPrevista, CasasValor),
			"margem_real": Arredondar(t.ValorVendido-t.CustoHoras-t.CustoIA, CasasValor),
			"custo_ia":    Arredondar(t.CustoIA, CasasValor),
		})
	}
	return models.DadosBarrasAgrupadas{
		Series: []models.Serie{
			{Chave: "previsto", Nome: "Margem prevista", Pilha: &previsto},
			{Chave: "margem_real", Nome: "Margem real", Pilha: &real},
			{Chave: "custo_ia", Nome: "Custo de IA", Pilha: &real},
		},
		Itens:   itens,
		Unidade: "R$",
	}
}

// ordenarPorTipo devolve os tipos na ordem do enum tipo_projeto.
func ordenarPorTipo(tipos []models.OrcadoRealTipo) []models.OrcadoRealTipo {
	porTipo := map[string]models.OrcadoRealTipo{}
	for _, t := range tipos {
		porTipo[t.Tipo] = t
	}
	ordenados := []models.OrcadoRealTipo{}
	for _, o := range models.TiposProjeto {
		if t, ok := porTipo[o.Valor]; ok {
			ordenados = append(ordenados, t)
		}
	}
	return ordenados
}
