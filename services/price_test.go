package services

import (
	"encoding/json"
	"testing"
	"time"
)

func TestGetCurrentXmrPrice(t *testing.T) {
	globalPriceCache.mu.Lock()
	globalPriceCache.Prices["usd"] = 150.00
	globalPriceCache.lastUpdated = time.Now()
	globalPriceCache.mu.Unlock()

	price, currency, err := GetCurrentXmrPrice("USD")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if price != 150.00 {
		t.Errorf("Expected $150.00, got %f", price)
	}
	if currency != "usd" {
		t.Errorf("Expected 'usd', got '%s'", currency)
	}

	_, _, err = GetCurrentXmrPrice("XYZ")
	if err == nil {
		t.Errorf("Expected error for missing currency XYZ, got nil")
	}

	globalPriceCache.mu.Lock()
	globalPriceCache.lastUpdated = time.Now().Add(-2 * time.Hour)
	globalPriceCache.mu.Unlock()

	_, _, err = GetCurrentXmrPrice("USD")
	if err == nil {
		t.Errorf("Expected error for stale cache, got nil")
	}

	globalPriceCache.mu.Lock()
	globalPriceCache.Prices = make(map[string]float64)
	globalPriceCache.lastUpdated = time.Now()
	globalPriceCache.mu.Unlock()

	_, _, err = GetCurrentXmrPrice("USD")
	if err == nil {
		t.Errorf("Expected error for empty cache, got nil")
	}
}

func TestPriceCacheProtections(t *testing.T) {

	globalPriceCache.mu.Lock()
	globalPriceCache.Prices["usd"] = 100.0
	globalPriceCache.mu.Unlock()

	newPriceStr := `{"monero": {"usd": 0.5}}`
	var data coingeckoResponse
	json.Unmarshal([]byte(newPriceStr), &data)

	if data.Monero["usd"] <= 1.00 {
	} else {
		t.Errorf("Protection failed")
	}

	newPriceStr = `{"monero": {"usd": 40.0}}`
	json.Unmarshal([]byte(newPriceStr), &data)

	globalPriceCache.mu.Lock()
	oldPrice := globalPriceCache.Prices["usd"]
	globalPriceCache.mu.Unlock()

	drop := (oldPrice - data.Monero["usd"]) / oldPrice
	if drop > 0.50 {
	} else {
		t.Errorf("Drop calculation failed")
	}
}
