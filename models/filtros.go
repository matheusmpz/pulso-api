package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// FormatoData é o formato das datas na API (ISO 8601, só a data).
const FormatoData = "2006-01-02"

// Data é um dia do calendário (meia-noite no fuso da aplicação).
// Em JSON vira "AAAA-MM-DD".
type Data struct {
	time.Time
}

// MarshalJSON grava a data como "AAAA-MM-DD" (ou null, se vazia).
func (d Data) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + d.Format(FormatoData) + `"`), nil
}

// UnmarshalJSON lê "AAAA-MM-DD" (ou null). A data lida fica à
// meia-noite em UTC; para o fuso do app, use utils.ParseData.
func (d *Data) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		*d = Data{}
		return nil
	}
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return fmt.Errorf("data inválida: %s", s)
	}
	t, err := time.Parse(FormatoData, s[1:len(s)-1])
	if err != nil {
		return fmt.Errorf("data inválida (use AAAA-MM-DD): %w", err)
	}
	d.Time = t
	return nil
}

// String devolve "AAAA-MM-DD".
func (d Data) String() string {
	if d.IsZero() {
		return ""
	}
	return d.Format(FormatoData)
}

// Escopos possíveis dos dashboards projetos e ia.
const (
	EscopoProjeto = "projeto"
	EscopoTime    = "time"
	EscopoSprint  = "sprint"
)

// Chaves dos dashboards.
const (
	DashboardProjetos = "projetos"
	DashboardVendas   = "vendas"
	DashboardIA       = "ia"
)

// FiltrosEntrada são os filtros como chegam do cliente, ainda sem
// validação. Campos nil = não informados. No REST vêm da query string;
// no WebSocket, do objeto "filtros" (JSON).
type FiltrosEntrada struct {
	Escopo        *string  `json:"escopo"`
	ID            *int64   `json:"id"`
	De            *string  `json:"de"`
	Ate           *string  `json:"ate"`
	TipoProjeto   []string `json:"tipo_projeto"`
	StatusProjeto []string `json:"status_projeto"`
	OrigemLead    []string `json:"origem_lead"`
	Modelo        []string `json:"modelo"`
}

// Filtros são os filtros já validados e normalizados para um dashboard:
// padrões aplicados, listas sem repetição e ordenadas, e campos que não
// se aplicam ao dashboard zerados. Listas nil = todos.
type Filtros struct {
	Dashboard     string   `json:"-"`
	Escopo        string   `json:"escopo,omitempty" example:"projeto"`
	ID            *int64   `json:"id" example:"3"`
	De            Data     `json:"de" swaggertype:"string" example:"2026-10-16"`
	Ate           Data     `json:"ate" swaggertype:"string" example:"2026-11-14"`
	TipoProjeto   []string `json:"tipo_projeto" example:"web"`
	StatusProjeto []string `json:"status_projeto"`
	OrigemLead    []string `json:"origem_lead"`
	Modelo        []string `json:"modelo"`
}

// Chave identifica de forma única o dashboard + filtros normalizados.
// Serve de chave de cache e de assinatura do WebSocket.
func (f Filtros) Chave() string {
	b, err := json.Marshal(f)
	if err != nil {
		// Não acontece com os tipos usados; mantém a assinatura simples
		return f.Dashboard + "|" + fmt.Sprintf("%+v", f)
	}
	return f.Dashboard + "|" + string(b)
}

// Sprint é uma sprint, com o nome do time.
type Sprint struct {
	ID       int64  `json:"id" example:"4"`
	TimeID   int64  `json:"time_id" example:"1"`
	TimeNome string `json:"time_nome" example:"Time Alfa"`
	Numero   int    `json:"numero" example:"12"`
	Inicio   Data   `json:"inicio" swaggertype:"string" example:"2026-11-03"`
	Fim      Data   `json:"fim" swaggertype:"string" example:"2026-11-14"`
}

// SprintOpcao é uma sprint na lista de opções, indicando se é a atual.
type SprintOpcao struct {
	Sprint
	Atual bool `json:"atual" example:"true"`
}

// ProjetoOpcao é um projeto na lista de opções.
type ProjetoOpcao struct {
	ID     int64  `json:"id" example:"3"`
	Nome   string `json:"nome" example:"Portal do Cliente"`
	Tipo   string `json:"tipo" example:"web"`
	Status string `json:"status" example:"em_andamento"`
}

// TimeOpcao é um time na lista de opções.
type TimeOpcao struct {
	ID   int64  `json:"id" example:"1"`
	Nome string `json:"nome" example:"Time Alfa"`
}

// OpcoesFiltros é o conteúdo de "dados" no GET /filtros/opcoes.
type OpcoesFiltros struct {
	TiposProjeto  []OpcaoEnum    `json:"tipos_projeto"`
	StatusProjeto []OpcaoEnum    `json:"status_projeto"`
	Origens       []string       `json:"origens" example:"indicacao"`
	Modelos       []string       `json:"modelos" example:"claude-sonnet"`
	Projetos      []ProjetoOpcao `json:"projetos"`
	Times         []TimeOpcao    `json:"times"`
	Sprints       []SprintOpcao  `json:"sprints"`
}
