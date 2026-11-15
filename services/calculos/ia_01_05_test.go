package calculos

import (
	"context"
	"testing"

	"pulso-api/models"
)

func (r repoIAFalso) TokensPeriodo(_ context.Context, f models.Filtros) (models.ResumoTokensPeriodo, error) {
	return r.tokens[f.De.String()+"|"+f.Ate.String()], r.err
}

func (r repoIAFalso) CustoAtividadesConcluidas(context.Context, models.Filtros) (models.ResumoCustoAtividades, error) {
	return r.custoAtiv, r.err
}

func (r repoIAFalso) ConsumoDiario(context.Context, models.Filtros) (models.ConsumoDiario, error) {
	return r.diario, r.err
}

func filtrosIA() models.Filtros {
	return models.Filtros{Dashboard: "ia", Escopo: "projeto", De: d("2026-11-11"), Ate: d("2026-11-20")}
}

func TestIA01e02(t *testing.T) {
	repo := repoIAFalso{tokens: map[string]models.ResumoTokensPeriodo{
		"2026-11-11|2026-11-20": {Custo: 79.271, Entrada: 900, Saida: 300},
		"2026-11-01|2026-11-10": {Custo: 100},
	}}
	if got := calcularComFiltros(t, NovoCustoIA(repo), filtrosIA()); got != `{"valor":79.27,"unidade":"R$","variacao":{"valor":-20.7,"unidade":"%"},"detalhe":null}` {
		t.Errorf("ia-01: %s", got)
	}
	if got := calcularComFiltros(t, NovoTokensConsumidos(repo), filtrosIA()); got != `{"valor":1200,"unidade":"tokens","variacao":null,"detalhe":[{"rotulo":"Entrada","valor":900,"unidade":"tokens"},{"rotulo":"Saída","valor":300,"unidade":"tokens"}]}` {
		t.Errorf("ia-02: %s", got)
	}
}

func TestIA03Quota(t *testing.T) {
	repo := repoIAFalso{consumo: models.ResumoConsumoIA{TokensAcumulados: 650000, Quota: 1000000}}
	if got := calcularComFiltros(t, NovoQuotaConsumida(repo), filtrosIA()); got != `{"valor":65,"meta":100,"unidade":"%","detalhe":[{"rotulo":"Consumido","valor":650000,"unidade":"tokens"},{"rotulo":"Limite","valor":1000000,"unidade":"tokens"}]}` {
		t.Errorf("ia-03: %s", got)
	}
	if got := calcularComFiltros(t, NovoQuotaConsumida(repoIAFalso{}), filtrosIA()); got != `{"valor":null,"meta":100,"unidade":"%","detalhe":[{"rotulo":"Consumido","valor":0,"unidade":"tokens"},{"rotulo":"Limite","valor":0,"unidade":"tokens"}]}` {
		t.Errorf("ia-03 sem quota: %s", got)
	}
}

func TestIA04CustoPorAtividade(t *testing.T) {
	got := calcularComFiltros(t, NovoCustoPorAtividade(repoIAFalso{custoAtiv: models.ResumoCustoAtividades{Atividades: 3, Custo: 10}}), filtrosIA())
	if got != `{"valor":3.33,"unidade":"R$","variacao":null,"detalhe":[{"rotulo":"Atividades concluídas","valor":3,"unidade":"atividades"},{"rotulo":"Custo total","valor":10,"unidade":"R$"}]}` {
		t.Errorf("ia-04: %s", got)
	}
	if got := calcularComFiltros(t, NovoCustoPorAtividade(repoIAFalso{}), filtrosIA()); got != `{"valor":null,"unidade":"R$","variacao":null,"detalhe":[{"rotulo":"Atividades concluídas","valor":0,"unidade":"atividades"},{"rotulo":"Custo total","valor":0,"unidade":"R$"}]}` {
		t.Errorf("ia-04 sem atividades: %s", got)
	}
}

// diasConsumo monta dias consecutivos a partir de inicio com os tokens dados.
func diasConsumo(inicio string, tokens ...int64) []models.DiaTokens {
	base := d(inicio)
	dias := make([]models.DiaTokens, len(tokens))
	for i, tk := range tokens {
		dias[i] = models.DiaTokens{Dia: models.Data{Time: base.AddDate(0, 0, i)}, Tokens: tk}
	}
	return dias
}

func TestMontarConsumoDiario(t *testing.T) {
	de, ate := d("2026-11-10"), d("2026-11-12")
	// 6 dias antes de de (04 a 09) + de a ate (10 a 12)
	r := models.ConsumoDiario{
		Dias:             diasConsumo("2026-11-04", 700, 700, 700, 700, 700, 700, 0, 1400, 700),
		TokensAcumulados: 6000,
		Quota:            10000,
	}
	dados := MontarConsumoDiario(r, de, ate)

	esperado := `[{"consumo":0,"media_movel":600,"x":"2026-11-10"},{"consumo":1400,"media_movel":700,"x":"2026-11-11"},{"consumo":700,"media_movel":700,"x":"2026-11-12"}]`
	if got := jsonDe(t, dados.Pontos); got != esperado {
		t.Errorf("pontos:\nesperado %s\nobtido   %s", esperado, got)
	}
	// restante 4000 ÷ média 700 = 5,7 → 5 dias → 17/11
	if got := jsonDe(t, dados.Extra); got != `{"projecao":{"data_esgotamento":"2026-11-17","dias_restantes":5,"esgotada":false,"media_diaria":700,"restante":4000}}` {
		t.Errorf("projeção: %s", got)
	}

	// Quota já estourada: restante 0, esgotada, esgota "hoje" (ate)
	r.TokensAcumulados = 12000
	if got := jsonDe(t, MontarConsumoDiario(r, de, ate).Extra); got != `{"projecao":{"data_esgotamento":"2026-11-12","dias_restantes":0,"esgotada":true,"media_diaria":700,"restante":0}}` {
		t.Errorf("esgotada: %s", got)
	}

	// Sem consumo nos últimos 7 dias: sem projeção
	semConsumo := models.ConsumoDiario{Dias: diasConsumo("2026-11-04", 0, 0, 0, 0, 0, 0, 0, 0, 0), Quota: 10000}
	if got := jsonDe(t, MontarConsumoDiario(semConsumo, de, ate).Extra); got != `{"projecao":{"data_esgotamento":null,"dias_restantes":null,"esgotada":false,"media_diaria":0,"restante":10000}}` {
		t.Errorf("sem consumo: %s", got)
	}

	// Sem quota (escopo vazio): sem projeção e não "esgotada"
	semQuota := models.ConsumoDiario{Dias: diasConsumo("2026-11-04", 7, 7, 7, 7, 7, 7, 7, 7, 7)}
	if got := jsonDe(t, MontarConsumoDiario(semQuota, de, ate).Extra); got != `{"projecao":{"data_esgotamento":null,"dias_restantes":null,"esgotada":false,"media_diaria":7,"restante":0}}` {
		t.Errorf("sem quota: %s", got)
	}
}
