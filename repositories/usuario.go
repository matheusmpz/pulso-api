package repositories

import (
	"context"
	"database/sql"

	"pulso-api/models"
)

// UsuarioRepository é o acesso aos usuários (só leitura: os usuários vêm do seed).
type UsuarioRepository interface {
	// BuscarAtivoPorEmail busca um usuário ativo pelo e-mail, sem
	// diferenciar maiúsculas. Devolve sql.ErrNoRows se não existir.
	BuscarAtivoPorEmail(ctx context.Context, email string) (*models.Usuario, error)
	// BuscarAtivoPorID busca um usuário ativo pelo id (UUID).
	// Devolve sql.ErrNoRows se não existir.
	BuscarAtivoPorID(ctx context.Context, id string) (*models.Usuario, error)
}

type usuarioRepository struct {
	db *sql.DB
}

// NewUsuarioRepository cria o repository de usuários.
func NewUsuarioRepository(db *sql.DB) UsuarioRepository {
	return &usuarioRepository{db}
}

// selectUsuario é a base das buscas: usuário com os dados do perfil.
const selectUsuario = `
	SELECT u.id, u.nome, u.email, u.senha_hash, u.perfil_id,
	       p.nome, p.descricao, u.ativo, u.criado_em
	FROM usuarios u
	JOIN perfis p ON p.id = u.perfil_id`

func (r *usuarioRepository) BuscarAtivoPorEmail(ctx context.Context, email string) (*models.Usuario, error) {
	query := selectUsuario + `
	WHERE lower(u.email) = lower($1) AND u.ativo = true`
	return r.buscarUm(ctx, query, email)
}

func (r *usuarioRepository) BuscarAtivoPorID(ctx context.Context, id string) (*models.Usuario, error) {
	// O texto do id é comparado com o UUID convertido para texto, para um
	// id malformado resultar em "não encontrado" em vez de erro de cast.
	query := selectUsuario + `
	WHERE u.id::text = $1 AND u.ativo = true`
	return r.buscarUm(ctx, query, id)
}

func (r *usuarioRepository) buscarUm(ctx context.Context, query string, arg string) (*models.Usuario, error) {
	var u models.Usuario
	err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&u.ID, &u.Nome, &u.Email, &u.SenhaHash, &u.PerfilID,
		&u.PerfilNome, &u.PerfilDescricao, &u.Ativo, &u.CriadoEm,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
