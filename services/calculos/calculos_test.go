package calculos

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"pulso-api/models"
)

// calcFalso é um calculador configurável para os testes.
type calcFalso struct {
	codigo  string
	espera  time.Duration
	err     error
	panico  bool
	ignorar bool // ignora o contexto (não para no timeout)
	emCurso *int32
	maximo  *int32
}

func (c calcFalso) Codigo() string     { return c.codigo }
func (c calcFalso) Gatilhos() []string { return []string{models.EventoConsumoTokens} }
func (c calcFalso) Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error) {
	if c.emCurso != nil {
		n := atomic.AddInt32(c.emCurso, 1)
		defer atomic.AddInt32(c.emCurso, -1)
		for {
			m := atomic.LoadInt32(c.maximo)
			if n <= m || atomic.CompareAndSwapInt32(c.maximo, m, n) {
				break
			}
		}
	}
	if c.panico {
		panic("bug no card")
	}
	if c.espera > 0 {
		if c.ignorar {
			time.Sleep(c.espera)
		} else {
			select {
			case <-time.After(c.espera):
			case <-ctx.Done():
				return models.CardResultado{}, ctx.Err()
			}
		}
	}
	if c.err != nil {
		return models.CardResultado{}, c.err
	}
	v := 1.0
	// Codigo/Titulo/Tipo errados de propósito: o serviço deve sobrescrever
	return models.CardResultado{Codigo: "x", Titulo: "x", Tipo: "x", Dados: models.DadosNumero{Valor: &v, Unidade: "%"}}, nil
}

func meta(codigo string) models.CardMeta {
	return models.CardMeta{Codigo: codigo, Titulo: "Título " + codigo, Tipo: "numero"}
}

func TestRegistroTemOs41CardsComGatilhosValidos(t *testing.T) {
	reg := NovoRegistro(nil)
	if len(reg) != 41 {
		t.Fatalf("esperados 41 cards, obtidos %d", len(reg))
	}
	por := map[string]int{}
	for codigo, calc := range reg {
		if calc.Codigo() != codigo {
			t.Errorf("chave %s aponta para calculador %s", codigo, calc.Codigo())
		}
		if len(calc.Gatilhos()) == 0 {
			t.Errorf("%s sem gatilhos", codigo)
		}
		for _, g := range calc.Gatilhos() {
			if _, ok := models.TabelaPorEvento[g]; !ok {
				t.Errorf("%s tem gatilho desconhecido %q", codigo, g)
			}
		}
		por[strings.SplitN(codigo, "-", 2)[0]]++
	}
	if por["proj"] != 15 || por["ven"] != 14 || por["ia"] != 12 {
		t.Errorf("esperados 15 proj, 14 ven, 12 ia; obtidos %v", por)
	}
	if _, ok := reg["ia-13"]; ok {
		t.Error("ia-13 não existe")
	}
}

func TestRegistroValidar(t *testing.T) {
	reg := Registro{}
	reg.Registrar(calcFalso{codigo: "proj-01"})
	reg.Registrar(calcFalso{codigo: "proj-02"})
	reg.Registrar(calcFalso{codigo: "velho-01"})

	faltando, sobrando := reg.Validar([]string{"proj-02", "proj-01", "ven-09", "ven-01"})
	if !reflect.DeepEqual(faltando, []string{"ven-01", "ven-09"}) {
		t.Errorf("faltando inesperado: %v", faltando)
	}
	if !reflect.DeepEqual(sobrando, []string{"velho-01"}) {
		t.Errorf("sobrando inesperado: %v", sobrando)
	}

	if f, s := NovoRegistro(nil).Validar(codigosDoRegistro()); len(f) != 0 || len(s) != 0 {
		t.Errorf("registro completo deveria bater com ele mesmo: %v %v", f, s)
	}
}

func codigosDoRegistro() []string {
	var c []string
	for codigo := range gatilhosPorCard {
		c = append(c, codigo)
	}
	return c
}

func TestCalcularCard(t *testing.T) {
	fixo := time.Date(2026, 11, 14, 15, 0, 0, 500, time.UTC)
	casos := []struct {
		nome string
		calc Calculador
		erro bool
	}{
		{"sucesso", calcFalso{codigo: "c"}, false},
		{"erro do cálculo", calcFalso{codigo: "c", err: errors.New("sql ruim")}, true},
		{"pânico", calcFalso{codigo: "c", panico: true}, true},
		{"timeout respeitando o contexto", calcFalso{codigo: "c", espera: time.Second}, true},
		{"timeout ignorando o contexto", calcFalso{codigo: "c", espera: time.Second, ignorar: true}, true},
		{"pendente (não implementado)", pendente{codigo: "c"}, true},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			reg := Registro{}
			reg.Registrar(c.calc)
			s := &Servico{Registro: reg, Timeout: 50 * time.Millisecond, Agora: func() time.Time { return fixo }}

			inicio := time.Now()
			r := s.CalcularCard(context.Background(), meta("c"), models.Filtros{})
			if time.Since(inicio) > 500*time.Millisecond {
				t.Errorf("o timeout não foi respeitado (%s)", time.Since(inicio))
			}
			if r.Codigo != "c" || r.Titulo != "Título c" || r.Tipo != "numero" {
				t.Errorf("envelope deve vir da tabela cards: %+v", r)
			}

			b, _ := json.Marshal(r)
			if c.erro {
				if r.Erro != models.MsgErroCard || r.Dados != nil || r.AtualizadoEm != nil {
					t.Errorf("falha deveria trazer só o erro padrão: %s", b)
				}
				return
			}
			if r.Erro != "" || r.Dados == nil || r.Alerta != nil {
				t.Errorf("sucesso inesperado: %s", b)
			}
			if r.AtualizadoEm.Format(time.RFC3339) != "2026-11-14T12:00:00-03:00" {
				t.Errorf("atualizado_em deve sair no fuso do app e sem frações: %s", r.AtualizadoEm.Format(time.RFC3339Nano))
			}
			if !strings.Contains(string(b), `"alerta":null`) {
				t.Errorf("alerta deve sair como null: %s", b)
			}
		})
	}

	// Card sem calculador
	s := &Servico{Registro: Registro{}}
	if r := s.CalcularCard(context.Background(), meta("sem-calc"), models.Filtros{}); r.Erro != models.MsgErroCard {
		t.Errorf("card sem calculador deveria falhar: %+v", r)
	}
}

func TestCalcularCardsConcorrenciaOrdemEIsolamento(t *testing.T) {
	var emCurso, maximo int32
	reg := Registro{}
	var cards []models.CardMeta
	for i := 0; i < 10; i++ {
		codigo := string(rune('a' + i))
		reg.Registrar(calcFalso{codigo: codigo, espera: 30 * time.Millisecond, emCurso: &emCurso, maximo: &maximo})
		cards = append(cards, meta(codigo))
	}
	// Um card que falha no meio não pode afetar os outros
	reg.Registrar(calcFalso{codigo: "e", err: errors.New("falhou")})

	s := &Servico{Registro: reg, Concorrencia: 4, Timeout: time.Second}
	resultados := s.CalcularCards(context.Background(), cards, models.Filtros{})

	if maximo > 4 {
		t.Errorf("no máximo 4 cálculos simultâneos; houve %d", maximo)
	}
	if maximo < 2 {
		t.Errorf("os cálculos deveriam rodar em paralelo; máximo simultâneo foi %d", maximo)
	}
	for i, r := range resultados {
		if r.Codigo != cards[i].Codigo {
			t.Errorf("ordem trocada na posição %d: %s", i, r.Codigo)
		}
		querErro := r.Codigo == "e"
		if (r.Erro != "") != querErro {
			t.Errorf("card %s: erro=%q", r.Codigo, r.Erro)
		}
	}
}
