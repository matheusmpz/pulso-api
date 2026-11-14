package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"pulso-api/controllers"
	"pulso-api/middlewares"
	"pulso-api/repositories"
	"pulso-api/utils"
)

// RegisterDashboards registra as rotas de metadados e dados dos dashboards.
func RegisterDashboards(r *gin.Engine, db *sql.DB, jwtCfg utils.ConfigJWT, calculos controllers.CalculadoraCards) {
	ctrl := &controllers.DashboardController{
		Repo:       repositories.NewDashboardRepository(db),
		CardRepo:   repositories.NewCardRepository(db),
		FiltroRepo: repositories.NewFiltroRepository(db),
		Calculos:   calculos,
	}

	g := r.Group("/dashboards")
	{
		g.Use(middlewares.AuthMiddleware(jwtCfg)) // protegido
		g.GET("", ctrl.Listar)
		g.GET("/:chave", ctrl.Buscar)
		g.GET("/:chave/dados", ctrl.Dados)
	}
}
