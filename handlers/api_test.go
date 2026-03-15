package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dnvie/pago/models"
	"github.com/dnvie/pago/services"
)

func setupHandlerTest() {
	models.InitDB(":memory:")

	services.InjectMockPrice("usd", 150.0)

	mockRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result": {"address": "mock_address", "address_index": 1}}`))
	}))
	services.InitWalletClient(mockRPC.URL, "user", "pass")
}

func TestHandleGetInvoiceStatus(t *testing.T) {
	setupHandlerTest()

	inv := &models.Invoice{
		PublicID:      "poll_123",
		FiatAmount:    5.00,
		XMRAmount:     1000,
		Status:        "pending",
		RequiredConfs: 1,
	}
	models.CreateInvoice(inv)

	req, _ := http.NewRequest(http.MethodGet, "/api/invoice/status?public_id=poll_123", nil)
	rr := httptest.NewRecorder()
	HandleGetInvoiceStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %v", rr.Code)
	}

	var resp InvoiceStatusResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp.Status != "pending" {
		t.Errorf("Expected pending, got %s", resp.Status)
	}
	if resp.PaymentCleared {
		t.Errorf("Expected PaymentCleared to be false")
	}
}

func TestHandleAddTip(t *testing.T) {
	setupHandlerTest()

	inv := &models.Invoice{
		PublicID:     "tip_123",
		FiatAmount:   10.00,
		FiatCurrency: "USD",
		XMRAmount:    2000,
		Status:       "pending",
	}
	models.CreateInvoice(inv)

	payload := AddTipRequest{InvoicePublicID: "tip_123", TipPercentage: 20}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/invoice/tip", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	HandleAddTip(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %v", rr.Code)
	}

	updated, _ := models.GetInvoiceByPublicID("tip_123")
	if updated.TipPercentage != 20 {
		t.Errorf("Expected tip percentage 20, got %d", updated.TipPercentage)
	}
	if updated.TipFiat != 2.00 {
		t.Errorf("Expected tip fiat $2.00, got %f", updated.TipFiat)
	}
	if updated.FiatAmount != 12.00 {
		t.Errorf("Expected total fiat to be $12.00, got %f", updated.FiatAmount)
	}
}

func TestHandleCreateInvoice_InvalidInputs(t *testing.T) {
	setupHandlerTest()

	tests := []struct {
		name           string
		payload        string
		expectedStatus int
	}{
		{
			name:           "Negative Fiat Amount",
			payload:        `{"fiat_amount": -10.50, "fiat_currency": "USD"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Zero Fiat Amount",
			payload:        `{"fiat_amount": 0.00, "fiat_currency": "USD"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			payload:        `{"fiat_amount": 10.50, "fiat_currency": "USD"`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "/api/invoice", bytes.NewBuffer([]byte(tt.payload)))
			rr := httptest.NewRecorder()
			HandleCreateInvoice(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestHandleGetInvoiceStatus_InvalidInputs(t *testing.T) {
	setupHandlerTest()

	req, _ := http.NewRequest(http.MethodGet, "/api/invoice/status", nil)
	rr := httptest.NewRecorder()
	HandleGetInvoiceStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for missing public_id, got %v", rr.Code)
	}

	req, _ = http.NewRequest(http.MethodGet, "/api/invoice/status?public_id=does_not_exist", nil)
	rr = httptest.NewRecorder()
	HandleGetInvoiceStatus(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected 404 Not Found for non-existent public_id, got %v", rr.Code)
	}
}

func TestHandleAddTip_InvalidInputs(t *testing.T) {
	setupHandlerTest()

	inv := &models.Invoice{
		PublicID:     "tip_invalid_123",
		FiatAmount:   10.00,
		FiatCurrency: "USD",
		XMRAmount:    2000,
		Status:       "confirmed",
	}
	models.CreateInvoice(inv)

	tests := []struct {
		name           string
		payload        string
		expectedStatus int
	}{
		{
			name:           "Negative Tip Percentage",
			payload:        `{"invoice_public_id": "tip_invalid_123", "tip_percentage": -10}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Zero Tip Percentage",
			payload:        `{"invoice_public_id": "tip_invalid_123", "tip_percentage": 0}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing Invoice ID",
			payload:        `{"tip_percentage": 10}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Not Pending Status",
			payload:        `{"invoice_public_id": "tip_invalid_123", "tip_percentage": 10}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Non-existent Invoice ID",
			payload:        `{"invoice_public_id": "does_not_exist", "tip_percentage": 10}`,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "/api/invoice/tip", bytes.NewBuffer([]byte(tt.payload)))
			rr := httptest.NewRecorder()
			HandleAddTip(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestHandleCreateInvoice_HappyPath(t *testing.T) {
	setupHandlerTest()

	payload := `{"fiat_amount": 100.0, "fiat_currency": "USD", "description": "Test order", "terminal_id": "Register-1"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/invoice", bytes.NewBuffer([]byte(payload)))
	rr := httptest.NewRecorder()

	HandleCreateInvoice(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %v", rr.Code)
	}

	var resp CreateInvoiceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	if resp.InvoicePublicID == "" {
		t.Errorf("Expected InvoicePublicID to be set")
	}
	if resp.Address == "" {
		t.Errorf("Expected Address to be set")
	}
	if resp.XMRAmount <= 0 {
		t.Errorf("Expected XMRAmount to be calculated")
	}

	inv, _ := models.GetInvoiceByPublicID(resp.InvoicePublicID)
	if inv.TerminalID != "Register-1" {
		t.Errorf("Expected terminal ID to be set, got %s", inv.TerminalID)
	}
}

func TestGeneratePublicID(t *testing.T) {
	id1, err := generatePublicID()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	id2, err := generatePublicID()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(id1) != 8 || len(id2) != 8 {
		t.Errorf("Expected ID length 8, got %d and %d", len(id1), len(id2))
	}
	if id1 == id2 {
		t.Errorf("Expected unique IDs, got identical: %s == %s", id1, id2)
	}
}

func TestHandleMockPayment(t *testing.T) {
	setupHandlerTest()

	inv := &models.Invoice{
		PublicID:      "mockpay_123",
		FiatAmount:    5.00,
		XMRAmount:     1000,
		Status:        "pending",
		RequiredConfs: 1,
	}
	models.CreateInvoice(inv)

	req, _ := http.NewRequest(http.MethodPost, "/api/mock-payment", nil)
	rr := httptest.NewRecorder()
	HandleMockPayment(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for missing public_id, got %v", rr.Code)
	}

	req, _ = http.NewRequest(http.MethodPost, "/api/mock-payment?public_id=does_not_exist", nil)
	rr = httptest.NewRecorder()
	HandleMockPayment(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found for non-existent public_id, got %v", rr.Code)
	}

	req, _ = http.NewRequest(http.MethodPost, "/api/mock-payment?public_id=mockpay_123", nil)
	rr = httptest.NewRecorder()
	HandleMockPayment(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %v", rr.Code)
	}

	updated, _ := models.GetInvoiceByPublicID("mockpay_123")
	if updated.Status != "in_mempool" {
		t.Errorf("Expected status in_mempool, got %s", updated.Status)
	}
}

func TestHandleGetActiveTerminalInvoice(t *testing.T) {
	setupHandlerTest()

	req, _ := http.NewRequest(http.MethodGet, "/api/terminal/active", nil)
	rr := httptest.NewRecorder()
	HandleGetActiveTerminalInvoice(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for missing terminal_id, got %v", rr.Code)
	}

	req, _ = http.NewRequest(http.MethodGet, "/api/terminal/active?terminal_id=Empty-Reg", nil)
	rr = httptest.NewRecorder()
	HandleGetActiveTerminalInvoice(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found for empty terminal, got %v", rr.Code)
	}

	inv := &models.Invoice{
		PublicID:   "term_active_123",
		TerminalID: "Active-Reg-1",
		Status:     "pending",
	}
	models.CreateInvoice(inv)

	req, _ = http.NewRequest(http.MethodGet, "/api/terminal/active?terminal_id=Active-Reg-1", nil)
	rr = httptest.NewRecorder()
	HandleGetActiveTerminalInvoice(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %v", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["invoice_public_id"] != "term_active_123" {
		t.Errorf("Expected term_active_123, got %s", resp["invoice_public_id"])
	}
}
