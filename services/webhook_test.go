package services

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dnvie/pago/models"
)

func TestDispatchWebhook(t *testing.T) {
	models.InitDB(":memory:")

	var receivedPayload WebhookPayload
	var webhookFired bool

	mockMerchant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookFired = true
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockMerchant.Close()

	inv := &models.Invoice{
		PublicID:       "wh_test_123",
		OrderID:        "Order-001",
		Status:         "confirmed",
		AmountReceived: 5000,
		FiatAmount:     10.0,
		FiatCurrency:   "USD",
		CallbackURL:    mockMerchant.URL,
	}
	models.CreateInvoice(inv)

	DispatchWebhook(inv)

	if !webhookFired {
		t.Fatalf("Expected webhook to be dispatched to merchant server, but it wasn't")
	}
	if receivedPayload.InvoicePublicID != "wh_test_123" {
		t.Errorf("Expected payload InvoicePublicID 'wh_test_123', got '%s'", receivedPayload.InvoicePublicID)
	}
	if receivedPayload.Status != "confirmed" {
		t.Errorf("Expected payload Status 'confirmed', got '%s'", receivedPayload.Status)
	}

	webhookFired = false
	DispatchWebhook(inv)

	if webhookFired {
		t.Errorf("Expected duplicate webhook to be blocked by the database lock, but it fired again!")
	}
}
