package identity

import "testing"

func TestHashPasswordAndCompare(t *testing.T) {
	hash, err := HashPassword("s3cr3t-pass")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}

	ok, err := ComparePasswordAndHash("s3cr3t-pass", hash)
	if err != nil {
		t.Fatalf("ComparePasswordAndHash error: %v", err)
	}
	if !ok {
		t.Fatal("contraseña correcta no validada")
	}

	ok, _ = ComparePasswordAndHash("otra", hash)
	if ok {
		t.Fatal("contraseña incorrecta validada")
	}
}

func TestComparePasswordAndHashRechazaFormatoInvalido(t *testing.T) {
	if _, err := ComparePasswordAndHash("x", "$argon2id$m=1,t=1,p=1$c2FsdA=="); err == nil {
		t.Fatal("formato inválido debería dar error")
	}
}

func TestHashesSonUnicos(t *testing.T) {
	h1, _ := HashPassword("pass")
	h2, _ := HashPassword("pass")
	if h1 == h2 {
		t.Fatal("el hash debe incluir salt aleatorio")
	}
}
