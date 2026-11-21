package repository

import (
	"context"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/currency-service/internal/currencydb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// DefaultAnchorCurrency is USD per ADR-014.
const DefaultAnchorCurrency = "USD"

// RateSource indicates how an exchange rate was derived.
type RateSource string

const (
	RateSourceIdentity     RateSource = "IDENTITY"
	RateSourceDirect       RateSource = "DIRECT"
	RateSourceInverse      RateSource = "INVERSE"
	RateSourceTriangulated RateSource = "TRIANGULATED"
)

// RateLookupResult holds the calculated rate and its lineage.
type RateLookupResult struct {
	FromCurrency  string
	ToCurrency    string
	RateTypeID    uuid.UUID
	Rate          *big.Rat
	RateString    string
	Source        RateSource
	Anchor        string
	EffectiveDate string
}

// ConvertResult represents the financial conversion outcome.
type ConvertResult struct {
	FromCurrency    string     `json:"fromCurrency"`
	ToCurrency      string     `json:"toCurrency"`
	RateTypeID      uuid.UUID  `json:"rateTypeId"`
	OriginalAmount  string     `json:"originalAmount"`
	ConvertedAmount string     `json:"convertedAmount"`
	Rate            string     `json:"rate"`
	RateSource      RateSource `json:"rateSource"`
	AnchorCurrency  string     `json:"anchorCurrency,omitempty"`
	EffectiveDate   string     `json:"effectiveDate"`
}

// ResolveExchangeRate calculates the exchange rate from fromCurr to toCurr on onDate.
// Precedence:
// 1. Identity (from == to) -> Rate = 1
// 2. Direct rate (from -> to)
// 3. Inverse rate (1 / (to -> from))
// 4. Triangulation via anchor currency (default: USD): (from -> anchor) * (anchor -> to)
func (r *CurrencyRepository) ResolveExchangeRate(
	ctx context.Context,
	scope []uuid.UUID,
	fromCurr, toCurr string,
	rateTypeID uuid.UUID,
	onDate pgtype.Date,
	anchorOverride string,
) (*RateLookupResult, error) {
	if fromCurr == "" || toCurr == "" {
		return nil, httpx.BadRequest("fromCurrency and toCurrency are required")
	}

	dateStr := platform.FormatDate(onDate)
	if fromCurr == toCurr {
		one := big.NewRat(1, 1)
		return &RateLookupResult{
			FromCurrency:  fromCurr,
			ToCurrency:    toCurr,
			RateTypeID:    rateTypeID,
			Rate:          one,
			RateString:    one.FloatString(8),
			Source:        RateSourceIdentity,
			EffectiveDate: dateStr,
		}, nil
	}

	var result *RateLookupResult
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)

		// 1. Direct rate
		directRate, err := q.GetLatestRate(ctx, currencydb.GetLatestRateParams{
			FromCurrency: fromCurr,
			ToCurrency:   toCurr,
			RateTypeID:   rateTypeID,
			ValidFrom:    onDate,
		})
		if err == nil {
			rat := platform.NumericToRat(directRate.Rate)
			if rat == nil || rat.Sign() <= 0 {
				return httpx.Unprocessable("INVALID_RATE", "exchange rate must be positive")
			}
			result = &RateLookupResult{
				FromCurrency:  fromCurr,
				ToCurrency:    toCurr,
				RateTypeID:    rateTypeID,
				Rate:          rat,
				RateString:    rat.FloatString(8),
				Source:        RateSourceDirect,
				EffectiveDate: platform.FormatDate(directRate.ValidFrom),
			}
			return nil
		}

		// 2. Inverse rate
		invRate, err := q.GetLatestRate(ctx, currencydb.GetLatestRateParams{
			FromCurrency: toCurr,
			ToCurrency:   fromCurr,
			RateTypeID:   rateTypeID,
			ValidFrom:    onDate,
		})
		if err == nil {
			invRat := platform.NumericToRat(invRate.Rate)
			if invRat == nil || invRat.Sign() <= 0 {
				return httpx.Unprocessable("INVALID_RATE", "inverse exchange rate must be positive")
			}
			rat := new(big.Rat).Inv(invRat)
			result = &RateLookupResult{
				FromCurrency:  fromCurr,
				ToCurrency:    toCurr,
				RateTypeID:    rateTypeID,
				Rate:          rat,
				RateString:    rat.FloatString(8),
				Source:        RateSourceInverse,
				EffectiveDate: platform.FormatDate(invRate.ValidFrom),
			}
			return nil
		}

		// 3. Triangulation via anchor currency (default: USD)
		anchor := DefaultAnchorCurrency
		if anchorOverride != "" {
			anchor = anchorOverride
		}

		if fromCurr == anchor || toCurr == anchor {
			// Direct and inverse already failed, cannot triangulate with anchor if one endpoint is the anchor
			return httpx.NotFound(fmt.Sprintf("no active exchange rate found between %s and %s on %s", fromCurr, toCurr, dateStr))
		}

		// Resolve fromCurr -> anchor
		ratFromAnchor, errFrom := resolveLeg(ctx, q, fromCurr, anchor, rateTypeID, onDate)
		if errFrom != nil {
			return httpx.NotFound(fmt.Sprintf("triangulation failed: no exchange rate between %s and anchor %s: %v", fromCurr, anchor, errFrom))
		}

		// Resolve anchor -> toCurr
		ratAnchorTo, errTo := resolveLeg(ctx, q, anchor, toCurr, rateTypeID, onDate)
		if errTo != nil {
			return httpx.NotFound(fmt.Sprintf("triangulation failed: no exchange rate between anchor %s and %s: %v", anchor, toCurr, errTo))
		}

		triangulated := new(big.Rat).Mul(ratFromAnchor, ratAnchorTo)
		result = &RateLookupResult{
			FromCurrency:  fromCurr,
			ToCurrency:    toCurr,
			RateTypeID:    rateTypeID,
			Rate:          triangulated,
			RateString:    triangulated.FloatString(8),
			Source:        RateSourceTriangulated,
			Anchor:        anchor,
			EffectiveDate: dateStr,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// resolveLeg resolves a single conversion leg (A -> B) via direct or inverse rate.
func resolveLeg(
	ctx context.Context,
	q *currencydb.Queries,
	from, to string,
	rateTypeID uuid.UUID,
	onDate pgtype.Date,
) (*big.Rat, error) {
	// Try direct
	direct, err := q.GetLatestRate(ctx, currencydb.GetLatestRateParams{
		FromCurrency: from,
		ToCurrency:   to,
		RateTypeID:   rateTypeID,
		ValidFrom:    onDate,
	})
	if err == nil {
		r := platform.NumericToRat(direct.Rate)
		if r != nil && r.Sign() > 0 {
			return r, nil
		}
	}

	// Try inverse
	inv, err := q.GetLatestRate(ctx, currencydb.GetLatestRateParams{
		FromCurrency: to,
		ToCurrency:   from,
		RateTypeID:   rateTypeID,
		ValidFrom:    onDate,
	})
	if err == nil {
		r := platform.NumericToRat(inv.Rate)
		if r != nil && r.Sign() > 0 {
			return new(big.Rat).Inv(r), nil
		}
	}

	return nil, fmt.Errorf("no exchange rate leg for %s to %s", from, to)
}

// ConvertAmount performs exact rational multiplication of amount by the resolved exchange rate.
func (r *CurrencyRepository) ConvertAmount(
	ctx context.Context,
	scope []uuid.UUID,
	fromCurr, toCurr string,
	rateTypeID uuid.UUID,
	onDate pgtype.Date,
	amountStr string,
	anchorOverride string,
) (*ConvertResult, error) {
	numAmount, err := platform.ParseNumeric(amountStr)
	if err != nil {
		return nil, httpx.BadRequest(fmt.Sprintf("invalid amount %q: %v", amountStr, err))
	}
	ratAmount := platform.NumericToRat(numAmount)
	if ratAmount == nil {
		return nil, httpx.BadRequest("amount rational conversion failed")
	}

	rateLookup, err := r.ResolveExchangeRate(ctx, scope, fromCurr, toCurr, rateTypeID, onDate, anchorOverride)
	if err != nil {
		return nil, err
	}

	convertedRat := new(big.Rat).Mul(ratAmount, rateLookup.Rate)

	// Fetch source and target currency precision if available (default 4)
	sourcePrecision := 4
	targetPrecision := 4
	_ = r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if fromC, e := q.GetCurrencyByCode(ctx, fromCurr); e == nil && fromC.Precision >= 0 {
			sourcePrecision = int(fromC.Precision)
		}
		if toC, e := q.GetCurrencyByCode(ctx, toCurr); e == nil && toC.Precision >= 0 {
			targetPrecision = int(toC.Precision)
		}
		return nil
	})

	return &ConvertResult{
		FromCurrency:    fromCurr,
		ToCurrency:      toCurr,
		RateTypeID:      rateTypeID,
		OriginalAmount:  ratAmount.FloatString(sourcePrecision),
		ConvertedAmount: convertedRat.FloatString(targetPrecision),
		Rate:            rateLookup.RateString,
		RateSource:      rateLookup.Source,
		AnchorCurrency:  rateLookup.Anchor,
		EffectiveDate:   rateLookup.EffectiveDate,
	}, nil
}

