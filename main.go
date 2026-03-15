package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dnvie/pago/cli"
	"github.com/dnvie/pago/handlers"
	"github.com/dnvie/pago/jobs"
	"github.com/dnvie/pago/models"
	"github.com/dnvie/pago/services"
)

func main() {

	if len(os.Args) > 1 && os.Args[1] == "admin" {
		cli.RunAdminCLI(os.Args[2:])
		return
	}

	log.Println("[Main] Starting Pago, a Monero Point-of-Sale System...")

	cgKeyFlag := flag.String("api-key", "", "CoinGecko API Key for fiat exchange rates")
	pagoKeyFlag := flag.String("pago-key", "", "API Key to protect invoice creation")
	mockFlag := flag.String("enable-mock-payments", "", "Enable the option to mock transactions for testing purposes")
	flag.Parse()

	coingeckoKey := *cgKeyFlag
	if coingeckoKey == "" {
		coingeckoKey = os.Getenv("COINGECKO_API_KEY")
		if coingeckoKey == "" {
			log.Fatal("[FATAL] COINGECKO_API_KEY environment variable is missing. Pago cannot start without it.")
		}
	}

	pagoAPIKey := *pagoKeyFlag
	if pagoAPIKey == "" {
		pagoAPIKey = os.Getenv("PAGO_API_KEY")
	}
	if pagoAPIKey == "" {
		log.Println("[Security Warning] No PAGO_API_KEY set. Anyone could create invoices on this server!")
	}

	enableMock := *mockFlag
	if enableMock == "" {
		enableMock = os.Getenv("ENABLE_MOCK_PAYMENTS")
	}

	isMockEnabled := (enableMock == "true" || enableMock == "1")

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		log.Fatal("[FATAL] DB_PATH environment variable is missing. Pago cannot start without it.")
	}

	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		log.Fatal("[FATAL] RPC_URL environment variable is missing. Pago cannot start without it.")
	}

	rpcUser := os.Getenv("RPC_USER")
	if rpcUser == "" {
		log.Fatal("[FATAL] RPC_USER environment variable is missing. Pago cannot start without it.")
	}

	rpcPass := os.Getenv("RPC_PASS")
	if rpcPass == "" {
		log.Fatal("[FATAL] RPC_PASS environment variable is missing. Pago cannot start without it.")
	}

	err := models.InitDB(dbPath)
	if err != nil {
		log.Printf("[Database] Database setup failed: %v", err)
	}

	jobs.StartPriceOracle(10*time.Minute, coingeckoKey)
	jobs.StartExpireOldInvoices(1 * time.Minute)
	jobs.StartConfirmationWorker(2 * time.Second)

	services.InitWalletClient(rpcURL, rpcUser, rpcPass)

	http.HandleFunc("/api/create-invoice", corsMiddleware(requireAPIKey(pagoAPIKey, handlers.HandleCreateInvoice)))
	http.HandleFunc("/api/terminal/active", corsMiddleware(handlers.HandleGetActiveTerminalInvoice))
	http.HandleFunc("/api/invoice/status", corsMiddleware(handlers.HandleGetInvoiceStatus))
	http.HandleFunc("/api/invoice/tip", corsMiddleware(handlers.HandleAddTip))
	http.HandleFunc("/api/monero-webhook", handlers.HandleTxNotify)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{
			"mock_payments_enabled": isMockEnabled,
		})
	})

	//DEV
	if isMockEnabled {
		log.Println("[Warning] Mock payments are ENABLED.")
		http.HandleFunc("/api/mock-payment", corsMiddleware(handlers.HandleMockPayment))
	}

	port := ":8080"
	log.Printf("[Main] Pago Backend API listening on port %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("[Main] Server failed to start: %v\n", err)
	}
}

// requireAPIKey is a middleware that enforces X-API-Key header validation if a key is configured.
func requireAPIKey(key string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if key != "" {
			reqKey := r.Header.Get("X-API-Key")
			if reqKey != key {
				http.Error(w, "Unauthorized: Invalid or missing X-API-Key header", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

// corsMiddleware adds basic CORS headers to every response.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
