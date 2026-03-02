# vssss - VSS Semantic Search

A single-binary CLI tool for semantic search over the [Vehicle Signal Specification (VSS)](https://github.com/COVESA/vehicle_signal_specification) catalog.

Instead of exact string matching, vssss uses OpenAI embeddings + sqlite-vec to find signals by meaning:

```bash
$ ./vssss -n 2 "is the driver wearing a seatbelt"
[{"signal":"Vehicle.Cabin.Seat.Row1.DriverSide.IsBelted","description":"Is the belt engaged."},{"signal":"Vehicle.Cabin.Seat.Row2.DriverSide.IsBelted","description":"Is the belt engaged."}]

$ ./vssss -n 1 "engine temperature" | jq -r '.[0].signal'
Vehicle.Powertrain.CombustionEngine.EngineCoolant.Temperature
```

## Features

- **Semantic search**: Find signals by meaning, not just keywords
- **Single binary**: Database embedded at compile time via `go:embed`
- **Fast**: Uses sqlite-vec for efficient vector similarity search
- **No CGO**: Pure Go + WASM (ncruces/go-sqlite3)
- **Composable**: JSON output to stdout, errors to stderr

## Quick Start

### For Users

**Install via Go:**

```bash
go install github.com/brojonat/vssss/cmd/vssss@latest
```

**Or download a pre-built binary** from [Releases](https://github.com/brojonat/vssss/releases).

**Run:**

```bash
export OPENAI_API_KEY=sk-...
vssss "battery charge level"
```

**Using an alternative OpenAI-compatible provider:**

```bash
export OPENAI_API_KEY=your-api-key
export OPENAI_BASE_URL=https://your-provider.com/v1
./vssss "battery charge level"
```

### For Developers

**Prerequisites:**
- Go 1.23+
- [sqlc](https://sqlc.dev/)
- OpenAI API key

**Build:**

```bash
# 1. Create .env file
echo "OPENAI_API_KEY=sk-..." > .env

# 2. Ensure vss.json exists (update VSS_JSON path in Makefile if needed)

# 3. Build everything
make all
```

This will:
1. Generate Go code from SQL (sqlc)
2. Create `signals.db` with embeddings (~1600 signals, ~60 seconds)
3. Build the `vssss` binary with embedded database

## Usage

```bash
# Basic search (outputs JSON to stdout)
./vssss "engine temperature"

# Limit results
./vssss -n 5 "wheel speed"

# Compose with jq
./vssss -n 1 "battery charge" | jq -r '.[0].signal'

# Help
./vssss --help
```

## Examples

| Query | Top Result |
|-------|------------|
| `"engine temperature"` | `Vehicle.Powertrain.CombustionEngine.EngineCoolant.Temperature` |
| `"how fast am I going"` | `Vehicle.Speed` |
| `"battery charge level"` | `Vehicle.Powertrain.TractionBattery.StateOfCharge.Current` |
| `"is the driver wearing a seatbelt"` | `Vehicle.Cabin.Seat.Row1.DriverSide.IsBelted` |
| `"tire pressure"` | `Vehicle.Chassis.Axle.Row1.Wheel.Left.Tire.Pressure` |

## How It Works

```
Build Time (maintainer):
┌─────────────────────────────────────────────────────────┐
│  vss.json ──► generate embeddings ──► signals.db       │
│                   (OpenAI API)            │             │
│                                     //go:embed          │
│                                           ▼             │
│  go build ────────────────────────► vssss binary       │
└─────────────────────────────────────────────────────────┘

Runtime (user):
┌─────────────────────────────────────────────────────────┐
│  $ ./vssss "query"                                      │
│       │                                                 │
│       ▼                                                 │
│  Embed query (1 API call) ──► KNN search ──► Results   │
└─────────────────────────────────────────────────────────┘
```

- **Build time**: ~1600 OpenAI API calls (one-time, done by maintainer)
- **Runtime**: 1 API call per search (done by user)

## Makefile Targets

```
$ make help

Targets:
  all             Generate DB and build binary (default)
  sqlc            Generate Go code from SQL using sqlc
  generate        Create signals.db with embeddings (requires OPENAI_API_KEY)
  build           Build vssss binary with embedded DB
  test            Run unit tests
  test-search     Build and run a test query
  clean           Remove build artifacts
  help            Show this help
```

## Project Structure

```
vssss/
├── cmd/
│   ├── generate/       # CLI to create signals.db from vss.json
│   └── vssss/          # Main search CLI (embeds signals.db)
├── internal/
│   ├── db/             # SQLite + sqlite-vec operations
│   └── search/         # OpenAI embeddings wrapper
├── embed.go            # //go:embed directive for signals.db
├── signals.db          # Generated embeddings database (13MB)
├── Makefile
└── README.md
```

## Dependencies

- [ncruces/go-sqlite3](https://github.com/ncruces/go-sqlite3) - Pure Go SQLite (WASM)
- [sqlite-vec](https://github.com/asg017/sqlite-vec) - Vector search extension
- [openai-go](https://github.com/openai/openai-go) - OpenAI API client
- [urfave/cli](https://github.com/urfave/cli) - CLI framework

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `OPENAI_API_KEY` | Yes | Your OpenAI API key |
| `OPENAI_BASE_URL` | No | Custom API endpoint for OpenAI-compatible providers |

## Output Format

Results are output as JSON to stdout, sorted by similarity (best match first):

```json
[
  {"signal": "Vehicle.Powertrain.CombustionEngine.EngineCoolant.Temperature", "description": "Engine coolant temperature."},
  {"signal": "Vehicle.Powertrain.ElectricMotor.EngineCoolant.Temperature", "description": "Engine coolant temperature."}
]
```

Errors go to stderr. No news is good news on stdout.

## License

MPL-2.0 (same as VSS)
