![pagoLogo](https://i.imgur.com/LzeFLc0.png)
# Pago: Self-Hosted Point-of-Sale for Monero Payments

*([/ˈpa.ɡo/] — Esperanto for payment)*

Pago is a lightweight, fully self-hosted Point-of-Sale (POS) backend and embeddable checkout widget for accepting Monero (XMR) payments. Built in Go and powered by SQLite, Pago enables merchants to accept Monero payments with zero reliance on third-party payment processors, zero fees, and total financial sovereignty.

Whether you are running a physical shop with a tablet register or an online e-commerce store, Pago handles the heavy lifting of XMR price conversions, subaddress generation, blockchain polling, 0-conf detection, tip prompts, and webhook dispatching.

![pagoPaymentScreens](https://i.imgur.com/8wmDHCB.jpeg)

---

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Installation & Setup (Docker)](#installation--setup-docker)
  - [Prerequisites](#prerequisites)
  - [Step 1: Clone the Repository](#step-1-clone-the-repository)
  - [Step 2: Place Your Wallet File](#step-2-place-your-wallet-file)
  - [Step 3: Configure Environment Variables](#step-3-configure-environment-variables)
  - [Step 4: Start Pago](#step-4-start-pago)
  - [Step 5: Verify](#step-5-verify)
- [In-Person POS Setup](#in-person-pos-setup)
- [Online Integration (E-Commerce)](#online-integration-e-commerce)
- [API Reference](#api-reference)
- [Webhook Callbacks](#webhook-callbacks-callback_url)
- [Invoice Lifecycle](#invoice-lifecycle)
- [Database Schema](#database-schema)
- [Admin CLI](#admin-cli)
- [Background Workers](#background-workers)
- [Supported Currencies](#supported-currencies)
- [Transaction Notification Script](#transaction-notification-script-notifysh)
- [Security Considerations](#security-considerations)
- [Contributing](#contributing)
- [Disclaimer](#disclaimer)
- [Donate](#donate)

## Features

-   **Self-Hosted & Non-Custodial:** Your keys, your coins. Pago connects directly to a `monero-wallet-rpc` instance that *you* control.
-   **Physical POS & Online Widget:** Includes a beautiful, embeddable JavaScript widget (`pago.js`) with built-in QR code generation and real-time status updates, alongside a dedicated merchant cashier interface and customer-facing idle screen.
-   **0-Conf Support:** Accept instant payments for low-value transactions using mempool detection. For high-value transactions, you can choose to require up to 10 confirmations.
-   **Real-time Price Oracle:** Automatically converts 46 fiat currencies to XMR in regular intervals via the CoinGecko API. Includes safety checks against stale or manipulated price data.
-   **Tipping:** Optionally prompts the customer to add a tip (10%, 15%, 20%, or a custom amount) which gets added to the invoice total before the QR code is displayed.
-   **Underpayment Handling:** If a customer sends too little XMR, the widget automatically detects it, updates the QR code to show the remaining balance, and waits for the difference.
-   **Webhook Callbacks:** Automatically `POST`s a JSON payload to your backend the second a payment is fully confirmed. Includes deduplication to prevent double-firing.
-   **Success & Cancel URLs:** For online integrations, redirect the customer to a success or cancellation page after the payment flow completes.
-   **Invoice Expiration:** Invoices expire after a configurable time (default: 15 minutes). A background worker automatically marks expired invoices.
-   **Admin CLI:** Manage your database, view revenue stats, inspect invoices, and void stuck orders directly from the command line.
-   **Multi-language:** Set the customer facing widget to English, German, Italian or Norwegian.
-   **Mock Payments (Dev Mode):** Simulate payment flows end-to-end without touching Monero mainnet or stagenet. Useful for testing.

---

## Architecture

Pago consists of three main components that run together inside Docker:

1.  **Pago Go Backend (`pago`):** The core engine. It serves the REST API, manages the SQLite database, runs background workers for confirmation polling, price updates, and invoice expiration, and serves the static frontend files.
2.  **Monero Wallet RPC (`wallet-rpc`):** The official `monero-wallet-rpc`. Pago communicates with it over HTTP to generate unique subaddresses for each invoice and to verify transaction details.
3.  **Static Frontend:** Three files (`index.html`, `merchant.html`, `pago.js`) served from the `/static` directory. `merchant.html` is the cashier register, `index.html` is the customer-facing idle/checkout screen, and `pago.js` is the embeddable widget that can also be used standalone on any website.

---

## Installation & Setup (Docker)

### Prerequisites

-   [Docker](https://www.docker.com/) and [Docker Compose](https://docs.docker.com/compose/) installed on your system.
-   A Monero **view-only wallet file** (`.keys` file). This is required for Pago to generate subaddresses and verify incoming transactions without having spending capabilities on the server. Learn how to create a view-only wallet [here](https://www.getmonero.org/resources/user-guides/view_only.html#:~:text=CLI%3A%20Creating%20a%20View%2DOnly,view%2Dkey%20wallet%2Dname%20.).
-   A [CoinGecko API key](https://www.coingecko.com/en/api) (Free tier is sufficient).

### Step 1: Clone the Repository

```bash
git clone https://github.com/dnvie/pago.git
cd pago
```

### Step 2: Place Your Wallet File

Copy your view-only wallet `.keys` file into `data/wallet`.

```bash
cp /path/to/your/wallet-file.keys data/wallet/
```

> **Important:** Only use a **view-only** wallet. This wallet can generate subaddresses and verify incoming transactions but cannot spend funds, keeping your coins safe even if the server is compromised.

### Step 3: Configure Environment Variables

Edit the `.env.example` file or create a new `.env`file in the project root.
If you edit `.env.example`, don't forget to rename it to `.env` after setting all environment variables!

```dotenv
# ==========================================
# PAGO CONFIGURATION
# ==========================================

# A secure API key to protect invoice creation via the API.
# Any request to /api/create-invoice must include this key
# in the X-API-Key header.
PAGO_API_KEY=your_secure_api_key_here

# CoinGecko API key for fetching XMR exchange rates.
COINGECKO_API_KEY=your_coingecko_key_here

# Set to "true" to enable the mock payment button on the
# merchant interface (for testing/development only).
ENABLE_MOCK_PAYMENTS=false

# ==========================================
# MONERO WALLET CONFIGURATION
# ==========================================

# The exact filename of your wallet file inside ./data/wallet/
WALLET_FILENAME=your-wallet-file.keys

# The password used to unlock your Monero wallet file.
WALLET_PASSWORD=your_wallet_password

# Optional: Your own Monero daemon/node. Defaults to a
# public node if left empty.
DAEMON_ADDRESS=node.sethforprivacy.com:443

# Basic auth credentials for the internal RPC connection
# between Pago and the wallet-rpc container.
RPC_USER=xmr
RPC_PASS=xmr
```

### Step 4: Start Pago

```bash
docker compose up -d --build && docker compose logs -f pago
```

This will:
1.  Build the Pago Go binary inside a Docker container.
2.  Start the `monero-wallet-rpc` container, loading your wallet file.
3.  Start the Pago backend on port `8080`.
4.  Continously print Pago's logs

> *Note: On first start, the wallet needs to scan the blockchain which might take some time*

### Step 5: Verify

-   **Merchant Interface:** Open `http://localhost:8080/merchant.html`
-   **Customer Terminal:** Open `http://localhost:8080/` in a separate browser window or device.

---

## In-Person POS Setup

Pago provides two browser-based interfaces designed to be opened on **separate screens** (e.g. a cashier tablet and a customer-facing display):

### Merchant Interface (`/merchant.html`)

This is the cashier's view. It allows the merchant to:
-   Enter a **fiat amount**, an optional **Order ID** and **Description**.
-   Toggle whether to **prompt the customer for a tip**.
-   Click **Create Invoice** to generate a Monero payment request.
-   See a **live status text** that polls the latest invoice every 3 seconds, showing a color-coded label (e.g. `Pending`, `In Mempool`, `Confirming`, `Confirmed`, `Expired`). Polling stops automatically once the invoice reaches a terminal state.

<img src="https://i.imgur.com/hSDObQG.jpeg" width="400">

#### Terminal Settings

Click the **gear icon** in the top-right corner of the widget to open the **Terminal Settings** modal:

| Setting | Purpose |
|---|---|
| **Device Name (Terminal ID)** | A unique identifier for this register (e.g. `Terminal-1` or `Bar-Register`). This is used to pair the merchant view with a specific customer display. |
| **Default Fiat Currency** | The currency to use for all invoices created from this terminal (e.g. `EUR`, `USD`, `CHF`). Supports 46 currencies. |
| **Required Confirmations** | How many blockchain confirmations to require before marking a payment as complete. `0` = instant (mempool detection), up to `10`. |
| **API Key** | The `PAGO_API_KEY` used to authenticate invoice creation requests. This must match the key configured in your `.env` file. |
| **Widget Language** | The language used on the customer-facing checkout widget. Supported: 🇬🇧 English (`en`), 🇩🇪 Deutsch (`de`), 🇮🇹 Italiano (`it`), 🇳🇴 Norsk (`no`). |
| **Mock Payment** | If mock payments are enabled in `.env`, a "Simulate Payment" appears at the bottom, which confirms the latest Invoice (USE ONLY FOR TESTING) |

<img src="https://i.imgur.com/e33CXso.jpeg" width="400">

These settings are persisted in the browser's `localStorage`.

### Customer Display (`/` aka `index.html`)

This is the customer-facing screen. It shows:
-   An idle "**Tap to Pay with Monero**" button.
-   When the customer taps it, the display checks if the merchant has created an invoice for this terminal. If yes, the checkout widget (`pago.js`) is mounted and the payment QR code is shown.

<img src="https://i.imgur.com/nR5ukvc.jpeg" width="400"> <img src="https://i.imgur.com/e4S6QUk.jpeg" width="400">


#### Pairing a Terminal

The customer display must be **paired** with the same Terminal ID as the merchant register:

1.  Hover over the **gear icon** (⚙️) in the bottom-right corner of the idle screen (Invisible unless hovered).
2.  Click it and enter the same **Terminal ID** you configured on the merchant interface (e.g. `Terminal-1`).
3.  The pairing indicator at the top of the screen will update to show `Paired with: Terminal-1`.

The pairing is stored in `localStorage`, so it persists across page reloads.

#### How the Flow Works

1.  **Merchant** enters an amount and clicks "Create Invoice" on `merchant.html`.
2.  **Customer** taps the "Pay with Monero" button on `index.html`.
3.  The customer display fetches the active invoice for the paired terminal via `GET /api/terminal/active?terminal_id=Terminal-1`.
4.  If an invoice exists, the `pago.js` widget is mounted and shows the QR code, amount, and real-time status.
5.  Once the payment is confirmed, the widget shows a success screen. After 10 seconds, the customer display automatically returns to the idle screen.

---

## Online Integration (E-Commerce)

For online stores, you embed the `pago.js` widget directly into your checkout page.

### Step 1: Create an Invoice (Server-Side)

Your backend creates an invoice by sending a `POST` request to the Pago API:

```bash
curl -X POST http://your-pago-server:8080/api/create-invoice \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_secure_api_key" \
  -d '{
    "fiat_amount": 49.99,
    "fiat_currency": "USD",
    "order_id": "ORDER-12345",
    "description": "Premium Widget x1",
    "required_confs": 1,
    "callback_url": "https://mystore.com/api/payment-webhook",
    "success_url": "https://mystore.com/order/12345/success",
    "cancel_url": "https://mystore.com/order/12345/cancelled",
    "tip_enabled": false,
    "expiration_minutes": 30,
    "metadata": {
      "customer_id": "usr_abc123",
      "sku": "WIDGET-001"
    }
  }'
```

**Response:**
```json
{
  "invoice_public_id": "aB3kx9Lm",
  "address": "8...monero_subaddress...",
  "xmr_amount": 0.312845000000,
  "uri": "monero:8...?tx_amount=0.312845000000",
  "metadata": { "customer_id": "usr_abc123", "sku": "WIDGET-001" },
  "expires_at": "2026-03-14T16:30:00Z"
}
```

### Step 2: Mount the Widget (Client-Side)

Include `pago.js` on your checkout page, then mount the widget using the `invoice_public_id` from the API response:

```html
<script src="https://your-pago-server:8080/pago.js"></script>

<div id="checkout"></div>

<script>
  const widget = Pago.mount({
    container: "#checkout",
    invoiceId: "aB3kx9Lm",          // from the API response
    enableTipping: false,             // or true (but tipping is uncommon for online purchases)
    language: "en",                   // "en", "de", "it", or "no"
    apiBase: "https://your-pago-server:8080",  // if hosting on a different domain
    onSuccess: function () {
      // Called when payment is fully confirmed.
      window.location.href = "/order/12345/thank-you";
    },
  });
</script>
```

The widget handles everything automatically:
-   Displays the Monero QR code and payment address.
-   Polls `GET /api/invoice/status` every 2 seconds to detect real-time state changes.
-   Updates the UI through each stage: `pending` → `in_mempool` → `confirming` → `confirmed`.
-   Handles underpayments by updating the QR code with the remaining balance.
-   Calls `onSuccess()` when the payment is fully confirmed.

To destroy the widget programmatically (e.g., if the user navigates away):

```javascript
widget.destroy();
```

---

## API Reference

All endpoints are served from `http://your-pago-server:8080`.

### `POST /api/create-invoice`

Creates a new payment invoice. Requires the `X-API-Key` header if `PAGO_API_KEY` is configured.

#### Request Body

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `fiat_amount` | `float` | Yes | — | The price in fiat currency (e.g. `5.50`). |
| `fiat_currency` | `string` | No | `USD` | ISO 4217 currency code. See [Supported Currencies](#supported-currencies). |
| `required_confs` | `int` | No | `0` | Number of blockchain confirmations required. `0` = accept on mempool detection. |
| `order_id` | `string` | No | `""` | Your internal order identifier (e.g. `Table 4`, `ORD-123`). |
| `description` | `string` | No | `""` | Human-readable description of the order. |
| `metadata` | `object` | No | `{}` | Arbitrary JSON object. Stored and returned in status queries. |
| `callback_url` | `string` | No | `""` | URL to receive a webhook `POST` when the payment is confirmed. See [Webhook Callbacks](#webhook-callbacks-callback_url). |
| `success_url` | `string` | No | `""` | URL to redirect the customer to after a successful payment. See [Success URL](#success-url-success_url). |
| `cancel_url` | `string` | No | `""` | URL to redirect the customer to if the invoice expires. See [Cancel URL](#cancel-url-cancel_url). |
| `tip_enabled` | `bool` | No | `false` | If `true`, the widget will display a tipping prompt before showing the QR code. |
| `expiration_minutes` | `int` | No | `15` | How long the invoice stays valid before being automatically expired. |
| `tax_amount` | `float` | No | `0` | Informational tax amount included in `fiat_amount` (not added on top). |
| `terminal_id` | `string` | No | `""` | For POS use. Links the invoice to a specific terminal. When set, any previous pending invoices for this terminal are automatically voided. |

#### Response Body

| Field | Type | Description |
|---|---|---|
| `invoice_public_id` | `string` | The unique 8-character public ID of the invoice. Use this to mount the widget or query status. |
| `address` | `string` | The Monero subaddress generated for this invoice. |
| `xmr_amount` | `float` | The XMR amount the customer needs to pay. |
| `uri` | `string` | A complete `monero:` URI with the address and amount, ready for QR encoding. |
| `metadata` | `object` | The metadata you passed in the request, echoed back. |
| `expires_at` | `string` | ISO 8601 timestamp of when the invoice expires. |

---

### `GET /api/invoice/status?public_id={id}`

Polls the current status of an invoice. This is the endpoint the `pago.js` widget calls every 2 seconds.

#### Response Body

| Field | Type | Description |
|---|---|---|
| `invoice_public_id` | `string` | The invoice's public ID. |
| `order_id` | `string` | The merchant's order ID. |
| `description` | `string` | The order description. |
| `metadata` | `object` | Arbitrary metadata. |
| `fiat_amount` | `float` | The total fiat amount (including any tip). |
| `fiat_currency` | `string` | The fiat currency code. |
| `exchange_rate` | `float` | The XMR/fiat exchange rate at the time of invoice creation. |
| `address` | `string` | The Monero subaddress. |
| `status` | `string` | One of: `pending`, `in_mempool`, `confirming`, `confirmed`, `underpaid`, `expired`, `failed`. |
| `payment_cleared` | `bool` | `true` if the payment has met the required confirmations. |
| `required_confs` | `int` | Required number of confirmations. |
| `current_confs` | `int` | Current number of confirmations. |
| `xmr_amount` | `int` | Expected XMR amount in **piconeros** (1 XMR = 10¹² piconeros). |
| `amount_received` | `int` | XMR received so far in **piconeros**. |
| `created_at` | `string` | ISO 8601. |
| `expires_at` | `string` | ISO 8601. |
| `in_mempool_at` | `string` | ISO 8601. Set when the transaction is first seen in the mempool. |
| `confirmed_at` | `string` | ISO 8601. Set when the payment reaches the required number of confirmations. |
| `tax_amount` | `float` | Tax amount. |
| `tip_enabled` | `bool` | Whether tipping is enabled. |
| `success_url` | `string` | Redirect URL on success. |
| `cancel_url` | `string` | Redirect URL on expiration/cancellation. |

---

### `POST /api/invoice/tip`

Adds a tip to a pending invoice. The backend recalculates the total fiat amount and converts it to XMR at the current exchange rate.

#### Request Body

```json
{
  "invoice_public_id": "aB3kx9Lm",
  "tip_percentage": 15
}
```

#### Response

`200 OK` with `{"success": true}` on success.

---

### `GET /api/terminal/active?terminal_id={id}`

Returns the most recent active invoice for a given terminal. Used by the customer display to check if there's a pending order.

#### Response

```json
{
  "invoice_public_id": "aB3kx9Lm"
}
```

Returns `404` if no active invoice exists for the terminal.

---

### `POST /api/monero-webhook`

Internal endpoint. Called by the `notify.sh` script when `monero-wallet-rpc` detects an incoming transaction. Not meant to be called manually.

#### Request Body

```json
{
  "txid": "abc123def456..."
}
```

---

### `GET /api/config`

Returns the current server configuration flags. Currently only exposes whether mock payments are enabled.

```json
{
  "mock_payments_enabled": false
}
```

---

### `POST /api/mock-payment?public_id={id}`

**Development only.** Simulates a full payment for a pending invoice. Only available when `ENABLE_MOCK_PAYMENTS=true`.

---

## Webhook Callbacks (`callback_url`)

When you set `callback_url` on an invoice, Pago will send a **single** HTTP `POST` request to that URL the moment the payment is fully confirmed (i.e., the required number of confirmations has been reached).

This is the primary mechanism for integrating Pago into your backend, e.g. to mark an order as paid in your database, trigger shipping, or send a confirmation email.

### Webhook Payload

```json
{
  "invoice_public_id": "aB3kx9Lm",
  "order_id": "ORDER-12345",
  "status": "confirmed",
  "amount_received": 312845000000,
  "fiat_amount": 49.99,
  "fiat_currency": "USD"
}
```

| Field | Type | Description |
|---|---|---|
| `invoice_public_id` | `string` | The invoice's public ID. Use this to match against your records. |
| `order_id` | `string` | The Order ID you passed when creating the invoice. |
| `status` | `string` | Always `confirmed` when the webhook fires. |
| `amount_received` | `int` | The total XMR received in piconeros. |
| `fiat_amount` | `float` | The final fiat total (including any tip). |
| `fiat_currency` | `string` | The fiat currency code. |

### Important Notes

-   The webhook is sent **exactly once** per invoice. Pago uses a `webhook_sent` flag in the database to prevent duplicate dispatches.
-   If your server returns a non-2xx status code, Pago will log the failure but will **not** retry automatically. You should use the `/api/invoice/status` endpoint as a fallback to verify payment status.
-   The webhook is dispatched asynchronously (in a goroutine). It does not block the confirmation worker.

---

## Success URL (`success_url`)

The `success_url` is a **client-side redirect URL**. When a payment is confirmed and the `pago.js` widget detects the `confirmed` status, it can expose this URL for the frontend to redirect the customer to a "Thank You" or order confirmation page.

**Use Case:** Online e-commerce. After the customer pays, the widget picks up the `success_url` from the status response and the frontend can redirect:

```
https://mystore.com/order/12345/success
```

> **Note:** The `success_url` is a convenience for the frontend. It is **not** a server-side redirect. Your backend should always verify payment via the `callback_url` webhook or by polling `/api/invoice/status`.

---

## Cancel URL (`cancel_url`)

The `cancel_url` serves a similar purpose to `success_url`, but for the case where the invoice **expires** or is otherwise cancelled before payment is received.

**Use Case:** If the customer doesn't pay within the expiration window, the widget detects the `expired` status and the frontend can redirect the customer:

```
https://mystore.com/order/12345/cancelled
```

This allows your store to show a "Payment expired, please try again" page and optionally offer to create a new invoice.

---

## Invoice Lifecycle

Every invoice transitions through these states:

| Status | Description |
|---|---|
| `pending` | Invoice created, waiting for payment. |
| `in_mempool` | Transaction detected in the mempool (0-conf). Full amount received. |
| `confirming` | Transaction is mined but hasn't reached `required_confs` yet. |
| `confirmed` | Payment fully confirmed. Webhook is dispatched. |
| `underpaid` | A transaction was received, but the amount is less than the total owed. The widget updates to show the remaining balance. |
| `expired` | The invoice wasn't paid within `expiration_minutes`. Automatically set by the expiration worker. |
| `failed` | Manually voided via the Admin CLI, or a double-spend was detected. |

---

## Database Schema

Pago uses a single `invoices` table in SQLite:

| Column | Type | Default | Description |
|---|---|---|---|
| `id` | `INTEGER` | Auto-increment | Internal primary key. |
| `public_id` | `TEXT` | — | Unique 8-character base62 public identifier. |
| `order_id` | `TEXT` | `''` | Merchant's order ID. |
| `description` | `TEXT` | `''` | Human-readable description. |
| `metadata` | `TEXT` | `'{}'` | Arbitrary JSON metadata. |
| `fiat_amount` | `REAL` | — | Total fiat amount (including tip). |
| `tip_percentage` | `INTEGER` | `0` | Tip percentage applied. |
| `tip_fiat` | `REAL` | `0` | Tip amount in fiat. |
| `fiat_currency` | `TEXT` | `'USD'` | ISO 4217 currency code (stored uppercase). |
| `exchange_rate` | `REAL` | — | XMR/fiat rate at time of creation. |
| `xmr_amount` | `INTEGER` | — | Expected XMR in piconeros. |
| `amount_received` | `INTEGER` | `0` | XMR received so far in piconeros. |
| `address` | `TEXT` | — | Monero subaddress for this invoice. |
| `subaddress_index` | `INTEGER` | — | Wallet subaddress index (unique). |
| `status` | `TEXT` | — | Current invoice status. |
| `required_confs` | `INTEGER` | `0` | Required blockchain confirmations. |
| `current_confs` | `INTEGER` | `0` | Current confirmation count. |
| `txids` | `TEXT` | `''` | Comma-separated transaction IDs. |
| `callback_url` | `TEXT` | `''` | Webhook URL. |
| `created_at` | `DATETIME` | — | Timestamp of creation. |
| `expires_at` | `DATETIME` | — | Timestamp of expiration. |
| `in_mempool_at` | `DATETIME` | `NULL` | Timestamp when first seen in mempool. |
| `confirmed_at` | `DATETIME` | `NULL` | Timestamp when fully confirmed. |
| `webhook_sent` | `INTEGER` | `0` | `1` if the webhook has been dispatched. |
| `tax_amount` | `REAL` | `0` | Informational tax amount. |
| `tip_enabled` | `INTEGER` | `0` | Whether tipping is enabled (`1`/`0`). |
| `expiration_minutes` | `INTEGER` | `15` | Invoice lifetime in minutes. |
| `success_url` | `TEXT` | `''` | Client-side redirect on success. |
| `cancel_url` | `TEXT` | `''` | Client-side redirect on expiration. |
| `terminal_id` | `TEXT` | `''` | POS terminal identifier. |

---

## Admin CLI

Pago includes a built-in command-line tool for managing the database.

### List Recent Invoices

```bash
docker exec -it pago_backend ./pago_app admin list [n]
```

Shows the last `n` invoices (default: 10, max: 100) in a formatted table with date, public ID, order ID, status, and amount.

### View System Statistics

```bash
docker exec -it pago_backend ./pago_app admin stats
```

Shows total confirmed orders, total XMR revenue, and count of active/pending orders.

### Inspect an Invoice

```bash
docker exec -it pago_backend ./pago_app admin info <public_id>
```

Displays detailed information about a specific invoice: status, fiat total, XMR expected/received, confirmations, terminal ID, and timestamps.

### Search Invoices

```bash
docker exec -it pago_backend ./pago_app admin search <query>
```

Searches invoices by Order ID or Description. Uses partial matching (e.g. `search Table` finds all invoices with "Table" in the order ID or description).

### Void an Invoice

```bash
docker exec -it pago_backend ./pago_app admin void <public_id>
```

Manually cancels a `pending` or `underpaid` invoice by setting its status to `failed`. Cannot void invoices that are already `confirmed` or `in_mempool`.

### Delete an Invoice

```bash
docker exec -it pago_backend ./pago_app admin delete <public_id>
```

Permanently removes an invoice from the database. Only `failed` or `expired` invoices can be deleted.

---

## Background Workers

Pago runs three background workers automatically:

| Worker | Interval | Purpose |
|---|---|---|
| **Price Oracle** | 10 minutes | Fetches the latest XMR exchange rates for 46+ fiat currencies from CoinGecko and caches them in memory. Includes safety checks: rejects prices below $1.00 and price drops exceeding 50%. |
| **Expiration Worker** | 1 minute | Scans for `pending` invoices past their `expires_at` timestamp and marks them as `expired`. |
| **Confirmation Worker** | 2 seconds | Polls `monero-wallet-rpc` for confirmation progress on all invoices with known transaction IDs. Updates confirmation counts, detects underpayments, handles reorgs and double-spends, and dispatches webhooks when payments are fully confirmed. |

---

## Supported Currencies

Pago supports the following 46 fiat currencies via the CoinGecko API:

`AED`, `ARS`, `AUD`, `BDT`, `BHD`, `BMD`, `BRL`, `CAD`, `CHF`, `CLP`, `CNY`, `CZK`, `DKK`, `EUR`, `GBP`, `GEL`, `HKD`, `HUF`, `IDR`, `ILS`, `INR`, `JPY`, `KRW`, `KWD`, `LKR`, `MMK`, `MXN`, `MYR`, `NGN`, `NOK`, `NZD`, `PHP`, `PKR`, `PLN`, `RUB`, `SAR`, `SEK`, `SGD`, `THB`, `TRY`, `TWD`, `UAH`, `USD`, `VEF`, `VND`, `ZAR`

---

## Transaction Notification Script (`notify.sh`)

Pago includes a `notify.sh` script in the project root that enables real-time payment detection. When `monero-wallet-rpc` receives an incoming transaction, it executes this script, which immediately posts the transaction ID to Pago's internal webhook endpoint.

The script is already configured to work inside Docker, using the `pago` service name for inter-container communication:

```bash
#!/bin/sh
# $1 represents the %s (txid) passed by the wallet
curl -s -X POST http://pago:8080/api/monero-webhook \
  -H "Content-Type: application/json" \
  -d "{\"txid\":\"$1\"}"
```

The `docker-compose.yml` already mounts this script into the `wallet-rpc` container and passes the `--tx-notify` flag automatically:

```yaml
wallet-rpc:
  volumes:
    - ./notify.sh:/home/monero/notify.sh
  command:
    - "--tx-notify=/home/monero/notify.sh %s"
```

> **Note:** The script uses `#!/bin/sh` (not `#!/bin/bash`) because the wallet-rpc container runs Alpine Linux. The file must have execute permissions (`chmod +x notify.sh`), which are preserved by Git automatically.

---

## Security Considerations

-   **Always use a view-only wallet.** Pago never needs spending keys.
-   **Set `PAGO_API_KEY`** in production. Without it, anyone can create invoices on your server.
-   **Never enable `ENABLE_MOCK_PAYMENTS` in production.** This allows anyone to simulate payments and mark invoices as paid.
-   **Verify webhooks server-side.** Never rely on `success_url` redirects to confirm payment. Always verify via the webhook callback or by polling `/api/invoice/status`.
-   **Use HTTPS** in production.

---

## Contributing

Pull requests are welcome! For major changes, please open an issue first to discuss what you would like to change. Please make sure to update tests as appropriate:

```bash
go test ./... -v
```

### Adding Additional Languages
To fully support a new language translation in Pago, you need to update three specific locations. Please use the standard [ISO 639-1 (Alpha-2)](https://en.wikipedia.org/wiki/List_of_ISO_639_language_codes) language codes (e.g 'en', 'fr').

#### 1. Merchant Settings (`/static/merchant.html`)
Add the new language to the [`setting-language`](https://github.com/dnvie/pago/blob/33b3bd9232b3c23fa6782be1b29d3a92bc3eacdc/static/merchant.html#L539-L545) dropdown so merchants can select it.
```html
<label>Widget Language</label>
            <select id="setting-language">
                <option value="en">🇬🇧 English</option>
                <option value="de">🇩🇪 Deutsch</option>
                <option value="it">🇮🇹 Italiano</option>
                <option value="no">🇳🇴 Norsk</option>
                <!-- Add here -->
                <option value="fr">🇫🇷 Français</option> 
            </select>
```

#### 2. Idle Screen (`/static/index.html`)
Update the [`idleTranslations`](https://github.com/dnvie/pago/blob/33b3bd9232b3c23fa6782be1b29d3a92bc3eacdc/static/index.html#L222-L243) object to localize the "Tap to Pay" standby screen.
```js
const idleTranslations = {
                en: {
                  ...
                },
                ...
                // Append here:
                fr: {
                    tapTo: "...",
                    payWith: "...",
                    paired: "...",
                },
                
            };
```
#### 3. Pago Widget (`/static/pago.js`)
Update the [`translations`](https://github.com/dnvie/pago/blob/33b3bd9232b3c23fa6782be1b29d3a92bc3eacdc/static/pago.js#L2-L143) object to localize the user-facing payment widget.
```js
const translations = {
    en: {
      paymentAddress: "Payment Address:",
      states: {
        awaiting: {
          ...
        },
        mempool: {
          ...
        },
        confirming: {
          ...
        },
        confirmed: {
          ...
        },
        underpayment: {
          ...
        },
        failed: {
          ...
        },
      },
    },
    ...
    // Append here
    fr: {
      paymentAddress: "...",
      states: {
        awaiting: {
          heading: "...",
          info: "...<br />{expiration_date}",
          tip_heading: "...",
          tip_other: "...",
          tip_no_tip: "...",
          tip_other_confirm: "...",
          tip_other_cancel: "...",
          tip_other_enter_amount: "...",
        },
        mempool: {
          heading: "...",
          info: "...",
        },
        confirming: {
          heading: "...",
          info: "{current_confirmations} ... {total_confirmations} ...",
        },
        confirmed: {
          heading: "...",
          info: "...<br />{confirmation_date}",
        },
        underpayment: {
          heading: "...",
          info: "... {missing_amount} XMR. ...",
        },
        failed: {
          heading: "...",
          info: "...",
          btn_back: "...",
        },
      },
    },
  };
```
#### 4. README
Update the `README` to include the added language (under `Features`, `Terminal Settings` and the comment for `Step 2: Mount the Widget (Client-Side`)

## Disclaimer

Pago has been extensively tested. However, this software is provided **"as is"**, without any warranties or guarantees of any kind.

By using Pago, you acknowledge that you are solely responsible for the operation and security of your system, infrastructure, and Monero wallet setup. The author of this project is **not liable for any damages**, including but not limited to:

* financial losses
* lost funds or incorrect payments
* downtime or service interruptions
* data loss
* security incidents

Always test your setup thoroughly (preferably using Monero **stagenet** or the built-in **mock payment mode**) before using Pago in a production environment.

If you are unsure about any part of the setup, do not deploy it in a live payment environment.

## Donate
If you find Pago useful and want to support development:
Monero (XMR): `82g5ZWQcE9se3b1SnxaWMqj1uAqstvpBeRrPiYYPrHXwMh8zx15bC7Vi8398c5ZhtVcoz9sJqgHSb6Bvok1wovYXQ5UaRh5`

---

>*For Transparency: This `README` was formatted and structured with the help of Claude Opus 4.6*
