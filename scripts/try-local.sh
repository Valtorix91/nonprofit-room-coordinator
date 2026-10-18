#!/bin/sh
set -eu

curl --fail-with-body --request POST http://127.0.0.1:8080/matchmaking \
  --header 'Content-Type: application/json' \
  --data '{"campaign_id":"food-drive","players":[{"id":"donor-1","role":"donor","display_name":"Kai"},{"id":"vol-2","role":"volunteer","display_name":"Rae"}],"receipt_ids":["receipt-8"],"reminder_ids":["reminder-3","reminder-4"],"report_period":"2026-Q3","idempotency_key":"match-food-drive-17"}'
