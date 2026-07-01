---
title: OMP Configuration Reference Audit
date: 2026-06-29
author: Dizan Vasquez
---

# OMP Configuration Reference Audit

## 1. Objective

The reference document at `docs/references/omp-configuration_reference.md` is incomplete and contains errors. It must describe every extension point in vanilla OMP using OMP's own terminology, but it currently misses several mechanisms, uses inconsistent names, and makes at least two claims contradicted by internal documentation.

## 2. Scope

| In scope | Out of scope |
|---|---|
| Audit `docs/references/omp-configuration_reference.md` against OMP internal docs | Fixing the reference document |
| Identify missing extension points | Carrel concepts, carrel deployment |
| Identify naming mismatches with internal OMP terminology | OMP settings schema enumeration |
| Identify claims contradicted by internal docs | OMP features that have no documentation at all |

## 3. Sources

1. Target: `docs/references/omp-configuration_reference.md` (669 lines, 12 capability sections)
2. Internal OMP docs under `omp://`:
   - `omp://slash-command-internals.md` — command resolution pipeline
   - `omp://custom-tools.md` — custom tool module contract
   - `omp://system-prompt-customization.md` — SYSTEM.md/APPEND_SYSTEM.md behavior
   - `omp://context-files.md` — context file discovery, shadowing, injection
   - `omp://task-agent-discovery.md` — agent definition fields, discovery
   - `omp://extension-loading.md` — extension discovery and loading
   - `omp://skills/authoring-extensions.md` — extension API surface
   - `omp://marketplace.md` — marketplace/plugin system
   - `omp://mcp-config.md`, `omp://mcp-protocol-transports.md` — MCP servers
   - `omp://rpc.md` — RPC mode
   - `omp://memory.md` — memory backend

## 4. Approach

Cross-reference of the reference document against OMP internal docs surfaced four categories of issues.

**Missing extension points (10).** Extension-registered tools (`pi.registerTool()`), extension-registered commands (`pi.registerCommand()`), MCP servers, plugins and marketplace, RPC mode, memory backend, four agent frontmatter fields (`spawns`, `blocking`, `autoloadSkills`, `readSummarize`), two thinking level values (`off`/`xhigh` missing, `medium` canonical vs `med` documented), the `omp-plugins` provider, and LSP configuration are all absent from the reference.

**Naming mismatches (5).** The filename `omp-configuration_reference.md` does not match the document's heading "OMP Configuration and Extensibility Reference" — it should be `omp-configuration-and-extensibility_reference.md`. §2.3 is titled "Slash Commands" (umbrella) but documents only file-based commands. §2.8 adds "(Task Subagents)" — not internal terminology. §2.9 labels hooks "Legacy" — internal docs do not. §2.10 uses "Prompts" where internal docs use "Prompt templates."

**Invocation/mechanism conflation.** The reference uses "slash command" to mean both the `/` invocation syntax and one specific mechanism (file-based template expansion). At least four distinct things sit behind `/` in OMP: built-in handlers (intercepted pre-pipeline), extension-registered commands (`pi.registerCommand`), file-based commands (template expansion), and prompt templates. Only the third is documented. The section structure does not distinguish invocation from mechanism.

**Incorrect claims (3).** §2.12 states SYSTEM.md is a "full replacement" retaining skills and rules; internal docs say it replaces only block 0, and generated skills/rules are explicitly not retained. §2.12 states SYSTEM.md triggers `custom-system-prompt.md`; internal docs say that template is "not the normal CLI SYSTEM.md path." §2.6 states the native context file provider walks up "to $HOME if not in a git repo"; internal docs mention $HOME fallback only for the `agents-md` provider.

Resolution order:

1. Confirm the two errors and one warning with the user.
2. Determine which missing extension points belong in the reference per its stated scope.
3. Document each included point using OMP internal terminology.
4. Align section titles with OMP canonical names.
5. Verify every corrected claim against its cited internal doc.

## 5. Constraints

- The reference must use OMP terminology, not carrel terminology.
- The reference must not duplicate internal docs — it indexes discovery paths and points to authoritative documentation.
- The reference currently interleaves "Writing effective X" guidance with factual discovery documentation.

## 6. Open Questions

- Should the reference enumerate the settings schema, or only document configuration files and their layering?
- Should MCP and LSP configuration live in this reference or in separate documents?
- Does the reference need a section distinguishing the three invocation mechanisms that share names with configuration directories (command, tool)?
