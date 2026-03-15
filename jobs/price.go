package jobs

import (
	"log"
	"time"

	"github.com/dnvie/pago/services"
)

var apiKey string

// StartPriceOracle initializes and maintains a background loop to update fiat/XMR exchange rates.
func StartPriceOracle(interval time.Duration, apiKey string) {

	if apiKey == "" {
		log.Println("[Price Worker] No CoinGecko API key provided. Using heavily rate-limited public tier.")
	} else {
		log.Println("[Price Worker] CoinGecko API key loaded.")
	}

	services.UpdatePriceCache(apiKey)

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			services.UpdatePriceCache(apiKey)
		}
	}()
}
