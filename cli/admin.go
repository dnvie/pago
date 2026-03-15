package cli

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/dnvie/pago/models"
)

// RunAdminCLI is the entry point for the admin command-line interface.
func RunAdminCLI(args []string) {
	if len(args) == 0 {
		printHelp()
		os.Exit(1)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		log.Fatalf("[Admin CLI] DB_PATH environment variable not specified")
	}

	err := models.InitDB(dbPath)
	if err != nil {
		log.Fatalf("[Admin CLI] Failed to open database: %v", err)
	}

	command := args[0]
	switch command {
	case "info":
		if len(args) < 2 {
			log.Fatalf("Usage: pago admin info <public_id>")
		}
		handleInfo(args[1])
	case "void":
		if len(args) < 2 {
			log.Fatalf("Usage: pago admin void <public_id>")
		}
		handleVoid(args[1])
	case "list":
		count := 10
		if len(args) >= 2 {
			requested, err := strconv.Atoi(args[1])
			if err == nil {
				count = requested
			}
		}
		handleList(count)
	case "search":
		if len(args) < 2 {
			log.Fatalf("Usage: pago admin search <query>")
		}
		handleSearch(args[1])
	case "delete":
		if len(args) < 2 {
			log.Fatalf("Usage: pago admin delete <public_id>")
		}
		handleDelete(args[1])
	case "stats":
		handleStats()
	default:
		fmt.Printf("[Admin CLI] Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("\n Pago Admin CLI")
	fmt.Println("Usage:")
	fmt.Println("  admin list [n]         - Show last n invoices (default 10)")
	fmt.Println("  admin info <id>        - View detailed status of an invoice")
	fmt.Println("  admin search <query>   - Search by OrderID or Description")
	fmt.Println("  admin void <id>        - Manually fail a pending/underpaid invoice")
	fmt.Println("  admin delete <id>      - Permanently remove a failed/expired invoice")
	fmt.Println("  admin stats            - View revenue and system metrics")
	fmt.Println("")
}

// handleDelete permanently deletes an invoice by its public ID. Only possible if the invoice is "failed" or "expired".
func handleDelete(publicID string) {
	inv, err := models.GetInvoiceByPublicID(publicID)
	if err != nil {
		log.Fatalf("[Admin CLI] Invoice not found: %v", err)
	}

	if inv.Status != "failed" && inv.Status != "expired" {
		log.Fatalf("[Admin CLI] Safety Refusal: Invoice %s is currently '%s'. "+
			"Only 'failed' or 'expired' invoices can be permanently deleted.",
			publicID, inv.Status)
	}

	res, err := models.DB.Exec("DELETE FROM invoices WHERE public_id = ?", publicID)
	if err != nil {
		log.Fatalf("[Admin CLI] Delete failed: %v", err)
	}

	count, _ := res.RowsAffected()
	if count == 0 {
		fmt.Printf("[Admin CLI] Warning: No rows deleted for ID %s\n", publicID)
	} else {
		fmt.Printf("[Admin CLI] Permanently deleted %s invoice: %s\n", inv.Status, publicID)
	}
}

// handleList prints a list of the last n invoices. Default: 10, Maximum: 100.
func handleList(limit int) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		fmt.Println("[Admin CLI] Note: Capping list at 100 entries.")
		limit = 100
	}

	rows, err := models.DB.Query(`
		SELECT public_id, order_id, status, fiat_amount, fiat_currency, created_at
		FROM invoices
		ORDER BY created_at DESC
		LIMIT ?`, limit)

	if err != nil {
		log.Fatalf("[Admin CLI] Failed to list invoices: %v", err)
	}
	defer rows.Close()

	fmt.Printf("\n Last %d Invoices:\n", limit)
	printInvoiceTable(rows)
}

// printInvoiceTable is a helper function to properly format the printed output.
func printInvoiceTable(rows *sql.Rows) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "CREATED\tPUBLIC ID\tORDER ID\tSTATUS\tAMOUNT")
	fmt.Fprintln(w, "-------\t---------\t--------\t------\t------")

	found := false
	for rows.Next() {
		found = true
		var pid, oid, status, curr string
		var amount float64
		var created time.Time

		err := rows.Scan(&pid, &oid, &status, &amount, &curr, &created)
		if err != nil {
			log.Printf("[Admin CLI] Error scanning row: %v", err)
			continue
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.2f %s\n",
			created.Format("01-02 15:04"), pid, oid, status, amount, curr)
	}

	if !found {
		fmt.Fprintln(w, "(No invoices found)\t\t\t\t")
	}

	w.Flush()
	fmt.Println("")
}

// handleSearch allows to search for invoices by their description or order ID
func handleSearch(query string) {
	searchQuery := "%" + query + "%"
	rows, err := models.DB.Query("SELECT public_id, order_id, status, fiat_amount, fiat_currency, created_at FROM invoices WHERE order_id LIKE ? OR description LIKE ? ORDER BY created_at DESC", searchQuery, searchQuery)
	if err != nil {
		log.Fatalf("[Admin CLI] Search failed: %v", err)
	}
	defer rows.Close()

	fmt.Printf("\n Search results for '%s':\n", query)
	printInvoiceTable(rows)
}

// handleInfo returns details for a specific invoice by its public ID.
func handleInfo(publicID string) {
	inv, err := models.GetInvoiceByPublicID(publicID)
	if err != nil {
		log.Fatalf("[Admin CLI] Invoice not found: %v", err)
	}

	fmt.Printf("\n Invoice Details: %s\n", inv.PublicID)
	fmt.Println("--------------------------------------------------")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Order ID:\t%s\n", inv.OrderID)
	fmt.Fprintf(w, "Status:\t%s\n", inv.Status)
	fmt.Fprintf(w, "Fiat Total:\t%.2f %s\n", inv.FiatAmount, inv.FiatCurrency)
	fmt.Fprintf(w, "XMR Expected:\t%.12f XMR\n", float64(inv.XMRAmount)/1e12)
	fmt.Fprintf(w, "XMR Received:\t%.12f XMR\n", float64(inv.AmountReceived)/1e12)
	fmt.Fprintf(w, "Confirmations:\t%d / %d\n", inv.CurrentConfs, inv.RequiredConfs)
	fmt.Fprintf(w, "Terminal:\t%s\n", inv.TerminalID)
	fmt.Fprintf(w, "Created At:\t%s\n", inv.CreatedAt.Format("2006-01-02 15:04:05"))
	w.Flush()
	fmt.Println("--------------------------------------------------")
}

// handleVoid sets the status of a specific invoice to "failed". Only works for invoices with status "pending" or "underpaid".
func handleVoid(publicID string) {
	inv, err := models.GetInvoiceByPublicID(publicID)
	if err != nil {
		log.Fatalf("[Admin CLI] Invoice not found: %v", err)
	}

	if inv.Status != "pending" && inv.Status != "underpaid" {
		log.Fatalf("[Admin CLI] Cannot void invoice in '%s' state. Only pending or underpaid invoices can be voided.", inv.Status)
	}

	_, err = models.DB.Exec("UPDATE invoices SET status = 'failed' WHERE public_id = ?", publicID)
	if err != nil {
		log.Fatalf("[Admin CLI] Failed to void invoice: %v", err)
	}

	fmt.Printf("[Admin CLI] Successfully voided Invoice %s\n", publicID)
}

// handleStats prints payment statistics of the POS system (Total Paid Orders, Total XMR Revenue, Active/Pending Orders).
func handleStats() {
	var totalXMR uint64
	var count int

	err := models.DB.QueryRow("SELECT COALESCE(SUM(amount_received), 0), COUNT(*) FROM invoices WHERE status = 'confirmed'").Scan(&totalXMR, &count)
	if err != nil {
		log.Fatalf("[Admin CLI] Failed to calculate stats: %v", err)
	}

	var pendingCount int
	models.DB.QueryRow("SELECT COUNT(*) FROM invoices WHERE status IN ('pending', 'confirming', 'in_mempool')").Scan(&pendingCount)

	fmt.Printf("\n Pago System Statistics\n")
	fmt.Println("--------------------------------------------------")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Total Paid Orders:\t%d\n", count)
	fmt.Fprintf(w, "Total XMR Revenue:\t%.12f XMR\n", float64(totalXMR)/1e12)
	fmt.Fprintf(w, "Active/Pending Orders:\t%d\n", pendingCount)
	w.Flush()
	fmt.Println("--------------------------------------------------")
}
