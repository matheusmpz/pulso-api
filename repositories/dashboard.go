package repositories

import (
	"context"
	"database/sql"

	"pulso-api/models"
)

// DashboardRepository é o acesso aos dashboards (só leitura).
type DashboardRepository interface {
	// ListarVisiveis devolve os dashboards ativos que têm ao menos um card
	// ativo permitido ao perfil, com a contagem desses cards.
	ListarVisiveis(ctx context.Context, perfilID int64) ([]models.DashboardResumo, error)
	// BuscarPorChave devolve o dashboard (ativo ou não) pela chave.
	// Devolve sql.ErrNoRows se não existir.
	BuscarPorChave(ctx context.Context, chave string) (*models.Dashboard, error)
}

type dashboardRepository struct {
	db *sql.DB
}

// NewDashboardRepository cria o repository de dashboards.
func NewDashboardRepository(db *sql.DB) DashboardRepository {
	return &dashboardRepository{db}
}

func (r *dashboardRepository) ListarVisiveis(ctx context.Context, perfilID int64) ([]models.DashboardResumo, error) {
	// Os JOINs só filtram (cada card aparece uma vez por perfil, pela PK de
	// perfil_cards), então o COUNT não é inflado.
	const query = `
		SELECT d.id, d.chave, d.titulo, d.descricao, COUNT(c.id)::int AS total_cards
		FROM dashboards d
		JOIN cards c         ON c.dashboard_id = d.id AND c.ativo = true
		JOIN perfil_cards pc ON pc.card_id = c.id AND pc.perfil_id = $1
		WHERE d.ativo = true
		GROUP BY d.id, d.chave, d.titulo, d.descricao
		ORDER BY d.id`

	rows, err := r.db.QueryContext(ctx, query, perfilID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dashboards := []models.DashboardResumo{}
	for rows.Next() {
		var d models.DashboardResumo
		if err := rows.Scan(&d.ID, &d.Chave, &d.Titulo, &d.Descricao, &d.TotalCards); err != nil {
			return nil, err
		}
		dashboards = append(dashboards, d)
	}
	return dashboards, rows.Err()
}

func (r *dashboardRepository) BuscarPorChave(ctx context.Context, chave string) (*models.Dashboard, error) {
	const query = `
		SELECT id, chave, titulo, descricao, ativo
		FROM dashboards
		WHERE chave = $1`

	var d models.Dashboard
	err := r.db.QueryRowContext(ctx, query, chave).Scan(&d.ID, &d.Chave, &d.Titulo, &d.Descricao, &d.Ativo)
	if err != nil {
		return nil, err
	}
	return &d, nil
}
