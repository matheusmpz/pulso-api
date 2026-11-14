package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"pulso-api/services/tempo_real"
	"pulso-api/utils"
)

func TestHandshakeWebSocket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := tempo_real.NovoHub(nil, nil, nil)
	ctrl := NovoWSController(hub, cfgJWTTeste, []string{"http://localhost:3000"})
	r := gin.New()
	r.GET("/ws", ctrl.Conectar)
	srv := httptest.NewServer(r)
	defer srv.Close()
	defer hub.Fechar()
	base := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	valido, _, _ := utils.GerarToken(cfgJWTTeste, "uuid-1", 1, "diretoria", "d@pulso.local", time.Now())
	expirado, _, _ := utils.GerarToken(cfgJWTTeste, "uuid-1", 1, "diretoria", "d@pulso.local", time.Now().Add(-48*time.Hour))

	casos := []struct {
		nome   string
		query  string
		origem string
		status int // 101 = upgrade aceito
	}{
		{"sem token", "", "", http.StatusUnauthorized},
		{"token inválido", "?token=abc", "", http.StatusUnauthorized},
		{"token expirado", "?token=" + expirado, "", http.StatusUnauthorized},
		{"origem não permitida", "?token=" + valido, "http://malicioso.com", http.StatusForbidden},
		{"origem permitida", "?token=" + valido, "http://localhost:3000", http.StatusSwitchingProtocols},
		{"sem origem (cliente que não é navegador)", "?token=" + valido, "", http.StatusSwitchingProtocols},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			cab := http.Header{}
			if c.origem != "" {
				cab.Set("Origin", c.origem)
			}
			ws, resp, err := websocket.DefaultDialer.Dial(base+c.query, cab)
			if ws != nil {
				defer ws.Close()
			}
			if resp == nil {
				t.Fatalf("sem resposta: %v", err)
			}
			if resp.StatusCode != c.status {
				t.Errorf("status esperado %d, obtido %d (%v)", c.status, resp.StatusCode, err)
			}
		})
	}
}
