package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dnvie/pago/models"
	"github.com/dnvie/pago/services"
)

func TestHandleTxNotify(t *testing.T) {
	models.InitDB(":memory:")

	inv := &models.Invoice{
		PublicID:        "webhook_inv",
		XMRAmount:       1000,
		AmountReceived:  0,
		Status:          "pending",
		SubaddressIndex: 5,
		RequiredConfs:   1,
	}
	err := models.CreateInvoice(inv)
	if err != nil {
		t.Fatalf("Failed to create mock invoice: %v", err)
	}

	mockRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockResponse := `{
			"result": {
				"transfer": {
					"amount": 1000,
					"type": "pool",
					"confirmations": 0,
					"subaddr_index": {"minor": 5}
				}
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponse))
	}))
	defer mockRPC.Close()

	services.InitWalletClient(mockRPC.URL, "user", "pass")

	payload := `{"txid":"test_tx_abc123"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/monero-webhook", bytes.NewBuffer([]byte(payload)))
	rr := httptest.NewRecorder()

	HandleTxNotify(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200 OK, got %v", rr.Code)
	}

	updatedInv, _ := models.GetInvoiceByPublicID("webhook_inv")
	if updatedInv.Status != "in_mempool" {
		t.Errorf("Expected status to change to 'in_mempool', got '%s'", updatedInv.Status)
	}
	if updatedInv.AmountReceived != 1000 {
		t.Errorf("Expected AmountReceived to be 1000, got %d", updatedInv.AmountReceived)
	}
	if updatedInv.TXIDs != "test_tx_abc123" {
		t.Errorf("Expected TXID to be saved in DB, got '%s'", updatedInv.TXIDs)
	}
}

func TestHandleTxNotify_InvalidIndex(t *testing.T) {
	models.InitDB(":memory:")

	mockRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockResponse := `{
			"result": {
				"transfer": {
					"amount": 1000,
					"type": "pool",
					"confirmations": 0,
					"subaddr_index": {"minor": 999}
				}
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponse))
	}))
	defer mockRPC.Close()

	services.InitWalletClient(mockRPC.URL, "user", "pass")

	payload := `{"txid":"test_tx_invalid"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/monero-webhook", bytes.NewBuffer([]byte(payload)))
	rr := httptest.NewRecorder()

	HandleTxNotify(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200 OK, got %v", rr.Code)
	}
}
