package jobs

import (
	"testing"
	"time"

	"github.com/dnvie/pago/models"
)

func TestStartExpireOldInvoices(t *testing.T) {
	models.InitDB(":memory:")
	models.DB.SetMaxOpenConns(1)

	inv := &models.Invoice{
		PublicID:  "expire_job_test",
		Status:    "pending",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	models.CreateInvoice(inv)

	StartExpireOldInvoices(10 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	updated, err := models.GetInvoiceByPublicID("expire_job_test")
	if err != nil {
		t.Fatalf("Failed to fetch invoice: %v", err)
	}

	if updated.Status != "expired" {
		t.Errorf("Expected background worker to change status to 'expired', got '%s'", updated.Status)
	}
}
