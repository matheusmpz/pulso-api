package calculos

import "testing"

func valorOuNil(p *float64) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func TestArredondar(t *testing.T) {
	casos := []struct {
		v        float64
		casas    int
		esperado float64
	}{
		{72.45, 1, 72.5},
		{72.44, 1, 72.4},
		{-3.25, 1, -3.3},
		{1234.5678, 2, 1234.57},
		{0.005, 2, 0.01},
		{10, 1, 10},
	}
	for _, c := range casos {
		if got := Arredondar(c.v, c.casas); got != c.esperado {
			t.Errorf("Arredondar(%v, %d): esperado %v, obtido %v", c.v, c.casas, c.esperado, got)
		}
	}
}

func TestPercentual(t *testing.T) {
	casos := []struct {
		num, den float64
		esperado interface{}
	}{
		{29, 40, 72.5},
		{1, 3, 33.3},
		{2, 3, 66.7},
		{0, 10, 0.0},
		{150, 100, 150.0}, // acima de 100% é permitido (estouro)
		{5, 0, nil},       // denominador 0 → null
		{0, 0, nil},
	}
	for _, c := range casos {
		if got := valorOuNil(Percentual(c.num, c.den)); got != c.esperado {
			t.Errorf("Percentual(%v, %v): esperado %v, obtido %v", c.num, c.den, c.esperado, got)
		}
	}
}

func TestVariacao(t *testing.T) {
	casos := []struct {
		atual, anterior float64
		esperado        interface{}
	}{
		{110, 100, 10.0},
		{90, 100, -10.0},
		{100, 100, 0.0},
		{50, 0, nil}, // anterior 0 → null
		{0, 80, -100.0},
	}
	for _, c := range casos {
		if got := valorOuNil(Variacao(c.atual, c.anterior)); got != c.esperado {
			t.Errorf("Variacao(%v, %v): esperado %v, obtido %v", c.atual, c.anterior, c.esperado, got)
		}
	}
}

func TestFormatarPercentual(t *testing.T) {
	casos := map[float64]string{66.666: "66,7%", 100: "100,0%", 0: "0,0%", 85.25: "85,3%", 3.04: "3,0%"}
	for v, esperado := range casos {
		if got := FormatarPercentual(v); got != esperado {
			t.Errorf("FormatarPercentual(%v): esperado %q, obtido %q", v, esperado, got)
		}
	}
}
