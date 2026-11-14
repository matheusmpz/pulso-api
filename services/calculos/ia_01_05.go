package calculos

import (
	"context"
	"math"

	"pulso-api/models"
	"pulso-api/repositories"
	"pulso-api/utils"
)

// ---------------------------------------------------------------------
// ia-01 · Custo de IA · numero · Período
// Soma de custo de T no período (R$), com variação contra o anterior.
// ---------------------------------------------------------------------

type custoIA struct {
	repo repositories.MetricasIARepository
}

// NovoCustoIA cria o calculador do ia-01.
func NovoCustoIA(repo repositories.MetricasIARepository) Calculador {
	return custoIA{repo}
}

func (custoIA) Codigo() string     { return "ia-01" }
func (custoIA) Gatilhos() []string { return gatilhosPorCard["ia-01"] }

func (c custoIA) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	atual, err := c.repo.TokensPeriodo(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	anterior, err := c.repo.TokensPeriodo(ctx, periodoAnterior(f))
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: models.DadosNumero{
		Valor:    PtrArredondado(atual.Custo, CasasValor),
		Unidade:  "R$",
		Variacao: &models.Variacao{Valor: Variacao(atual.Custo, anterior.Custo), Unidade: "%"},
	}}, nil
}

// ---------------------------------------------------------------------
// ia-02 · Tokens consumidos · numero · Período
// Soma de tokens_entrada + tokens_saida de T. detalhe: entrada e saída.
// ---------------------------------------------------------------------

type tokensConsumidos struct {
	repo repositories.MetricasIARepository
}

// NovoTokensConsumidos cria o calculador do ia-02.
func NovoTokensConsumidos(repo repositories.MetricasIARepository) Calculador {
	return tokensConsumidos{repo}
}

func (tokensConsumidos) Codigo() string     { return "ia-02" }
func (tokensConsumidos) Gatilhos() []string { return gatilhosPorCard["ia-02"] }

func (c tokensConsumidos) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	r, err := c.repo.TokensPeriodo(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: models.DadosNumero{
		Valor:   Ptr(float64(r.Entrada + r.Saida)),
		Unidade: "tokens",
		Detalhe: []models.Detalhe{
			{Rotulo: "Entrada", Valor: Ptr(float64(r.Entrada)), Unidade: "tokens"},
			{Rotulo: "Saída", Valor: Ptr(float64(r.Saida)), Unidade: "tokens"},
		},
	}}, nil
}

// ---------------------------------------------------------------------
// ia-03 · Quota consumida · radial · Acumulado
// valor = tokens de T até ate ÷ quota × 100 (null se a quota for 0);
// meta = 100; detalhe: consumido e limite.
// ---------------------------------------------------------------------

type quotaConsumida struct {
	repo repositories.MetricasIARepository
}

// NovoQuotaConsumida cria o calculador do ia-03.
func NovoQuotaConsumida(repo repositories.MetricasIARepository) Calculador {
	return quotaConsumida{repo}
}

func (quotaConsumida) Codigo() string     { return "ia-03" }
func (quotaConsumida) Gatilhos() []string { return gatilhosPorCard["ia-03"] }

func (c quotaConsumida) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	r, err := c.repo.ConsumoAcumulado(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: models.DadosRadial{
		Valor:   Percentual(float64(r.TokensAcumulados), float64(r.Quota)),
		Meta:    100,
		Unidade: "%",
		Detalhe: []models.Detalhe{
			{Rotulo: "Consumido", Valor: Ptr(float64(r.TokensAcumulados)), Unidade: "tokens"},
			{Rotulo: "Limite", Valor: Ptr(float64(r.Quota)), Unidade: "tokens"},
		},
	}}, nil
}

// ---------------------------------------------------------------------
// ia-04 · Custo por atividade concluída · numero · Período
// Custo de IA das atividades de A concluídas no período (todo o consumo
// ligado a elas até ate) ÷ quantidade (R$; null se nenhuma).
// ---------------------------------------------------------------------

type custoPorAtividade struct {
	repo repositories.MetricasIARepository
}

// NovoCustoPorAtividade cria o calculador do ia-04.
func NovoCustoPorAtividade(repo repositories.MetricasIARepository) Calculador {
	return custoPorAtividade{repo}
}

func (custoPorAtividade) Codigo() string     { return "ia-04" }
func (custoPorAtividade) Gatilhos() []string { return gatilhosPorCard["ia-04"] }

func (c custoPorAtividade) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	r, err := c.repo.CustoAtividadesConcluidas(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	var valor *float64
	if r.Atividades > 0 {
		valor = PtrArredondado(r.Custo/float64(r.Atividades), CasasValor)
	}
	return models.CardResultado{Dados: models.DadosNumero{
		Valor:   valor,
		Unidade: "R$",
		Detalhe: []models.Detalhe{
			{Rotulo: "Atividades concluídas", Valor: Ptr(float64(r.Atividades)), Unidade: "atividades"},
			{Rotulo: "Custo total", Valor: PtrArredondado(r.Custo, CasasValor), Unidade: "R$"},
		},
	}}, nil
}

// ---------------------------------------------------------------------
// ia-05 · Consumo diário e projeção · area · Período
// ---------------------------------------------------------------------

type consumoDiario struct {
	repo repositories.MetricasIARepository
}

// NovoConsumoDiario cria o calculador do ia-05.
func NovoConsumoDiario(repo repositories.MetricasIARepository) Calculador {
	return consumoDiario{repo}
}

func (consumoDiario) Codigo() string     { return "ia-05" }
func (consumoDiario) Gatilhos() []string { return gatilhosPorCard["ia-05"] }

func (c consumoDiario) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	r, err := c.repo.ConsumoDiario(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: MontarConsumoDiario(r, f.De, f.Ate)}, nil
}

// MontarConsumoDiario monta o ia-05 (seção 12):
//   - consumo: tokens por dia de de a ate (dias sem consumo = 0);
//   - media_movel: média dos 7 dias terminados naquele dia (usa dias
//     anteriores a de quando preciso);
//   - extra.projecao: media_diaria (média dos 7 dias até ate), restante
//     (quota − acumulado, mínimo 0), dias_restantes (piso de restante ÷
//     media_diaria; null se a média ou a quota for 0), data_esgotamento
//     (ate + dias_restantes) e esgotada (restante 0 com quota > 0).
//
// Tokens e médias saem arredondados para inteiro.
func MontarConsumoDiario(r models.ConsumoDiario, de, ate models.Data) models.DadosSerieTemporal {
	janela := repositories.JanelaMediaDias
	pontos := []map[string]interface{}{}
	var mediaFinal float64

	for i, d := range r.Dias {
		// Média dos até 7 dias terminados em i
		inicio := i - janela + 1
		if inicio < 0 {
			inicio = 0
		}
		soma := 0.0
		for _, x := range r.Dias[inicio : i+1] {
			soma += float64(x.Tokens)
		}
		media := soma / float64(janela)
		if d.Dia.Before(de.Time) {
			continue // dias antes de de só servem para a média
		}
		pontos = append(pontos, map[string]interface{}{
			"x":           d.Dia.String(),
			"consumo":     d.Tokens,
			"media_movel": Arredondar(media, 0),
		})
		mediaFinal = media
	}

	restante := r.Quota - r.TokensAcumulados
	if restante < 0 {
		restante = 0
	}
	var diasRestantes, dataEsgotamento interface{}
	if mediaFinal > 0 && r.Quota > 0 {
		n := int(math.Floor(float64(restante) / mediaFinal))
		diasRestantes = n
		dataEsgotamento = utils.SomarDias(ate, n).String()
	}

	return models.DadosSerieTemporal{
		Series: []models.Serie{
			{Chave: "consumo", Nome: "Consumo diário"},
			{Chave: "media_movel", Nome: "Média móvel (7 dias)"},
		},
		Pontos:  pontos,
		EixoX:   "Dia",
		EixoY:   "Tokens",
		Unidade: "tokens",
		Extra: map[string]interface{}{"projecao": map[string]interface{}{
			"media_diaria":     Arredondar(mediaFinal, 0),
			"restante":         restante,
			"dias_restantes":   diasRestantes,
			"data_esgotamento": dataEsgotamento,
			"esgotada":         restante == 0 && r.Quota > 0,
		}},
	}
}
