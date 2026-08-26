# LocalNet

A hand-rolled, authoritative DNS server written in Go from scratch — no DNS libraries, raw UDP packet parsing and construction using `encoding/binary`. Built as a learning project to understand the DNS wire protocol at the byte level, while also serving as a working local resolver for Docker containers and self-built apps.

## What it does

- Listens for DNS queries over UDP
- Parses the raw DNS header and question section by hand (length-prefixed label decoding, big-endian field parsing)
- Resolves names found in a local JSON config file with a hand-built A-record response
- Forwards any unresolved query upstream to `8.8.8.8` and relays the response back unmodified
- Handles each incoming query in its own goroutine, so a slow upstream lookup for one query never blocks or delays others

## How it works

DNS messages have a fixed structure: a 12-byte header, followed by a Question section, then Answer/Authority/Additional sections. LocalNet reads and writes this format directly:

- **Header** — 6 fixed 2-byte fields (ID, flags, record counts), parsed/written with `encoding/binary.BigEndian`
- **Names** — encoded as length-prefixed labels (e.g. `loky.local` → `04 6c6f6b79 05 6c6f63616c 00`), not plain dot-separated strings
- **Answer** — NAME + TYPE + CLASS + TTL + RDLENGTH + RDATA, matching the query's echoed name

For a name found in `records.json`, LocalNet builds this answer section by hand and writes it back to the client. For anything else, it opens a fresh UDP connection to `8.8.8.8:53`, forwards the original query bytes untouched, and relays whatever comes back — with a 3-second timeout so a slow/unreachable upstream never hangs the server.

## Project structure

```
LocalNet/
├── main.go       # entry point, UDP listener, request loop
├── parser.go     # DNS header/name parsing, response building
├── records.json  # name → IP map (auto-created empty on first run if missing)
└── .env          # optional — PORT, RECORDS_PATH overrides
```

## Configuration

Config is read from environment variables (via `.env`, using `github.com/joho/godotenv`), with sane defaults:

| Variable       | Default                      | Description                                                    |
| -------------- | ---------------------------- | -------------------------------------------------------------- |
| `PORT`         | `53`                         | UDP port to listen on (needs root/`sudo` for ports below 1024) |
| `RECORDS_PATH` | `/etc/LocalNet/records.json` | Path to the name→IP JSON map                                   |

### `records.json` format

```json
{
  "loky.local": "127.0.0.1",
  "api.internal": "172.18.0.5",
  "redis.internal": "172.18.0.7"
}
```

If the file doesn't exist at startup, LocalNet creates an empty one (`{}`) at the configured path automatically.

> **Note:** `.local` is conventionally reserved for mDNS and will trigger a warning from tools like `dig`. Prefer `.internal` or `.test` for your own entries to avoid any ambiguity with multicast DNS.

## Running it

```bash
go mod init localnet   # if not already initialized
go run .
```

Since the default port (53) is a privileged port on Linux, you'll need:

```bash
sudo go run .
```

To run on an unprivileged port during development instead:

```bash
PORT=5353 go run .
```

## Testing it

```bash
dig @127.0.0.1 -p 5353 loky.local     # should return your configured IP
dig @127.0.0.1 -p 5353 google.com     # should forward to 8.8.8.8 and return a real answer
```

## Using it as your system resolver

To route your machine's DNS traffic through LocalNet, point your OS resolver at `127.0.0.1`. On systems using `systemd-resolved`:

```bash
resolvectl dns <interface> 127.0.0.1
resolvectl status   # confirm 127.0.0.1 is the active DNS server
```

This is a runtime-only change and reverts on reconnect/reboot — for a persistent change, configure it at the NetworkManager connection-profile level.

**Note on browsers:** Chromium-based browsers (Chrome, Brave, Edge) often have DNS-over-HTTPS (Secure DNS) enabled by default, which bypasses the OS resolver entirely. Disable it in the browser's security settings if you want browser traffic to route through LocalNet too.

## Known limitations / not yet implemented

- Only A records are resolved locally — AAAA, CNAME, and other types always fall through to upstream forwarding
- No DNS name compression on outgoing responses (valid but slightly larger packets than necessary)
- Config is loaded once at startup — no hot-reload on file changes
- No caching of upstream responses
- Binds to a single hardcoded interface/address — not yet configurable for LAN-wide use

## Security note

LocalNet forwards arbitrary unresolved queries to a public upstream resolver. If bound to `0.0.0.0` (all interfaces) rather than `127.0.0.1`, it becomes reachable from other devices on the network — and, without care, this pattern is exactly what's known as an **open DNS resolver**, a known vector for DNS amplification abuse. Keep this bound to loopback-only for personal use unless you've deliberately added access controls (firewall rules restricting the source subnet).
