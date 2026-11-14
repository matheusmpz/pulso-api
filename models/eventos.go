package models

// Eventos de tempo real (seção 6.4): cada INSERT numa tabela monitorada
// é um evento que pode invalidar cards.
const (
	EventoHorasLancadas     = "horas_lancadas"
	EventoStatusAtividade   = "status_atividade"
	EventoAtividadeCriada   = "atividade_criada"
	EventoConsumoTokens     = "consumo_tokens"
	EventoVendaFechada      = "venda_fechada"
	EventoEtapaOportunidade = "etapa_oportunidade"
	EventoReuniaoRegistrada = "reuniao_registrada"
	EventoOrcamentoEmitido  = "orcamento_emitido"
	EventoOrcamentoStatus   = "orcamento_status"
)

// TabelaPorEvento liga cada evento à tabela que o poller monitora.
var TabelaPorEvento = map[string]string{
	EventoHorasLancadas:     "registros_horas",
	EventoStatusAtividade:   "atividade_eventos",
	EventoAtividadeCriada:   "atividades",
	EventoConsumoTokens:     "consumo_tokens",
	EventoVendaFechada:      "vendas",
	EventoEtapaOportunidade: "oportunidade_eventos",
	EventoReuniaoRegistrada: "reunioes",
	EventoOrcamentoEmitido:  "orcamentos",
	EventoOrcamentoStatus:   "orcamento_eventos",
}
