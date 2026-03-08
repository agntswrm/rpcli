# rpcli Implementation Plan

## Overview
Build an agent-first CLI in Go for managing RunPod infrastructure. RunPod uses a **GraphQL API** at `https://api.runpod.io/graphql` with API key auth via `?api_key=KEY` query parameter.

---

## Phase 1: Project Skeleton & Config

### 1.1 Initialize Go module and directory structure
- `go mod init github.com/agntswrm/rpcli`
- Create directory layout:
  ```
  cmd/rpcli/main.go
  internal/api/          # GraphQL client
  internal/commands/     # cobra command definitions
  internal/config/       # config file + API key management
  internal/output/       # JSON/table/YAML formatters
  pkg/client/            # public client package
  skills/rpcli/          # skill.sh + skill.json
  tests/unit/
  tests/integration/
  tests/evals/
  ```

### 1.2 CLI framework with cobra
- Root command with global flags: `--api-key`, `--output (json|table|yaml)`, `--yes`, `--dry-run`
- `rpcli version` command (embed version via ldflags)
- `rpcli config set-key` — store API key to `~/.config/rpcli/config.json`
- `rpcli config show` — display current config (mask key)

### 1.3 Config system
- File: `internal/config/config.go`
- Config file at `~/.config/rpcli/config.json`
- Environment variable override: `RUNPOD_API_KEY`
- Flag override: `--api-key`
- Priority: flag > env > config file

### 1.4 Output system
- File: `internal/output/output.go`
- JSON output (default): `encoding/json`
- Table output: `text/tabwriter`
- YAML output: `gopkg.in/yaml.v3`
- Structured error format: `{"error": {"code": "...", "message": "..."}}`

---

## Phase 2: GraphQL Client

### 2.1 Core GraphQL client
- File: `internal/api/client.go`
- HTTP POST to `https://api.runpod.io/graphql?api_key=KEY`
- Request body: `{"query": "...", "variables": {...}}`
- Generic `Execute(query string, variables map[string]any, result any) error`
- Structured error handling from GraphQL error responses
- User-Agent header: `rpcli/<version>`

### 2.2 API types
- File: `internal/api/types.go`
- Go structs for: Pod, Endpoint, Template, Volume, Registry, GPU, Secret, Billing
- Match RunPod GraphQL schema field names with JSON tags

---

## Phase 3: Read-Only Commands

### 3.1 GPU commands
- `rpcli gpu list` — query `gpuTypes` → list all GPU types with VRAM, display name
- `rpcli gpu availability` — query `gpuTypes` with `lowestPrice` availability data

### 3.2 Pod commands (read)
- `rpcli pod list` — query `myself { pods { ... } }`
- `rpcli pod get <id>` — query `pod(input: {podId: "..."}) { ... }`

### 3.3 Endpoint commands (read)
- `rpcli endpoint list` — query `myself { endpoints { ... } }`
- `rpcli endpoint get <id>` — query by endpoint ID

### 3.4 Template commands (read)
- `rpcli template list` — query `myself { podTemplates { ... } }`

### 3.5 Volume commands (read)
- `rpcli volume list` — query for network volumes

### 3.6 Registry commands (read)
- `rpcli registry list` — query container registry credentials

### 3.7 Secret commands (read)
- `rpcli secret list` — list secrets

### 3.8 Billing commands
- `rpcli billing pods` — pod billing/spend
- `rpcli billing endpoints` — endpoint billing/spend
- `rpcli billing volumes` — volume billing/spend

---

## Phase 4: Mutation Commands

All mutations support `--dry-run` (print what would happen, don't execute).
All destructive mutations require `--yes` or fail with confirmation message.

### 4.1 Pod mutations
- `rpcli pod create` — flags: `--gpu-type`, `--gpu-count`, `--image`, `--name`, `--volume-size`, `--container-disk`, `--template-id`, `--env KEY=VAL`, `--ports`
  - GraphQL: `podFindAndDeployOnDemand` mutation
- `rpcli pod update <id>` — update pod config
- `rpcli pod start <id>` — `podResume` mutation
- `rpcli pod stop <id>` — `podStop` mutation (destructive → `--yes`)
- `rpcli pod restart <id>` — stop then start
- `rpcli pod reset <id>` — reset pod
- `rpcli pod delete <id>` — `podTerminate` mutation (destructive → `--yes`)

### 4.2 Endpoint mutations
- `rpcli endpoint create` — flags: `--name`, `--template-id`, `--gpu-ids`, `--workers-min`, `--workers-max`, `--idle-timeout`
- `rpcli endpoint update <id>` — update endpoint config
- `rpcli endpoint delete <id>` — destructive → `--yes`

### 4.3 Template mutations
- `rpcli template create` — flags: `--name`, `--image`, `--docker-start-cmd`, `--env`, `--ports`, etc.
- `rpcli template update <id>`
- `rpcli template delete <id>` — destructive → `--yes`

### 4.4 Volume mutations
- `rpcli volume create` — flags: `--name`, `--size`, `--datacenter`
- `rpcli volume update <id>`
- `rpcli volume delete <id>` — destructive → `--yes`

### 4.5 Registry mutations
- `rpcli registry create` — flags: `--name`, `--url`, `--username`, `--password`
- `rpcli registry delete <id>` — destructive → `--yes`

### 4.6 Secret mutations
- `rpcli secret create` — flags: `--name`, `--value`
- `rpcli secret delete <name>` — destructive → `--yes`

---

## Phase 5: Safety Guards

- `--dry-run` on all mutations: print JSON of what would be sent, exit 0
- `--yes` required for destructive ops (stop, delete, reset): without it, print error message and exit 1
- No interactive prompts ever (agent-first)
- Structured error JSON on all failures
- Non-zero exit codes on errors

---

## Phase 6: Testing

### 6.1 Unit tests
- Command flag parsing tests
- Output formatter tests (JSON, table, YAML)
- Config loading priority tests
- Dry-run behavior tests
- Safety guard tests (--yes required)
- Located in `tests/unit/` and alongside source files

### 6.2 API contract tests
- Mock HTTP server returning expected GraphQL responses
- Validate request payloads match expected queries/mutations
- Validate response parsing

### 6.3 Integration tests (tests/integration/)
- Require `RUNPOD_API_KEY` env var
- Smoke: `rpcli version`, `rpcli gpu list`
- CRUD cycle: create pod → get → stop → delete

---

## Phase 7: CI/CD & Release

### 7.1 GitHub Actions: build + test
- `.github/workflows/ci.yml`
- Matrix: linux-amd64, darwin-amd64, darwin-arm64, windows-amd64
- Steps: checkout, setup Go, test, build

### 7.2 GitHub Actions: release
- `.github/workflows/release.yml`
- Trigger on tag push `v*`
- Build all targets, create GitHub release with artifacts

### 7.3 Skills
- `skills/rpcli/skill.sh` — wrapper script
- `skills/rpcli/skill.json` — skill metadata

---

## Phase 8: Implementation File List

Key files to create (in order):

1. `go.mod` — module init
2. `cmd/rpcli/main.go` — entrypoint
3. `internal/config/config.go` — config management
4. `internal/output/output.go` — formatters
5. `internal/api/client.go` — GraphQL client
6. `internal/api/types.go` — API types
7. `internal/commands/root.go` — root command + global flags
8. `internal/commands/version.go` — version command
9. `internal/commands/config.go` — config commands
10. `internal/commands/gpu.go` — GPU commands
11. `internal/commands/pod.go` — pod commands
12. `internal/commands/endpoint.go` — endpoint commands
13. `internal/commands/template.go` — template commands
14. `internal/commands/volume.go` — volume commands
15. `internal/commands/registry.go` — registry commands
16. `internal/commands/secret.go` — secret commands
17. `internal/commands/billing.go` — billing commands
18. `skills/rpcli/skill.sh` — skill wrapper
19. `skills/rpcli/skill.json` — skill metadata
20. `.github/workflows/ci.yml` — CI workflow
21. `.github/workflows/release.yml` — release workflow

---

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `gopkg.in/yaml.v3` — YAML output
- No other external dependencies needed (stdlib for HTTP, JSON, tabwriter)

---

## Implementation Notes

- RunPod has both **GraphQL** (`https://api.runpod.io/graphql`) and **REST** (`https://rest.runpod.io/v1/`) APIs — we use GraphQL as it covers the full surface
- Auth is via query parameter `?api_key=KEY` for GraphQL
- Key mutations: `podFindAndDeployOnDemand`, `podRentInterruptable`, `podResume`, `podStop`, `podTerminate`, `saveEndpoint`, `deleteEndpoint`
- Key queries: `myself { pods {...} endpoints {...} podTemplates {...} }`, `pod(input: {podId: "..."})`, `gpuTypes`
- GraphQL spec reference: https://graphql-spec.runpod.io/
- Default output is JSON (agent-first)
- All commands are non-interactive
- Version embedded at build time via `-ldflags "-X main.version=..."`
