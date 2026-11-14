package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"pulso-api/controllers"
	"pulso-api/middlewares"
	"pulso-api/repositories"
	"pulso-api/utils"
)

// RegisterAuth registra as rotas de autenticação.
func RegisterAuth(r *gin.Engine, db *sql.DB, jwtCfg utils.ConfigJWT) {
	repo := repositories.NewUsuarioRepository(db)
	ctrl := &controllers.AuthController{Repo: repo, JWT: jwtCfg}

	g := r.Group("/auth")
	{
		// Público
		g.POST("/login", ctrl.Login)

		// Protegido
		g.GET("/me", middlewares.AuthMiddleware(jwtCfg), ctrl.Me)
	}
}
