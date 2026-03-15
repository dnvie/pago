package jobs

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dnvie/pago/models"
	"github.com/dnvie/pago/services"
)

func setupJobsTest() *httptest.Server {
	models.InitDB(":memory:")
	models.DB.SetMaxOpenConns(1)

	mockRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"result": {
				"transfer": {
					"amount": 5000,
					"type": "in",
					"confirmations": 2,
					"subaddr_index": {"minor": 1}
				}
			}
		}`))
	}))

	services.InitWalletClient(mockRPC.URL, "user", "pass")
	return mockRPC
}

func TestPollConfirmations_MockPayment(t *testing.T) {
	mockRPC := setupJobsTest()
	defer mockRPC.Close()

	inv := &models.Invoice{
		PublicID:      "job_mock_1",
		Status:        "confirming",
		TXIDs:         "mock_tx_123",
		RequiredConfs: 5,
		CurrentConfs:  1,
	}
	models.CreateInvoice(inv)

	pollConfirmations()

	updated, _ := models.GetInvoiceByPublicID("job_mock_1")
	if updated.CurrentConfs != 2 {
		t.Errorf("Expected CurrentConfs to increment to 2, got %d", updated.CurrentConfs)
	}
	if updated.Status != "confirming" {
		t.Errorf("Expected status to remain 'confirming', got '%s'", updated.Status)
	}
}

func TestPollConfirmations_RealPaymentCleared(t *testing.T) {
	mockRPC := setupJobsTest()
	defer mockRPC.Close()

	inv := &models.Invoice{
		PublicID:       "job_real_1",
		Status:         "in_mempool",
		TXIDs:          "real_tx_abc",
		XMRAmount:      5000,
		AmountReceived: 5000,
		RequiredConfs:  2,
		CurrentConfs:   0,
	}
	models.CreateInvoice(inv)

	pollConfirmations()

	updated, _ := models.GetInvoiceByPublicID("job_real_1")
	if updated.CurrentConfs != 2 {
		t.Errorf("Expected CurrentConfs to jump to 2, got %d", updated.CurrentConfs)
	}
	if updated.Status != "confirmed" {
		t.Errorf("Expected status to change to 'confirmed', got '%s'", updated.Status)
	}
}

func TestPollConfirmations_Underpayment(t *testing.T) {
	mockRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result": {"transfer": {"amount": 2500, "type": "in", "confirmations": 1, "subaddr_index": {"minor": 2}}}}`))
	}))
	defer mockRPC.Close()
	services.InitWalletClient(mockRPC.URL, "user", "pass")
	models.InitDB(":memory:")

	inv := &models.Invoice{
		PublicID:       "job_under_1",
		Status:         "in_mempool",
		TXIDs:          "underpay_tx",
		XMRAmount:      5000,
		AmountReceived: 2500,
		RequiredConfs:  2,
		CurrentConfs:   0,
	}
	models.CreateInvoice(inv)

	pollConfirmations()

	updated, _ := models.GetInvoiceByPublicID("job_under_1")
	if updated.Status != "underpaid" {
		t.Errorf("Expected status to be 'underpaid', got '%s'", updated.Status)
	}
}

func TestPollConfirmations_Overpayment(t *testing.T) {
	mockRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result": {"transfer": {"amount": 10000, "type": "in", "confirmations": 2, "subaddr_index": {"minor": 3}}}}`))
	}))
	defer mockRPC.Close()
	services.InitWalletClient(mockRPC.URL, "user", "pass")
	models.InitDB(":memory:")

	inv := &models.Invoice{
		PublicID:       "job_over_1",
		Status:         "in_mempool",
		TXIDs:          "overpay_tx",
		XMRAmount:      5000,
		AmountReceived: 10000,
		RequiredConfs:  2,
		CurrentConfs:   0,
	}
	models.CreateInvoice(inv)

	pollConfirmations()

	updated, _ := models.GetInvoiceByPublicID("job_over_1")
	if updated.Status != "confirmed" {
		t.Errorf("Expected overpayment to successfully confirm, got '%s'", updated.Status)
	}
}

func TestPollConfirmations_MempoolDrop(t *testing.T) {
	mockRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result": {"transfer": {"amount": 5000, "type": "pool", "confirmations": 0, "subaddr_index": {"minor": 4}}}}`))
	}))
	defer mockRPC.Close()
	services.InitWalletClient(mockRPC.URL, "user", "pass")
	models.InitDB(":memory:")

	inv := &models.Invoice{
		PublicID:       "job_reorg_1",
		Status:         "confirming",
		TXIDs:          "reorg_tx",
		XMRAmount:      5000,
		AmountReceived: 5000,
		RequiredConfs:  2,
		CurrentConfs:   1,
	}
	models.CreateInvoice(inv)

	pollConfirmations()

	updated, _ := models.GetInvoiceByPublicID("job_reorg_1")
	if updated.Status != "in_mempool" {
		t.Errorf("Expected invoice to drop back to 'in_mempool' after reorg, got '%s'", updated.Status)
	}
	if updated.CurrentConfs != 0 {
		t.Errorf("Expected confirmations to drop to 0, got %d", updated.CurrentConfs)
	}
}

func TestPollConfirmations_TransactionFailed(t *testing.T) {
	mockRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result": {"transfer": {"amount": 5000, "type": "failed", "confirmations": 0, "subaddr_index": {"minor": 5}}}}`))
	}))
	defer mockRPC.Close()
	services.InitWalletClient(mockRPC.URL, "user", "pass")
	models.InitDB(":memory:")

	inv := &models.Invoice{
		PublicID:       "job_fail_1",
		Status:         "in_mempool",
		TXIDs:          "fail_tx",
		XMRAmount:      5000,
		AmountReceived: 5000,
		RequiredConfs:  2,
		CurrentConfs:   0,
	}
	models.CreateInvoice(inv)

	pollConfirmations()

	updated, _ := models.GetInvoiceByPublicID("job_fail_1")
	if updated.Status != "failed" {
		t.Errorf("Expected status to be 'failed', got '%s'", updated.Status)
	}
}
