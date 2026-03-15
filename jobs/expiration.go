package jobs

import (
	"log"
	"time"

	"github.com/dnvie/pago/models"
)

// StartExpireOldInvoices runs a background worker to periodically expire unpaid invoices.
func StartExpireOldInvoices(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		for range ticker.C {
			count, err := models.ExpireOldInvoices()
			if err != nil {
				log.Printf("[Expiration Worker] Error: %v\n", err)
			} else if count > 0 {
				log.Printf("[Expiration Worker] Cleaned up %d expired invoice(s).\n", count)
			}
		}
	}()
}
