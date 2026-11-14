package calculos

import (
	"context"
	"fmt"
	"math"

	"pulso-api/models"
	"pulso-api/repositories"
)

// ---------------------------------------------------------------------
// proj-12 · Atividades estourando a estimativa · barras (horizontal) · Acumulado
// Top 5 por horas reais − estimativa; extra = {desvio: "130%", projeto};
// total = quantidade de atividades estouradas (todas).
// ---------------------------------------------------------------------

type estouros struct {
	repo repositories.MetricasProjetosRepository
}

// NovoEstouros cria o calculador do proj-12.
func NovoEstouros(repo repositories.MetricasProjetosRepository) Calculador {
	return estouros{repo}
}

func (estouros) Codigo() string     { return "proj-12" }
func (estouros) Gatilhos() []string { return gatilhosPorCard["proj-12"] }

func (c estouros) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	e, err := c.repo.Estouros(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: MontarEstouros(e)}, nil
}

// MontarEstouros monta as barras do proj-12.
func MontarEstouros(e models.Estouros) models.DadosBarras {
	itens := []models.ItemBarra{}
	for _, a := range e.Top {
		desvio := 0.0
		if a.Estimativa > 0 {
			desvio = a.Reais / a.Estimativa * 100
		}
		itens = append(itens, models.ItemBarra{
			Rotulo: a.Titulo,
			Valor:  PtrArredondado(a.Reais-a.Estimativa, CasasValor),
			Extra:  map[string]interface{}{"desvio": fmt.Sprintf("%.0f%%", math.Round(desvio)), "projeto": a.Projeto},
		})
	}
	total := e.Total
	return models.DadosBarras{Itens: itens, Unidade: "h", Horizontal: true, Total: &total}
}

// ---------------------------------------------------------------------
// proj-13 · Histórico de sprints · barras_agrupadas · independe de de
// ---------------------------------------------------------------------

type historicoSprints struct {
	repo repositories.MetricasProjetosRepository
}

// NovoHistoricoSprints cria o calculador do proj-13.
func NovoHistoricoSprints(repo repositories.MetricasProjetosRepository) Calculador {
	return historicoSprints{repo}
}

func (historicoSprints) Codigo() string     { return "proj-13" }
func (historicoSprints) Gatilhos() []string { return gatilhosPorCard["proj-13"] }

func (c historicoSprints) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	h, err := c.repo.HistoricoSprints(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: MontarHistoricoSprints(h)}, nil
}

// MontarHistoricoSprints monta os itens {rotulo: "Sprint N", planejado,
// concluido, sprint_id, destaque}. Se a lista tiver sprints de mais de um
// time, o rótulo leva o nome do time ("Sprint 5 · Time Alfa").
func MontarHistoricoSprints(h []models.SprintHistorico) models.DadosBarrasAgrupadas {
	times := map[int64]bool{}
	for _, s := range h {
		times[s.Sprint.TimeID] = true
	}

	itens := []map[string]interface{}{}
	for _, s := range h {
		rotulo := fmt.Sprintf("Sprint %d", s.Sprint.Numero)
		if len(times) > 1 {
			rotulo += " · " + s.Sprint.TimeNome
		}
		itens = append(itens, map[string]interface{}{
			"rotulo":    rotulo,
			"planejado": Arredondar(s.Planejado, CasasValor),
			"concluido": Arredondar(s.Concluido, CasasValor),
			"sprint_id": s.Sprint.ID,
			"destaque":  s.Destaque,
		})
	}
	return models.DadosBarrasAgrupadas{
		Series: []models.Serie{
			{Chave: "planejado", Nome: "Planejado"},
			{Chave: "concluido", Nome: "Concluído"},
		},
		Itens:   itens,
		Unidade: "h",
	}
}

// ---------------------------------------------------------------------
// proj-14 · Distribuição de horas · barras (horizontal) · Período
// Por desenvolvedor (escopo=projeto) ou por projeto (time e sprint).
// Top 10 mais "Outros".
// ---------------------------------------------------------------------

type distribuicaoHoras struct {
	repo repositories.MetricasProjetosRepository
}

// NovoDistribuicaoHoras cria o calculador do proj-14.
func NovoDistribuicaoHoras(repo repositories.MetricasProjetosRepository) Calculador {
	return distribuicaoHoras{repo}
}

func (distribuicaoHoras) Codigo() string     { return "proj-14" }
func (distribuicaoHoras) Gatilhos() []string { return gatilhosPorCard["proj-14"] }

func (c distribuicaoHoras) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	g, err := c.repo.DistribuicaoHoras(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: MontarTopComOutros(g, repositories.LimiteTopHoras, "h")}, nil
}

// MontarTopComOutros monta barras horizontais com os "limite" primeiros
// grupos (já em ordem decrescente) e soma o resto num item "Outros".
func MontarTopComOutros(grupos []models.GrupoHoras, limite int, unidade string) models.DadosBarras {
	itens := []models.ItemBarra{}
	outros := 0.0
	for i, g := range grupos {
		if i < limite {
			itens = append(itens, models.ItemBarra{Rotulo: g.Rotulo, Valor: PtrArredondado(g.Horas, CasasValor)})
			continue
		}
		outros += g.Horas
	}
	if len(grupos) > limite {
		itens = append(itens, models.ItemBarra{Rotulo: "Outros", Valor: PtrArredondado(outros, CasasValor)})
	}
	return models.DadosBarras{Itens: itens, Unidade: unidade, Horizontal: true}
}

// ---------------------------------------------------------------------
// proj-15 · Custo de IA no tempo · area · Acumulado (ignora de)
// Série "consumo" = tokens de T acumulados por dia, do primeiro consumo
// até ate; referencia = limite de tokens (null se a quota for 0).
// ---------------------------------------------------------------------

type consumoNoTempo struct {
	repo repositories.MetricasIARepository
}

// NovoConsumoNoTempo cria o calculador do proj-15.
func NovoConsumoNoTempo(repo repositories.MetricasIARepository) Calculador {
	return consumoNoTempo{repo}
}

func (consumoNoTempo) Codigo() string     { return "proj-15" }
func (consumoNoTempo) Gatilhos() []string { return gatilhosPorCard["proj-15"] }

func (c consumoNoTempo) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	r, err := c.repo.ConsumoNoTempo(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: MontarConsumoNoTempo(r)}, nil
}

// MontarConsumoNoTempo monta a área de tokens acumulados.
func MontarConsumoNoTempo(r models.ConsumoNoTempo) models.DadosSerieTemporal {
	dados := models.DadosSerieTemporal{
		Series:  []models.Serie{{Chave: "consumo", Nome: "Consumo acumulado"}},
		Pontos:  []map[string]interface{}{},
		EixoX:   "Dia",
		EixoY:   "Tokens acumulados",
		Unidade: "tokens",
	}
	for _, d := range r.Dias {
		dados.Pontos = append(dados.Pontos, map[string]interface{}{"x": d.Dia.String(), "consumo": d.TokensAcumulados})
	}
	if r.Quota > 0 {
		dados.Referencia = &models.Referencia{Nome: "Limite de tokens", Valor: float64(r.Quota)}
	}
	return dados
}
