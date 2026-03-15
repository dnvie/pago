package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type PriceCache struct {
	mu          sync.RWMutex
	Prices      map[string]float64
	lastUpdated time.Time
}

type coingeckoResponse struct {
	Monero map[string]float64 `json:"monero"`
}

var globalPriceCache = &PriceCache{
	Prices: make(map[string]float64),
}

// GetCurrentXmrPrice returns the cached price for a given fiat currency.
func GetCurrentXmrPrice(fiatCurrency string) (float64, string, error) {
	globalPriceCache.mu.RLock()
	defer globalPriceCache.mu.RUnlock()

	if len(globalPriceCache.Prices) == 0 {
		return 0, "", fmt.Errorf("[Price Oracle] price oracle is not ready yet.")
	}

	if time.Since(globalPriceCache.lastUpdated) > time.Hour {
		return 0, "", fmt.Errorf("[Price Oracle] cached price is dangerously out of sync")
	}

	fiatLower := strings.ToLower(fiatCurrency)
	price, exists := globalPriceCache.Prices[fiatLower]
	if !exists {
		return 0, "", fmt.Errorf("[Price Oracle] price for currency %s is not available", fiatLower)
	}

	return price, fiatLower, nil
}

// InjectMockPrice manually sets a price in the cache for testing purposes.
func InjectMockPrice(fiat string, price float64) {
	globalPriceCache.mu.Lock()
	defer globalPriceCache.mu.Unlock()
	globalPriceCache.Prices[strings.ToLower(fiat)] = price
	globalPriceCache.lastUpdated = time.Now()
}

// UpdatePriceCache fetches fresh prices from the API and updates the local cache. Includes safety checks to reject suspiciously large price drops.
func UpdatePriceCache(apiKey string) {
	newPrices, err := fetchPriceFromAPI(apiKey)
	if err != nil {
		log.Printf("[Price Oracle] Warning: Failed to fetch new price: %v\n", err)
		return
	}

	globalPriceCache.mu.Lock()
	defer globalPriceCache.mu.Unlock()

	for currency, newPrice := range newPrices {
		oldPrice, exists := globalPriceCache.Prices[currency]

		if newPrice <= 1.00 {
			log.Printf("[Price Oracle] Critical: API returned a suspiciously low price for %s ($%.2f). Rejecting update.\n", currency, newPrice)
			continue
		}

		if exists && oldPrice > 0 {
			dropPercentage := (oldPrice - newPrice) / oldPrice
			if dropPercentage > 0.50 {
				log.Printf("[Price Oracle] Critical: Price for %s dropped by >50%% (from $%.2f to $%.2f). Rejecting update to protect margins.\n", currency, oldPrice, newPrice)
				continue
			}
		}

		globalPriceCache.Prices[currency] = newPrice
	}

	globalPriceCache.lastUpdated = time.Now()
	log.Printf("[Price Oracle] XMR Prices updated for %d currencies\n", len(globalPriceCache.Prices))
}

// fetchPriceFromAPI performs the actual HTTP request to CoinGecko.
func fetchPriceFromAPI(apiKey string) (map[string]float64, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	vsCurrencies := "usd,aed,ars,aud,bdt,bhd,bmd,brl,cad,chf,clp,cny,czk,dkk,eur,gbp,gel,hkd,huf,idr,ils,inr,jpy,krw,kwd,lkr,mmk,mxn,myr,ngn,nok,nzd,php,pkr,pln,rub,sar,sek,sgd,thb,try,twd,uah,vef,vnd,zar,xdr"

	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?vs_currencies=%s&ids=monero&names=Monero&symbols=xmr&include_market_cap=false&include_24hr_vol=false&include_24hr_change=false&include_last_updated_at=false&precision=full", vsCurrencies)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if apiKey != "" {
		req.Header.Add("x-cg-api-key", apiKey)
	} else {
		log.Fatalf("[Price Worker] No Coingecko API key set")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[Price Worker] API returned non-200 status code: %d", resp.StatusCode)
	}

	var data coingeckoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("[Price Worker] failed to decode JSON response: %v", err)
	}

	if len(data.Monero) == 0 {
		return nil, fmt.Errorf("[Price Worker] API response did not contain prices")
	}

	return data.Monero, nil
}
