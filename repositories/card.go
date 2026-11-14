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
