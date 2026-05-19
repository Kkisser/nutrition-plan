package auth

import (
	"testing"

	"github.com/google/uuid"
)

func TestHashAndCheck(t *testing.T) {
	h, err := HashPassword("Secret123")
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckPassword(h, "Secret123"); err != nil {
		t.Errorf("correct password rejected: %v", err)
	}
	if err := CheckPassword(h, "WrongPass1"); err == nil {
		t.Error("wrong password accepted")
	}
}

func TestTokenRoundtrip(t *testing.T) {
	id := uuid.New()
	tok, err := GenerateToken(id, "user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParseToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.UserID != id || c.Email != "user@example.com" {
		t.Errorf("claims mismatch: %+v", c)
	}
}

func TestParseToken_Invalid(t *testing.T) {
	if _, err := ParseToken("not-a-token"); err == nil {
		t.Error("expected error for bogus token")
	}
}

func TestValidatePassword(t *testing.T) {
	good := []string{"Secret12", "Abc12345", "MyPass1Word"}
	bad := []string{"short1A", "alllower1", "ALLUPPER1", "NoDigits", "Has space1A", "Кир1Latin"}
	for _, p := range good {
		if err := ValidatePassword(p); err != nil {
			t.Errorf("good password rejected %q: %v", p, err)
		}
	}
	for _, p := range bad {
		if err := ValidatePassword(p); err == nil {
			t.Errorf("bad password accepted %q", p)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	if err := ValidateEmail("a@b.co"); err != nil {
		t.Errorf("good email rejected: %v", err)
	}
	if err := ValidateEmail("bad"); err == nil {
		t.Error("bad email accepted")
	}
}
