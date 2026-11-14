package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"pulso-api/controllers"
	"pulso-api/utils"
)

// Register registra todas as rotas da API.
func Register(r *gin.Engine, db *sql.DB, jwtCfg utils.ConfigJWT) {
	// Health check (público)
	health := &controllers.HealthController{DB: db}
	r.GET("/health", health.Verificar)

	RegisterAuth(r, db, jwtCfg)
	RegisterDashboards(r, db, jwtCfg)

	// Rotas dos domínios — descomentar conforme as etapas forem entregues
	// RegisterCards(r, db, jwtCfg)
	// RegisterFiltros(r, db, jwtCfg)
	// RegisterWS(r, db, jwtCfg)
}
