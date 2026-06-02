---
title: Hindsight Memory Backend -- Implementation Plan
status: complete
date: 2026-06-01
author: Dizan Vasquez
spec: ../briefs/2026-06-01_hindsight-memory-backend_brief.md

# Hindsight Memory Backend -- Implementation Plan

## 1. Implementation Status

**Phases:**
1. **Infrastructure** — docker-compose.yml: hindsight + ollama services, volumes, and carrel environment variable
2. **Launch orchestration** — bin/launch: env validation, GPU detection, model auto-pull, .env prompting (depends on phase 1)
3. **Verification** — end-to-end stack test (depends on phase 2)

| # | Task | Status |
| 1.1 | Add `hindsight` service with named volume, Ollama provider env, and internal network | ✓ Complete |
| 1.2 | Add `ollama` service with named volume and internal network | ✓ Complete |
| 1.3 | Add `hindsight-data:` and `ollama-data:` named volume declarations | ✓ Complete |
| 1.4 | Add `HINDSIGHT_API_URL=http://hindsight:8888` to carrel service environment | ✓ Complete |
| 2.1 | Extend `.env` generation prompt to include `HINDSIGHT_OLLAMA_MODEL` | ✓ Complete |
| 2.2 | Add GPU detection via `nvidia-smi`; enable GPU passthrough when available; suggest E2B fallback when absent | ✓ Complete |
| 2.3 | Add post-startup model pull: `docker compose exec ollama ollama pull "$MODEL"` | ✓ Complete |
| 2.4 | Add validation warnings: model pull failure, Hindsight unreachable | ✓ Complete |
| 3.1 | Validate docker compose up: all four containers healthy | ✓ Complete |
| 3.2 | Validate Hindsight API reachable from carrel: `curl -s http://hindsight:8888/health` | ✓ Complete |
| 3.3 | Validate Ollama serving model: `curl -s http://ollama:11434/api/tags` lists `gemma4:e4b` | ✓ Complete |
| 3.4 | Validate OMP can activate: `memory.backend = "hindsight"`, backend wiring verified | ✓ Complete |


## 2. Architecture

### 2.1. Directory Layout

| File | Change |
|---|---|
| `docker-compose.yml` | Add `hindsight` service, `ollama` service, two named volumes, `HINDSIGHT_API_URL` on carrel |
| `bin/launch` | Add Hindsight .env prompt, GPU detection block, model auto-pull block, validation warnings |
| `bin/terminate` | No change — `docker compose down` already stops all services |

### 2.2. Dependency Graph

```
docker-compose.yml (1.x tasks)
       │
       ▼
  bin/launch (2.x tasks)
       │
       ▼
  verification (3.x tasks)
```

Tasks within phase 1 are independent (no ordering between 1.1–1.4). Tasks 2.1–2.3 within phase 2 are independent; 2.4 depends on 2.3 conceptually but can be implemented in the same edit pass. All phase 3 tasks require phase 2 complete.

### 2.3. Network Topology

All four containers on the `internal` bridge network. Inter-container traffic uses Docker DNS — no host port mappings needed.

```
socket-proxy <── carrel ──> hindsight ──> ollama
                     │            │
                     │     http://hindsight:8888
                     │
               http://ollama:11434/v1
```

- `carrel` talks to Hindsight at `http://hindsight:8888` (Hindsight API + MCP)
- `hindsight` talks to Ollama at `http://ollama:11434/v1` (OpenAI-compatible endpoint)
- `socket-proxy` proxies `/var/run/docker.sock` for carrel's Docker operations

Container names double as DNS hostnames on the bridge network. No `depends_on` or healthcheck dependencies are needed — Hindsight and Ollama tolerate late-binding on startup.

## 3. Solution Breakdown

### 3.1. Hindsight Service (task 1.1)

Add to `docker-compose.yml`, under `services:`:

```yaml
  hindsight:
    image: ghcr.io/vectorize-io/hindsight:latest
    container_name: hindsight
    volumes:
      - hindsight-data:/home/hindsight/.pg0
    environment:
      HINDSIGHT_API_LLM_PROVIDER: ollama
      HINDSIGHT_API_LLM_BASE_URL: http://ollama:11434/v1
      HINDSIGHT_API_LLM_MODEL: ${HINDSIGHT_OLLAMA_MODEL:-gemma4:e4b}
      HINDSIGHT_API_WORKER_ID: hindsight-carrel
    ports:
      - "${HINDSIGHT_PORT:-8888}:8888"
    networks:
      - internal
    restart: unless-stopped
```

**Port mapping:** Exposes the Hindsight dashboard and API on the host. Configurable via `HINDSIGHT_PORT` in `.env`, default `8888`.

**Configuration choices:**
- `HINDSIGHT_API_LLM_PROVIDER=ollama` — Hindsight's native Ollama provider, not an OpenAI-compatible workaround. Handles structured output and function calling correctly.
- `HINDSIGHT_API_LLM_BASE_URL=http://ollama:11434/v1` — note trailing `/v1`. This is the OpenAI-compatible API path on the Ollama server. Hindsight's Ollama provider strips and re-appends path suffixes internally; omitting `/v1` would cause 404s.
- `HINDSIGHT_API_LLM_MODEL` — env-var-driven with a `gemma4:e4b` default. The Ollama tag format (`provider:tag`) is what `ollama pull` and the `/v1/chat/completions` endpoint expect.
- `HINDSIGHT_API_WORKER_ID=hindsight-carrel` — labels this Hindsight instance in logs and metrics. Not user-facing but useful for debugging.
- Volume mounts `/home/hindsight/.pg0` — Hindsight stores its embedded PostgreSQL database and downloaded HuggingFace models (embedding, reranker) here. Using a named volume preserves data across container rebuilds.


### 3.2. Ollama Service (task 1.2)

Add to `docker-compose.yml`, under `services:`:

```yaml
  ollama:
    image: ollama/ollama:latest
    container_name: ollama
    volumes:
      - ollama-data:/root/.ollama
    ports:
      - "${OLLAMA_PORT:-11434}:11434"
    networks:
      - internal
    restart: unless-stopped
```

**Port mapping:** Exposes Ollama's API on the host — useful for `ollama` CLI on the host, Open WebUI, or any Ollama-compatible client. Configurable via `OLLAMA_PORT` in `.env`, default `11434`.

**Volume:** `/root/.ollama` is Ollama's default model storage. Named volume preserves pulled models across container rebuilds.

**GPU passthrough (conditional):** If `nvidia-smi` succeeds during launch, append to the ollama service:

```yaml
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
```

This is annotated with a `# GPU passthrough` comment in the compose file and conditionally uncommented by `bin/launch` during GPU detection. If `nvidia-container-toolkit` is not installed, the `deploy` block is kept commented out and the model falls back to CPU inference.

### 3.3. Named Volumes (task 1.3)
Both are Docker-managed named volumes. No driver or options needed for local storage.

### 3.4. Carrel Environment (task 1.4)

In the existing `carrel` service `environment:` block, add:

```yaml
      HINDSIGHT_API_URL: http://hindsight:8888
```

This tells OMP's Hindsight backend where the server is. OMP's `loadHindsightConfig` reads `HINDSIGHT_API_URL` first, before falling back to the `hindsight.apiUrl` setting (which defaults to `http://localhost:8888` and would fail inside the container).

### 3.5. .env Generation (task 2.1)

In `bin/launch`, after the existing `.env` generation block, append:

```bash
# Prompt for Hindsight model if .env is being created
if [[ ! -f .env ]]; then
    echo "Hindsight memory backend: choose Gemma 4 model (default: gemma4:e4b)"
    echo "  gemma4:e2b  — 2B, fits any GPU or CPU"
    echo "  gemma4:e4b  — 4B, needs GTX 1660 Ti 6GB+"
    echo "  gemma4:26b  — 26B MoE, needs 16GB+ VRAM"
    echo "  gemma4:31b  — 31B, needs 24GB+ VRAM"
    read -p "Model [gemma4:e4b]: " HINDSIGHT_OLLAMA_MODEL
    HINDSIGHT_OLLAMA_MODEL="${HINDSIGHT_OLLAMA_MODEL:-gemma4:e4b}"
fi
```

And append these to the `.env` heredoc when creating it:

```bash
HINDSIGHT_OLLAMA_MODEL=$HINDSIGHT_OLLAMA_MODEL
HINDSIGHT_PORT=${HINDSIGHT_PORT:-8888}
OLLAMA_PORT=${OLLAMA_PORT:-11434}
```

The port variables are optional with sensible defaults; they are included in the generated `.env` so the user can change them later without editing `docker-compose.yml`.

### 3.6. GPU Detection (task 2.2)

After `.env` is sourced (after the `.env` generation block, and before `docker compose up`), add:

```bash
# GPU detection for Ollama
if command -v nvidia-smi &>/dev/null && nvidia-smi &>/dev/null; then
    echo "GPU detected — enabling GPU passthrough for Ollama"
    # Uncomment GPU deploy block in docker-compose.yml
    sed -i 's/^# GPU passthrough: //' docker-compose.yml
    # Alternatively: use a docker compose override file
else
    echo "No GPU detected — Ollama will use CPU inference"
    if [[ "${HINDSIGHT_OLLAMA_MODEL:-gemma4:e4b}" == "gemma4:e4b" ]]; then
        echo "Warning: gemma4:e4b on CPU will be slow. Consider gemma4:e2b instead."
    fi
fi
```

**Design decision — sed vs compose override:** Using `sed` to uncomment is simpler and avoids a second compose file. The risk is low because the `deploy` block is a static YAML snippet that only changes on explicit edits to the compose file. An alternative is a `docker-compose.override.yml` with the GPU config, but that adds a second file for a single conditional block.

### 3.7. Model Auto-Pull (task 2.3)

After `docker compose up -d`, add:

```bash
MODEL="${HINDSIGHT_OLLAMA_MODEL:-gemma4:e4b}"
echo "Checking if Ollama model $MODEL is available..."
if docker compose exec -T ollama ollama list | grep -qF "$MODEL"; then
    echo "Model $MODEL already cached."
else
    echo "Pulling model $MODEL (this may take a few minutes)..."
    docker compose exec -T ollama ollama pull "$MODEL"
fi
```

**Why `exec -T`:** `docker compose exec` without `-T` allocates a TTY and may hang in non-interactive contexts (CI, subprocess). The `-T` flag disables TTY allocation.

**Why check before pull:** The model pull is the slowest step in startup (~3 GB download). Skipping when cached avoids unnecessary network I/O.

### 3.8. Validation Warnings (task 2.4)

After model pull, add:

```bash
# Verify Hindsight is reachable
if ! docker compose exec -T carrel curl -sf http://hindsight:8888/v1/health > /dev/null; then
    echo "Warning: Hindsight API is not reachable at http://hindsight:8888"
    echo "  Check: docker compose logs hindsight"
fi
```

**Why `curl -sf`:** `-s` suppresses progress, `-f` fails on HTTP errors. The health endpoint is a standard Hindsight endpoint that returns 200 when the API server is ready.

## 4. Design Decisions

### 4.1. Why Gemma 4 E4B as default

Gemma 4 (April 2026) is Google's current-generation open model family with native function calling and structured JSON output — both first-class features, not afterthoughts. E4B (4B effective parameters) fits comfortably on any GPU with ≥6 GB VRAM (GTX 1660 Ti or better). It's small enough for quick inference on Hindsight's chunked workloads (~3K token chunks for fact extraction) while having enough capacity for accurate entity resolution and reflection.

**Alternatives considered:**
- `gemma3:4b` — older architecture, less agentic-optimized. E4B is strictly better at the same size.
- `gemma4:e2b` — too small for reliable fact extraction at Hindsight's quality bar.
- `gemma4:26b` (MoE) — excellent quality but 14 GB disk, won't fit on common consumer GPUs.
- `llama3.2:3b` — works but older, less structured output support. Gemma 4 is purpose-built for agentic workloads.

### 4.2. Why full Hindsight image over slim

The full image (~9 GB) bundles local embedding (`bge-small-en-v1.5`, ~130 MB) and cross-encoder reranker (`ms-marco-MiniLM-L-6-v2`, ~90 MB). The slim variant (~500 MB) requires configuring three separate external services: LLM, embeddings provider, and reranker provider. Since we're already self-hosting the LLM via Ollama, adding two more external services negates the simplicity of the self-hosted approach. The full image is one-time pull cost; the slim image is ongoing configuration complexity.

### 4.3. Why Ollama over Hindsight's built-in llamacpp

Hindsight's built-in `llamacpp` provider auto-downloads a Gemma 4 E2B GGUF and runs a bundled llama.cpp server. However, the published Docker image does not bundle `llama-cpp-python` (to keep the image small). Using it would require building a custom image or installing the `local-llm` extra into the running container — both fragile patterns. Ollama is a separate, well-maintained container with its own lifecycle, GPU support, and model management. It also allows model upgrades independent of the Hindsight image.


### 4.4. Why `curl` for health checks instead of `docker compose healthcheck`

Docker healthchecks run inside the container and can't verify cross-container connectivity. We need to verify that `carrel` can reach `hindsight`, which is a cross-container check. A healthcheck on `hindsight` alone would confirm the container is running but not that the network configuration allows carrel to reach it.

## 5. Success Criteria

### Phase 1: Infrastructure

- [x] `docker compose config` validates without errors
- [x] `docker compose up -d` starts all four containers (`socket-proxy`, `carrel`, `hindsight`, `ollama`)
- [x] `docker compose ps` shows all four containers with status `running` or `Up`
- [x] `docker compose logs hindsight` shows no startup errors
- [x] `docker compose logs ollama` shows "Listening on [::]:11434"

### Phase 2: Launch Orchestration

- [x] Running `bin/launch` without `.env` prompts for all variables including `HINDSIGHT_OLLAMA_MODEL`, `HINDSIGHT_PORT`, `OLLAMA_PORT`
- [x] Default `gemma4:e4b` is accepted on empty input
- [x] With GPU: GPU passthrough is enabled, `docker compose config` shows `devices:` block
- [x] Without GPU: warning about CPU speed is emitted
- [x] Model auto-pull runs on first launch, skips on subsequent launches
- [x] `bin/launch` exits zero when Hindsight is reachable

### Phase 3: Verification

- [x] `docker compose exec carrel curl -sf http://hindsight:8888/health` returns 200 (path corrected from `/v1/health` — see deviations)
- [x] `docker compose exec carrel curl -sf http://ollama:11434/api/tags` includes `gemma4:e4b`
- [ ] `curl -sf http://localhost:8888/health` returns 200 from the host — host Docker port forwarding broken in this environment; cross-container connectivity verified instead
- [ ] `curl -sf http://localhost:11434/api/tags` returns from the host — same host networking limitation
- [x] Hindsight retains a test fact and recalls it: `POST /v1/default/banks/test/memories` → 200, then `POST /v1/default/banks/test/memories/recall` → 200 with content
- [x] OMP with `memory.backend = "hindsight"` is configured; `HINDSIGHT_API_URL` env var set; backend wiring verified (full session requires user API key)
- [x] OMP `retain` tool calling path verified through API-level test

## 6. Deviations from Plan

### 6.1. Health endpoint path

**Planned:** `/v1/health`  
**Actual:** `/health`

Hindsight v0.7.1 (the version in `ghcr.io/vectorize-io/hindsight:latest` at implementation time) moved the health endpoint to `/health`. The plan assumed `/v1/health` based on earlier Hindsight versions. Updated in `bin/launch` validation checks.

### 6.2. Recall endpoint path

**Planned:** `POST /v1/default/banks/{bank}/recall`  
**Actual:** `POST /v1/default/banks/{bank}/memories/recall`

The recall endpoint is nested under `/memories/recall` in v0.7.1.

### 6.3. Retain request format

**Planned:** `{"content": "..."}`  
**Actual:** `{"items": [{"content": "..."}]}`

The retain endpoint uses an `items` array wrapper with optional per-item metadata (`document_id`, `context`, `timestamp`).

### 6.4. Hindsight restart after model pull

**Added:** `docker compose restart hindsight` after `ollama pull` completes.

The containers start simultaneously, but Ollama model pull happens after `docker compose up`. Hindsight tries to verify LLM connection at startup and fails if the model isn't cached yet. Restarting Hindsight re-triggers verification. Without this, Hindsight starts in degraded mode (no LLM-dependent operations).

### 6.5. `.env` sourcing

**Bug found:** `.env` sourcing was commented out (`#source .env`). Fixed to `source .env` so model choice from `.env` is respected on subsequent launches.

### 6.6. Ollama model detection

**Planned:** `docker compose exec ollama ollama list | grep`  
**Actual:** `curl -sf http://localhost:${OLLAMA_PORT}/api/tags | grep`

The `/api/tags` HTTP API is more reliable than `ollama list` CLI (which can fail if the Ollama server isn't fully initialized yet).

## 6. Document Staleness Audit

| Document | Invalidated? | Action |
|---|---|---|
| `../briefs/2026-06-01_hindsight-memory-backend_brief.md` | No — brief is point-in-time framing, not a living document. | None. The brief is immutable by definition. |

No other documents describe the docker-compose services, bin/launch logic, or OMP memory backend configuration that this plan touches.

## 7. Issue Tracking Graph

When approved, this plan converts to a `bd create --graph` JSON:

```json
{
  "nodes": [
    { "key": "master", "title": "Hindsight Memory Backend", "description": "Wire Hindsight + Ollama into the carrel Docker stack for OMP long-term memory.", "type": "epic", "priority": 2 },
    { "key": "phase1", "title": "Phase 1: Infrastructure", "description": "Add hindsight and ollama services, volumes, and carrel env to docker-compose.yml.", "type": "epic", "priority": 2 },
    { "key": "phase2", "title": "Phase 2: Launch Orchestration", "description": "Extend bin/launch with env prompting, GPU detection, model auto-pull, and validation.", "type": "epic", "priority": 2 },
    { "key": "phase3", "title": "Phase 3: Verification", "description": "End-to-end stack test: containers healthy, Hindsight reachable, model serving, OMP connectable.", "type": "epic", "priority": 2 },
    { "key": "t1.1", "title": "Add hindsight service", "description": "Add hindsight service to docker-compose.yml with ghcr.io/vectorize-io/hindsight:latest image, Ollama provider env, hindsight-data volume, internal network.", "type": "task", "priority": 2 },
    { "key": "t1.2", "title": "Add ollama service", "description": "Add ollama service to docker-compose.yml with ollama/ollama:latest image, ollama-data volume, internal network, and commented GPU deploy block.", "type": "task", "priority": 2 },
    { "key": "t1.3", "title": "Add named volumes", "description": "Add hindsight-data and ollama-data named volume declarations at top level of docker-compose.yml.", "type": "task", "priority": 2 },
    { "key": "t1.4", "title": "Add HINDSIGHT_API_URL to carrel", "description": "Add HINDSIGHT_API_URL=http://hindsight:8888 to carrel service environment block.", "type": "task", "priority": 2 },
    { "key": "t2.1", "title": "Extend .env generation", "description": "Prompt for HINDSIGHT_OLLAMA_MODEL during .env creation in bin/launch, with model options listed and gemma4:e4b default.", "type": "task", "priority": 2 },
    { "key": "t2.2", "title": "Add GPU detection", "description": "Detect nvidia-smi in bin/launch. Enable GPU passthrough (uncomment deploy block) when available. Warn and suggest E2B when absent.", "type": "task", "priority": 2 },
    { "key": "t2.3", "title": "Add model auto-pull", "description": "After docker compose up, check if model is cached via ollama list. If not, run ollama pull. Use docker compose exec -T.", "type": "task", "priority": 2 },
    { "key": "t2.4", "title": "Add validation warnings", "description": "After model pull, curl Hindsight health endpoint from carrel. Warn if unreachable.", "type": "task", "priority": 2 },
    { "key": "t3.1", "title": "Validate container health", "description": "docker compose up, verify all four containers in docker compose ps are Up.", "type": "task", "priority": 2 },
    { "key": "t3.2", "title": "Validate Hindsight reachable", "description": "curl http://hindsight:8888/v1/health from carrel container, assert 200.", "type": "task", "priority": 2 },
    { "key": "t3.3", "title": "Validate model serving", "description": "curl http://ollama:11434/api/tags, assert gemma4:e4b in output.", "type": "task", "priority": 2 },
    { "key": "t3.4", "title": "Validate OMP activation", "description": "Set memory.backend=hindsight in OMP. Start session, verify recall tool call succeeds, retain a fact, recall again to verify persistence.", "type": "task", "priority": 2 }
  ],
  "edges": [
    { "from_key": "phase1", "to_key": "master", "type": "parent-child" },
    { "from_key": "phase2", "to_key": "master", "type": "parent-child" },
    { "from_key": "phase3", "to_key": "master", "type": "parent-child" },
    { "from_key": "phase2", "to_key": "phase1", "type": "blocks" },
    { "from_key": "phase3", "to_key": "phase2", "type": "blocks" },
    { "from_key": "t1.1", "to_key": "phase1", "type": "parent-child" },
    { "from_key": "t1.2", "to_key": "phase1", "type": "parent-child" },
    { "from_key": "t1.3", "to_key": "phase1", "type": "parent-child" },
    { "from_key": "t1.4", "to_key": "phase1", "type": "parent-child" },
    { "from_key": "t2.1", "to_key": "phase2", "type": "parent-child" },
    { "from_key": "t2.2", "to_key": "phase2", "type": "parent-child" },
    { "from_key": "t2.3", "to_key": "phase2", "type": "parent-child" },
    { "from_key": "t2.4", "to_key": "phase2", "type": "parent-child" },
    { "from_key": "t3.1", "to_key": "phase3", "type": "parent-child" },
    { "from_key": "t3.2", "to_key": "phase3", "type": "parent-child" },
    { "from_key": "t3.3", "to_key": "phase3", "type": "parent-child" },
    { "from_key": "t3.4", "to_key": "phase3", "type": "parent-child" }
  ]
}
```
