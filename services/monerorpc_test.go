package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateSubaddress(t *testing.T) {
	mockRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"result": {
				"address": "44AFFq5kSiGBoZ4NMDwYtN18obc8AemS33DBLWs3H7otXft3XjrpDtQGv7SqSsaBYBb98uNbr2VBBEt7f2wfn3RVGQBEP3A",
				"address_index": 99
			}
		}`))
	}))
	defer mockRPC.Close()

	InitWalletClient(mockRPC.URL, "testuser", "testpass")

	addr, idx, err := GlobalWalletClient.GenerateSubaddress("test_label")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if addr == "" {
		t.Errorf("Expected address, got empty string")
	}
	if idx != 99 {
		t.Errorf("Expected index 99, got %d", idx)
	}
}

func TestGetTransferByTxID_Success(t *testing.T) {
	mockRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"result": {
				"transfer": {
					"amount": 1500000000000,
					"type": "in",
					"confirmations": 10,
					"subaddr_index": {"minor": 5}
				}
			}
		}`))
	}))
	defer mockRPC.Close()

	InitWalletClient(mockRPC.URL, "testuser", "testpass")

	amount, minorIdx, txType, confs, err := GlobalWalletClient.GetTransferByTxID("fake_txid_123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if amount != 1500000000000 {
		t.Errorf("Expected amount 1500000000000, got %d", amount)
	}
	if minorIdx != 5 {
		t.Errorf("Expected minor index 5, got %d", minorIdx)
	}
	if txType != "in" {
		t.Errorf("Expected type 'in', got '%s'", txType)
	}
	if confs != 10 {
		t.Errorf("Expected 10 confirmations, got %d", confs)
	}
}

func TestGetTransferByTxID_WalletError(t *testing.T) {
	mockRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"error": {
				"message": "tx not found"
			}
		}`))
	}))
	defer mockRPC.Close()

	InitWalletClient(mockRPC.URL, "testuser", "testpass")

	_, _, _, _, err := GlobalWalletClient.GetTransferByTxID("invalid_txid")
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	if err.Error() != "[Wallet RPC] monero-wallet-rpc error: tx not found" {
		t.Errorf("Unexpected error message: %v", err)
	}
}
