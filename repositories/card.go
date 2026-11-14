package repositories

import (
	"context"
	"database/sql"

	"pulso-api/models"
)

// CardRepository é o acesso aos metadados dos cards (só leitura).
type CardRepository interface {
	// ListarVisiveis devolve os cards ativos do dashboard que o perfil pode
	// ver, ordenados por código. Lista vazia se não houver nenhum.
	ListarVisiveis(ctx context.Context, dashboardID, perfilID int64) ([]models.CardMeta, error)
	// ListarCodigosAtivos devolve o código de todos os cards ativos de
	// dashboards ativos (usado para validar o registro na inicialização).
	ListarCodigosAtivos(ctx context.Context) ([]string, error)
	// BuscarAcesso devolve o card, o dashboard dele e se o perfil pode
	// vê-lo. Devolve sql.ErrNoRows se o código não existir.
	BuscarAcesso(ctx context.Context, codigo string, perfilID int64) (*models.CardAcesso, error)
}

type cardRepository struct {
	db *sql.DB
}

// NewCardRepository cria o repository de cards.
func NewCardRepository(db *sql.DB) CardRepository {
	return &cardRepository{db}
}

func (r *cardRepository) ListarVisiveis(ctx context.Context, dashboardID, perfilID int64) ([]models.CardMeta, error) {
	const query = `
		SELECT c.codigo, c.titulo, c.descricao, c.tipo::text
		FROM cards c
		JOIN perfil_cards pc ON pc.card_id = c.id AND pc.perfil_id = $2
		WHERE c.dashboard_id = $1 AND c.ativo = true
		ORDER BY c.codigo`

	rows, err := r.db.QueryContext(ctx, query, dashboardID, perfilID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cards := []models.CardMeta{}
	for rows.Next() {
		var c models.CardMeta
		if err := rows.Scan(&c.Codigo, &c.Titulo, &c.Descricao, &c.Tipo); err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

func (r *cardRepository) ListarCodigosAtivos(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.codigo
		FROM cards c
		JOIN dashboards d ON d.id = c.dashboard_id
		WHERE c.ativo = true AND d.ativo = true
		ORDER BY c.codigo`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	codigos := []string{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		codigos = append(codigos, c)
	}
	return codigos, rows.Err()
}

func (r *cardRepository) BuscarAcesso(ctx context.Context, codigo string, perfilID int64) (*models.CardAcesso, error) {
	const query = `
		SELECT c.codigo, c.titulo, c.descricao, c.tipo::text,
		       d.id, d.chave, (c.ativo AND d.ativo) AS ativo,
		       EXISTS (SELECT 1 FROM perfil_cards pc
		               WHERE pc.card_id = c.id AND pc.perfil_id = $2) AS permitido
		FROM cards c
		JOIN dashboards d ON d.id = c.dashboard_id
		WHERE c.codigo = $1`

	var a models.CardAcesso
	err := r.db.QueryRowContext(ctx, query, codigo, perfilID).Scan(
		&a.Codigo, &a.Titulo, &a.Descricao, &a.Tipo,
		&a.DashboardID, &a.DashboardChave, &a.Ativo, &a.Permitido,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}
