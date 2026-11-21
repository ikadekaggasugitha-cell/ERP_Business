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
	// - BPJS JKK+JKM employer 0.54%: 81,000.0000
	// - Monthly taxable gross = 15,000,000 + 480,000 + 81,000 = 15,561,000
	// - Annual taxable gross = 186,732,000; Biaya Jabatan capped at 6,000,000
	// - Annual JHT+JP emp = 4,805,076; Annual net = 175,926,924; PKP = 121,926,000
	// - Annual tax = 60M*5% (3M) + 61,926,000*15% (9,288,900) = 12,288,900
	// - Monthly tax = 12,288,900 / 12 = 1,024,075.0000
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
	if got := platform.FormatNumeric(res.TaxPPh21, 4); got != "1024075.0000" {
		t.Fatalf("expected PPh21 1024075.0000, got %s", got)
	}
	if got := platform.FormatNumeric(res.Net, 4); got != "13455502.0000" {
		t.Fatalf("expected net 13455502.0000, got %s", got)
	}

	// Irregular one-time bonus (Rp 10,000,000) must NOT increase monthly BPJS and must NOT be multiplied by 12.
	resBonus, err := ComputeEmployee(EmployeeInput{
		EmployeeID:  uuid.New(),
		BasicSalary: big.NewRat(15_000_000, 1),
		Bonus:       big.NewRat(10_000_000, 1),
		PTKP:        "TK/0",
		HasNPWP:     true,
	})
	if err != nil {
		t.Fatalf("ComputeEmployee with bonus: %v", err)
	}
	if got := platform.FormatNumeric(resBonus.BPJSJHTEmp, 4); got != "300000.0000" {
		t.Fatalf("expected BPJS JHT unchanged by bonus (300000.0000), got %s", got)
	}
	// Marginal tax on 10M bonus in 15% bracket = 1,500,000 -> total monthly PPh21 = 1,024,075 + 1,500,000 = 2,524,075.0000
	if got := platform.FormatNumeric(resBonus.TaxPPh21, 4); got != "2524075.0000" {
		t.Fatalf("expected PPh21 with 10M bonus = 2524075.0000, got %s", got)
	}
}

