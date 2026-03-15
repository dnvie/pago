package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/dnvie/pago/models"
	"github.com/dnvie/pago/services"
)

type WebhookPayload struct {
	TxID string `json:"txid"`
}

// HandleTxNotify processes incoming transaction notifications from the Monero wallet.
// It verifies the transaction, updates invoice status, and triggers webhooks on completion.
func HandleTxNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.TxID == "" {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	log.Printf("[Webhook] Received txid: %s", payload.TxID)

	amountReceived, subaddressIndex, txType, _, err := services.GlobalWalletClient.GetTransferByTxID(payload.TxID)
	if err != nil {
		log.Printf("[Webhook] Failed to get transfer info: %v", err)
		http.Error(w, "Wallet error", http.StatusInternalServerError)
		return
	}

	invoice, err := models.GetActiveInvoiceByIndex(subaddressIndex)
	if err != nil {
		log.Printf("[Webhook] No active invoice found for index %d", subaddressIndex)
		w.WriteHeader(http.StatusOK)
		return
	}

	txidsList := []string{}
	if invoice.TXIDs != "" {
		txidsList = strings.Split(invoice.TXIDs, ",")
	}

	alreadyKnown := slices.Contains(txidsList, payload.TxID)

	newTotalAmount := invoice.AmountReceived
	newTxidsStr := invoice.TXIDs

	if !alreadyKnown {
		newTotalAmount += amountReceived
		if newTxidsStr == "" {
			newTxidsStr = payload.TxID
		} else {
			newTxidsStr = newTxidsStr + "," + payload.TxID
		}
	}

	if newTotalAmount >= invoice.XMRAmount {

		var newStatus string
		if txType == "pool" {
			newStatus = "in_mempool"
		} else if txType == "in" {
			if invoice.RequiredConfs <= 1 {
				newStatus = "confirmed"
			} else {
				newStatus = "confirming"
			}
		}

		if invoice.Status == newStatus && alreadyKnown {
			w.WriteHeader(http.StatusOK)
			return
		}

		err = models.UpdateInvoiceStatus(invoice.ID, newStatus, newTxidsStr, newTotalAmount)
		if err != nil {
			log.Printf("[Webhook] Failed to update DB status: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		log.Printf("[Webhook] SUCCESS! Invoice #%d transitioned to: %s. Amount: %d piconeros", invoice.ID, newStatus, newTotalAmount)

		if newStatus == "confirmed" {
			invoice.Status = newStatus
			invoice.AmountReceived = newTotalAmount
			go services.DispatchWebhook(invoice)
		}
	} else {
		err = models.UpdateInvoiceStatus(invoice.ID, "underpaid", newTxidsStr, newTotalAmount)
		if err != nil {
			log.Printf("[Webhook] Failed to update DB status for underpayment: %v", err)
		} else {
			log.Printf("[Webhook] PARTIAL PAYMENT! Invoice #%d expects %d but only has %d piconeros total", invoice.ID, invoice.XMRAmount, newTotalAmount)
		}
	}

	w.WriteHeader(http.StatusOK)
}
