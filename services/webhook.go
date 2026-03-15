package services

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/dnvie/pago/models"
)

type WebhookPayload struct {
	InvoicePublicID string  `json:"invoice_public_id"`
	OrderID         string  `json:"order_id"`
	Status          string  `json:"status"`
	AmountReceived  uint64  `json:"amount_received"`
	FiatAmount      float64 `json:"fiat_amount"`
	FiatCurrency    string  `json:"fiat_currency"`
}

// DispatchWebhook sends an asynchronous POST request to the merchant's callback URL once an invoice is confirmed.
func DispatchWebhook(inv *models.Invoice) {
	if inv.CallbackURL == "" {
		return
	}

	shouldSend, err := models.MarkWebhookSent(inv.ID)
	if err != nil {
		log.Printf("[Webhook Dispatcher] DB Error checking lock for Invoice %s: %v", inv.PublicID, err)
		return
	}

	if !shouldSend {
		log.Printf("[Webhook Dispatcher] Webhook already sent for Invoice %s. Skipping duplicate.", inv.PublicID)
		return
	}

	payload := WebhookPayload{
		InvoicePublicID: inv.PublicID,
		OrderID:         inv.OrderID,
		Status:          inv.Status,
		AmountReceived:  inv.AmountReceived,
		FiatAmount:      inv.FiatAmount,
		FiatCurrency:    inv.FiatCurrency,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Webhook Dispatcher] Failed to marshal payload: %v", err)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodPost, inv.CallbackURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[Webhook Dispatcher] Failed to create request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Webhook Dispatcher] Failed to send webhook for Invoice %s: %v", inv.PublicID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("[Webhook Dispatcher] Successfully notified store for Invoice %s (Status: %d)", inv.PublicID, resp.StatusCode)
	} else {
		log.Printf("[Webhook Dispatcher] Merchant store rejected webhook for Invoice %s (Status: %d)", inv.PublicID, resp.StatusCode)
	}
}
