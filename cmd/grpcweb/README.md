# grpcweb — tiny HTML tester for hiddify-core

Browser UI and **Core gRPC on the same TCP port** (HTTP/2 cleartext / h2c). Shows **CoreInfoListener** states and **OutboundsInfo** as JSON, and can Start / Stop / SelectOutbound / UrlTestActive.

One config box accepts a **file path**, **subscription URL**, **JSON**, or **share links**. URLs are fetched with `HiddifyNext/5.0.0 (…)` User-Agent; content is run through **Core.Parse** (ray2sing). **Preview / Parse** (and Start) open a dialog with raw + parsed JSON.

Use this to confirm `STARTING → WAITING → STARTED` without the Flutter app (which may stay on “Connecting” if it never sees `WAITING`).

## Run

From `hiddify-core` (via `cmd/main`, same tags as `cmd.sh`):

```bash
./cmd.sh grpcweb \
  --config "$HOME/Library/Application Support/apple.hiddify.com/configs/<profile>.json"
```

Or:

```bash
go run -tags "$TAGS" ./cmd/main grpcweb --config /path/to/config.json
```

Open [http://127.0.0.1:18088](http://127.0.0.1:18088) — gRPC clients dial the same `127.0.0.1:18088`.

In **Cursor**: Command Palette → **Simple Browser: Show** → paste `http://127.0.0.1:18088`.

| Flag | Default | Meaning |
|------|---------|---------|
| `--listen` | `0.0.0.0:18088` | One port for Core gRPC + HTTP UI |
| `--config` | | Prefills the config path in the UI |
| `--workdir` | temp dir | Core data dir |
| `--connect` | false | Attach only; UI on `--listen`, dial `--grpc` |
| `--grpc` | | Remote Core address (required with `--connect`) |
| `--mixed-port` | `12334` | Local mixed inbound |

## What to check

1. Click **Start** → CoreInfo JSON should show `STARTING` then `WAITING` (core up), then `STARTED` when a ping succeeds.
2. Proxies JSON should list groups; **Select** changes `selected`.
3. If this UI reaches `WAITING`/`STARTED` but the Mac app stays on Connecting, the bug is in the Flutter status mapping, not the core start path.
