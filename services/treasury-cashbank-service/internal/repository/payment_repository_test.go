package repository

import (
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/treasury-cashbank-service/internal/treasurydb"
)

func strPtr(s string) *string { return &s }

func TestMatchStatementLinesPreventsCrossDirectionAndAmbiguousMatches(t *testing.T) {
	amt500, _ := platform.ParseNumeric("500.0000")
	amtNeg500, _ := platform.ParseNumeric("-500.0000")
	amt750, _ := platform.ParseNumeric("750.0000")
	amtNeg750, _ := platform.ParseNumeric("-750.0000")

	// 1. Opposite direction with signed batch must NOT match (negative line vs Incoming payment).
	pairs := MatchStatementLines(
		[]treasurydb.ListUnreconciledStatementLinesRow{
			{ID: uuid.New(), Amount: amtNeg500},
		},
		[]treasurydb.ListSettledPaymentsForBankRow{
			{ID: uuid.New(), PaymentNo: "PAY-IN-1", PaymentType: "Incoming", Amount: amt500},
		},
	)
	if len(pairs) != 0 {
		t.Fatalf("expected 0 matches for opposite cashflow direction, got %d", len(pairs))
	}

	// 2. Ambiguous duplicate amounts without reference must stay unmatched for manual reconciliation.
	pairs = MatchStatementLines(
		[]treasurydb.ListUnreconciledStatementLinesRow{
			{ID: uuid.New(), Amount: amtNeg500},
		},
		[]treasurydb.ListSettledPaymentsForBankRow{
			{ID: uuid.New(), PaymentNo: "PAY-OUT-1", PaymentType: "Outgoing", Amount: amt500},
			{ID: uuid.New(), PaymentNo: "PAY-OUT-2", PaymentType: "Outgoing", Amount: amt500},
		},
	)
	if len(pairs) != 0 {
		t.Fatalf("expected 0 matches for ambiguous duplicate-amount payments, got %d", len(pairs))
	}

	// 3. Unique signed match + exact ExternalRef match both succeed cleanly.
	p1ID := uuid.New()
	p2ID := uuid.New()
	pairs = MatchStatementLines(
		[]treasurydb.ListUnreconciledStatementLinesRow{
			{ID: uuid.New(), Amount: amtNeg500, ExternalRef: strPtr("PAY-OUT-1")},
			{ID: uuid.New(), Amount: amt750},
		},
		[]treasurydb.ListSettledPaymentsForBankRow{
			{ID: p1ID, PaymentNo: "PAY-OUT-1", PaymentType: "Outgoing", Amount: amt500},
			{ID: uuid.New(), PaymentNo: "PAY-OUT-2", PaymentType: "Outgoing", Amount: amt500},
			{ID: p2ID, PaymentNo: "PAY-IN-1", PaymentType: "Incoming", Amount: amt750},
		},
	)
	if len(pairs) != 2 {
		t.Fatalf("expected 2 deterministic matches, got %d", len(pairs))
	}
	if pairs[0].payment.ID != p1ID || pairs[1].payment.ID != p2ID {
		t.Fatalf("unexpected matched payments: %+v", pairs)
	}

	// 4. Positive-only bank export still works when unique 1-to-1, and refuses ambiguity.
	p3ID := uuid.New()
	pairs = MatchStatementLines(
		[]treasurydb.ListUnreconciledStatementLinesRow{
			{ID: uuid.New(), Amount: amt750},
		},
		[]treasurydb.ListSettledPaymentsForBankRow{
			{ID: p3ID, PaymentNo: "PAY-OUT-3", PaymentType: "Outgoing", Amount: amt750},
		},
	)
	if len(pairs) != 1 || pairs[0].payment.ID != p3ID {
		t.Fatalf("expected 1 unique positive-only fallback match, got %+v", pairs)
	}

	// 5. Negative line matched with Outgoing payment succeeds uniquely.
	p4ID := uuid.New()
	pairs = MatchStatementLines(
		[]treasurydb.ListUnreconciledStatementLinesRow{
			{ID: uuid.New(), Amount: amtNeg750},
		},
		[]treasurydb.ListSettledPaymentsForBankRow{
			{ID: p4ID, PaymentNo: "PAY-OUT-4", PaymentType: "Outgoing", Amount: amt750},
		},
	)
	if len(pairs) != 1 || pairs[0].payment.ID != p4ID {
		t.Fatalf("expected 1 unique signed Outgoing match, got %+v", pairs)
	}
}
