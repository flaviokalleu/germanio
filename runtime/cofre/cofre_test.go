package cofre

import (
	"errors"
	"strings"
	"testing"
)

const (
	k1 = "chave-numero-um-com-mais-de-32-caracteres"
	k2 = "chave-numero-dois-com-mais-de-32-caracteres"
)

func TestSelarEAbrir(t *testing.T) {
	c, err := Novo(k1)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := c.Selar("webhook.token", "s3nh4")
	b, _ := c.Selar("webhook.token", "s3nh4")
	if a == b {
		t.Fatal("o mesmo valor cifrado duas vezes deve sair diferente (nonce aleatório)")
	}
	if !strings.HasPrefix(a, c.PrefixoAtual()) || strings.Contains(a, "s3nh4") || !c.Atual(a) || !c.Conhece(a) {
		t.Fatalf("formato: %s", a)
	}
	if v, err := c.Abrir("webhook.token", a); err != nil || v != "s3nh4" {
		t.Fatalf("abrir: %q %v", v, err)
	}
	// bound to its purpose: moved to another column it does not open
	if _, err := c.Abrir("espelho.credencial", a); !errors.Is(err, ErrIlegivel) {
		t.Fatalf("outro propósito: %v", err)
	}
	// altered: refused
	alt := a[:len(a)-2] + "AA"
	if alt == a {
		alt = a[:len(a)-2] + "BB"
	}
	if _, err := c.Abrir("webhook.token", alt); !errors.Is(err, ErrIlegivel) {
		t.Fatalf("alterado: %v", err)
	}
	if _, err := c.Abrir("webhook.token", "ge1:semcorpo"); !errors.Is(err, ErrIlegivel) {
		t.Fatalf("truncado: %v", err)
	}
	// legacy plain text passes through, to be sealed by the caller
	if v, err := c.Abrir("webhook.token", "antigo"); err != nil || v != "antigo" {
		t.Fatalf("legado: %q %v", v, err)
	}
	if e, _ := c.Selar("x.y", ""); !Selado(e) {
		t.Fatal("o vazio também é cifrado quando pedido")
	}
}

func TestRotacao(t *testing.T) {
	velho, _ := Novo(k1)
	antigo, _ := velho.Selar("t.c", "valor")
	novo, err := Novo(k2, k1)
	if err != nil {
		t.Fatal(err)
	}
	if novo.Atual(antigo) || !novo.Conhece(antigo) {
		t.Fatal("o valor antigo é conhecido, mas não está na chave atual")
	}
	if v, err := novo.Abrir("t.c", antigo); err != nil || v != "valor" {
		t.Fatalf("a chave anterior abre: %q %v", v, err)
	}
	sem, _ := Novo(k2)
	if _, err := sem.Abrir("t.c", antigo); !errors.Is(err, ErrChaveDesconhecida) || sem.Conhece(antigo) {
		t.Fatalf("sem a chave anterior: %v", err)
	}
	if len(novo.ChavesLegadas("rotulo")) != 2 {
		t.Fatal("chaves legadas: uma por chave")
	}
}

func TestAmbiente(t *testing.T) {
	t.Setenv("GERMANIO_SEGREDO", "")
	t.Setenv("GERMANIO_SEGREDO_ANTERIOR", "")
	if c, err := DoAmbiente(); c != nil || err != nil {
		t.Fatalf("sem chave: %v %v", c, err)
	}
	t.Setenv("GERMANIO_SEGREDO", "curta")
	if _, err := DoAmbiente(); err == nil || !strings.Contains(err.Error(), "pelo menos 32") {
		t.Fatalf("chave curta: %v", err)
	}
	t.Setenv("GERMANIO_SEGREDO", "")
	t.Setenv("GERMANIO_SEGREDO_ANTERIOR", k1)
	if _, err := DoAmbiente(); err == nil || !strings.Contains(err.Error(), "falta GERMANIO_SEGREDO") {
		t.Fatalf("só a anterior: %v", err)
	}
	t.Setenv("GERMANIO_SEGREDO", k2)
	t.Setenv("GERMANIO_SEGREDO_ANTERIOR", k1+" curta")
	if _, err := DoAmbiente(); err == nil {
		t.Fatal("uma anterior curta é erro")
	}
	t.Setenv("GERMANIO_SEGREDO_ANTERIOR", k1)
	c, err := DoAmbiente()
	if err != nil || len(c.todas) != 2 {
		t.Fatalf("atual e anterior: %v", err)
	}
}
