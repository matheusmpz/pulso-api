package models

import (
	"encoding/json"
	"testing"
)

func TestDataJSONIdaEVolta(t *testing.T) {
	var d Data
	if err := json.Unmarshal([]byte(`"2026-11-14"`), &d); err != nil {
		t.Fatalf("erro ao ler: %v", err)
	}
	b, _ := json.Marshal(d)
	if string(b) != `"2026-11-14"` {
		t.Errorf("ida e volta: esperado \"2026-11-14\", obtido %s", b)
	}

	if err := json.Unmarshal([]byte(`null`), &d); err != nil || !d.IsZero() {
		t.Errorf("null deveria virar data vazia: %v %v", err, d)
	}
	if b, _ := json.Marshal(Data{}); string(b) != "null" {
		t.Errorf("data vazia deveria virar null, obtido %s", b)
	}
	for _, ruim := range []string{`"14/11/2026"`, `"2026-02-30"`, `20261114`, `"2026-11-14T10:00:00Z"`} {
		if err := json.Unmarshal([]byte(ruim), &d); err == nil {
			t.Errorf("%s deveria dar erro", ruim)
		}
	}
}
