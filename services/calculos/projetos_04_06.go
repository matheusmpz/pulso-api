package calculos

import (
	"context"
	"math"

	"pulso-api/models"
	"pulso-api/repositories"
	"pulso-api/utils"
)

// ---------------------------------------------------------------------
// proj-04 · Evolução do projeto (burn-up) · linhas · Acumulado (ignora de)
// ---------------------------------------------------------------------

type burnUp struct {
	repo repositories.MetricasProjetosRepository
}

// NovoBurnUp cria o calculador do proj-04.
func NovoBurnUp(repo repositories.MetricasProjetosRepository) Calculador {
	return burnUp{repo}
}

func (burnUp) Codigo() string     { return "proj-04" }
func (burnUp) Gatilhos() []string { return gatilhosPorCard["proj-04"] }

func (c burnUp) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	b, err := c.repo.BurnUp(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: MontarBurnUp(b)}, nil
}

// MontarBurnUp monta os pontos do burn-up (seção 12, proj-04):
//   - um ponto por dia: x = horas acumuladas; real = evolução % acumulada
//     sobre o total de A; ideal = min(100, x ÷ estimativa total × 100);
//     data = o dia;
//   - se o maior x ficar abaixo da estimativa total, um ponto final
//     {x: estimativa, ideal: 100, real: null} leva a linha ideal a 100%;
//   - sem horas: [{x:0, ideal:0, real:0}, {x: estimativa, ideal:100, real:null}];
//   - sem atividades (estimativa total 0): nenhum ponto.
func MontarBurnUp(b models.BurnUp) models.DadosSerieTemporal {
	dados := models.DadosSerieTemporal{
		Series: []models.Serie{
			{Chave: "ideal", Nome: "Ideal"},
			{Chave: "real", Nome: "Real"},
		},
		Pontos:  []map[string]interface{}{},
		EixoX:   "Horas gastas (acumuladas)",
		EixoY:   "Evolução (%)",
		Unidade: "%",
	}
	total := b.EstimativaTotal
	if total <= 0 {
		return dados
	}
	pontoFinal := map[string]interface{}{"x": Arredondar(total, CasasValor), "ideal": 100.0, "real": nil}

	semHoras := len(b.Dias) == 0 || b.Dias[len(b.Dias)-1].HorasAcumuladas == 0
	if semHoras {
		dados.Pontos = []map[string]interface{}{{"x": 0.0, "ideal": 0.0, "real": 0.0}, pontoFinal}
		return dados
	}

	maiorX := 0.0
	for _, d := range b.Dias {
		x := Arredondar(d.HorasAcumuladas, CasasValor)
		maiorX = math.Max(maiorX, x)
		dados.Pontos = append(dados.Pontos, map[string]interface{}{
			"x":     x,
			"ideal": Arredondar(math.Min(100, d.HorasAcumuladas/total*100), CasasPercentual),
			"real":  Arredondar(d.EstimativaConcluida/total*100, CasasPercentual),
			"data":  d.Dia.String(),
		})
	}
	if maiorX < Arredondar(total, CasasValor) {
		dados.Pontos = append(dados.Pontos, pontoFinal)
	}
	return dados
}

// ---------------------------------------------------------------------
// proj-05 · Issues por status · donut · Acumulado (foto até ate)
// ---------------------------------------------------------------------

type issuesPorStatus struct {
	repo repositories.MetricasProjetosRepository
}

// NovoIssuesPorStatus cria o calculador do proj-05.
func NovoIssuesPorStatus(repo repositories.MetricasProjetosRepository) Calculador {
	return issuesPorStatus{repo}
}

func (issuesPorStatus) Codigo() string     { return "proj-05" }
func (issuesPorStatus) Gatilhos() []string { return gatilhosPorCard["proj-05"] }

func (c issuesPorStatus) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	s, err := c.repo.ContagemStatus(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	// Sempre as 3 fatias, mesmo com zero
	return models.CardResultado{Dados: models.DadosDonut{
		Fatias: []models.Fatia{
			{Rotulo: "Abertas", Chave: "aberta", Valor: float64(s.Abertas)},
			{Rotulo: "Em andamento", Chave: "em_andamento", Valor: float64(s.EmAndamento)},
			{Rotulo: "Concluídas", Chave: "concluida", Valor: float64(s.Concluidas)},
		},
		Total: float64(s.Abertas + s.EmAndamento + s.Concluidas),
	}}, nil
}

// ---------------------------------------------------------------------
// proj-06 · Burndown da sprint · area
// ---------------------------------------------------------------------

type burndown struct {
	repo repositories.MetricasProjetosRepository
	hoje func() models.Data
}

// NovoBurndown cria o calculador do proj-06.
func NovoBurndown(repo repositories.MetricasProjetosRepository) Calculador {
	return burndown{repo: repo, hoje: utils.Hoje}
}

func (burndown) Codigo() string     { return "proj-06" }
func (burndown) Gatilhos() []string { return gatilhosPorCard["proj-06"] }

func (c burndown) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	hoje := c.hoje()
	b, err := c.repo.Burndown(ctx, f, hoje)
	if err != nil {
		return models.CardResultado{}, err
	}
	// Dias depois de hoje (ou de ate, se for anterior) ficam sem valor real
	limite := hoje
	if f.Ate.Before(hoje.Time) && !f.Ate.IsZero() {
		limite = f.Ate
	}
	return models.CardResultado{Dados: MontarBurndown(b, limite)}, nil
}

// MontarBurndown monta os pontos do burndown (seção 12, proj-06):
// um ponto por dia da sprint; ideal = reta do total inicial até 0 no
// último dia; restante = estimativa ainda não concluída ao fim do dia
// (null depois de limite). Sem sprint alvo: nenhum ponto e sprint null.
func MontarBurndown(b models.Burndown, limite models.Data) models.DadosSerieTemporal {
	dados := models.DadosSerieTemporal{
		Series: []models.Serie{
			{Chave: "ideal", Nome: "Ideal"},
			{Chave: "restante", Nome: "Restante"},
		},
		Pontos:  []map[string]interface{}{},
		EixoX:   "Dia da sprint",
		EixoY:   "Horas restantes (estimativa)",
		Unidade: "h",
		Extra:   map[string]interface{}{"sprint": nil},
	}
	if b.Sprint == nil {
		return dados
	}
	dados.Extra["sprint"] = map[string]interface{}{
		"id": b.Sprint.ID, "numero": b.Sprint.Numero, "time_id": b.Sprint.TimeID,
		"inicio": b.Sprint.Inicio.String(), "fim": b.Sprint.Fim.String(),
	}

	n := len(b.Dias)
	for i, d := range b.Dias {
		ideal := 0.0
		if n > 1 {
			ideal = b.Total * (1 - float64(i)/float64(n-1))
		}
		var restante interface{}
		if !d.Dia.After(limite.Time) {
			restante = Arredondar(d.Restante, CasasValor)
		}
		dados.Pontos = append(dados.Pontos, map[string]interface{}{
			"x":        d.Dia.String(),
			"ideal":    Arredondar(ideal, CasasValor),
			"restante": restante,
		})
	}
	return dados
}
