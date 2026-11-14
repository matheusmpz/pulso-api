package calculos

import (
	"context"

	"pulso-api/models"
	"pulso-api/repositories"
	"pulso-api/utils"
)

// ---------------------------------------------------------------------
// ven-01 · Receita fechada · numero · Período
// Soma de vendas.valor com fechada_em no período (R$), com variação contra
// o período anterior de mesma duração. detalhe: quantidade de vendas.
// ---------------------------------------------------------------------

type receitaFechada struct {
	repo repositories.MetricasVendasRepository
}

// NovoReceitaFechada cria o calculador do ven-01.
func NovoReceitaFechada(repo repositories.MetricasVendasRepository) Calculador {
	return receitaFechada{repo}
}

func (receitaFechada) Codigo() string     { return "ven-01" }
func (receitaFechada) Gatilhos() []string { return gatilhosPorCard["ven-01"] }

func (c receitaFechada) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	atual, err := c.repo.Receita(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	anterior, err := c.repo.Receita(ctx, periodoAnterior(f))
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: models.DadosNumero{
		Valor:    PtrArredondado(atual.Valor, CasasValor),
		Unidade:  "R$",
		Variacao: &models.Variacao{Valor: Variacao(atual.Valor, anterior.Valor), Unidade: "%"},
		Detalhe:  []models.Detalhe{{Rotulo: "Vendas", Valor: Ptr(float64(atual.Vendas)), Unidade: "vendas"}},
	}}, nil
}

// periodoAnterior devolve os filtros com de/ate trocados pelo período
// anterior de mesma duração (seção 7.3).
func periodoAnterior(f models.Filtros) models.Filtros {
	f.De, f.Ate = utils.PeriodoAnterior(f.De, f.Ate)
	return f
}

// ---------------------------------------------------------------------
// ven-02 · Taxa de conversão · numero · Período
// Oportunidades cujo estado final (ganha ou perdida) ocorreu no período:
// ganhas ÷ (ganhas + perdidas) × 100. detalhe: ganhas e perdidas.
// ---------------------------------------------------------------------

type taxaConversao struct {
	repo repositories.MetricasVendasRepository
}

// NovoTaxaConversao cria o calculador do ven-02.
func NovoTaxaConversao(repo repositories.MetricasVendasRepository) Calculador {
	return taxaConversao{repo}
}

func (taxaConversao) Codigo() string     { return "ven-02" }
func (taxaConversao) Gatilhos() []string { return gatilhosPorCard["ven-02"] }

func (c taxaConversao) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	r, err := c.repo.Conversao(ctx, semTipo(f))
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: models.DadosNumero{
		Valor:   Percentual(float64(r.Ganhas), float64(r.Ganhas+r.Perdidas)),
		Unidade: "%",
		Detalhe: []models.Detalhe{
			{Rotulo: "Ganhas", Valor: Ptr(float64(r.Ganhas)), Unidade: "oportunidades"},
			{Rotulo: "Perdidas", Valor: Ptr(float64(r.Perdidas)), Unidade: "oportunidades"},
		},
	}}, nil
}

// semTipo zera tipo_projeto: nos cards de funil o filtro não vale,
// porque lead ainda não tem tipo (seção 7.1).
func semTipo(f models.Filtros) models.Filtros {
	f.TipoProjeto = nil
	return f
}

// ---------------------------------------------------------------------
// ven-03 · Ticket médio · numero · Período
// Receita ÷ número de vendas do período (R$; null se não houver vendas).
// ---------------------------------------------------------------------

type ticketMedio struct {
	repo repositories.MetricasVendasRepository
}

// NovoTicketMedio cria o calculador do ven-03.
func NovoTicketMedio(repo repositories.MetricasVendasRepository) Calculador {
	return ticketMedio{repo}
}

func (ticketMedio) Codigo() string     { return "ven-03" }
func (ticketMedio) Gatilhos() []string { return gatilhosPorCard["ven-03"] }

func (c ticketMedio) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	r, err := c.repo.Receita(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	var valor *float64
	if r.Vendas > 0 {
		valor = PtrArredondado(r.Valor/float64(r.Vendas), CasasValor)
	}
	return models.CardResultado{Dados: models.DadosNumero{
		Valor:   valor,
		Unidade: "R$",
		Detalhe: []models.Detalhe{{Rotulo: "Vendas", Valor: Ptr(float64(r.Vendas)), Unidade: "vendas"}},
	}}, nil
}

// ---------------------------------------------------------------------
// ven-04 · Ciclo médio de venda · numero · Período
// Média de dias entre oportunidades.criado_em e o evento ganha, nas
// oportunidades ganhas no período (dias, 1 casa; null se nenhuma).
// ---------------------------------------------------------------------

type cicloVenda struct {
	repo repositories.MetricasVendasRepository
}

// NovoCicloVenda cria o calculador do ven-04.
func NovoCicloVenda(repo repositories.MetricasVendasRepository) Calculador {
	return cicloVenda{repo}
}

func (cicloVenda) Codigo() string     { return "ven-04" }
func (cicloVenda) Gatilhos() []string { return gatilhosPorCard["ven-04"] }

func (c cicloVenda) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	r, err := c.repo.Ciclo(ctx, semTipo(f))
	if err != nil {
		return models.CardResultado{}, err
	}
	var valor *float64
	if r.MediaDias != nil {
		valor = PtrArredondado(*r.MediaDias, CasasDias)
	}
	return models.CardResultado{Dados: models.DadosNumero{
		Valor:   valor,
		Unidade: "dias",
		Detalhe: []models.Detalhe{{Rotulo: "Ganhas", Valor: Ptr(float64(r.Ganhas)), Unidade: "oportunidades"}},
	}}, nil
}

// ---------------------------------------------------------------------
// ven-05 · Funil de vendas · barras (horizontal) · Período
// Uma barra por etapa de lead a ganha; extra = "x%" de conversão em
// relação à etapa anterior (null na primeira ou se a anterior for 0).
// ---------------------------------------------------------------------

type funilVendas struct {
	repo repositories.MetricasVendasRepository
}

// NovoFunilVendas cria o calculador do ven-05.
func NovoFunilVendas(repo repositories.MetricasVendasRepository) Calculador {
	return funilVendas{repo}
}

func (funilVendas) Codigo() string     { return "ven-05" }
func (funilVendas) Gatilhos() []string { return gatilhosPorCard["ven-05"] }

func (c funilVendas) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	etapas, err := c.repo.Funil(ctx, semTipo(f))
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: MontarFunil(etapas)}, nil
}

// MontarFunil monta as barras do funil.
func MontarFunil(etapas []models.EtapaFunil) models.DadosBarras {
	itens := []models.ItemBarra{}
	for i, e := range etapas {
		var extra interface{}
		if i > 0 {
			if anterior := etapas[i-1].Oportunidades; anterior > 0 {
				extra = FormatarPercentual(float64(e.Oportunidades) / float64(anterior) * 100)
			}
		}
		itens = append(itens, models.ItemBarra{
			Rotulo: models.RotuloEnum(models.EtapasFunil, e.Etapa),
			Valor:  Ptr(float64(e.Oportunidades)),
			Extra:  extra,
		})
	}
	return models.DadosBarras{Itens: itens, Unidade: "oportunidades", Horizontal: true}
}
