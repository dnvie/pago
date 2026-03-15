package models

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// Invoice represents a single customer order
type Invoice struct {
	ID                int          `json:"id"`
	PublicID          string       `json:"public_id"`
	OrderID           string       `json:"order_id"`
	Description       string       `json:"description"`
	Metadata          string       `json:"metadata"`
	FiatAmount        float64      `json:"fiat_amount"`
	TipPercentage     int          `json:"tip_percentage"`
	TipFiat           float64      `json:"tip_fiat"`
	FiatCurrency      string       `json:"fiat_currency"`
	ExchangeRate      float64      `json:"exchange_rate"`
	XMRAmount         uint64       `json:"xmr_amount"`
	AmountReceived    uint64       `json:"amount_received"`
	Address           string       `json:"address"`
	SubaddressIndex   uint32       `json:"subaddress_index"`
	Status            string       `json:"status"`
	RequiredConfs     int          `json:"required_confs"`
	CurrentConfs      int          `json:"current_confs"`
	TXIDs             string       `json:"txids"`
	CallbackURL       string       `json:"callback_url"`
	CreatedAt         time.Time    `json:"created_at"`
	ExpiresAt         time.Time    `json:"expires_at"`
	InMempoolAt       sql.NullTime `json:"in_mempool_at"`
	ConfirmedAt       sql.NullTime `json:"confirmed_at"`
	WebhookSent       int          `json:"webhook_sent"`
	TaxAmount         float64      `json:"tax_amount"`
	TipEnabled        bool         `json:"tip_enabled"`
	ExpirationMinutes int          `json:"expiration_minutes"`
	SuccessURL        string       `json:"success_url"`
	CancelURL         string       `json:"cancel_url"`
	TerminalID        string       `json:"terminal_id"`
}

// InitDB initializes the SQLite database and ensures the invoices table exists.
func InitDB(filepath string) error {
	var err error
	DB, err = sql.Open("sqlite3", filepath)
	if err != nil {
		return fmt.Errorf("[Database] failed to open database: %v", err)
	}

	createTableQuery := `
		CREATE TABLE IF NOT EXISTS invoices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			public_id TEXT UNIQUE,
			order_id TEXT DEFAULT '',
			description TEXT DEFAULT '',
			metadata TEXT DEFAULT '{}',
			fiat_amount REAL,
			tip_percentage INTEGER DEFAULT 0,
			tip_fiat REAL DEFAULT 0,
			fiat_currency TEXT DEFAULT 'USD',
			exchange_rate REAL,
			xmr_amount INTEGER,
			amount_received INTEGER DEFAULT 0,
			address TEXT,
			subaddress_index INTEGER UNIQUE,
			status TEXT,
			required_confs INTEGER DEFAULT 0,
			current_confs INTEGER DEFAULT 0,
			txids TEXT DEFAULT '',
			callback_url TEXT DEFAULT '',
			created_at DATETIME,
			expires_at DATETIME,
			in_mempool_at DATETIME,
			confirmed_at DATETIME,
			webhook_sent INTEGER DEFAULT 0,
			tax_amount REAL DEFAULT 0,
			tip_enabled INTEGER DEFAULT 0,
			expiration_minutes INTEGER DEFAULT 15,
			success_url TEXT DEFAULT '',
			cancel_url TEXT DEFAULT '',
			terminal_id TEXT DEFAULT ''
		);`

	_, err = DB.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("[Database] failed to create table: %v", err)
	}

	log.Println("[Database] SQLite initialized successfully.")
	return nil
}

// CreateInvoice inserts a new invoice record into the database.
func CreateInvoice(inv *Invoice) error {
	query := `
			INSERT INTO invoices (
				public_id, order_id, description, metadata, fiat_amount, tip_percentage, tip_fiat, fiat_currency, exchange_rate,
				xmr_amount, amount_received, address, subaddress_index, status,
				required_confs, current_confs, txids, callback_url, created_at, expires_at,
				tax_amount, tip_enabled, expiration_minutes, success_url, cancel_url, terminal_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := DB.Exec(query,
		inv.PublicID, inv.OrderID, inv.Description, inv.Metadata, inv.FiatAmount, inv.TipPercentage, inv.TipFiat, inv.FiatCurrency, inv.ExchangeRate,
		inv.XMRAmount, inv.AmountReceived, inv.Address, inv.SubaddressIndex, inv.Status,
		inv.RequiredConfs, inv.CurrentConfs, inv.TXIDs, inv.CallbackURL, inv.CreatedAt, inv.ExpiresAt,
		inv.TaxAmount, inv.TipEnabled, inv.ExpirationMinutes, inv.SuccessURL, inv.CancelURL, inv.TerminalID,
	)
	if err != nil {
		return fmt.Errorf("failed to insert invoice: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %v", err)
	}
	inv.ID = int(id)
	return nil
}

// GetActiveInvoiceByIndex retrieves a pending/confirming invoice associated with a specific Monero subaddress index.
func GetActiveInvoiceByIndex(index uint32) (*Invoice, error) {
	query := `SELECT id, public_id, order_id, callback_url, fiat_currency, metadata, fiat_amount, tip_percentage, tip_fiat, exchange_rate, xmr_amount, amount_received, address, subaddress_index, status, required_confs, current_confs, txids, created_at, expires_at, in_mempool_at, confirmed_at, webhook_sent, tax_amount, tip_enabled, expiration_minutes, success_url, cancel_url
	          FROM invoices WHERE subaddress_index = ? AND status IN ('pending', 'in_mempool', 'underpaid', 'confirming')`

	row := DB.QueryRow(query, index)

	var inv Invoice
	err := row.Scan(
		&inv.ID, &inv.PublicID, &inv.OrderID, &inv.CallbackURL, &inv.FiatCurrency,
		&inv.Metadata, &inv.FiatAmount, &inv.TipPercentage, &inv.TipFiat, &inv.ExchangeRate,
		&inv.XMRAmount, &inv.AmountReceived, &inv.Address, &inv.SubaddressIndex,
		&inv.Status, &inv.RequiredConfs, &inv.CurrentConfs, &inv.TXIDs,
		&inv.CreatedAt, &inv.ExpiresAt, &inv.InMempoolAt, &inv.ConfirmedAt, &inv.WebhookSent,
		&inv.TaxAmount, &inv.TipEnabled, &inv.ExpirationMinutes, &inv.SuccessURL, &inv.CancelURL,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no pending invoice found for index %d", index)
		}
		return nil, err
	}
	return &inv, nil
}

// UpdateInvoiceStatus updates the status, received amount, and mempool/confirmation timestamps of an invoce.
func UpdateInvoiceStatus(id int, newStatus, txids string, amount_received uint64) error {
	now := time.Now()

	if newStatus == "in_mempool" {
		query := `UPDATE invoices SET status = ?, in_mempool_at = COALESCE(in_mempool_at, ?), txids = ?, current_confs = 0, amount_received = ? WHERE id = ?`
		_, err := DB.Exec(query, newStatus, now, txids, amount_received, id)
		return err
	}

	if newStatus == "confirming" {
		query := `UPDATE invoices SET status = ?, in_mempool_at = COALESCE(in_mempool_at, ?), txids = ?, current_confs = 1, amount_received = ? WHERE id = ?`
		_, err := DB.Exec(query, newStatus, now, txids, amount_received, id)
		return err
	}

	if newStatus == "confirmed" {
		query := `UPDATE invoices SET status = ?, in_mempool_at = COALESCE(in_mempool_at, ?), confirmed_at = COALESCE(confirmed_at, ?), txids = ?, current_confs = 1, amount_received = ? WHERE id = ?`
		_, err := DB.Exec(query, newStatus, now, now, txids, amount_received, id)
		return err
	}

	query := `UPDATE invoices SET status = ?, amount_received = ?, txids = ? WHERE id = ?`
	_, err := DB.Exec(query, newStatus, amount_received, txids, id)
	return err
}

// GetInvoiceByID retrieves a single invoice by its internal primary key.
func GetInvoiceByID(id int) (*Invoice, error) {
	query := `SELECT id, public_id, order_id, description, metadata, fiat_amount, tip_percentage, tip_fiat, fiat_currency,
		                 exchange_rate, xmr_amount, amount_received, address, subaddress_index,
		                 status, required_confs, current_confs, txids, callback_url,
		                 created_at, expires_at, in_mempool_at, confirmed_at, webhook_sent,
		                 tax_amount, tip_enabled, expiration_minutes, success_url, cancel_url, terminal_id
	          FROM invoices WHERE id = ?`

	row := DB.QueryRow(query, id)

	var inv Invoice
	err := row.Scan(
		&inv.ID, &inv.PublicID, &inv.OrderID, &inv.Description, &inv.Metadata, &inv.FiatAmount,
		&inv.TipPercentage, &inv.TipFiat, &inv.FiatCurrency, &inv.ExchangeRate, &inv.XMRAmount, &inv.AmountReceived,
		&inv.Address, &inv.SubaddressIndex, &inv.Status, &inv.RequiredConfs,
		&inv.CurrentConfs, &inv.TXIDs, &inv.CallbackURL, &inv.CreatedAt,
		&inv.ExpiresAt, &inv.InMempoolAt, &inv.ConfirmedAt, &inv.WebhookSent,
		&inv.TaxAmount, &inv.TipEnabled, &inv.ExpirationMinutes, &inv.SuccessURL, &inv.CancelURL, &inv.TerminalID,
	)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// GetInvoiceByPublicID retrieves a single invoice using its external public_id.
func GetInvoiceByPublicID(publicID string) (*Invoice, error) {
	query := `SELECT id, public_id, order_id, description, metadata, fiat_amount, tip_percentage, tip_fiat, fiat_currency,
		                 exchange_rate, xmr_amount, amount_received, address, subaddress_index,
		                 status, required_confs, current_confs, txids, callback_url,
		                 created_at, expires_at, in_mempool_at, confirmed_at, webhook_sent,
		                 tax_amount, tip_enabled, expiration_minutes, success_url, cancel_url, terminal_id
	          FROM invoices WHERE public_id = ?`

	row := DB.QueryRow(query, publicID)

	var inv Invoice
	err := row.Scan(
		&inv.ID, &inv.PublicID, &inv.OrderID, &inv.Description, &inv.Metadata, &inv.FiatAmount,
		&inv.TipPercentage, &inv.TipFiat, &inv.FiatCurrency, &inv.ExchangeRate, &inv.XMRAmount, &inv.AmountReceived,
		&inv.Address, &inv.SubaddressIndex, &inv.Status, &inv.RequiredConfs,
		&inv.CurrentConfs, &inv.TXIDs, &inv.CallbackURL, &inv.CreatedAt,
		&inv.ExpiresAt, &inv.InMempoolAt, &inv.ConfirmedAt, &inv.WebhookSent,
		&inv.TaxAmount, &inv.TipEnabled, &inv.ExpirationMinutes, &inv.SuccessURL, &inv.CancelURL, &inv.TerminalID,
	)
	if err != nil {
		return nil, err
	}

	return &inv, nil
}

// ExpireOldInvoices marks all pending invoices that have passed their expiration date as 'expired'.
func ExpireOldInvoices() (int64, error) {
	query := `UPDATE invoices SET status = 'expired' WHERE status = 'pending' AND expires_at < ?`
	result, err := DB.Exec(query, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetInvoicesAwaitingConfirmations returns a list of invoices that have incoming transactions but haven't reached required confirmations.
func GetInvoicesAwaitingConfirmations() ([]*Invoice, error) {
	query := `SELECT id, public_id, order_id, callback_url, fiat_currency, fiat_amount, status, required_confs, current_confs, txids, xmr_amount, amount_received
		          FROM invoices
		          WHERE txids != ''
		          AND (current_confs < required_confs OR status = 'in_mempool')
		          AND status NOT IN ('expired', 'failed')`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*Invoice
	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(
			&inv.ID, &inv.PublicID, &inv.OrderID, &inv.CallbackURL, &inv.FiatCurrency, &inv.FiatAmount,
			&inv.Status, &inv.RequiredConfs, &inv.CurrentConfs, &inv.TXIDs,
			&inv.XMRAmount, &inv.AmountReceived,
		); err != nil {
			log.Printf("[Database] Error scanning confirming invoice: %v", err)
			continue
		}
		invoices = append(invoices, &inv)
	}
	return invoices, nil
}

// UpdateInvoiceConfirmations increments the current confirmation count for an invoice in the database.
func UpdateInvoiceConfirmations(id int, status string, currentConfs int) error {
	query := `UPDATE invoices SET status = ?, current_confs = ? WHERE id = ?`
	_, err := DB.Exec(query, status, currentConfs, id)
	return err
}

// UpdateInvoiceTip updates the fiat amount and XMR amount after a customer chooses to add a tip.
func UpdateInvoiceTip(id int, totalFiat float64, xmrAmount uint64, tipPercentage int, tipFiat float64) error {
	query := `UPDATE invoices SET fiat_amount = ?, xmr_amount = ?, tip_percentage = ?, tip_fiat = ? WHERE id = ?`
	_, err := DB.Exec(query, totalFiat, xmrAmount, tipPercentage, tipFiat, id)
	return err
}

// MarkWebhookSent marks the invoice's webhook as sent to prevent multiple triggers.
func MarkWebhookSent(id int) (bool, error) {
	query := `UPDATE invoices SET webhook_sent = 1 WHERE id = ? AND webhook_sent = 0`
	result, err := DB.Exec(query, id)
	if err != nil {
		return false, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rows > 0, nil
}

// VoidPendingTerminalInvoices voids any existing pending invoices for a specific terminal, allowing a new invoice to take precedence on that terminal.
func VoidPendingTerminalInvoices(terminalID string) error {
	if terminalID == "" {
		return nil
	}

	query := `UPDATE invoices SET status = 'failed'
	          WHERE terminal_id = ? AND status IN ('pending', 'underpaid')`

	_, err := DB.Exec(query, terminalID)
	return err
}
