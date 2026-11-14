package tempo_real

import (
	"context"
	"database/sql"
	"errors"

	"pulso-api/models"
	"pulso-api/repositories"
	"pulso-api/utils"
)

// AcessoBanco aplica no WebSocket as mesmas regras de acesso do REST
// (GET /dashboards/:chave), usando os repositories.
type AcessoBanco struct {
	Dashboards repositories.DashboardRepository
	Cards      repositories.CardRepository
	Filtros    repositories.FiltroRepository
}

// DashboardVisivel devolve o dashboard e os cards ativos que o perfil vê,
// com o resultado da regra de acesso.
func (a AcessoBanco) DashboardVisivel(ctx context.Context, chave string, perfilID int64) (*models.Dashboard, []models.CardMeta, utils.Acesso, error) {
	dashboard, err := a.Dashboards.BuscarPorChave(ctx, chave)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, utils.AcessoNaoEncontrado, err
	}
	existe := err == nil

	var cards []models.CardMeta
	if existe && dashboard.Ativo {
		if cards, err = a.Cards.ListarVisiveis(ctx, dashboard.ID, perfilID); err != nil {
			return nil, nil, utils.AcessoNaoEncontrado, err
		}
	}
	acesso := utils.AvaliarAcesso(existe, existe && dashboard.Ativo, len(cards) > 0)
	return dashboard, cards, acesso, nil
}

// BuscarSprint repassa ao repository de filtros.
func (a AcessoBanco) BuscarSprint(ctx context.Context, id int64) (models.Sprint, error) {
	return a.Filtros.BuscarSprint(ctx, id)
}
