package models

import "time"

// Usuario é um usuário do dashboard, já com os dados do perfil.
// SenhaHash nunca sai em JSON.
type Usuario struct {
	ID              string    `json:"id" db:"id"`
	Nome            string    `json:"nome" db:"nome"`
	Email           string    `json:"email" db:"email"`
	SenhaHash       string    `json:"-" db:"senha_hash"`
	PerfilID        int64     `json:"perfil_id" db:"perfil_id"`
	PerfilNome      string    `json:"perfil" db:"perfil_nome"`
	PerfilDescricao string    `json:"-" db:"perfil_descricao"`
	Ativo           bool      `json:"ativo" db:"ativo"`
	CriadoEm        time.Time `json:"criado_em" db:"criado_em"`
}

// LoginInput é o corpo do POST /auth/login.
type LoginInput struct {
	Email string `json:"email" binding:"required" example:"diretoria@pulso.local"`
	Senha string `json:"senha" binding:"required" example:"sua_senha"`
}

// UsuarioResumo são os dados do usuário devolvidos no login.
type UsuarioResumo struct {
	ID     string `json:"id" example:"5b1f8a2e-3c4d-4e5f-8a9b-0c1d2e3f4a5b"`
	Nome   string `json:"nome" example:"Diretoria (dev)"`
	Email  string `json:"email" example:"diretoria@pulso.local"`
	Perfil string `json:"perfil" example:"diretoria"`
}

// LoginResposta é o conteúdo de "dados" no login com sucesso.
type LoginResposta struct {
	Token    string        `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiraEm time.Time     `json:"expira_em" example:"2026-11-15T12:00:00-03:00"`
	Usuario  UsuarioResumo `json:"usuario"`
}

// PerfilResumo são os dados do perfil devolvidos no /auth/me.
type PerfilResumo struct {
	ID        int64  `json:"id" example:"1"`
	Nome      string `json:"nome" example:"diretoria"`
	Descricao string `json:"descricao" example:"Acesso a todos os dashboards e cards, incluindo vendas."`
}

// MeResposta é o conteúdo de "dados" no GET /auth/me.
type MeResposta struct {
	ID     string       `json:"id" example:"5b1f8a2e-3c4d-4e5f-8a9b-0c1d2e3f4a5b"`
	Nome   string       `json:"nome" example:"Diretoria (dev)"`
	Email  string       `json:"email" example:"diretoria@pulso.local"`
	Perfil PerfilResumo `json:"perfil"`
}
