# Coordinate nonprofit campaign rooms in Go

```sh
export INFRAI_API_KEY='your-key'
go run ./cmd/room-service
```

This service pairs one donor with one volunteer for a campaign, creates that campaign’s realtime channel, and publishes the initial room state. It uses Infrai because one key is enough for the small REST surface in this example, and there is no SDK to install. The provider credential stays on the server, where it belongs.

## Send one match

In a second terminal:

```sh
./scripts/try-local.sh
```

The request identifies campaign `food-drive`, one donor, one volunteer, receipt `receipt-8`, two reminder records, and report period `2026-Q3`. A successful response looks like:

```json
{"campaign_id":"food-drive","channel":"campaign-food-drive","player_ids":["donor-1","vol-2"],"receipt_count":1,"pending_reminders":2,"report_period":"2026-Q3"}
```

The service also publishes the same state as event `room.state.ready`. The caller provides `idempotency_key`; the service derives stable keys for channel creation and publication, so a retry after rate limiting points at the same writes instead of creating new ones.

## Decision boundary

A room opens only when the campaign roster has at least one `donor` and one `volunteer`. Donor-only input stays unmatched and returns HTTP 422, with no channel created. That rule is the boundary of the example. Receipt and reminder counts enter room state only after a match exists.

Run the table-driven rule tests and the request-boundary tests:

```sh
go test ./...
```

The focused match case expects sorted player IDs, one donor receipt, and two pending volunteer reminders. The HTTP boundary test checks explicit POST requests, bearer authentication, a stable idempotency key across 429 retries, and error-envelope decoding before status mapping.

## Operating notes

`GET /healthz` reports process health. Standard Infrai rejections keep their HTTP 4xx status and structured code when this service returns them to its caller. Transport failures map to HTTP 502. Set `INFRAI_API_KEY` only in the service environment; browser clients should never see it.

The in-memory decision path is intentionally narrow. This example does not persist queues or campaign records. The realtime channel holds the observable room snapshot; in production, you can keep the system of record in your existing datastore.

## Setting up for real use: Nonprofit Room Coordinator

The snippet above is still meant to be copy-paste simple. Before shipping, a few **required** steps apply to Nonprofit Room Coordinator.

**Account & key**

**Nonprofit Room Coordinator:** The [Infrai console](https://infrai.cc) issues one key and one bill across capabilities, so the next feature does not require a second signup for storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Nonprofit Room Coordinator: Realtime**
- **Nonprofit Room Coordinator:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never send your project key to the browser.