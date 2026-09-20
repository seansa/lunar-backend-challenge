# Lunar rockets

Service that consumes the rockets message stream and exposes the state of every
rocket through a REST API.

## Run

```bash
docker compose up -d        # MySQL 8 + migrations
go run ./cmd/api            # API on :8088
```

Configuration comes from the environment (HTTP port, MySQL connection and consumer
pool settings; `internal/config` has the full list). The defaults match
`docker-compose.yml`, so the service starts with no extra setup.

Then run the test program from the challenge against it:

```bash
./rockets launch "http://localhost:8088/messages" --message-delay=500ms --concurrency-level=1
```

## API

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/messages` | Ingests a message through the consumer pool. Returns `202 Accepted`. |
| `POST` | `/events` | Synchronous variant of `/messages`. Returns `200 OK`. |
| `GET` | `/rockets` | All rockets, sorted (`?sort=id\|type\|mission\|status`, `?order=asc\|desc`). |
| `GET` | `/rockets/:channel` | Current state of one rocket, `404` if the channel is unknown. |
| `GET` | `/rockets/:channel/events` | The stored event log of a channel. |

Both ingestion endpoints validate, normalise and store the message the same way;
they only differ in how the work is executed. They answer with the same body:
`channel`, `messageNumber`, `messageType`, `duplicate` and `applied`. The
`applied` flag is `false` when the message was already folded or when the stream
in front of it still has a gap.

```bash
curl -s http://localhost:8088/messages -H 'Content-Type: application/json' -d '{
  "metadata": {"channel": "193270a9-c9cf-404a-8f83-838e71d9ae67", "messageNumber": 1,
               "messageTime": "2022-02-02T19:39:05.86337+01:00", "messageType": "RocketLaunched"},
  "message": {"type": "Falcon-9", "launchSpeed": 500, "mission": "ARTEMIS"}
}'

curl -s 'http://localhost:8088/rockets?sort=mission&order=desc'
```

## Design

The state is a read model built from an immutable event log (Event Sourcing):

- `events`, keyed by `(channel, message_number)`, is the source of truth. Every
  message is appended once; a redelivery hits the primary key and is reported as a
  duplicate instead of failing.
- `rockets` is a projection: the state of a channel is the fold of its events in
  `messageNumber` order. A message that arrives early is stored and folded as soon
  as the gap in front of it is filled; when a message is not contiguous, the
  projection is rebuilt from the log, so no state is ever guessed. Reads are a
  single row lookup.
- Folding is serialised per channel (`pkg/lock`), so the read-modify-write of one
  rocket can never be interleaved with another message of the same channel.
- The endpoints answer `2xx` only once the message is durable. Anything else is a
  retryable status code and the sender redelivers; deduplication by
  `(channel, messageNumber)` plus a projection that only moves forward makes those
  retries harmless.
- Unknown message types and payloads that do not decode are skipped and logged, but
  still consumed: a tolerant reader keeps one bad message from blocking a channel.
  The raw payload stays in the log for later reprocessing.
- The pool bounds concurrency and back-pressures the handlers, so the API keeps
  answering while messages are being processed.

## Trade-offs and limitations

- The per-channel lock is in-process. More than one instance would need a database
  level guard, e.g. `UPDATE ... WHERE last_message_number < ?` or a row lock.
- The event log has no retention and the read endpoints have no pagination. Fine at
  this scale; a snapshot or compaction step would be the next step.
- A repeated `(channel, messageNumber)` is always treated as a redelivery, the
  payload is not compared.
- A skipped message is consumed but not applied: its effect is missing from the
  projection until a rebuild reads the log again (any out-of-order message triggers
  one). There is no explicit marker for it, so visibility is the `event skipped`
  warning and the raw payload in `GET /rockets/:channel/events`.
- No authentication, no rate limiting.

## Tests

```bash
go test ./...
go generate ./...   # regenerates internal/mocks (mockgen, see the tool directive in go.mod)
```

Unit tests cover the fold and the ingestion/projection logic; the interfaces used
by the tests are mocked with `go:generate`.

`lunar-rocket-challenge.postman_collection` is the set of requests used to drive
the API by hand: a `Read` folder with the three GET endpoints on `:8088` (the list
one with `?sort=mission&order=desc`, the other two carrying a `channel` variable)
and a `Write` folder that posts a `RocketLaunched` to `/messages` and to `/events`.
The bodies of the other four message types are saved as response examples on those
two requests. The collection has no test scripts, so it documents the calls rather
than asserting on the responses.
