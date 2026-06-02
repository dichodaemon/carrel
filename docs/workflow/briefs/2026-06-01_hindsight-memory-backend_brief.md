---
title: Hindsight Memory Backend Integration
date: 2026-06-01
author: Dizan Vasquez
---

# Hindsight Memory Backend Integration

## 1. Objective

Wire the [Hindsight](https://hindsight.vectorize.io/) memory system into the carrel container stack so OMP gains persistent long-term memory across sessions — retaining facts, recalling context, and reflecting on past work — with a self-hosted Gemma 4 LLM for privacy and zero inference cost.

## 2. Scope

| In scope | Out of scope |
|---|---|
| Add Hindsight as a docker-compose service alongside carrel | Modifying OMP code (backends already implemented) |
| Add Ollama service serving Gemma 4 E4B | Configuring Hindsight Cloud (hosted) |
| Wire carrel container env to Hindsight API | External PostgreSQL instead of embedded pg0 |
| Validate environment in `bin/launch` | Hindsight API authentication setup |
| Model auto-pull on launch | Tuning Hindsight recall/retain/reflect parameters |
| Document the OMP settings change (`memory.backend = "hindsight"`) | Custom bank mission or directive authoring |

## 3. Sources

1. **OMP Hindsight backend** — `/workspace/oh-my-pi/packages/coding-agent/src/hindsight/` (client, config, state, bank, mental-models, backend lifecycle)
2. **OMP memory backend resolver** — `/workspace/oh-my-pi/packages/coding-agent/src/memory-backend/resolve.ts` (selects backend from `memory.backend` setting)
3. **OMP settings schema** — `/workspace/oh-my-pi/packages/coding-agent/src/config/settings-schema.ts:1522-1661` (Hindsight configuration defaults, UI metadata)
4. **Hindsight docs** — `https://hindsight.vectorize.io/developer/` (installation, configuration, models, API)
5. **Carrel docker-compose** — `/workspace/carrel/docker-compose.yml` (existing services and network)
6. **Carrel launch/terminate** — `/workspace/carrel/bin/launch`, `/workspace/carrel/bin/terminate`
7. **Hindsight Docker image** — `ghcr.io/vectorize-io/hindsight:latest` (full image with local embedding and reranker models)
8. **Ollama Gemma 4 library** — `https://ollama.com/library/gemma4` (E2B, E4B, 26B, 31B variants)
9. **Gemma 4 announcement** — Google blog, April 2026 (native function calling, structured JSON output, agentic-optimized architecture)

## 4. Approach

### 4.1. Architecture

Four-container stack on the `internal` bridge network:

| Container | Image | Role |
|---|---|---|
| `socket-proxy` | `tecnativa/docker-socket-proxy` | Docker API proxy (existing) |
| `carrel` | Built from `Dockerfile` | OMP runtime (existing) |
| `hindsight` | `ghcr.io/vectorize-io/hindsight:latest` | Memory API + embeddings + reranker |
| `ollama` | `ollama/ollama:latest` | Serves Gemma 4 for LLM tasks |

Inter-container DNS:
- `carrel` → `http://hindsight:8888` (Hindsight API)
- `hindsight` → `http://ollama:11434/v1` (Ollama OpenAI-compatible endpoint)

**Why hindsight:latest (full, ~9 GB):** The full image bundles local embedding (`bge-small-en-v1.5`, ~130 MB) and cross-encoder reranker (`ms-marco-MiniLM-L-6-v2`, ~90 MB) models — both CPU-friendly and auto-downloaded on first run. The slim variant would require configuring three separate external services (LLM + embeddings + reranker). The full image works out of the box with just an LLM endpoint.

**Why Ollama for Gemma:** Hindsight has native Ollama provider support. `HINDSIGHT_API_LLM_PROVIDER=ollama` with `HINDSIGHT_API_LLM_BASE_URL=http://ollama:11434/v1`. The chosen model is `gemma4:e4b` — Google's current-generation 4B-effective-parameter dense model, purpose-built for agentic workflows with native function calling and structured JSON output.

### 4.2. docker-compose changes

Add two services, two named volumes, and one environment variable on carrel:

```yaml
# In services: (alongside socket-proxy and carrel)
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
  networks:
    - internal
  restart: unless-stopped
ollama:
  image: ollama/ollama:latest
  container_name: ollama
  volumes:
    - ollama-data:/root/.ollama
  networks:
    - internal
  restart: unless-stopped
# At the top level:
volumes:
  hindsight-data:
  ollama-data:
```

In the existing `carrel` service environment block, add:

```yaml
    HINDSIGHT_API_URL: http://hindsight:8888
```

No host port mappings are needed — inter-container traffic uses the internal bridge network DNS.

For GPU passthrough, add to the `ollama` service if `nvidia-container-toolkit` is installed:


```yaml
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
```

### 4.3. Model choice

Gemma 4 is Google's current-generation open model family (April 2026), designed for reasoning and agentic workflows under an Apache 2.0 license. All variants support native function calling and structured JSON output.

| Variant | Parameters | Architecture | Disk (Q4) | Fits GPU? |
|---|---|---|---|---|
| `gemma4:e2b` | 2B effective | Dense, edge-optimized | ~1.6 GB | Any GPU |
| `gemma4:e4b` | 4B effective | Dense, edge-optimized | ~2.5–3 GB | GTX 1660 Ti 6 GB or better |
| `gemma4:26b` | 26B total / 3.8B active | Mixture of Experts | ~14 GB | 16 GB+ VRAM |
| `gemma4:31b` | 31B | Dense | ~18 GB | 24 GB+ VRAM |

Default: `gemma4:e4b`. Configurable via `HINDSIGHT_OLLAMA_MODEL` in `.env`. GPU lower bound: NVIDIA GeForce GTX 1660 Ti (6 GB VRAM) or equivalent. On CPU without a GPU, the E2B variant (~1.6 GB) is the practical choice; E4B on CPU will be slow.

**Structured output:** Gemma 4 has native function calling and structured JSON output — a first-class design feature, not retrofitted. This directly satisfies Hindsight's requirement for structured output in the `retain` fact-extraction pipeline.

### 4.4. bin/launch changes

1. **`.env` generation:** Add `HINDSIGHT_OLLAMA_MODEL` prompt with default `gemma4:e4b`.
2. **GPU detection:** Run `nvidia-smi` during launch. If a GPU is found and `nvidia-container-toolkit` is available, enable GPU passthrough on the ollama service. Otherwise, suggest falling back to `gemma4:e2b` for CPU inference.
3. **Post-startup model pull:** After containers are up, run `docker compose exec ollama ollama pull "$HINDSIGHT_OLLAMA_MODEL"` if the model is not already cached.
4. **Validation:** Warn if model pull fails or if Hindsight is unreachable.

### 4.5. OMP activation

After infrastructure is running, the user sets one setting in OMP:

```
memory.backend = "hindsight"
```

All other defaults are sensible: `autoRecall: true`, `autoRetain: true`, `retainMode: "full-session"`, `scoping: "per-project-tagged"`. The `HINDSIGHT_API_URL` env var on the carrel container tells the backend where the server is (overrides `hindsight.apiUrl` setting, which defaults to `http://localhost:8888`).

### 4.6. Runtime behavior

- **Session start:** `recall()` fetches relevant memories from prior sessions. Mental models (project conventions, user preferences, decisions) are loaded from the bank.
- **Every 3 turns + session end:** `retain()` stores conversation transcript into the bank. A debounced batch queue (16 items / 5 second flush) handles tool-initiated retains.
- **Mental models:** Auto-seeded on first use (`project-conventions`, `project-decisions`, `user-preferences`), refreshed every 5 minutes.
- **Bank scoping:** `per-project-tagged` — all projects share one bank (`omp-`) with `project:<cwd-basename>` tags for isolation.

## 5. Constraints

- **Total disk footprint:** ~13 GB (9 GB hindsight image, 1 GB ollama image, ~3 GB model, ~0.2 GB embedding/reranker models). One-time pull cost. Embedding and reranker models auto-download on first Hindsight startup (~5 minutes).
- **Total RAM:** ~4–5 GB under load (1.5–2 GB hindsight, 2.5–3 GB ollama + Gemma 4 E4B, plus carrel).
- **GPU:** Gemma 4 E4B fits on any NVIDIA GPU with ≥6 GB VRAM (GTX 1660 Ti or better). The model runs at near-zero latency with GPU acceleration. CPU fallback (E2B) is available for machines without a GPU.
- **Hindsight requires structured output support** for retain — Gemma 4 provides this natively.
- **bin/terminate needs no changes** — `docker compose down` stops all four containers.
- **OMP backend code is complete** — no OMP changes required. The `hindsight` backend, tools (`retain`, `recall`, `reflect`), lifecycle management, and mental models are all implemented and shipping.

## 6. Open Questions

1. **GPU availability at launch time:** `nvidia-smi` detection in `bin/launch` should handle the common case, but GPU hot-plug or driver changes while the stack is running are out of scope. Answer: detect once at launch; restart required to change GPU configuration.
2. **Should Hindsight be opt-in or always-on?** Always-on simplifies the configuration surface. Opt-in (via `HINDSIGHT_ENABLED` env var) avoids the 13 GB pull for users who don't want memory. Answer: start always-on; add an opt-out if needed.
3. **Should the `memory.backend` setting be pre-configured?** OMP settings are stored in the OMP config store and can't be set via env var. The user must toggle this manually or we could seed it in the OMP defaults. Answer: document as a manual step; investigate seeding in a follow-up.