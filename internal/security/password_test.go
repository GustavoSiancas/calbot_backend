package security

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("safe-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	matched, err := VerifyPassword("safe-password", hash)
	if err != nil || !matched {
		t.Fatalf("VerifyPassword() = %v, %v; want true, nil", matched, err)
	}

	matched, err = VerifyPassword("incorrect", hash)
	if err != nil || matched {
		t.Fatalf("VerifyPassword() = %v, %v; want false, nil", matched, err)
	}
}
