package tempo_real

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Prazos do protocolo (seção 13.3).
const (
	intervaloPing     = 30 * time.Second // ping do servidor
	prazoLeitura      = 60 * time.Second // sem pong/mensagem nesse tempo, a conexão cai
	prazoEscrita      = 10 * time.Second
	tamanhoBuffer     = 32        // mensagens na fila de envio
	tamanhoMaxLeitura = 16 * 1024 // bytes por mensagem do cliente
)

// Conexao é um cliente WebSocket autenticado.
type Conexao struct {
	ws       *websocket.Conn
	hub      *Hub
	PerfilID int64
	expira   time.Time

	envio        chan []byte
	feito        chan struct{}
	pedidoFechar chan string
	fechar       sync.Once

	grupo string // chave do grupo de assinatura (protegido por hub.mu)
}

// Servir registra a conexão no hub e roda a leitura e a escrita até ela
// cair. Bloqueia até o fim da conexão.
func (h *Hub) Servir(ws *websocket.Conn, perfilID int64, expira time.Time) {
	c := &Conexao{
		ws: ws, hub: h, PerfilID: perfilID, expira: expira,
		envio: make(chan []byte, tamanhoBuffer), feito: make(chan struct{}), pedidoFechar: make(chan string, 1),
	}
	h.adicionar(c)
	go c.escrever()
	c.ler()
}

// Enviar põe a mensagem na fila. Se a fila estiver cheia (cliente lento),
// fecha a conexão em vez de travar o hub.
func (c *Conexao) Enviar(msg []byte) {
	select {
	case <-c.feito:
	case c.envio <- msg:
	default:
		log.Println("Tempo real: cliente lento, conexão encerrada")
		c.encerrar()
	}
}

// Fechar pede à goroutine de escrita que envie o aviso (motivo) e encerre
// a conexão. Só ela escreve mensagens de texto (o gorilla/websocket aceita
// um escritor por vez).
func (c *Conexao) Fechar(motivo string) {
	select {
	case c.pedidoFechar <- motivo:
	case <-c.feito:
	default: // já há um pedido de fechamento
	}
}

// encerrar fecha uma única vez: tira do hub, manda o frame de fechamento
// (WriteControl é seguro em paralelo) e fecha o socket.
func (c *Conexao) encerrar() {
	c.fechar.Do(func() {
		close(c.feito)
		c.hub.remover(c)
		_ = c.ws.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		_ = c.ws.Close()
	})
}

// ler trata as mensagens do cliente e o pong do keepalive.
func (c *Conexao) ler() {
	defer c.encerrar()
	c.ws.SetReadLimit(tamanhoMaxLeitura)
	_ = c.ws.SetReadDeadline(time.Now().Add(prazoLeitura))
	c.ws.SetPongHandler(func(string) error {
		return c.ws.SetReadDeadline(time.Now().Add(prazoLeitura))
	})

	for {
		_, dados, err := c.ws.ReadMessage()
		if err != nil {
			return // cliente saiu, prazo vencido ou conexão fechada
		}
		_ = c.ws.SetReadDeadline(time.Now().Add(prazoLeitura))

		var msg MensagemCliente
		if err := json.Unmarshal(dados, &msg); err != nil || msg.Tipo == "" {
			c.Enviar(mensagemErro(MsgMensagemInvalida))
			continue
		}
		switch msg.Tipo {
		case TipoAssinar:
			ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
			c.hub.Assinar(ctx, c, msg)
			cancelar()
		default:
			c.Enviar(mensagemErro(MsgTipoDesconhecido))
		}
	}
}

// escrever envia a fila, o ping periódico e o aviso de sessão expirada.
func (c *Conexao) escrever() {
	ping := time.NewTicker(intervaloPing)
	defer ping.Stop()
	expiracao := time.NewTimer(time.Until(c.expira))
	defer expiracao.Stop()

	for {
		select {
		case <-c.feito:
			return
		case msg := <-c.envio:
			_ = c.ws.SetWriteDeadline(time.Now().Add(prazoEscrita))
			if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.encerrar()
				return
			}
		case <-ping.C:
			_ = c.ws.SetWriteDeadline(time.Now().Add(prazoEscrita))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.encerrar()
				return
			}
		case <-expiracao.C:
			c.avisarEEncerrar(MsgSessaoExpirada)
			return
		case motivo := <-c.pedidoFechar:
			c.avisarEEncerrar(motivo)
			return
		}
	}
}

// avisarEEncerrar envia {"tipo":"erro","mensagem":motivo} e encerra.
// Só é chamado pela goroutine de escrita.
func (c *Conexao) avisarEEncerrar(motivo string) {
	if motivo != "" {
		_ = c.ws.SetWriteDeadline(time.Now().Add(prazoEscrita))
		_ = c.ws.WriteMessage(websocket.TextMessage, mensagemErro(motivo))
	}
	c.encerrar()
}
