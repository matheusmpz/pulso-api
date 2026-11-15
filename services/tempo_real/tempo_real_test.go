package tempo_real

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"pulso-api/models"
	"pulso-api/utils"
)

// ---------------------------------------------------------------------
// Poller
// ---------------------------------------------------------------------

type fonteFalsa struct {
	mu  sync.Mutex
	ids map[string]int64
	err error
}

func (f *fonteFalsa) MaxIDs(context.Context) (map[string]int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	copia := map[string]int64{}
	for k, v := range f.ids {
		copia[k] = v
	}
	return copia, f.err
}

func TestPollerDetectaSoOQueCresceu(t *testing.T) {
	fonte := &fonteFalsa{ids: map[string]int64{"registros_horas": 10, "vendas": 3, "atividade_eventos": 7}}
	p := &Poller{Fonte: fonte}
	if err := p.Iniciar(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Nada mudou: nenhum evento (dados antigos não viram rajada)
	if ev := p.Ciclo(context.Background()); len(ev) != 0 {
		t.Errorf("sem mudança não deveria haver eventos: %v", ev)
	}

	// Duas tabelas cresceram
	fonte.ids["registros_horas"] = 12
	fonte.ids["vendas"] = 4
	if ev := p.Ciclo(context.Background()); !reflect.DeepEqual(ev, []string{"horas_lancadas", "venda_fechada"}) {
		t.Errorf("eventos inesperados: %v", ev)
	}
	// O cursor avançou: o mesmo id não gera evento de novo
	if ev := p.Ciclo(context.Background()); len(ev) != 0 {
		t.Errorf("cursor não avançou: %v", ev)
	}

	// Erro de banco: nada muda e o próximo ciclo segue
	fonte.err = errors.New("caiu")
	fonte.ids["vendas"] = 9
	if ev := p.Ciclo(context.Background()); len(ev) != 0 {
		t.Errorf("com erro não deveria haver eventos: %v", ev)
	}
	fonte.err = nil
	if ev := p.Ciclo(context.Background()); !reflect.DeepEqual(ev, []string{"venda_fechada"}) {
		t.Errorf("depois do erro deveria detectar a venda: %v", ev)
	}
}

func TestPollerRodarNotificaEPara(t *testing.T) {
	fonte := &fonteFalsa{ids: map[string]int64{"consumo_tokens": 1}}
	recebidos := make(chan []string, 10)
	p := &Poller{Fonte: fonte, Intervalo: 10 * time.Millisecond, Notificar: func(_ context.Context, ev []string) { recebidos <- ev }}
	if err := p.Iniciar(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, parar := context.WithCancel(context.Background())
	feito := make(chan struct{})
	go func() { p.Rodar(ctx); close(feito) }()

	fonte.mu.Lock()
	fonte.ids["consumo_tokens"] = 2
	fonte.mu.Unlock()
	select {
	case ev := <-recebidos:
		if !reflect.DeepEqual(ev, []string{"consumo_tokens"}) {
			t.Errorf("evento inesperado: %v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("o poller não notificou")
	}
	parar()
	select {
	case <-feito:
	case <-time.After(time.Second):
		t.Fatal("o poller não parou com o contexto")
	}
}

// ---------------------------------------------------------------------
// Hub + WebSocket
// ---------------------------------------------------------------------

// calcFalsa devolve um valor por card (alterável) e conta os cálculos.
type calcFalsa struct {
	mu       sync.Mutex
	valores  map[string]float64
	calculos int64
}

func (c *calcFalsa) CalcularCard(_ context.Context, card models.CardMeta, _ models.Filtros) models.CardResultado {
	atomic.AddInt64(&c.calculos, 1)
	c.mu.Lock()
	v := c.valores[card.Codigo]
	c.mu.Unlock()
	agora := time.Now()
	return models.CardResultado{Codigo: card.Codigo, Titulo: card.Titulo, Tipo: card.Tipo,
		Dados: map[string]float64{"valor": v}, AtualizadoEm: &agora}
}

func (c *calcFalsa) mudar(codigo string, v float64) {
	c.mu.Lock()
	c.valores[codigo] = v
	c.mu.Unlock()
}

// acessoFalso: "projetos" (proj-01, proj-02) para os perfis 1 e 2;
// "vendas" (ven-01) só para o perfil 1. Sprint 4 existe.
type acessoFalso struct{}

func (acessoFalso) DashboardVisivel(_ context.Context, chave string, perfilID int64) (*models.Dashboard, []models.CardMeta, utils.Acesso, error) {
	switch chave {
	case "projetos":
		return &models.Dashboard{ID: 1, Chave: "projetos", Ativo: true},
			[]models.CardMeta{{Codigo: "proj-01", Tipo: "numero"}, {Codigo: "proj-02", Tipo: "numero"}}, utils.AcessoPermitido, nil
	case "vendas":
		if perfilID != 1 {
			return &models.Dashboard{ID: 2, Chave: "vendas", Ativo: true}, nil, utils.AcessoNegado, nil
		}
		return &models.Dashboard{ID: 2, Chave: "vendas", Ativo: true}, []models.CardMeta{{Codigo: "ven-01", Tipo: "numero"}}, utils.AcessoPermitido, nil
	}
	return nil, nil, utils.AcessoNaoEncontrado, nil
}

func (acessoFalso) BuscarSprint(_ context.Context, id int64) (models.Sprint, error) {
	if id == 4 {
		return models.Sprint{ID: 4, Inicio: utils.Hoje(), Fim: utils.Hoje()}, nil
	}
	return models.Sprint{}, sql.ErrNoRows
}

var gatilhosTeste = map[string][]string{
	"proj-01": {models.EventoStatusAtividade},
	"proj-02": {models.EventoHorasLancadas},
	"ven-01":  {models.EventoVendaFechada},
}

// servidorTeste sobe um WebSocket que autentica pela query ?perfil=N&expira_ms=M.
func servidorTeste(t *testing.T) (*Hub, *calcFalsa, string) {
	t.Helper()
	_ = utils.ConfigurarFuso("America/Sao_Paulo")
	calc := &calcFalsa{valores: map[string]float64{"proj-01": 10, "proj-02": 20, "ven-01": 30}}
	hub := NovoHub(calc, acessoFalso{}, gatilhosTeste)
	upgrader := websocket.Upgrader{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		perfil := int64(1)
		if r.URL.Query().Get("perfil") == "2" {
			perfil = 2
		}
		expira := time.Now().Add(time.Hour)
		if r.URL.Query().Get("expira_ms") != "" {
			expira = time.Now().Add(200 * time.Millisecond)
		}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.Servir(ws, perfil, expira)
	}))
	t.Cleanup(func() { hub.Fechar(); srv.Close() })
	return hub, calc, "ws" + strings.TrimPrefix(srv.URL, "http")
}

// cliente é um cliente WebSocket de teste com um leitor contínuo: no
// gorilla/websocket uma leitura que estoura o prazo inutiliza a conexão,
// então as mensagens vão para um canal e os testes esperam nele.
type cliente struct {
	ws   *websocket.Conn
	msgs chan map[string]interface{}
	fim  chan struct{} // fechado quando o servidor encerra a conexão
}

func conectar(t *testing.T, url string) *cliente {
	t.Helper()
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("falha ao conectar: %v", err)
	}
	t.Cleanup(func() { ws.Close() })
	cl := &cliente{ws: ws, msgs: make(chan map[string]interface{}, 32), fim: make(chan struct{})}
	go func() {
		defer close(cl.fim)
		for {
			_, dados, err := ws.ReadMessage()
			if err != nil {
				return
			}
			var m map[string]interface{}
			if json.Unmarshal(dados, &m) == nil {
				cl.msgs <- m
			}
		}
	}()
	return cl
}

// receber espera uma mensagem JSON (ou falha após o prazo).
func receber(t *testing.T, cl *cliente, prazo time.Duration) map[string]interface{} {
	t.Helper()
	select {
	case m := <-cl.msgs:
		return m
	case <-cl.fim:
		t.Fatal("a conexão foi fechada sem mensagem")
	case <-time.After(prazo):
		t.Fatal("nenhuma mensagem recebida no prazo")
	}
	return nil
}

// nadaRecebido confere que nenhuma mensagem chega no prazo.
func nadaRecebido(t *testing.T, cl *cliente, prazo time.Duration) {
	t.Helper()
	select {
	case m := <-cl.msgs:
		t.Fatalf("não deveria receber nada, recebeu %v", m)
	case <-time.After(prazo):
	}
}

func assinar(t *testing.T, cl *cliente, dashboard string, filtros map[string]interface{}) map[string]interface{} {
	t.Helper()
	if err := cl.ws.WriteJSON(map[string]interface{}{"tipo": "assinar", "dashboard": dashboard, "filtros": filtros}); err != nil {
		t.Fatal(err)
	}
	return receber(t, cl, 2*time.Second)
}

// esperarBase espera o cálculo da base do grupo terminar.
func esperarBase(t *testing.T, calc *calcFalsa, minimo int64) {
	t.Helper()
	limite := time.Now().Add(2 * time.Second)
	for atomic.LoadInt64(&calc.calculos) < minimo {
		if time.Now().After(limite) {
			t.Fatalf("a base não foi calculada (%d cálculos)", calc.calculos)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestAssinarRespondeComOsCardsDoPerfil(t *testing.T) {
	_, _, url := servidorTeste(t)
	ws := conectar(t, url)

	m := assinar(t, ws, "projetos", nil)
	if m["tipo"] != "assinado" || m["dashboard"] != "projetos" || !reflect.DeepEqual(m["cards"], []interface{}{"proj-01", "proj-02"}) {
		t.Errorf("assinado inesperado: %v", m)
	}
}

func TestAssinarRecusada(t *testing.T) {
	_, _, url := servidorTeste(t)
	casos := []struct {
		nome, query, dashboard string
		filtros                map[string]interface{}
		mensagem               string
	}{
		{"sem permissão", "?perfil=2", "vendas", nil, utils.MsgSemAcesso},
		{"dashboard inexistente", "", "xyz", nil, MsgDashboardInvalido},
		{"filtro inválido", "", "projetos", map[string]interface{}{"escopo": "cliente"}, "escopo inválido"},
		{"sprint sem id", "", "projetos", map[string]interface{}{"escopo": "sprint"}, "exige o id"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			ws := conectar(t, url+c.query)
			m := assinar(t, ws, c.dashboard, c.filtros)
			if m["tipo"] != "erro" || !strings.Contains(m["mensagem"].(string), c.mensagem) {
				t.Errorf("esperado erro com %q, obtido %v", c.mensagem, m)
			}
		})
	}

	ws := conectar(t, url)
	_ = ws.ws.WriteMessage(websocket.TextMessage, []byte("isso não é json"))
	if m := receber(t, ws, time.Second); m["mensagem"] != MsgMensagemInvalida {
		t.Errorf("mensagem inválida: %v", m)
	}
	_ = ws.ws.WriteJSON(map[string]string{"tipo": "outro"})
	if m := receber(t, ws, time.Second); m["mensagem"] != MsgTipoDesconhecido {
		t.Errorf("tipo desconhecido: %v", m)
	}
}

func TestNotificarEnviaSoOsCardsAfetadosQueMudaram(t *testing.T) {
	hub, calc, url := servidorTeste(t)
	ws := conectar(t, url)
	assinar(t, ws, "projetos", nil)
	esperarBase(t, calc, 2)

	// Evento sem mudança de resultado: nada é enviado
	hub.Notificar(context.Background(), []string{models.EventoHorasLancadas})
	nadaRecebido(t, ws, 150*time.Millisecond)

	// proj-02 mudou e o evento é dele: só proj-02 chega
	calc.mudar("proj-02", 21)
	calc.mudar("proj-01", 99) // mudou, mas o evento não é dele
	hub.Notificar(context.Background(), []string{models.EventoHorasLancadas})
	m := receber(t, ws, time.Second)
	dados := m["dados"].(map[string]interface{})
	if m["tipo"] != "card_atualizado" || m["card"] != "proj-02" || dados["codigo"] != "proj-02" || dados["dados"].(map[string]interface{})["valor"] != 21.0 {
		t.Errorf("card_atualizado inesperado: %v", m)
	}
	nadaRecebido(t, ws, 150*time.Millisecond)

	// Evento de um card de outro dashboard: nada
	hub.Notificar(context.Background(), []string{models.EventoVendaFechada})
	nadaRecebido(t, ws, 150*time.Millisecond)

	// Agora o evento do proj-01 chega
	hub.Notificar(context.Background(), []string{models.EventoStatusAtividade})
	if m := receber(t, ws, time.Second); m["card"] != "proj-01" {
		t.Errorf("esperado proj-01: %v", m)
	}
}

func TestGrupoCalculaUmaVezParaTodasAsConexoes(t *testing.T) {
	hub, calc, url := servidorTeste(t)
	a, b := conectar(t, url), conectar(t, url)
	outroPerfil := conectar(t, url+"?perfil=2")
	assinar(t, a, "projetos", nil)
	assinar(t, b, "projetos", nil)
	assinar(t, outroPerfil, "projetos", nil)
	esperarBase(t, calc, 4) // base de 2 grupos (perfil 1 e perfil 2) × 2 cards

	antes := atomic.LoadInt64(&calc.calculos)
	calc.mudar("proj-02", 50)
	hub.Notificar(context.Background(), []string{models.EventoHorasLancadas})
	for _, ws := range []*cliente{a, b, outroPerfil} {
		if m := receber(t, ws, time.Second); m["card"] != "proj-02" {
			t.Errorf("todas as conexões deveriam receber proj-02: %v", m)
		}
	}
	// 2 grupos (perfil 1 e perfil 2) → 2 cálculos, não 3
	if n := atomic.LoadInt64(&calc.calculos) - antes; n != 2 {
		t.Errorf("esperados 2 cálculos (um por grupo), houve %d", n)
	}
}

func TestNovaAssinaturaSubstituiAAnterior(t *testing.T) {
	hub, calc, url := servidorTeste(t)
	ws := conectar(t, url)
	assinar(t, ws, "projetos", nil)
	assinar(t, ws, "vendas", nil)
	esperarBase(t, calc, 3)

	calc.mudar("proj-02", 77)
	hub.Notificar(context.Background(), []string{models.EventoHorasLancadas})
	nadaRecebido(t, ws, 150*time.Millisecond) // não assina mais projetos

	calc.mudar("ven-01", 31)
	hub.Notificar(context.Background(), []string{models.EventoVendaFechada})
	if m := receber(t, ws, time.Second); m["card"] != "ven-01" {
		t.Errorf("esperado ven-01: %v", m)
	}
}

func TestSessaoExpiradaAvisaEFecha(t *testing.T) {
	_, _, url := servidorTeste(t)
	ws := conectar(t, url+"?expira_ms=200")
	m := receber(t, ws, 2*time.Second)
	if m["tipo"] != "erro" || m["mensagem"] != MsgSessaoExpirada {
		t.Errorf("esperado aviso de sessão expirada: %v", m)
	}
	select {
	case <-ws.fim:
	case <-time.After(time.Second):
		t.Error("a conexão deveria ser fechada depois do aviso")
	}
}

func TestFecharEncerraTodasAsConexoes(t *testing.T) {
	hub, _, url := servidorTeste(t)
	ws := conectar(t, url)
	assinar(t, ws, "projetos", nil)
	if hub.TotalConexoes() != 1 {
		t.Fatalf("esperada 1 conexão, há %d", hub.TotalConexoes())
	}
	hub.Fechar()
	if m := receber(t, ws, time.Second); m["tipo"] != "erro" {
		t.Errorf("esperado aviso antes de fechar: %v", m)
	}
	limite := time.Now().Add(time.Second)
	for hub.TotalConexoes() != 0 && time.Now().Before(limite) {
		time.Sleep(5 * time.Millisecond)
	}
	if hub.TotalConexoes() != 0 {
		t.Errorf("o hub deveria ficar sem conexões, há %d", hub.TotalConexoes())
	}
}

func TestSemAssinantesNaoCalcula(t *testing.T) {
	hub, calc, _ := servidorTeste(t)
	hub.Notificar(context.Background(), []string{models.EventoHorasLancadas, models.EventoVendaFechada})
	if calc.calculos != 0 {
		t.Errorf("sem assinantes não deveria calcular nada: %d cálculos", calc.calculos)
	}
}

func TestHashIgnoraAtualizadoEm(t *testing.T) {
	a, b := time.Now(), time.Now().Add(time.Hour)
	r1 := models.CardResultado{Codigo: "x", Dados: 1, AtualizadoEm: &a}
	r2 := models.CardResultado{Codigo: "x", Dados: 1, AtualizadoEm: &b}
	if hashResultado(r1) != hashResultado(r2) {
		t.Error("atualizado_em não pode mudar o hash")
	}
	if hashResultado(r1) == hashResultado(models.CardResultado{Codigo: "x", Dados: 2}) {
		t.Error("dados diferentes devem mudar o hash")
	}
}
