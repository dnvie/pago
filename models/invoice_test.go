package models

import (
	"testing"
	"time"
)

func setupTestDB(t *testing.T) {
	err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize in-memory DB: %v", err)
	}
}

func TestCreateAndGetInvoice(t *testing.T) {
	setupTestDB(t)

	newInv := &Invoice{
		PublicID:        "test_inv_123",
		OrderID:         "Order-99",
		FiatAmount:      10.50,
		FiatCurrency:    "USD",
		XMRAmount:       50000000000,
		Address:         "44AFFq5...",
		SubaddressIndex: 1,
		Status:          "pending",
		RequiredConfs:   1,
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(15 * time.Minute),
	}

	err := CreateInvoice(newInv)
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	fetchedInv, err := GetInvoiceByPublicID("test_inv_123")
	if err != nil {
		t.Fatalf("Failed to fetch invoice: %v", err)
	}

	if fetchedInv.OrderID != "Order-99" {
		t.Errorf("Expected OrderID 'Order-99', got '%s'", fetchedInv.OrderID)
	}
}

func TestUpdateInvoiceStatus(t *testing.T) {
	setupTestDB(t)
	inv := &Invoice{PublicID: "stat_1", Status: "pending", SubaddressIndex: 10}
	CreateInvoice(inv)

	err := UpdateInvoiceStatus(inv.ID, "in_mempool", "tx_abc", 1000)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	updated, _ := GetInvoiceByID(inv.ID)
	if updated.Status != "in_mempool" {
		t.Errorf("Expected status in_mempool, got %s", updated.Status)
	}
	if !updated.InMempoolAt.Valid {
		t.Errorf("Expected InMempoolAt to be populated")
	}
	if updated.TXIDs != "tx_abc" {
		t.Errorf("Expected TXID to be saved")
	}

	UpdateInvoiceStatus(inv.ID, "confirmed", "tx_abc", 1000)
	final, _ := GetInvoiceByID(inv.ID)
	if !final.ConfirmedAt.Valid {
		t.Errorf("Expected ConfirmedAt to be populated")
	}
}

func TestExpireOldInvoices(t *testing.T) {
	setupTestDB(t)
	inv1 := &Invoice{PublicID: "exp_1", Status: "pending", ExpiresAt: time.Now().Add(-1 * time.Hour)}
	CreateInvoice(inv1)

	inv2 := &Invoice{PublicID: "exp_2", Status: "pending", ExpiresAt: time.Now().Add(1 * time.Hour)}
	CreateInvoice(inv2)

	count, err := ExpireOldInvoices()
	if err != nil || count != 1 {
		t.Fatalf("Expected 1 expired invoice, got %d. Err: %v", count, err)
	}

	check1, _ := GetInvoiceByPublicID("exp_1")
	if check1.Status != "expired" {
		t.Errorf("Expected exp_1 to be expired, got %s", check1.Status)
	}
}

func TestVoidPendingTerminalInvoices(t *testing.T) {
	setupTestDB(t)
	inv := &Invoice{PublicID: "term_1", TerminalID: "Register-1", Status: "pending"}
	CreateInvoice(inv)

	err := VoidPendingTerminalInvoices("Register-1")
	if err != nil {
		t.Fatalf("Failed to void: %v", err)
	}

	check, _ := GetInvoiceByPublicID("term_1")
	if check.Status != "failed" {
		t.Errorf("Expected terminal invoice to be voided (failed), got %s", check.Status)
	}
}

func TestGetActiveInvoiceByIndex(t *testing.T) {
	setupTestDB(t)

	_, err := GetActiveInvoiceByIndex(999)
	if err == nil {
		t.Errorf("Expected error for missing index")
	}

	inv := &Invoice{PublicID: "idx_1", SubaddressIndex: 5, Status: "in_mempool"}
	CreateInvoice(inv)

	found, err := GetActiveInvoiceByIndex(5)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if found.PublicID != "idx_1" {
		t.Errorf("Expected idx_1, got %s", found.PublicID)
	}
}

func TestGetInvoicesAwaitingConfirmations(t *testing.T) {
	setupTestDB(t)

	inv1 := &Invoice{PublicID: "await_1", Status: "in_mempool", TXIDs: "tx_123", RequiredConfs: 2, CurrentConfs: 0}
	CreateInvoice(inv1)

	inv2 := &Invoice{PublicID: "await_2", Status: "confirmed", TXIDs: "tx_456", RequiredConfs: 1, CurrentConfs: 1}
	CreateInvoice(inv2)

	list, err := GetInvoicesAwaitingConfirmations()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("Expected exactly 1 invoice awaiting confirmations, got %d", len(list))
	}
	if list[0].PublicID != "await_1" {
		t.Errorf("Expected await_1, got %s", list[0].PublicID)
	}
}

func TestUpdateInvoiceConfirmations(t *testing.T) {
	setupTestDB(t)
	inv := &Invoice{PublicID: "conf_update_1", Status: "in_mempool", CurrentConfs: 0}
	CreateInvoice(inv)

	err := UpdateInvoiceConfirmations(inv.ID, "confirming", 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	updated, _ := GetInvoiceByID(inv.ID)
	if updated.Status != "confirming" {
		t.Errorf("Expected status confirming, got %s", updated.Status)
	}
	if updated.CurrentConfs != 1 {
		t.Errorf("Expected 1 conf, got %d", updated.CurrentConfs)
	}
}

func TestUpdateInvoiceTip(t *testing.T) {
	setupTestDB(t)
	inv := &Invoice{PublicID: "tip_update_1", FiatAmount: 10.0, XMRAmount: 1000}
	CreateInvoice(inv)

	err := UpdateInvoiceTip(inv.ID, 12.0, 1200, 20, 2.0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	updated, _ := GetInvoiceByID(inv.ID)
	if updated.FiatAmount != 12.0 {
		t.Errorf("Expected FiatAmount 12.0, got %f", updated.FiatAmount)
	}
	if updated.XMRAmount != 1200 {
		t.Errorf("Expected XMRAmount 1200, got %d", updated.XMRAmount)
	}
	if updated.TipPercentage != 20 {
		t.Errorf("Expected TipPercentage 20, got %d", updated.TipPercentage)
	}
	if updated.TipFiat != 2.0 {
		t.Errorf("Expected TipFiat 2.0, got %f", updated.TipFiat)
	}
}

func TestMarkWebhookSent(t *testing.T) {
	setupTestDB(t)
	inv := &Invoice{PublicID: "wh_sent_1", Status: "confirmed"}
	CreateInvoice(inv)

	success, err := MarkWebhookSent(inv.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !success {
		t.Errorf("Expected MarkWebhookSent to return true for first attempt")
	}

	success2, _ := MarkWebhookSent(inv.ID)
	if success2 {
		t.Errorf("Expected MarkWebhookSent to return false for duplicate attempt")
	}
}
