#!/bin/sh
# $1 represents the %s (txid) passed by the wallet
curl -s -X POST http://pago:8080/api/monero-webhook -H "Content-Type: application/json" -d "{\"txid\":\"$1\"}"
