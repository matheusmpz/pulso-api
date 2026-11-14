package models

// Formatos de "dados" de cada tipo de card (seção 11). Os nomes das chaves
// seguem o formato de dados do Recharts, usado pelo shadcn/ui no front.
//
// Convenções:
//   - razão com denominador 0 → null (o front mostra "—");
//   - percentuais com 1 casa, R$ e horas com 2, dias com 1;
//   - dado vazio nunca é erro: zeros ou listas vazias, nunca null nas listas.

// ---------------------------------------------------------------------
// numero
// ---------------------------------------------------------------------

// DadosNumero: um valor em destaque, com variação e detalhes opcionais.
type DadosNumero struct {
	Valor    *float64  `json:"valor" example:"72.5"`
	Unidade  string    `json:"unidade" example:"%"`
	Variacao *Variacao `json:"variacao"`
	Detalhe  []Detalhe `json:"detalhe"`
}

// Variacao em relação ao período anterior.
type Variacao struct {
	Valor   *float64 `json:"valor" example:"3.2"`
	Unidade string   `json:"unidade" example:"%"`
}

// Detalhe é uma linha de informação complementar (numero e radial).
type Detalhe struct {
	Rotulo  string   `json:"rotulo" example:"Concluídas"`
	Valor   *float64 `json:"valor" example:"12"`
	Unidade string   `json:"unidade" example:"atividades"`
}

// ---------------------------------------------------------------------
// linhas e area
// ---------------------------------------------------------------------

// DadosSerieTemporal: pontos com uma chave por série (formato Recharts).
// Cada ponto é {"x": ..., "<chave da série>": valor | null, ...}.
type DadosSerieTemporal struct {
	Series     []Serie                  `json:"series"`
	Pontos     []map[string]interface{} `json:"pontos"`
	EixoX      string                   `json:"eixo_x" example:"Horas gastas (acumuladas)"`
	EixoY      string                   `json:"eixo_y" example:"Evolução (%)"`
	Unidade    string                   `json:"unidade" example:"%"`
	Referencia *Referencia              `json:"referencia"`
	Extra      map[string]interface{}   `json:"extra"`
}

// Serie descreve uma série de um gráfico. Pilha só existe em
// barras_agrupadas: séries com o mesmo texto empilham juntas.
type Serie struct {
	Chave string  `json:"chave" example:"real"`
	Nome  string  `json:"nome" example:"Real"`
	Pilha *string `json:"pilha,omitempty"`
}

// Referencia é uma linha horizontal de referência (ex.: limite de tokens).
type Referencia struct {
	Nome  string  `json:"nome" example:"Limite de tokens"`
	Valor float64 `json:"valor" example:"1000000"`
}

// ---------------------------------------------------------------------
// barras
// ---------------------------------------------------------------------

// DadosBarras: uma barra por item.
type DadosBarras struct {
	Itens      []ItemBarra `json:"itens"`
	Unidade    string      `json:"unidade" example:"h"`
	Horizontal bool        `json:"horizontal" example:"true"`
	Total      *int        `json:"total" example:"12"`
}

// ItemBarra é uma barra. Extra é um texto ou objeto complementar
// (ex.: "38%"), documentado por card.
type ItemBarra struct {
	Rotulo string      `json:"rotulo" example:"Integração com ERP"`
	Valor  *float64    `json:"valor" example:"12.5"`
	Extra  interface{} `json:"extra"`
}

// ---------------------------------------------------------------------
// barras_agrupadas
// ---------------------------------------------------------------------

// DadosBarrasAgrupadas: várias séries por item. Cada item é
// {"rotulo": ..., "<chave da série>": valor | null, ...chaves extras}.
type DadosBarrasAgrupadas struct {
	Series  []Serie                  `json:"series"`
	Itens   []map[string]interface{} `json:"itens"`
	Unidade string                   `json:"unidade" example:"h"`
}

// ---------------------------------------------------------------------
// donut
// ---------------------------------------------------------------------

// DadosDonut: fatias e total.
type DadosDonut struct {
	Fatias []Fatia `json:"fatias"`
	Total  float64 `json:"total" example:"10"`
}

// Fatia de um donut. Tokens só aparece no ia-07 (custo por modelo).
type Fatia struct {
	Rotulo string   `json:"rotulo" example:"Abertas"`
	Chave  string   `json:"chave" example:"aberta"`
	Valor  float64  `json:"valor" example:"4"`
	Tokens *float64 `json:"tokens,omitempty"`
}

// ---------------------------------------------------------------------
// radial
// ---------------------------------------------------------------------

// DadosRadial: um valor contra uma meta.
type DadosRadial struct {
	Valor   *float64  `json:"valor" example:"65"`
	Meta    float64   `json:"meta" example:"100"`
	Unidade string    `json:"unidade" example:"%"`
	Detalhe []Detalhe `json:"detalhe"`
}

// ---------------------------------------------------------------------
// lista
// ---------------------------------------------------------------------

// DadosLista: uma tabela simples.
type DadosLista struct {
	Colunas []string   `json:"colunas" example:"Projeto,Alerta,Valor,Quando"`
	Linhas  [][]string `json:"linhas"`
	Total   int        `json:"total" example:"3"`
}
