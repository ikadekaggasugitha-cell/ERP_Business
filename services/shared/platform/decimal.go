package platform

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ParseNumeric strictly parses a fixed-point decimal string into pgtype.Numeric.
// Never silently zeroes invalid input.
func ParseNumeric(raw string) (pgtype.Numeric, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return pgtype.Numeric{}, fmt.Errorf("decimal value is required")
	}
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil || !n.Valid || n.NaN || n.InfinityModifier != pgtype.Finite {
		return pgtype.Numeric{}, fmt.Errorf("invalid finite decimal: %q", raw)
	}
	return n, nil
}

// ParseOptionalNumeric treats empty string as "0", but rejects malformed values.
func ParseOptionalNumeric(raw string) (pgtype.Numeric, error) {
	if strings.TrimSpace(raw) == "" {
		return ParseNumeric("0")
	}
	return ParseNumeric(raw)
}

// FormatNumeric converts pgtype.Numeric into a fixed-scale canonical decimal string (e.g. "1000.0000").
func FormatNumeric(n pgtype.Numeric, scale int) string {
	if !n.Valid || n.NaN {
		return fmt.Sprintf("0.%s", strings.Repeat("0", scale))
	}
	r := NumericToRat(n)
	if r == nil {
		return fmt.Sprintf("0.%s", strings.Repeat("0", scale))
	}
	return r.FloatString(scale)
}

// NumericToRat returns exact *big.Rat representation of pgtype.Numeric.
func NumericToRat(n pgtype.Numeric) *big.Rat {
	if !n.Valid || n.NaN || n.InfinityModifier != pgtype.Finite || n.Int == nil {
		return big.NewRat(0, 1)
	}
	num := new(big.Int).Set(n.Int)
	ten := big.NewInt(10)
	if n.Exp >= 0 {
		pow := new(big.Int).Exp(ten, big.NewInt(int64(n.Exp)), nil)
		num.Mul(num, pow)
		return new(big.Rat).SetInt(num)
	}
	den := new(big.Int).Exp(ten, big.NewInt(int64(-n.Exp)), nil)
	return new(big.Rat).SetFrac(num, den)
}

// RatToNumeric converts a *big.Rat rounded half-up to the specified decimal scale (e.g. 4 for NUMERIC(18,4)).
func RatToNumeric(r *big.Rat, scale int) (pgtype.Numeric, error) {
	if r == nil {
		return pgtype.Numeric{}, fmt.Errorf("nil rational")
	}
	return ParseNumeric(r.FloatString(scale))
}

// SumRatStrings sums exact decimal strings without float64 precision loss.
func SumRatStrings(vals ...string) (*big.Rat, error) {
	total := new(big.Rat)
	for _, v := range vals {
		n, err := ParseOptionalNumeric(v)
		if err != nil {
			return nil, err
		}
		total.Add(total, NumericToRat(n))
	}
	return total, nil
}

// ParseDate strictly parses a YYYY-MM-DD calendar date into pgtype.Date.
func ParseDate(raw string) (pgtype.Date, error) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("date must be YYYY-MM-DD: %w", err)
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

// FormatDate formats a pgtype.Date as YYYY-MM-DD (or empty string if invalid).
func FormatDate(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
}
