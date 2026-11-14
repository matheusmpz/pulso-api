package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"pulso-api/controllers"
	"pulso-api/utils"
)

// Register registra todas as rotas da API.
func Register(r *gin.Engine, db *sql.DB, jwtCfg utils.ConfigJWT, calculos controllers.CalculadoraCards) {
	// Health check (público)
	health := &controllers.HealthController{DB: db}
	r.GET("/health", health.Verificar)

	RegisterAuth(r, db, jwtCfg)
	RegisterDashboards(r, db, jwtCfg, calculos)
	RegisterCards(r, db, jwtCfg, calculos)
	RegisterFiltros(r, db, jwtCfg)

	// Rotas dos domínios — descomentar conforme as etapas forem entregues
	// RegisterWS(r, db, jwtCfg)
}
