package repository_test

import (
	"math/big"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/currency-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func TestScopeWithSystemOrg(t *testing.T) {
	t.Parallel()

	if got := repository.ScopeWithSystemOrg(nil); got != nil {
		t.Fatalf("expected nil for empty scope, got %v", got)
	}

	orgID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	sysID := uuid.MustParse("00000000-0000-0000-0000-000000000000")

	got := repository.ScopeWithSystemOrg([]uuid.UUID{orgID})
	if len(got) != 2 || got[0] != orgID || got[1] != sysID {
		t.Fatalf("expected [orgID, sysID], got %v", got)
	}

	gotExisting := repository.ScopeWithSystemOrg([]uuid.UUID{orgID, sysID})
	if len(gotExisting) != 2 || gotExisting[0] != orgID || gotExisting[1] != sysID {
		t.Fatalf("expected no duplicate sysID, got %v", gotExisting)
	}
}

// TestRationalTriangulationMath verifies the exact rational arithmetic used
// by the triangulation engine without database dependencies.
func TestRationalTriangulationMath(t *testing.T) {
	t.Parallel()

	// 1. Identity conversion (e.g. USD -> USD)
	identityRat := big.NewRat(1, 1)
	amtRat, _ := platform.SumRatStrings("1500.25")
	converted := new(big.Rat).Mul(amtRat, identityRat)
	if converted.FloatString(2) != "1500.25" {
		t.Fatalf("identity failed: got %s, want 1500.25", converted.FloatString(2))
	}

	// 2. Direct rate: 1 USD = 16250.50000000 IDR
	rateDirect, _ := platform.ParseNumeric("16250.50000000")
	ratDirect := platform.NumericToRat(rateDirect)
	usdAmount, _ := platform.ParseNumeric("100.00")
	idrResult := new(big.Rat).Mul(platform.NumericToRat(usdAmount), ratDirect)
	if idrResult.FloatString(2) != "1625050.00" {
		t.Fatalf("direct conversion failed: got %s, want 1625050.00", idrResult.FloatString(2))
	}

	// 3. Inverse rate: 1 IDR = 1 / 16250.50000000 USD
	ratInverse := new(big.Rat).Inv(ratDirect)
	// Convert 1625050.00 IDR back to USD
	usdBack := new(big.Rat).Mul(idrResult, ratInverse)
	if usdBack.FloatString(2) != "100.00" {
		t.Fatalf("inverse conversion failed: got %s, want 100.00", usdBack.FloatString(2))
	}

	// 4. Triangulation via USD anchor:
	// Suppose 1 USD = 16000 IDR, and 1 EUR = 1.08 USD
	// We want to convert 1000 EUR -> IDR:
	eurToUsd := big.NewRat(108, 100) // 1.08
	usdToIdr := big.NewRat(16000, 1)
	eurToIdr := new(big.Rat).Mul(eurToUsd, usdToIdr)
	if eurToIdr.FloatString(2) != "17280.00" {
		t.Fatalf("triangulated rate failed: got %s, want 17280.00", eurToIdr.FloatString(2))
	}

	eurAmt := big.NewRat(1000, 1)
	convertedIdr := new(big.Rat).Mul(eurAmt, eurToIdr)
	if convertedIdr.FloatString(2) != "17280000.00" {
		t.Fatalf("triangulated amount conversion failed: got %s, want 17280000.00", convertedIdr.FloatString(2))
	}
}

func TestDateParsing(t *testing.T) {
	t.Parallel()

	d, err := platform.ParseDate("2026-11-13")
	if err != nil {
		t.Fatalf("ParseDate valid failed: %v", err)
	}
	if !d.Valid {
		t.Fatal("expected date to be valid")
	}
	if platform.FormatDate(d) != "2026-11-13" {
		t.Fatalf("FormatDate mismatch: got %s", platform.FormatDate(d))
	}

	_, err = platform.ParseDate("invalid-date")
	if err == nil {
		t.Fatal("expected error on invalid date")
	}
}

func TestNumericHandlingEdgeCases(t *testing.T) {
	t.Parallel()

	// Empty string rejection
	_, err := platform.ParseNumeric("")
	if err == nil {
		t.Fatal("expected error on empty numeric")
	}

	// Valid high precision
	n, err := platform.ParseNumeric("0.00001234")
	if err != nil {
		t.Fatalf("ParseNumeric failed: %v", err)
	}
	s := platform.FormatNumeric(n, 8)
	if s != "0.00001234" {
		t.Fatalf("FormatNumeric mismatch: got %s, want 0.00001234", s)
	}
}
