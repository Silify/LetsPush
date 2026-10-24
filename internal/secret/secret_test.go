package secret

import (
	"strings"
	"testing"
)

func TestRandomHexIsRandomAndTheRightSize(t *testing.T) {
	a, err := RandomHex(16)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := RandomHex(16)
	if len(a) != 32 || a == b {
		t.Fatalf("unexpected values %q %q", a, b)
	}
}

func TestHashAndEqual(t *testing.T) {
	if Hash("a") == Hash("b") || Hash("a") != Hash("a") {
		t.Fatal("hash is not stable and distinct")
	}
	if !Equal("same", "same") || Equal("same", "other") || Equal("a", "ab") {
		t.Fatal("Equal is wrong")
	}
}

func TestPasswords(t *testing.T) {
	stored, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, "correct") {
		t.Fatal("the password leaked into the hash")
	}
	if !CheckPassword("correct horse battery", stored) {
		t.Fatal("the right password was refused")
	}
	if CheckPassword("wrong", stored) {
		t.Fatal("a wrong password was accepted")
	}
	again, _ := HashPassword("correct horse battery")
	if again == stored {
		t.Fatal("the salt is not random")
	}
}

func TestCheckPasswordRefusesMalformedValues(t *testing.T) {
	for _, stored := range []string{"", "plain", "a$b$c$d", "pbkdf2-sha256$x$AAAA$AAAA", "pbkdf2-sha256$1$!!$AAAA"} {
		if CheckPassword("x", stored) {
			t.Errorf("accepted %q", stored)
		}
	}
}

func TestBoxSealsAndOpens(t *testing.T) {
	box, err := NewBox(strings.Repeat("k", 32))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal([]byte(`{"private_key":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sealed, "private_key") {
		t.Fatal("the value was not encrypted")
	}
	plain, err := box.Open(sealed)
	if err != nil || string(plain) != `{"private_key":"x"}` {
		t.Fatalf("round trip failed: %v %q", err, plain)
	}

	other, _ := NewBox(strings.Repeat("z", 32))
	if _, err := other.Open(sealed); err != ErrCannotOpen {
		t.Fatalf("opened with the wrong secret: %v", err)
	}
	if _, err := box.Open("not base64!"); err != ErrCannotOpen {
		t.Fatalf("opened garbage: %v", err)
	}
}
