package controllers

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"pulso-api/services/tempo_real"
	"pulso-api/utils"
)

// WSController faz o handshake do WebSocket.
type WSController struct {
	Hub     *tempo_real.Hub
	JWT     utils.ConfigJWT
	Origens []string // CORS_ORIGINS: origens aceitas no upgrade

	upgrader *websocket.Upgrader
}

// NovoWSController cria o controller com a checagem de origem.
func NovoWSController(hub *tempo_real.Hub, jwt utils.ConfigJWT, origens []string) *WSController {
	ctrl := &WSController{Hub: hub, JWT: jwt, Origens: origens}
	ctrl.upgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		CheckOrigin:     ctrl.origemPermitida,
	}
	return ctrl
}

// origemPermitida aceita as origens de CORS_ORIGINS. Clientes que não são
// navegador (sem cabeçalho Origin) são aceitos: o token já é obrigatório.
func (h *WSController) origemPermitida(r *http.Request) bool {
	origem := r.Header.Get("Origin")
	if origem == "" {
		return true
	}
	for _, o := range h.Origens {
		if strings.EqualFold(o, origem) {
			return true
		}
	}
	return false
}

// Conectar godoc
//
// GET /ws — WebSocket de atualizações dos cards
//
// @Summary      WebSocket (tempo real)
// @Description  Abre o WebSocket de atualizações. O token vai na query (?token=<jwt>), porque o navegador não envia o cabeçalho Authorization no upgrade; ele é validado antes do upgrade (401 se inválido). A origem precisa estar em CORS_ORIGINS (403 se não estiver). Depois de conectar, o cliente envia {"tipo":"assinar","dashboard":"projetos","filtros":{...}} e recebe {"tipo":"assinado",...} e, a cada mudança, {"tipo":"card_atualizado","card":"proj-04","dados":{CardResultado}}. Detalhes do protocolo na seção "Tempo real (WebSocket)" do README.
// @Tags         tempo real
// @Param        token  query     string  true  "JWT obtido no POST /auth/login"
// @Success      101    {string}  string  "Switching Protocols"
// @Failure      401    {object}  ErroDoc  "Token ausente, inválido ou expirado"
// @Failure      403    {object}  ErroDoc  "Origem não permitida"
// @Router       /ws [get]
func (h *WSController) Conectar(c *gin.Context) {
	claims, err := utils.ValidarToken(h.JWT, c.Query("token"))
	if err != nil {
		utils.RespostaErro(c, http.StatusUnauthorized, "Token inválido ou expirado.")
		return
	}
	if !h.origemPermitida(c.Request) {
		utils.RespostaErro(c, http.StatusForbidden, "Origem não permitida.")
		return
	}

	ws, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Tempo real: falha no upgrade: ", err) // o upgrader já respondeu
		return
	}
	h.Hub.Servir(ws, claims.PerfilID, claims.ExpiresAt.Time)
}
