package repository

import (
	"testing"

	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func TestPayrollAmountEncryptionRoundTrip(t *testing.T) {
	key, err := platform.LoadPIIKey()
	if err != nil {
		t.Fatalf("LoadPIIKey: %v", err)
	}
	repo := NewPayrollRunRepository(nil, key)
	encG, encT, encN, err := repo.EncryptAmounts("15000000.0000", "939925.0000", "13539652.0000")
	if err != nil {
		t.Fatalf("EncryptAmounts: %v", err)
	}
	if encG == "15000000.0000" || encT == "939925.0000" || encN == "13539652.0000" {
		t.Fatal("sensitive payroll amounts must be encrypted")
	}
	g, err := platform.DecryptField(key, encG)
	if err != nil || g != "15000000.0000" {
		t.Fatalf("decrypt gross mismatch: %q %v", g, err)
	}
	n, err := platform.DecryptField(key, encN)
	if err != nil || n != "13539652.0000" {
		t.Fatalf("decrypt net mismatch: %q %v", n, err)
	}
}

