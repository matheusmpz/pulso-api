package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"pulso-api/controllers"
	"pulso-api/middlewares"
	"pulso-api/repositories"
	"pulso-api/utils"
)

// RegisterFiltros registra as rotas de opções de filtro.
func RegisterFiltros(r *gin.Engine, db *sql.DB, jwtCfg utils.ConfigJWT) {
	ctrl := &controllers.FiltroController{Repo: repositories.NewFiltroRepository(db)}

	g := r.Group("/filtros")
	{
		g.Use(middlewares.AuthMiddleware(jwtCfg)) // protegido
		g.GET("/opcoes", ctrl.Opcoes)
	}
}
