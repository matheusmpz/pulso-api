package routes

import (
	"github.com/gin-gonic/gin"

	"pulso-api/controllers"
	"pulso-api/services/tempo_real"
	"pulso-api/utils"
)

// RegisterWS registra o WebSocket. O token vai na query (?token=), e não
// no cabeçalho, por isso a rota não usa o AuthMiddleware.
func RegisterWS(r *gin.Engine, hub *tempo_real.Hub, jwtCfg utils.ConfigJWT, origens []string) {
	ctrl := controllers.NovoWSController(hub, jwtCfg, origens)
	r.GET("/ws", ctrl.Conectar)
}
