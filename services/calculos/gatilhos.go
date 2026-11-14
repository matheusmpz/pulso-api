package calculos

import m "pulso-api/models"

// gatilhosPorCard: os 41 cards e os eventos que podem mudar cada um
// (seção 12). É a fonte dos códigos do registro.
var gatilhosPorCard = map[string][]string{
	// ---- Projetos ----
	"proj-01": {m.EventoAtividadeCriada, m.EventoStatusAtividade},
	"proj-02": {m.EventoHorasLancadas, m.EventoAtividadeCriada},
	"proj-03": {m.EventoConsumoTokens},
	"proj-04": {m.EventoHorasLancadas, m.EventoStatusAtividade},
	"proj-05": {m.EventoAtividadeCriada, m.EventoStatusAtividade},
	"proj-06": {m.EventoAtividadeCriada, m.EventoStatusAtividade},
	"proj-07": {m.EventoHorasLancadas, m.EventoAtividadeCriada},
	"proj-08": {m.EventoHorasLancadas, m.EventoAtividadeCriada},
	"proj-09": {m.EventoHorasLancadas, m.EventoAtividadeCriada},
	"proj-10": {m.EventoStatusAtividade},
	"proj-11": {m.EventoStatusAtividade},
	"proj-12": {m.EventoHorasLancadas},
	"proj-13": {m.EventoAtividadeCriada, m.EventoStatusAtividade},
	"proj-14": {m.EventoHorasLancadas},
	"proj-15": {m.EventoConsumoTokens},

	// ---- Vendas ----
	"ven-01": {m.EventoVendaFechada},
	"ven-02": {m.EventoEtapaOportunidade},
	"ven-03": {m.EventoVendaFechada},
	"ven-04": {m.EventoEtapaOportunidade},
	"ven-05": {m.EventoEtapaOportunidade},
	"ven-06": {m.EventoEtapaOportunidade, m.EventoOrcamentoEmitido, m.EventoOrcamentoStatus},
	"ven-07": {m.EventoEtapaOportunidade},
	"ven-08": {m.EventoEtapaOportunidade},
	"ven-09": {m.EventoReuniaoRegistrada, m.EventoEtapaOportunidade},
	"ven-10": {m.EventoVendaFechada},
	"ven-11": {m.EventoOrcamentoEmitido, m.EventoOrcamentoStatus},
	"ven-12": {m.EventoOrcamentoEmitido},
	"ven-13": {m.EventoHorasLancadas, m.EventoVendaFechada},
	"ven-14": {m.EventoHorasLancadas, m.EventoConsumoTokens, m.EventoVendaFechada},

	// ---- IA ----
	"ia-01": {m.EventoConsumoTokens},
	"ia-02": {m.EventoConsumoTokens},
	"ia-03": {m.EventoConsumoTokens},
	"ia-04": {m.EventoConsumoTokens, m.EventoStatusAtividade},
	"ia-05": {m.EventoConsumoTokens},
	"ia-06": {m.EventoConsumoTokens},
	"ia-07": {m.EventoConsumoTokens},
	"ia-08": {m.EventoConsumoTokens},
	"ia-09": {m.EventoConsumoTokens},
	"ia-10": {m.EventoStatusAtividade, m.EventoConsumoTokens, m.EventoHorasLancadas},
	"ia-11": {m.EventoHorasLancadas, m.EventoConsumoTokens},
	"ia-12": {m.EventoConsumoTokens},
}
