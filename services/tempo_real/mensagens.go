package tempo_real

import (
	"encoding/json"

	"pulso-api/models"
)

// Mensagens do protocolo (seção 13.3).

// MensagemCliente é o que o cliente envia. Hoje só existe "assinar":
//
//	{ "tipo": "assinar", "dashboard": "projetos",
//	  "filtros": { "escopo": "projeto", "id": 3, "de": "2026-10-15", ... } }
type MensagemCliente struct {
	Tipo      string                `json:"tipo"`
	Dashboard string                `json:"dashboard"`
	Filtros   models.FiltrosEntrada `json:"filtros"`
}

// MensagemAssinado confirma a assinatura e lista os cards que o perfil vê.
type MensagemAssinado struct {
	Tipo      string   `json:"tipo"` // "assinado"
	Dashboard string   `json:"dashboard"`
	Cards     []string `json:"cards"`
}

// MensagemCard leva o resultado completo de um card que mudou.
type MensagemCard struct {
	Tipo  string               `json:"tipo"` // "card_atualizado"
	Card  string               `json:"card"`
	Dados models.CardResultado `json:"dados"`
}

// MensagemErro informa um problema (assinatura recusada, sessão expirada...).
type MensagemErro struct {
	Tipo     string `json:"tipo"` // "erro"
	Mensagem string `json:"mensagem"`
}

// Mensagens de erro do tempo real.
const (
	MsgSessaoExpirada    = "Sessão expirada."
	MsgMensagemInvalida  = "Mensagem inválida: envie JSON com \"tipo\"."
	MsgTipoDesconhecido  = "Tipo de mensagem desconhecido: use \"assinar\"."
	MsgDashboardInvalido = "Dashboard não encontrado."
	MsgErroInternoTR     = "Erro interno. Tente novamente mais tarde."
	TipoAssinar          = "assinar"
	TipoAssinado         = "assinado"
	TipoCardAtualizado   = "card_atualizado"
	TipoErro             = "erro"
)

func codificar(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(MensagemErro{Tipo: TipoErro, Mensagem: MsgErroInternoTR})
	}
	return b
}

func mensagemErro(texto string) []byte {
	return codificar(MensagemErro{Tipo: TipoErro, Mensagem: texto})
}
