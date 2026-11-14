package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"pulso-api/controllers"
	"pulso-api/middlewares"
	"pulso-api/repositories"
	"pulso-api/utils"
)

// RegisterDashboards registra as rotas de metadados dos dashboards.
func RegisterDashboards(r *gin.Engine, db *sql.DB, jwtCfg utils.ConfigJWT) {
	ctrl := &controllers.DashboardController{
		Repo:     repositories.NewDashboardRepository(db),
		CardRepo: repositories.NewCardRepository(db),
	}

	g := r.Group("/dashboards")
	{
		g.Use(middlewares.AuthMiddleware(jwtCfg)) // protegido
		g.GET("", ctrl.Listar)
		g.GET("/:chave", ctrl.Buscar)
		// GET /:chave/dados entra junto com os cálculos dos cards
	}
}
