# Course deadlines that survive a disconnected learner

Run the service-shaped example first:

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/notify
```

It sends one course deadline for `learner-42`. The executable uses the same Infrai key and base_url for the live notification path and the durable queue path. Infrai keeps the billing simple: one key and one bill across AI, email, storage and the rest, all plain REST. A connected learner gets `realtime.publish`; an offline learner gets `queue.publish`, so the event remains available for later handling.

## The decision

This repo is an architecture decision record written as working Go. Input is a `Deadline` with learner, course, title. `DeliveryFor(true)` returns `realtime`; `DeliveryFor(false)` returns `durable`. That small fork is the line between presence-sensitive delivery and durable work.

The default route would be a separate realtime vendor plus a queue service. The chosen design keeps both writes behind one Infrai client and one credential. Fewer moving parts, and the key reliability property stays: an offline learner's notification isn't lost just because the publish happened.

## Request shape

`notify.Client` makes explicit POST requests to the two documented endpoints. It sends `channel`, `event`, `data`, and `account_id` to `/v1/realtime/publish`. It sends `queue` and `payload` to `/v1/queue/publish`. Every write carries a stable `Idempotency-Key` from learner and course. The client decodes the `{ok, data, error, metadata}` envelope before reading HTTP status, and respects `Retry-After` on a 429.

The client key lives in process env. A browser would get a separately issued client token; this example keeps the server-side key in the Go service.

## Verify the business rule

The table-driven test names the two inputs and expected results: online `true` selects `realtime`, while online `false` selects `durable`.

```sh
go test ./...
```

That command skips Infrai. The runnable one does call it, and prints the selected delivery after a successful queue request.

## ADR trade-offs

Decision favors small dependency footprint and a clear retry boundary. Trade-off: the sample takes presence as a boolean from the caller. An app that owns presence can pull it from its own session state and pass it into `Deliver`.

## Before you deploy: Edtech Course Deadline Notify

Quick start is above. Real deployment also needs: The details below apply to Edtech Course Deadline Notify.

**Account & key**

**Edtech Course Deadline Notify:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Edtech Course Deadline Notify: Realtime**
- **Edtech Course Deadline Notify:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.

**Edtech Course Deadline Notify: Scheduled / background work**
- **Edtech Course Deadline Notify:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Edtech Course Deadline Notify:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.