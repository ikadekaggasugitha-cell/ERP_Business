// Package calc implements exact rational Indonesian statutory payroll math
// (PPh 21 UU HPP annualized Pasal 17 + BPJS Kesehatan + BPJS Ketenagakerjaan).
package calc

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type EmployeeInput struct {
	EmployeeID      uuid.UUID
	BasicSalary     *big.Rat
	Allowance       *big.Rat
	Bonus           *big.Rat
	OtherDeductions *big.Rat
	PTKP            string
	HasNPWP         bool
}

type EmployeeResult struct {
	EmployeeID      uuid.UUID
	Gross           pgtype.Numeric
	Deduction       pgtype.Numeric
	TaxPPh21        pgtype.Numeric
	Net             pgtype.Numeric
	Basic           pgtype.Numeric
	Allowance       pgtype.Numeric
	Bonus           pgtype.Numeric
	BPJSKesEmp      pgtype.Numeric
	BPJSKesEmployer pgtype.Numeric
	BPJSJHTEmp      pgtype.Numeric
	BPJSJHTEmployer pgtype.Numeric
	BPJSJPEmp       pgtype.Numeric
	BPJSJPEmployer  pgtype.Numeric
	BPJSJKKJKMEmp   pgtype.Numeric
}

var (
	bpjsKesCap = big.NewRat(12_000_000, 1)
	bpjsJPCap  = big.NewRat(10_042_300, 1)
	biayaCap   = big.NewRat(500_000, 1)
	twelve     = big.NewRat(12, 1)
	thousand   = big.NewInt(1000)
)

var ptkpAnnual = map[string]int64{
	"TK/0": 54_000_000,
	"TK/1": 58_500_000,
	"TK/2": 63_000_000,
	"TK/3": 67_500_000,
	"K/0":  58_500_000,
	"K/1":  63_000_000,
	"K/2":  67_500_000,
	"K/3":  72_000_000,
}

type bracket struct {
	limit *big.Rat
	rate  *big.Rat
}

var pph21Brackets = []bracket{
	{limit: big.NewRat(60_000_000, 1), rate: big.NewRat(5, 100)},
	{limit: big.NewRat(190_000_000, 1), rate: big.NewRat(15, 100)},  // 60M..250M
	{limit: big.NewRat(250_000_000, 1), rate: big.NewRat(25, 100)},  // 250M..500M
	{limit: big.NewRat(4_500_000_000, 1), rate: big.NewRat(30, 100)}, // 500M..5B
	{limit: nil, rate: big.NewRat(35, 100)},                          // >5B
}

func minRat(a, b *big.Rat) *big.Rat {
	if a.Cmp(b) < 0 {
		return new(big.Rat).Set(a)
	}
	return new(big.Rat).Set(b)
}

func roundHalfUp(r *big.Rat, scale int) *big.Rat {
	factor := big.NewInt(1)
	for i := 0; i < scale; i++ {
		factor.Mul(factor, big.NewInt(10))
	}
	scaled := new(big.Rat).Mul(r, new(big.Rat).SetInt(factor))
	half := big.NewRat(1, 2)
	if scaled.Sign() >= 0 {
		scaled.Add(scaled, half)
	} else {
		scaled.Sub(scaled, half)
	}
	q := new(big.Int).Quo(scaled.Num(), scaled.Denom())
	return new(big.Rat).SetFrac(q, factor)
}

// ComputeEmployee calculates gross, BPJS Kesehatan/TK, PPh 21, and net pay using exact rationals.
func ComputeEmployee(in EmployeeInput) (EmployeeResult, error) {
	basic := zeroIfNil(in.BasicSalary)
	allow := zeroIfNil(in.Allowance)
	bonus := zeroIfNil(in.Bonus)
	otherDed := zeroIfNil(in.OtherDeductions)

	gross := new(big.Rat).Add(basic, allow)
	gross.Add(gross, bonus)
	if gross.Sign() < 0 || otherDed.Sign() < 0 {
		return EmployeeResult{}, fmt.Errorf("gross and deduction amounts must be non-negative")
	}

	// BPJS Kesehatan (1% employee, 4% employer, capped at Rp 12M).
	kesBase := minRat(gross, bpjsKesCap)
	kesEmp := roundHalfUp(new(big.Rat).Mul(kesBase, big.NewRat(1, 100)), 2)
	kesEmpr := roundHalfUp(new(big.Rat).Mul(kesBase, big.NewRat(4, 100)), 2)

	// BPJS Ketenagakerjaan: JHT (2%/3.7%), JP (1%/2% capped), JKK+JKM (0.54% employer).
	jhtEmp := roundHalfUp(new(big.Rat).Mul(gross, big.NewRat(2, 100)), 2)
	jhtEmpr := roundHalfUp(new(big.Rat).Mul(gross, big.NewRat(37, 1000)), 2)
	jpBase := minRat(gross, bpjsJPCap)
	jpEmp := roundHalfUp(new(big.Rat).Mul(jpBase, big.NewRat(1, 100)), 2)
	jpEmpr := roundHalfUp(new(big.Rat).Mul(jpBase, big.NewRat(2, 100)), 2)
	jkkJkmEmpr := roundHalfUp(new(big.Rat).Mul(gross, big.NewRat(54, 10000)), 2)

	// PPh 21 annualized Pasal 17 UU HPP.
	biayaJabatan := minRat(new(big.Rat).Mul(gross, big.NewRat(5, 100)), biayaCap)
	monthlyNetTaxable := new(big.Rat).Sub(gross, biayaJabatan)
	monthlyNetTaxable.Sub(monthlyNetTaxable, jhtEmp)
	monthlyNetTaxable.Sub(monthlyNetTaxable, jpEmp)
	if monthlyNetTaxable.Sign() < 0 {
		monthlyNetTaxable = new(big.Rat)
	}
	annualNet := new(big.Rat).Mul(monthlyNetTaxable, twelve)

	ptkpKey := strings.ToUpper(strings.TrimSpace(in.PTKP))
	if ptkpKey == "" {
		ptkpKey = "TK/0"
	}
	ptkpVal, ok := ptkpAnnual[ptkpKey]
	if !ok {
		return EmployeeResult{}, fmt.Errorf("unsupported PTKP status %q", in.PTKP)
	}
	pkp := new(big.Rat).Sub(annualNet, big.NewRat(ptkpVal, 1))
	annualTax := new(big.Rat)
	if pkp.Sign() > 0 {
		// Statutory PKP is rounded down to full multiples of Rp 1,000.
		intPKP := new(big.Int).Quo(pkp.Num(), pkp.Denom())
		intPKP.Quo(intPKP, thousand)
		intPKP.Mul(intPKP, thousand)
		rem := new(big.Rat).SetInt(intPKP)
		for _, br := range pph21Brackets {
			if rem.Sign() <= 0 {
				break
			}
			slice := rem
			if br.limit != nil && rem.Cmp(br.limit) > 0 {
				slice = br.limit
			}
			annualTax.Add(annualTax, new(big.Rat).Mul(slice, br.rate))
			rem = new(big.Rat).Sub(rem, slice)
		}
		if !in.HasNPWP {
			annualTax.Mul(annualTax, big.NewRat(120, 100))
		}
	}
	pph21 := roundHalfUp(new(big.Rat).Quo(annualTax, twelve), 2)

	dedTotal := new(big.Rat).Add(kesEmp, jhtEmp)
	dedTotal.Add(dedTotal, jpEmp)
	dedTotal.Add(dedTotal, otherDed)

	net := new(big.Rat).Sub(gross, dedTotal)
	net.Sub(net, pph21)
	if net.Sign() < 0 {
		return EmployeeResult{}, fmt.Errorf("deductions and PPh 21 exceed gross salary for employee %s", in.EmployeeID)
	}

	res := EmployeeResult{EmployeeID: in.EmployeeID}
	var err error
	if res.Gross, err = platform.RatToNumeric(gross, 4); err != nil {
		return res, err
	}
	if res.Deduction, err = platform.RatToNumeric(dedTotal, 4); err != nil {
		return res, err
	}
	if res.TaxPPh21, err = platform.RatToNumeric(pph21, 4); err != nil {
		return res, err
	}
	if res.Net, err = platform.RatToNumeric(net, 4); err != nil {
		return res, err
	}
	if res.Basic, err = platform.RatToNumeric(basic, 4); err != nil {
		return res, err
	}
	if res.Allowance, err = platform.RatToNumeric(allow, 4); err != nil {
		return res, err
	}
	if res.Bonus, err = platform.RatToNumeric(bonus, 4); err != nil {
		return res, err
	}
	if res.BPJSKesEmp, err = platform.RatToNumeric(kesEmp, 4); err != nil {
		return res, err
	}
	if res.BPJSKesEmployer, err = platform.RatToNumeric(kesEmpr, 4); err != nil {
		return res, err
	}
	if res.BPJSJHTEmp, err = platform.RatToNumeric(jhtEmp, 4); err != nil {
		return res, err
	}
	if res.BPJSJHTEmployer, err = platform.RatToNumeric(jhtEmpr, 4); err != nil {
		return res, err
	}
	if res.BPJSJPEmp, err = platform.RatToNumeric(jpEmp, 4); err != nil {
		return res, err
	}
	if res.BPJSJPEmployer, err = platform.RatToNumeric(jpEmpr, 4); err != nil {
		return res, err
	}
	if res.BPJSJKKJKMEmp, err = platform.RatToNumeric(jkkJkmEmpr, 4); err != nil {
		return res, err
	}
	return res, nil
}

func zeroIfNil(r *big.Rat) *big.Rat {
	if r == nil {
		return new(big.Rat)
	}
	return new(big.Rat).Set(r)
}

