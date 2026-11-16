package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func TestEmployeePIIEncryptionRoundTripAndMasking(t *testing.T) {
	key, err := platform.LoadPIIKey()
	if err != nil {
		t.Fatalf("LoadPIIKey: %v", err)
	}
	repo := NewEmployeeRepository(nil, key)

	enc, err := repo.encryptPII("Budi Santoso", "3174011234560009", "1992-08-17", "budi@sinergi.co.id")
	if err != nil {
		t.Fatalf("encryptPII: %v", err)
	}
	if enc.FullName == "Budi Santoso" {
		t.Fatal("full_name must be encrypted at rest, got plaintext")
	}
	if enc.NationalID == nil || *enc.NationalID == "3174011234560009" {
		t.Fatal("national_id column must contain blind HMAC hash, not plaintext")
	}
	if enc.NationalIDHash == nil || *enc.NationalIDHash != *enc.NationalID || len(*enc.NationalIDHash) != 64 {
		t.Fatalf("unexpected blind hash: %+v", enc.NationalIDHash)
	}
	if enc.EncryptedDob == nil || enc.EncryptedEmail == nil {
		t.Fatal("expected encrypted DOB and email ciphertext")
	}

	dto, err := repo.toDTO(
		uuid.New(), uuid.New(), "EMP-001", enc.FullName,
		enc.EncryptedFullName, enc.EncryptedNationalID, enc.EncryptedDob, enc.EncryptedEmail,
		pgtype.Date{}, nil,
		pgtype.Date{Time: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC), Valid: true},
		"Active", nil, 1,
	)
	if err != nil {
		t.Fatalf("toDTO: %v", err)
	}
	if dto.FullName != "Budi Santoso" {
		t.Fatalf("expected decrypted FullName 'Budi Santoso', got %q", dto.FullName)
	}
	if dto.NationalIDMasked != "************0009" {
		t.Fatalf("expected masked NIK '************0009', got %q", dto.NationalIDMasked)
	}
	if dto.DateOfBirth != "1992-08-17" || dto.Email != "budi@sinergi.co.id" {
		t.Fatalf("unexpected decrypted DOB/email: %+v", dto)
	}
}

