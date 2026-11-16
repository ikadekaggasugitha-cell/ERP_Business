package calc

import (
	"math/big"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func TestComputeEmployeeBPJSAndPPh21(t *testing.T) {
	// Rp 15,000,000/month salary, TK/0, with NPWP:
	// - BPJS Kes capped at 12M: emp = 120,000.0000, employer = 480,000.0000
	// - BPJS JHT uncapped 15M: emp = 300,000.0000, employer = 555,000.0000
	// - BPJS JP capped at 10,042,300: emp = 100,423.0000, employer = 200,846.0000
	// - Biaya Jabatan capped at 500,000
	// - Monthly taxable = 15M - 500k - 300k - 100,423 = 14,099,577
	// - Annual = 169,194,924; PKP = 115,194,000 (floor to 1000)
	// - Annual tax = 60M*5% (3M) + 55,194,000*15% (8,279,100) = 11,279,100
	// - Monthly tax = 11,279,100 / 12 = 939,925.0000
	res, err := ComputeEmployee(EmployeeInput{
		EmployeeID:  uuid.New(),
		BasicSalary: big.NewRat(15_000_000, 1),
		PTKP:        "TK/0",
		HasNPWP:     true,
	})
	if err != nil {
		t.Fatalf("ComputeEmployee: %v", err)
	}
	if got := platform.FormatNumeric(res.Gross, 4); got != "15000000.0000" {
		t.Fatalf("expected gross 15000000.0000, got %s", got)
	}
	if got := platform.FormatNumeric(res.BPJSKesEmp, 4); got != "120000.0000" {
		t.Fatalf("expected BPJS Kes emp 120000.0000, got %s", got)
	}
	if got := platform.FormatNumeric(res.BPJSJHTEmp, 4); got != "300000.0000" {
		t.Fatalf("expected BPJS JHT emp 300000.0000, got %s", got)
	}
	if got := platform.FormatNumeric(res.BPJSJPEmp, 4); got != "100423.0000" {
		t.Fatalf("expected BPJS JP emp 100423.0000, got %s", got)
	}
	if got := platform.FormatNumeric(res.TaxPPh21, 4); got != "939925.0000" {
		t.Fatalf("expected PPh21 939925.0000, got %s", got)
	}
	if got := platform.FormatNumeric(res.Net, 4); got != "13539652.0000" {
		t.Fatalf("expected net 13539652.0000, got %s", got)
	}
}

