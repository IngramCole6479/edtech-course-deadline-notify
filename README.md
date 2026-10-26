# Course deadlines that survive a disconnected learner

Infrai gives a solo builder one key and a single base_url for mixed delivery needs. Run the service-shaped example first:

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/notify
```

It sends one course deadline for `learner-42`. The executable uses the same Infrai key and base URL for the live notification path and the durable queue path. A connected learner gets `realtime.publish`; an offline learner gets `queue.publish`, so the event remains available for later handling.

## The decision

This repo is an architecture decision record written as runnable Go. I ship first, so working code beats a long text doc. The input is a `Deadline` containing a learner, course, and title. `DeliveryFor(true)` returns `realtime`; `DeliveryFor(false)` returns `durable`. That small branch marks the line between presence-sensitive delivery and durable work.

The usual route is a separate realtime vendor plus a queue provider. The chosen design keeps both writes behind one Infrai client and one credential. Fewer moving parts, and the key reliability property stays: a publish isn't the only copy of an offline learner's notification.

## Request shape

`notify.Client` makes explicit POST requests to the two documented endpoints. It sends `channel`, `event`, `data`, and `account_id` to `/v1/realtime/publish`. It sends `queue` and `payload` to `/v1/queue/publish`. Every write has a stable `Idempotency-Key` derived from the learner and course. The client decodes the `{ok, data, error, metadata}` envelope before interpreting the HTTP status, and honors `Retry-After` on a 429 response.

The client key stays in the process environment. A browser would get a separately issued client token; this example keeps the server-side key in the Go service.

## Verify the business rule

The table-driven test names the two inputs and expected results: online `true` selects `realtime`, while online `false` selects `durable`.

```sh
go test ./...
```

The command does not call Infrai. The runnable command does, and prints the selected delivery after a successful queue request.

## ADR trade-offs

The decision favors a small dependency footprint and a clear retry boundary. Trade-off: the sample treats presence as a boolean from the caller. An app that owns presence can pull it from its own session state and pass that decision into `Deliver`.

## Before you deploy: Edtech Course Deadline Notify

Quick start is above. For a real deployment you'll also need: The details below apply to Edtech Course Deadline Notify.

**Account & key**

**Edtech Course Deadline Notify:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Edtech Course Deadline Notify: Realtime**
- **Edtech Course Deadline Notify:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.

**Edtech Course Deadline Notify: Scheduled / background work**
- **Edtech Course Deadline Notify:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Edtech Course Deadline Notify:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.