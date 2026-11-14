package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"pulso-api/controllers"
)

// Register registra todas as rotas da API.
func Register(r *gin.Engine, db *sql.DB) {
	// Health check (público)
	health := &controllers.HealthController{DB: db}
	r.GET("/health", health.Verificar)

	// Rotas dos domínios — descomentar conforme as etapas forem entregues
	// RegisterAuth(r, db)
	// RegisterDashboards(r, db)
	// RegisterCards(r, db)
	// RegisterFiltros(r, db)
	// RegisterWS(r, db)
}
