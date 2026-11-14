package tempo_real

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"

	"pulso-api/models"
	"pulso-api/utils"
)

// Calculadora calcula um card (o *calculos.Servico satisfaz; ele já
// aplica o timeout de 10 s e isola falhas).
type Calculadora interface {
	CalcularCard(ctx context.Context, card models.CardMeta, f models.Filtros) models.CardResultado
}

// Acesso é o que o hub precisa para validar uma assinatura com as mesmas
// regras do REST.
type Acesso interface {
	// DashboardVisivel devolve o dashboard, os cards ativos que o perfil vê
	// e o resultado da regra de acesso (404/403/permitido).
	DashboardVisivel(ctx context.Context, chave string, perfilID int64) (*models.Dashboard, []models.CardMeta, utils.Acesso, error)
	// BuscarSprint é usada na normalização dos filtros (escopo=sprint).
	BuscarSprint(ctx context.Context, id int64) (models.Sprint, error)
}

// concorrenciaPadrao: recálculos simultâneos (seção 13.2).
const concorrenciaPadrao = 4

// grupo reúne as conexões com a mesma assinatura (dashboard, filtros
// normalizados, perfil): cada card afetado é recalculado uma vez por grupo.
type grupo struct {
	chave    string
	filtros  models.Filtros
	cards    []models.CardMeta
	conexoes map[*Conexao]bool

	mu     sync.Mutex
	hashes map[string]string // codigo → hash do último resultado enviado (ou base)
}

// Hub mantém as conexões e os grupos de assinatura.
type Hub struct {
	Calc         Calculadora
	Acesso       Acesso
	Gatilhos     map[string][]string // codigo → eventos que afetam o card
	Concorrencia int
	// Hoje devolve o dia atual (substituível nos testes).
	Hoje func() models.Data

	mu       sync.Mutex
	grupos   map[string]*grupo
	conexoes map[*Conexao]bool
}

// NovoHub cria o hub.
func NovoHub(calc Calculadora, acesso Acesso, gatilhos map[string][]string) *Hub {
	return &Hub{
		Calc: calc, Acesso: acesso, Gatilhos: gatilhos,
		Concorrencia: concorrenciaPadrao, Hoje: utils.Hoje,
		grupos: map[string]*grupo{}, conexoes: map[*Conexao]bool{},
	}
}

// adicionar registra uma conexão nova (sem assinatura ainda).
func (h *Hub) adicionar(c *Conexao) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conexoes[c] = true
}

// remover tira a conexão do hub e do grupo dela.
func (h *Hub) remover(c *Conexao) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conexoes, c)
	h.sairDoGrupo(c)
}

// sairDoGrupo tira a conexão do grupo atual (h.mu travado). Grupos vazios
// são descartados.
func (h *Hub) sairDoGrupo(c *Conexao) {
	if c.grupo == "" {
		return
	}
	if g, ok := h.grupos[c.grupo]; ok {
		delete(g.conexoes, c)
		if len(g.conexoes) == 0 {
			delete(h.grupos, c.grupo)
		}
	}
	c.grupo = ""
}

// Assinar valida e registra a assinatura da conexão (uma nova substitui a
// anterior) e responde "assinado" ou "erro".
func (h *Hub) Assinar(ctx context.Context, c *Conexao, msg MensagemCliente) {
	dashboard, cards, acesso, err := h.Acesso.DashboardVisivel(ctx, msg.Dashboard, c.PerfilID)
	if err != nil {
		log.Println("Tempo real: erro ao validar assinatura: ", err)
		c.Enviar(mensagemErro(MsgErroInternoTR))
		return
	}
	switch acesso {
	case utils.AcessoNaoEncontrado:
		c.Enviar(mensagemErro(MsgDashboardInvalido))
		return
	case utils.AcessoNegado:
		c.Enviar(mensagemErro(utils.MsgSemAcesso))
		return
	}

	filtros, err := utils.NormalizarFiltros(ctx, msg.Filtros, dashboard.Chave, h.Hoje(), h.Acesso.BuscarSprint)
	if err != nil {
		var ef *utils.ErroFiltro
		if errors.As(err, &ef) {
			c.Enviar(mensagemErro(ef.Mensagem))
			return
		}
		log.Println("Tempo real: erro ao normalizar filtros: ", err)
		c.Enviar(mensagemErro(MsgErroInternoTR))
		return
	}

	chave := fmt.Sprintf("%s|perfil=%d", filtros.Chave(), c.PerfilID)
	h.mu.Lock()
	h.sairDoGrupo(c)
	g, existe := h.grupos[chave]
	if !existe {
		g = &grupo{chave: chave, filtros: filtros, cards: cards, conexoes: map[*Conexao]bool{}, hashes: map[string]string{}}
		h.grupos[chave] = g
	}
	g.conexoes[c] = true
	c.grupo = chave
	h.mu.Unlock()

	codigos := make([]string, len(cards))
	for i, card := range cards {
		codigos[i] = card.Codigo
	}
	c.Enviar(codificar(MensagemAssinado{Tipo: TipoAssinado, Dashboard: dashboard.Chave, Cards: codigos}))

	// Grupo novo: calcula a base (o que o REST acabou de mostrar), para o
	// primeiro evento só enviar o que de fato mudou
	if !existe {
		go h.calcularBase(context.WithoutCancel(ctx), g)
	}
}

// calcularBase guarda o hash atual de cada card do grupo, sem enviar.
func (h *Hub) calcularBase(ctx context.Context, g *grupo) {
	h.emParalelo(g.cards, func(card models.CardMeta) {
		hash := hashResultado(h.Calc.CalcularCard(ctx, card, g.filtros))
		g.mu.Lock()
		if _, ja := g.hashes[card.Codigo]; !ja {
			g.hashes[card.Codigo] = hash
		}
		g.mu.Unlock()
	})
}

// Notificar recalcula, em cada grupo com conexões, os cards visíveis
// afetados pelos eventos (uma vez por grupo e por card) e envia só os que
// mudaram. Sem assinantes, não calcula nada.
func (h *Hub) Notificar(ctx context.Context, eventos []string) {
	afetados := h.cardsAfetados(eventos)
	if len(afetados) == 0 {
		return
	}

	type tarefa struct {
		g    *grupo
		card models.CardMeta
	}
	var tarefas []tarefa
	h.mu.Lock()
	for _, g := range h.grupos {
		if len(g.conexoes) == 0 {
			continue
		}
		for _, card := range g.cards {
			if afetados[card.Codigo] {
				tarefas = append(tarefas, tarefa{g, card})
			}
		}
	}
	h.mu.Unlock()
	if len(tarefas) == 0 {
		return
	}

	limite := h.Concorrencia
	if limite <= 0 {
		limite = concorrenciaPadrao
	}
	vagas := make(chan struct{}, limite)
	var wg sync.WaitGroup
	for _, t := range tarefas {
		wg.Add(1)
		go func(t tarefa) {
			defer wg.Done()
			vagas <- struct{}{}
			defer func() { <-vagas }()
			h.recalcular(ctx, t.g, t.card)
		}(t)
	}
	wg.Wait()
}

// recalcular calcula o card do grupo e envia às conexões se o resultado
// mudou em relação ao último enviado.
func (h *Hub) recalcular(ctx context.Context, g *grupo, card models.CardMeta) {
	resultado := h.Calc.CalcularCard(ctx, card, g.filtros)
	hash := hashResultado(resultado)

	g.mu.Lock()
	mudou := g.hashes[card.Codigo] != hash
	if mudou {
		g.hashes[card.Codigo] = hash
	}
	g.mu.Unlock()
	if !mudou {
		return
	}

	msg := codificar(MensagemCard{Tipo: TipoCardAtualizado, Card: card.Codigo, Dados: resultado})
	h.mu.Lock()
	destinos := make([]*Conexao, 0, len(g.conexoes))
	for c := range g.conexoes {
		destinos = append(destinos, c)
	}
	h.mu.Unlock()
	for _, c := range destinos {
		c.Enviar(msg)
	}
}

// cardsAfetados devolve os códigos cujos gatilhos incluem algum evento.
func (h *Hub) cardsAfetados(eventos []string) map[string]bool {
	ocorreram := map[string]bool{}
	for _, e := range eventos {
		ocorreram[e] = true
	}
	afetados := map[string]bool{}
	for codigo, gatilhos := range h.Gatilhos {
		for _, g := range gatilhos {
			if ocorreram[g] {
				afetados[codigo] = true
				break
			}
		}
	}
	return afetados
}

// emParalelo roda fn para cada card com a concorrência limitada.
func (h *Hub) emParalelo(cards []models.CardMeta, fn func(models.CardMeta)) {
	limite := h.Concorrencia
	if limite <= 0 {
		limite = concorrenciaPadrao
	}
	vagas := make(chan struct{}, limite)
	var wg sync.WaitGroup
	for _, card := range cards {
		wg.Add(1)
		go func(card models.CardMeta) {
			defer wg.Done()
			vagas <- struct{}{}
			defer func() { <-vagas }()
			fn(card)
		}(card)
	}
	wg.Wait()
}

// TotalConexoes devolve quantas conexões estão abertas.
func (h *Hub) TotalConexoes() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.conexoes)
}

// Fechar encerra todas as conexões (desligamento da API).
func (h *Hub) Fechar() {
	h.mu.Lock()
	todas := make([]*Conexao, 0, len(h.conexoes))
	for c := range h.conexoes {
		todas = append(todas, c)
	}
	h.mu.Unlock()
	for _, c := range todas {
		c.Fechar("Servidor encerrando.")
	}
}

// hashResultado resume o que o front vê de um card. atualizado_em fica de
// fora: muda a cada cálculo e não é uma mudança de dado.
func hashResultado(r models.CardResultado) string {
	b, _ := json.Marshal(struct {
		Dados interface{} `json:"dados"`
		Erro  string      `json:"erro"`
	}{r.Dados, r.Erro})
	soma := sha256.Sum256(b)
	return hex.EncodeToString(soma[:])
}
