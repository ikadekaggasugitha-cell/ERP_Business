package platform

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestDecimalParseFormatAndSum(t *testing.T) {
	if _, err := ParseNumeric("invalid-num"); err == nil {
		t.Fatal("expected error for malformed decimal")
	}
	n, err := ParseNumeric("1234.5600")
	if err != nil {
		t.Fatalf("ParseNumeric: %v", err)
	}
	if got := FormatNumeric(n, 4); got != "1234.5600" {
		t.Fatalf("FormatNumeric = %q, want 1234.5600", got)
	}
	sum, err := SumRatStrings("0.10", "0.20")
	if err != nil {
		t.Fatalf("SumRatStrings: %v", err)
	}
	if sum.Cmp(big.NewRat(3, 10)) != 0 {
		t.Fatalf("expected exact 0.3, got %s", sum.FloatString(10))
	}
}

func TestCryptoEncryptDecryptAndBlindIndex(t *testing.T) {
	t.Setenv("APP_ENV", "dev")
	key, err := LoadPIIKey()
	if err != nil {
		t.Fatalf("LoadPIIKey: %v", err)
	}
	nik := "3174012345678901"
	enc, err := EncryptField(key, nik)
	if err != nil {
		t.Fatalf("EncryptField: %v", err)
	}
	if enc == nik {
		t.Fatal("ciphertext must differ from plaintext")
	}
	dec, err := DecryptField(key, enc)
	if err != nil || dec != nik {
		t.Fatalf("DecryptField = %q, err=%v, want %q", dec, err, nik)
	}
	h1 := HashLookup(key, " "+nik+" ")
	h2 := HashLookup(key, nik)
	if h1 == "" || h1 != h2 {
		t.Fatalf("HashLookup mismatch: %q vs %q", h1, h2)
	}
	if masked := MaskNationalID(nik); masked != "************8901" {
		t.Fatalf("MaskNationalID = %q", masked)
	}
}

func TestClassifyPGErrorAndEmptyScope(t *testing.T) {
	if ClassifyPGError(pgx.ErrNoRows) != PGNotFound {
		t.Fatal("expected PGNotFound for ErrNoRows")
	}
	if ClassifyPGError(&pgconn.PgError{Code: "42501"}) != PGRLS {
		t.Fatal("expected PGRLS for 42501")
	}
	if ClassifyPGError(errors.New("other")) != PGUnknown {
		t.Fatal("expected PGUnknown")
	}
	db := &DB{}
	if err := db.WithOrgScope(context.Background(), nil, nil); err == nil {
		t.Fatal("expected empty org scope to fail")
	}
}

