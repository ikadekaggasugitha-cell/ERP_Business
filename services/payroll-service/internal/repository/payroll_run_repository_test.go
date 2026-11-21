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

func TestSumPayrollResultTotals(t *testing.T) {
	gross, tax, net, err := sumPayrollResultTotals([]PayrollResultDTO{
		{GrossAmount: "10000000.0000", TaxAmount: "250000.2500", NetAmount: "9350000.7500"},
		{GrossAmount: "5000000.5000", TaxAmount: "100000.7500", NetAmount: "4699999.2500"},
	})
	if err != nil {
		t.Fatalf("sumPayrollResultTotals: %v", err)
	}
	if gross != "15000000.5000" || tax != "350001.0000" || net != "14050000.0000" {
		t.Fatalf("unexpected totals: gross=%s tax=%s net=%s", gross, tax, net)
	}
}

