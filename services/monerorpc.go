package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/icholy/digest"
)

type WalletClient struct {
	client *http.Client
	url    string
}

var GlobalWalletClient *WalletClient

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type createAddressResponse struct {
	Result struct {
		Address      string `json:"address"`
		AddressIndex uint32 `json:"address_index"`
	} `json:"result"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type transferResponse struct {
	Result struct {
		Transfer struct {
			Amount        uint64 `json:"amount"`
			Type          string `json:"type"`
			Confirmations int    `json:"confirmations"`
			SubaddrIndex  struct {
				Minor uint32 `json:"minor"`
			} `json:"subaddr_index"`
		} `json:"transfer"`
		Transfers []struct {
			Amount        uint64 `json:"amount"`
			Type          string `json:"type"`
			Confirmations int    `json:"confirmations"`
			SubaddrIndex  struct {
				Minor uint32 `json:"minor"`
			} `json:"subaddr_index"`
		} `json:"transfers"`
	} `json:"result"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// InitWalletClient initializes the global wallet RPC client with digest authentication.
func InitWalletClient(rpcURL, username, password string) {
	transport := &digest.Transport{
		Username: username,
		Password: password,
	}

	GlobalWalletClient = &WalletClient{
		url: rpcURL,
		client: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		},
	}
	log.Println("[Wallet RPC] Wallet client initialized")
}

// GenerateSubaddress creates a new Monero subaddress through the wallet RPC.
func (wc *WalletClient) GenerateSubaddress(label string) (string, uint32, error) {
	reqBody := rpcRequest{
		JSONRPC: "2.0",
		ID:      "0",
		Method:  "create_address",
		Params: map[string]any{
			"account_index": 0,
			"label":         label,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("[Wallet RPC] failed to marshal request: %v", err)
	}

	resp, err := wc.client.Post(wc.url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", 0, fmt.Errorf("[Wallet RPC] RPC request failed: %v", err)
	}
	defer resp.Body.Close()

	var rpcResp createAddressResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return "", 0, fmt.Errorf("[Wallet RPC] failed to decod RPC response: %v", err)
	}

	if rpcResp.Error != nil {
		return "", 0, fmt.Errorf("[Wallet RPC] wallet error: %s", rpcResp.Error.Message)
	}

	return rpcResp.Result.Address, rpcResp.Result.AddressIndex, nil
}

// GetTransferByTxID retrieves transaction details (amount, type, subaddress) for a given entry.
func (wc *WalletClient) GetTransferByTxID(txid string) (uint64, uint32, string, int, error) {
	reqBody := rpcRequest{
		JSONRPC: "2.0",
		ID:      "0",
		Method:  "get_transfer_by_txid",
		Params: map[string]any{
			"txid": txid,
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := wc.client.Post(wc.url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, 0, "", 0, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, "", 0, err
	}

	var rpcResp transferResponse
	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		return 0, 0, "", 0, err
	}
	if rpcResp.Error != nil {
		return 0, 0, "", 0, fmt.Errorf("[Wallet RPC] monero-wallet-rpc error: %s", rpcResp.Error.Message)
	}

	if len(rpcResp.Result.Transfers) > 0 {
		return rpcResp.Result.Transfers[0].Amount, rpcResp.Result.Transfers[0].SubaddrIndex.Minor, rpcResp.Result.Transfers[0].Type, rpcResp.Result.Transfers[0].Confirmations, nil
	}

	return rpcResp.Result.Transfer.Amount, rpcResp.Result.Transfer.SubaddrIndex.Minor, rpcResp.Result.Transfer.Type, rpcResp.Result.Transfer.Confirmations, nil
}
