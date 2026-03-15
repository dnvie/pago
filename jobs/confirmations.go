package jobs

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/dnvie/pago/models"
	"github.com/dnvie/pago/services"
)

// StartConfirmationWorker begins a background loop that polls the wallet for transaction confirmations.
func StartConfirmationWorker(duration time.Duration) {
	ticker := time.NewTicker(duration)
	log.Println("[Confirmation Worker] Confirmation polling engine started.")

	go func() {
		for range ticker.C {
			pollConfirmations()
		}
	}()
}

func pollConfirmations() {
	invoices, err := models.GetInvoicesAwaitingConfirmations()
	if err != nil {
		log.Printf("[Confirmation Worker] Error fetching invoices: %v", err)
		return
	}

	for _, inv := range invoices {

		// Only for mock transactions
		if strings.HasPrefix(inv.TXIDs, "mock_tx_") {

			enableMock := os.Getenv("ENABLE_MOCK_PAYMENTS")
			isMockEnabled := (enableMock == "true" || enableMock == "1")

			if !isMockEnabled {
				log.Printf("[Security] WARNING: Invoice #%d has a mock TXID but mock payments are disabled! Skipping.", inv.ID)
				continue
			}

			newConfs := inv.CurrentConfs + 1
			newStatus := "confirming"

			if newConfs >= inv.RequiredConfs {
				newStatus = "confirmed"
				log.Printf("[Mock Cleared] Invoice #%d reached %d/%d confirmations.", inv.ID, newConfs, inv.RequiredConfs)

				inv.Status = newStatus
				go services.DispatchWebhook(inv)
			} else {
				log.Printf("[Mock Progress] Invoice #%d reached %d/%d confirmations.", inv.ID, newConfs, inv.RequiredConfs)
			}

			models.UpdateInvoiceConfirmations(inv.ID, newStatus, newConfs)
			continue
		}

		txidsList := []string{}
		if inv.TXIDs != "" {
			txidsList = strings.Split(inv.TXIDs, ",")
		}

		if len(txidsList) == 0 {
			continue
		}

		lowestConfs := -1
		allConfirmed := true
		hasFailed := false

		for _, txid := range txidsList {
			_, _, txType, confs, err := services.GlobalWalletClient.GetTransferByTxID(txid)
			if err != nil {
				log.Printf("[Confirmation Worker] Failed to query TX %s: %v", txid, err)
				allConfirmed = false
				continue
			}

			if lowestConfs == -1 || confs < lowestConfs {
				lowestConfs = confs
			}

			if txType == "failed" {
				hasFailed = true
				log.Printf("[Confirmation Worker] [DOUBLE SPEND] Invoice #%d (TX: %s) was destroyed or invalidated!", inv.ID, txid)
			} else if txType == "pool" {
				allConfirmed = false
				if inv.CurrentConfs > 0 {
					log.Printf("[Confirmation Worker] [REORG DETECTED] Invoice #%d (TX: %s) dropped from block back to mempool!", inv.ID, txid)
				}
			} else if txType == "in" {
				if confs < inv.RequiredConfs {
					allConfirmed = false
				}
			}
		}

		if lowestConfs == -1 {
			lowestConfs = 0
		}

		newStatus := inv.Status

		if hasFailed {
			newStatus = "failed"
		} else if inv.AmountReceived < inv.XMRAmount {
			newStatus = "underpaid"
		} else if lowestConfs == 0 {
			newStatus = "in_mempool"
		} else if allConfirmed {
			newStatus = "confirmed"
		} else {
			newStatus = "confirming"
		}

		if newStatus != inv.Status || lowestConfs != inv.CurrentConfs {
			err = models.UpdateInvoiceConfirmations(inv.ID, newStatus, lowestConfs)
			if err != nil {
				log.Printf("[Confirmation Worker] Failed to save DB update for Invoice #%d: %v", inv.ID, err)
			} else {
				if newStatus == "confirmed" {
					log.Printf("[Confirmation Worker] [CLEARED] Invoice #%d reached fully confirmed state across all transactions.", inv.ID)

					inv.Status = newStatus
					go services.DispatchWebhook(inv)
				} else {
					log.Printf("[Confirmation Worker] Invoice #%d progressed: %d/%d lowest confirmations (Status: %s)", inv.ID, lowestConfs, inv.RequiredConfs, newStatus)
				}
			}
		}
	}
}
