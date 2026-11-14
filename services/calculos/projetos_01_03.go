package calculos

import (
	"context"

	"pulso-api/models"
	"pulso-api/repositories"
)

// ---------------------------------------------------------------------
// proj-01 · Progresso · numero · Acumulado
// valor = Evolução % de A: estimativa das atividades concluídas ÷
// estimativa de todas as atividades × 100 (ponderada pela estimativa).
// detalhe: atividades concluídas e total.
// ---------------------------------------------------------------------

type progresso struct {
	repo repositories.MetricasProjetosRepository
}

// NovoProgresso cria o calculador do proj-01.
func NovoProgresso(repo repositories.MetricasProjetosRepository) Calculador {
	return progresso{repo}
}

func (progresso) Codigo() string     { return "proj-01" }
func (progresso) Gatilhos() []string { return gatilhosPorCard["proj-01"] }

func (c progresso) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	p, err := c.repo.Progresso(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: models.DadosNumero{
		Valor:   Percentual(p.EstimativaConcluida, p.EstimativaTotal),
		Unidade: "%",
		Detalhe: []models.Detalhe{
			{Rotulo: "Concluídas", Valor: Ptr(float64(p.Concluidas)), Unidade: "atividades"},
			{Rotulo: "Total", Valor: Ptr(float64(p.Total)), Unidade: "atividades"},
		},
	}}, nil
}

// ---------------------------------------------------------------------
// proj-02 · Horas · numero · Acumulado
// valor = horas reais de H até ate ÷ horas estimadas de A × 100.
// detalhe: horas reais e horas estimadas.
// ---------------------------------------------------------------------

type horas struct {
	repo repositories.MetricasProjetosRepository
}

// NovoHoras cria o calculador do proj-02.
func NovoHoras(repo repositories.MetricasProjetosRepository) Calculador {
	return horas{repo}
}

func (horas) Codigo() string     { return "proj-02" }
func (horas) Gatilhos() []string { return gatilhosPorCard["proj-02"] }

func (c horas) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	h, err := c.repo.Horas(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: models.DadosNumero{
		Valor:   Percentual(h.Reais, h.Estimadas),
		Unidade: "%",
		Detalhe: []models.Detalhe{
			{Rotulo: "Horas reais", Valor: PtrArredondado(h.Reais, CasasValor), Unidade: "h"},
			{Rotulo: "Horas estimadas", Valor: PtrArredondado(h.Estimadas, CasasValor), Unidade: "h"},
		},
	}}, nil
}

// ---------------------------------------------------------------------
// proj-03 · Custo de IA · numero · Acumulado (valor) e Período (detalhe)
// valor = tokens de T até ate ÷ quota × 100 (null se a quota for 0).
// detalhe: custo em R$ de T no período e tokens consumidos até ate.
// ---------------------------------------------------------------------

type custoIAProjeto struct {
	repo repositories.MetricasIARepository
}

// NovoCustoIAProjeto cria o calculador do proj-03.
func NovoCustoIAProjeto(repo repositories.MetricasIARepository) Calculador {
	return custoIAProjeto{repo}
}

func (custoIAProjeto) Codigo() string     { return "proj-03" }
func (custoIAProjeto) Gatilhos() []string { return gatilhosPorCard["proj-03"] }

func (c custoIAProjeto) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	r, err := c.repo.ConsumoAcumulado(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: models.DadosNumero{
		Valor:   Percentual(float64(r.TokensAcumulados), float64(r.Quota)),
		Unidade: "%",
		Detalhe: []models.Detalhe{
			{Rotulo: "Custo no período", Valor: PtrArredondado(r.CustoPeriodo, CasasValor), Unidade: "R$"},
			{Rotulo: "Tokens consumidos", Valor: Ptr(float64(r.TokensAcumulados)), Unidade: "tokens"},
		},
	}}, nil
}
