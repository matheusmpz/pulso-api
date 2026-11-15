package models

// Dashboard é um registro da tabela dashboards.
type Dashboard struct {
	ID        int64  `json:"id" db:"id" example:"1"`
	Chave     string `json:"chave" db:"chave" example:"projetos"`
	Titulo    string `json:"titulo" db:"titulo" example:"Projetos"`
	Descricao string `json:"descricao" db:"descricao" example:"Andamento dos projetos em desenvolvimento."`
	// Cor: nome de uma cor da paleta do Tailwind usada pelo front nos
	// gráficos do dashboard (emerald, violet ou orange); nil = sem cor.
	Cor   *string `json:"cor" db:"cor" example:"violet"`
	Ativo bool    `json:"-" db:"ativo"`
}

// DashboardResumo é um item do GET /dashboards: o dashboard e quantos
// cards ativos o perfil pode ver nele.
type DashboardResumo struct {
	ID         int64   `json:"id" example:"1"`
	Chave      string  `json:"chave" example:"projetos"`
	Titulo     string  `json:"titulo" example:"Projetos"`
	Descricao  string  `json:"descricao" example:"Andamento dos projetos em desenvolvimento."`
	Cor        *string `json:"cor" example:"violet"`
	TotalCards int     `json:"total_cards" example:"15"`
}

// ListaDashboards é o conteúdo de "dados" no GET /dashboards.
type ListaDashboards struct {
	Dashboards []DashboardResumo `json:"dashboards"`
}

// CardMeta são os metadados de um card, sem os dados calculados.
// A descricao é o texto do modal de informação do front.
type CardMeta struct {
	Codigo    string `json:"codigo" example:"proj-01"`
	Titulo    string `json:"titulo" example:"Progresso"`
	Descricao string `json:"descricao" example:"Mostra quanto do trabalho planejado já foi concluído."`
	Tipo      string `json:"tipo" example:"numero"`
}

// DashboardDetalhe é o conteúdo de "dados" no GET /dashboards/:chave.
type DashboardDetalhe struct {
	ID        int64      `json:"id" example:"1"`
	Chave     string     `json:"chave" example:"projetos"`
	Titulo    string     `json:"titulo" example:"Projetos"`
	Descricao string     `json:"descricao" example:"Andamento dos projetos em desenvolvimento."`
	Cor       *string    `json:"cor" example:"violet"`
	Cards     []CardMeta `json:"cards"`
}
