package calculos

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"pulso-api/models"
	"pulso-api/repositories"
)

// ---------------------------------------------------------------------
// ia-10 · Intensidade de IA x desempenho · barras_agrupadas · Período
// Base: atividades de A concluídas no período com tokens > 0. Por
// complexidade, tercis da intensidade (tokens ÷ estimativa): 1 baixa,
// 2 média, 3 alta. Itens = faixas; séries pequena, media, grande; valor =
// desvio médio (%); chaves extras n_pequena, n_media, n_grande.
// referencia = 100 e amostra_minima = 5. Mede quanto se usa IA, não se usa.
// ---------------------------------------------------------------------

type intensidadeDesempenho struct {
	repo repositories.MetricasIARepository
}

// NovoIntensidadeDesempenho cria o calculador do ia-10.
func NovoIntensidadeDesempenho(repo repositories.MetricasIARepository) Calculador {
	return intensidadeDesempenho{repo}
}

func (intensidadeDesempenho) Codigo() string     { return "ia-10" }
func (intensidadeDesempenho) Gatilhos() []string { return gatilhosPorCard["ia-10"] }

func (c intensidadeDesempenho) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	faixas, err := c.repo.IntensidadeDesempenho(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: MontarIntensidade(faixas)}, nil
}

// MontarIntensidade monta os 3 itens (Baixa, Média, Alta).
func MontarIntensidade(faixas []models.FaixaIntensidade) models.DadosIntensidade {
	porFaixa := map[int]map[string]models.FaixaIntensidade{}
	for _, x := range faixas {
		if porFaixa[x.Faixa] == nil {
			porFaixa[x.Faixa] = map[string]models.FaixaIntensidade{}
		}
		porFaixa[x.Faixa][x.Complexidade] = x
	}

	itens := []map[string]interface{}{}
	for i, rotulo := range []string{"Baixa", "Média", "Alta"} {
		item := map[string]interface{}{"rotulo": rotulo}
		for _, comp := range models.Complexidades {
			var valor interface{}
			n := 0
			if x, ok := porFaixa[i+1][comp.Valor]; ok && x.N > 0 {
				valor = Arredondar(x.DesvioMedio, CasasPercentual)
				n = x.N
			}
			item[comp.Valor] = valor
			item["n_"+comp.Valor] = n
		}
		itens = append(itens, item)
	}

	series := []models.Serie{}
	for _, comp := range models.Complexidades {
		series = append(series, models.Serie{Chave: comp.Valor, Nome: comp.Rotulo})
	}
	return models.DadosIntensidade{
		DadosBarrasAgrupadas: models.DadosBarrasAgrupadas{Series: series, Itens: itens, Unidade: "%"},
		Referencia:           100,
		AmostraMinima:        AmostraMinima,
	}
}

// ---------------------------------------------------------------------
// ia-11 · Peso da IA no custo · barras_agrupadas (empilhada) · Período
// Top 10 projetos de P por custo total. Séries custo_horas e custo_ia, as
// duas com pilha "custo"; chave extra percentual_ia = custo_ia ÷ total × 100.
// ---------------------------------------------------------------------

type pesoIA struct {
	repo repositories.MetricasIARepository
}

// NovoPesoIA cria o calculador do ia-11.
func NovoPesoIA(repo repositories.MetricasIARepository) Calculador {
	return pesoIA{repo}
}

func (pesoIA) Codigo() string     { return "ia-11" }
func (pesoIA) Gatilhos() []string { return gatilhosPorCard["ia-11"] }

func (c pesoIA) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	projetos, err := c.repo.PesoIA(ctx, f)
	if err != nil {
		return models.CardResultado{}, err
	}
	pilha := "custo"
	itens := []map[string]interface{}{}
	for i, p := range projetos {
		if i >= limiteTopProjetosIA {
			break
		}
		var percentual interface{}
		if v := Percentual(p.CustoIA, p.CustoHoras+p.CustoIA); v != nil {
			percentual = *v
		}
		itens = append(itens, map[string]interface{}{
			"rotulo":        p.Projeto,
			"custo_horas":   Arredondar(p.CustoHoras, CasasValor),
			"custo_ia":      Arredondar(p.CustoIA, CasasValor),
			"percentual_ia": percentual,
		})
	}
	return models.CardResultado{Dados: models.DadosBarrasAgrupadas{
		Series: []models.Serie{
			{Chave: "custo_horas", Nome: "Custo das horas", Pilha: &pilha},
			{Chave: "custo_ia", Nome: "Custo de IA", Pilha: &pilha},
		},
		Itens:   itens,
		Unidade: "R$",
	}}, nil
}

// ---------------------------------------------------------------------
// ia-12 · Alertas de consumo · lista · Misto
// (a) Quota: projetos de P com quota usada ≥ 80% até ate ("crítica" a
//     partir de 100%); quando = primeiro dia em que passou de 80%.
// (b) Pico: dias do período com consumo acima de 3× a média dos 7 dias
//     anteriores (média > 0); valor = razão ("3,4x"); quando = o dia.
// Ordem: quota crítica, quota atenção, picos por razão decrescente.
// Até 20 linhas; total = linhas antes do limite. O "alerta" do envelope
// continua null.
// ---------------------------------------------------------------------

// limiteLinhasAlertas: quantas linhas o ia-12 mostra.
const limiteLinhasAlertas = 20

type alertasConsumo struct {
	repo repositories.MetricasIARepository
}

// NovoAlertasConsumo cria o calculador do ia-12.
func NovoAlertasConsumo(repo repositories.MetricasIARepository) Calculador {
	return alertasConsumo{repo}
}

func (alertasConsumo) Codigo() string     { return "ia-12" }
func (alertasConsumo) Gatilhos() []string { return gatilhosPorCard["ia-12"] }

func (c alertasConsumo) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	quotas, err := c.repo.AlertasQuota(ctx, f, LimiarQuotaAtencao)
	if err != nil {
		return models.CardResultado{}, err
	}
	picos, err := c.repo.AlertasPico(ctx, f, FatorPico, JanelaMediaDias)
	if err != nil {
		return models.CardResultado{}, err
	}
	return models.CardResultado{Dados: MontarAlertas(quotas, picos)}, nil
}

// MontarAlertas ordena por gravidade, formata e limita as linhas.
func MontarAlertas(quotas []models.AlertaQuota, picos []models.AlertaPico) models.DadosLista {
	type linha struct {
		gravidade int // 0 quota crítica, 1 quota atenção, 2 pico
		peso      float64
		desempate string
		colunas   []string
	}
	var linhas []linha
	for _, q := range quotas {
		g, alerta := 1, "Quota em atenção"
		if q.Percentual >= LimiarQuotaCritica {
			g, alerta = 0, "Quota crítica"
		}
		linhas = append(linhas, linha{g, q.Percentual, q.Projeto,
			[]string{q.Projeto, alerta, FormatarPercentual(q.Percentual), q.Quando.String()}})
	}
	for _, p := range picos {
		linhas = append(linhas, linha{2, p.Razao, p.Projeto + p.Dia.String(),
			[]string{p.Projeto, "Pico", formatarRazao(p.Razao), p.Dia.String()}})
	}

	// Gravidade; dentro dela, maior percentual/razão primeiro
	sort.SliceStable(linhas, func(i, j int) bool {
		if linhas[i].gravidade != linhas[j].gravidade {
			return linhas[i].gravidade < linhas[j].gravidade
		}
		if linhas[i].peso != linhas[j].peso {
			return linhas[i].peso > linhas[j].peso
		}
		return linhas[i].desempate < linhas[j].desempate
	})

	dados := models.DadosLista{
		Colunas: []string{"Projeto", "Alerta", "Valor", "Quando"},
		Linhas:  [][]string{},
		Total:   len(linhas),
	}
	for i, l := range linhas {
		if i >= limiteLinhasAlertas {
			break
		}
		dados.Linhas = append(dados.Linhas, l.colunas)
	}
	return dados
}

// formatarRazao formata a razão do pico com 1 casa e vírgula: 3.42 → "3,4x".
func formatarRazao(v float64) string {
	return strings.Replace(strconv.FormatFloat(Arredondar(v, 1), 'f', 1, 64), ".", ",", 1) + "x"
}
