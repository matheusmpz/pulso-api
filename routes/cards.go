package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"pulso-api/controllers"
	"pulso-api/middlewares"
	"pulso-api/repositories"
	"pulso-api/utils"
)

// RegisterCards registra a rota de dados de um card isolado.
func RegisterCards(r *gin.Engine, db *sql.DB, jwtCfg utils.ConfigJWT, calculos controllers.CalculadoraCards) {
	ctrl := &controllers.CardController{
		Repo:       repositories.NewCardRepository(db),
		FiltroRepo: repositories.NewFiltroRepository(db),
		Calculos:   calculos,
	}

	g := r.Group("/cards")
	{
		g.Use(middlewares.AuthMiddleware(jwtCfg)) // protegido
		g.GET("/:codigo/dados", ctrl.Dados)
	}
}
