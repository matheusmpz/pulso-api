package models

import "time"

// MsgErroCard é a mensagem devolvida no lugar dos dados quando um card
// não pode ser calculado (o erro real só vai para o log).
const MsgErroCard = "Não foi possível calcular este card."

// CardResultado é o envelope de um card calculado (seção 11).
//
// Em caso de sucesso: codigo, titulo, tipo, dados, alerta (sempre null no
// MVP) e atualizado_em. Em caso de falha: codigo, titulo, tipo e erro.
//
// O formato de "dados" depende do tipo do card; veja os tipos Dados* em
// cards_dados.go.
type CardResultado struct {
	Codigo       string      `json:"codigo" example:"proj-04"`
	Titulo       string      `json:"titulo" example:"Evolução do projeto (burn-up)"`
	Tipo         string      `json:"tipo" example:"linhas"`
	Dados        interface{} `json:"dados,omitempty" swaggertype:"object"`
	Alerta       interface{} `json:"alerta" swaggertype:"object"`
	AtualizadoEm *time.Time  `json:"atualizado_em,omitempty" example:"2026-11-14T12:00:00-03:00"`
	Erro         string      `json:"erro,omitempty" example:""`
}

// CardAcesso são os dados de um card para checar acesso e calcular.
type CardAcesso struct {
	CardMeta
	DashboardID    int64
	DashboardChave string
	// Ativo: o card e o dashboard dele estão ativos.
	Ativo bool
	// Permitido: o perfil tem o card em perfil_cards.
	Permitido bool
}

// DashboardRef identifica o dashboard na resposta dos dados.
type DashboardRef struct {
	ID     int64  `json:"id" example:"1"`
	Chave  string `json:"chave" example:"projetos"`
	Titulo string `json:"titulo" example:"Projetos"`
}

// DadosDashboard é o conteúdo de "dados" no GET /dashboards/:chave/dados.
type DadosDashboard struct {
	Dashboard        DashboardRef    `json:"dashboard"`
	FiltrosAplicados Filtros         `json:"filtros_aplicados"`
	Cards            []CardResultado `json:"cards"`
}

// DadosCard é o conteúdo de "dados" no GET /cards/:codigo/dados.
type DadosCard struct {
	FiltrosAplicados Filtros       `json:"filtros_aplicados"`
	Card             CardResultado `json:"card"`
}
