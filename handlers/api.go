package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/dnvie/pago/models"
	"github.com/dnvie/pago/services"
)

type CreateInvoiceRequest struct {
	FiatAmount        float64        `json:"fiat_amount"`
	FiatCurrency      string         `json:"fiat_currency"`
	RequiredConfs     int            `json:"required_confs"`
	OrderID           string         `json:"order_id"`
	Description       string         `json:"description"`
	Metadata          map[string]any `json:"metadata,omitempty"`
	CallbackURL       string         `json:"callback_url"`
	TaxAmount         float64        `json:"tax_amount,omitempty"`
	TipEnabled        bool           `json:"tip_enabled"`
	ExpirationMinutes int            `json:"expiration_minutes,omitempty"`
	SuccessURL        string         `json:"success_url,omitempty"`
	CancelURL         string         `json:"cancel_url,omitempty"`
	TerminalID        string         `json:"terminal_id,omitempty"`
}

type CreateInvoiceResponse struct {
	InvoicePublicID string         `json:"invoice_public_id"`
	Address         string         `json:"address"`
	XMRAmount       float64        `json:"xmr_amount"`
	URI             string         `json:"uri"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	ExpiresAt       time.Time      `json:"expires_at"`
}

type AddTipRequest struct {
	InvoicePublicID string `json:"invoice_public_id"`
	TipPercentage   int    `json:"tip_percentage"`
}

type InvoiceStatusResponse struct {
	InvoicePublicID string         `json:"invoice_public_id"`
	OrderID         string         `json:"order_id,omitempty"`
	Description     string         `json:"description,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	FiatAmount      float64        `json:"fiat_amount"`
	FiatCurrency    string         `json:"fiat_currency"`
	ExchangeRate    float64        `json:"exchange_rate"`
	Address         string         `json:"address"`
	Status          string         `json:"status"`
	PaymentCleared  bool           `json:"payment_cleared"`
	RequiredConfs   int            `json:"required_confs"`
	CurrentConfs    int            `json:"current_confs"`
	XMRAmount       uint64         `json:"xmr_amount"`
	AmountReceived  uint64         `json:"amount_received"`
	CreatedAt       time.Time      `json:"created_at"`
	ExpiresAt       time.Time      `json:"expires_at"`
	InMempoolAt     *time.Time     `json:"in_mempool_at,omitempty"`
	ConfirmedAt     *time.Time     `json:"confirmed_at,omitempty"`
	TaxAmount       float64        `json:"tax_amount"`
	TipEnabled      bool           `json:"tip_enabled"`
	SuccessURL      string         `json:"success_url,omitempty"`
	CancelURL       string         `json:"cancel_url,omitempty"`
}

// HandleCreateInvoice fetches the current fiat price for the specified currency, converts to XMR,
// generates a publicID and subaddress, and saves the invoice
func HandleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FiatAmount <= 0 {
		http.Error(w, "Invalid fiat amount", http.StatusBadRequest)
		return
	}

	fiatCurrency := req.FiatCurrency
	if fiatCurrency == "" {
		fiatCurrency = "usd"
	}

	currentPrice, currencyMatched, err := services.GetCurrentXmrPrice(fiatCurrency)
	if err != nil {
		log.Printf("[API] Error getting price: %v", err)
		http.Error(w, fmt.Sprintf("Price oracle unavailable for currency: %s", fiatCurrency), http.StatusInternalServerError)
		return
	}

	rawXmrTotal := req.FiatAmount / currentPrice
	piconeros := uint64(rawXmrTotal * 1e12)

	cleanXmrTotal := float64(piconeros) / 1e12

	label := fmt.Sprintf("Order_%d", time.Now().Unix())
	address, index, err := services.GlobalWalletClient.GenerateSubaddress(label)
	if err != nil {
		log.Printf("[API] Wallet RPC error: %v", err)
		http.Error(w, "Failed to generate payment address", http.StatusInternalServerError)
		return
	}

	metadataStr := "{}"
	if req.Metadata != nil {
		if mdBytes, err := json.Marshal(req.Metadata); err == nil {
			metadataStr = string(mdBytes)
		}
	}

	expiresMins := req.ExpirationMinutes
	if expiresMins <= 0 {
		expiresMins = 15
	}
	expires := time.Now().Add(time.Duration(expiresMins) * time.Minute)

	if req.TerminalID != "" {
		err := models.VoidPendingTerminalInvoices(req.TerminalID)
		if err != nil {
			log.Printf("[API] Failed to void old terminal invoices: %v", err)
		}
	}

	newInvoice := models.Invoice{
		FiatAmount:        req.FiatAmount,
		FiatCurrency:      strings.ToUpper(currencyMatched),
		ExchangeRate:      currentPrice,
		XMRAmount:         piconeros,
		AmountReceived:    0,
		Address:           address,
		SubaddressIndex:   index,
		Status:            "pending",
		RequiredConfs:     req.RequiredConfs,
		CurrentConfs:      0,
		TXIDs:             "",
		OrderID:           req.OrderID,
		Description:       req.Description,
		Metadata:          metadataStr,
		CallbackURL:       req.CallbackURL,
		CreatedAt:         time.Now(),
		ExpiresAt:         expires,
		TaxAmount:         req.TaxAmount,
		TipEnabled:        req.TipEnabled,
		ExpirationMinutes: expiresMins,
		SuccessURL:        req.SuccessURL,
		CancelURL:         req.CancelURL,
		TerminalID:        req.TerminalID,
	}

	var dbErr error
	maxAttempts := 3

	for range maxAttempts {
		newInvoice.PublicID, err = generatePublicID()
		if err != nil {
			log.Printf("Error creating invoice id: %v", dbErr)
			http.Error(w, "Failed to create id for invoice", http.StatusInternalServerError)
		}

		dbErr = models.CreateInvoice(&newInvoice)
		if dbErr == nil {
			break
		}

		log.Printf("[API] ID Collision detected (%s). Retrying...", newInvoice.PublicID)
	}

	if dbErr != nil {
		log.Printf("Database error: %v", dbErr)
		http.Error(w, "Failed to save invoice", http.StatusInternalServerError)
		return
	}

	uri := fmt.Sprintf("monero:%s?tx_amount=%.12f", address, cleanXmrTotal)

	resp := CreateInvoiceResponse{
		InvoicePublicID: newInvoice.PublicID,
		Address:         address,
		XMRAmount:       cleanXmrTotal,
		URI:             uri,
		Metadata:        req.Metadata,
		ExpiresAt:       expires,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	log.Printf("[API] Generated Invoice #%d for %.2f %s", newInvoice.ID, req.FiatAmount, strings.ToUpper(fiatCurrency))
}

// HandleGetInvoiceStatus returns current status and payment details for a specific invoice.
func HandleGetInvoiceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	publicID := r.URL.Query().Get("public_id")
	if publicID == "" {
		http.Error(w, "Missing invoice public_id", http.StatusBadRequest)
		return
	}

	invoice, err := models.GetInvoiceByPublicID(publicID)
	if err != nil {
		log.Printf("[API] Database error fetching invoice %s: %v", publicID, err)
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	var inMempoolAt, confirmedAt *time.Time
	if invoice.InMempoolAt.Valid {
		inMempoolAt = &invoice.InMempoolAt.Time
	}
	if invoice.ConfirmedAt.Valid {
		confirmedAt = &invoice.ConfirmedAt.Time
	}

	cleared := false
	if invoice.CurrentConfs >= invoice.RequiredConfs {
		if invoice.Status != "pending" && invoice.Status != "expired" {
			cleared = true
		}
	}

	var metadata map[string]any
	if invoice.Metadata != "" && invoice.Metadata != "{}" {
		if err := json.Unmarshal([]byte(invoice.Metadata), &metadata); err != nil {
			log.Printf("[API] Failed to parse metadata for invoice %s", invoice.PublicID)
		}
	}

	resp := InvoiceStatusResponse{
		InvoicePublicID: invoice.PublicID,
		OrderID:         invoice.OrderID,
		Description:     invoice.Description,
		Metadata:        metadata,
		FiatAmount:      invoice.FiatAmount,
		FiatCurrency:    invoice.FiatCurrency,
		ExchangeRate:    invoice.ExchangeRate,
		Address:         invoice.Address,
		Status:          invoice.Status,
		PaymentCleared:  cleared,
		RequiredConfs:   invoice.RequiredConfs,
		CurrentConfs:    invoice.CurrentConfs,
		XMRAmount:       invoice.XMRAmount,
		AmountReceived:  invoice.AmountReceived,
		CreatedAt:       invoice.CreatedAt,
		ExpiresAt:       invoice.ExpiresAt,
		InMempoolAt:     inMempoolAt,
		ConfirmedAt:     confirmedAt,
		TaxAmount:       invoice.TaxAmount,
		TipEnabled:      invoice.TipEnabled,
		SuccessURL:      invoice.SuccessURL,
		CancelURL:       invoice.CancelURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

const base62Charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var max = big.NewInt(int64(len(base62Charset)))

// generatePublicID creates a random 8-character base62 string for public invoice identification.
func generatePublicID() (string, error) {
	result := make([]byte, 8)

	for i := range result {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		result[i] = base62Charset[n.Int64()]
	}

	return string(result), nil
}

// HandleMockPayment simulates a payment for testing purposes (when mock payments are enabled).
func HandleMockPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	publicID := r.URL.Query().Get("public_id")
	if publicID == "" {
		http.Error(w, "Missing public_id", http.StatusBadRequest)
		return
	}

	inv, err := models.GetInvoiceByPublicID(publicID)
	if err != nil {
		log.Printf("[Dev Mock] Invoice not found: %s", publicID)
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	mockTxID := fmt.Sprintf("mock_tx_%d", time.Now().Unix())

	err = models.UpdateInvoiceStatus(inv.ID, "in_mempool", mockTxID, inv.XMRAmount)
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	log.Printf("[Dev Mock] Simulated payment for Invoice #%d (%s)", inv.ID, publicID)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

// HandleAddTip adds a percentage-based tip to a pending invoice and updates its XMR total.
func HandleAddTip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AddTipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid tip request body", http.StatusBadRequest)
		return
	}

	if req.InvoicePublicID == "" || req.TipPercentage <= 0 {
		http.Error(w, "Invalid tip parameters", http.StatusBadRequest)
		return
	}

	inv, err := models.GetInvoiceByPublicID(req.InvoicePublicID)
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	if inv.Status != "pending" {
		http.Error(w, "Invoice is not pending, cannot add tip", http.StatusBadRequest)
		return
	}

	baseFiat := inv.FiatAmount - inv.TipFiat
	tipFiat := baseFiat * (float64(req.TipPercentage) / 100.0)
	newTotalFiat := baseFiat + tipFiat

	currentPrice, _, err := services.GetCurrentXmrPrice(inv.FiatCurrency)
	if err != nil {
		log.Printf("[API] Error getting price during tip: %v", err)
		http.Error(w, "Price oracle unavailable, cannot calculate tip XMR", http.StatusInternalServerError)
		return
	}

	rawXmrTotal := newTotalFiat / currentPrice
	newPiconeros := uint64(rawXmrTotal * 1e12)

	err = models.UpdateInvoiceTip(inv.ID, newTotalFiat, newPiconeros, req.TipPercentage, tipFiat)
	if err != nil {
		log.Printf("[API] Failed to update tip for invoice %s: %v", inv.PublicID, err)
		http.Error(w, "Failed to apply tip", http.StatusInternalServerError)
		return
	}

	log.Printf("[API] Invoice %s tipped %d%% (%.2f %s)! New Total: %.2f %s", inv.PublicID, req.TipPercentage, tipFiat, inv.FiatCurrency, newTotalFiat, inv.FiatCurrency)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

// HandleGetActiveTerminalInvoice retrieves the most recent active invoice for a specific terminal ID.
func HandleGetActiveTerminalInvoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	terminalID := r.URL.Query().Get("terminal_id")
	if terminalID == "" {
		http.Error(w, "Missing terminal_id parameter", http.StatusBadRequest)
		return
	}

	query := `SELECT public_id FROM invoices
	          WHERE terminal_id = ? AND status IN ('pending', 'underpaid', 'in_mempool', 'confirming')
	          ORDER BY created_at DESC LIMIT 1`

	var publicID string
	err := models.DB.QueryRow(query, terminalID).Scan(&publicID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "No active invoice found for this terminal", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"invoice_public_id": publicID,
	})
}
