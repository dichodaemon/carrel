---
name: create-dependency-diagram
description: >
  Create a standards-compliant dependency diagram for a package or subsystem.
  Reads BUILD files and palette.yaml to determine scope, edges, and colors.
  Produces a mermaid diagram with correct containment, border styles, palette,
  and legend. Usage: /create-dependency-diagram <package-path> --level=<L1|L2|L3>
---

# Create Dependency Diagram

Create a dependency diagram that complies with the Visual Vocabulary Standards.
The diagram is built by reading BUILD files and palette.yaml -- never from
memory, never with invented colors. Every node, edge, border style, and color
is verified against the build system and palette before the diagram is
committed.

## Trigger

User says "create dependency diagram", "draw dependency diagram", "add a
dependency diagram", or invokes `/create-dependency-diagram`.

## Arguments

- **package-path** (required): Path to the package or subsystem to diagram.
- **--level** (required): `L1`, `L2`, or `L3`.
  - `L1`: Targets within one package. Container = package. External = other
    packages (collapsed to package level).
  - `L2`: One package within its subsystem. Container = owning subsystem.
    External = other subsystems.
  - `L3`: All subsystems. No container. All nodes solid.

## Step 1: Read the standards

Read the Visual Vocabulary Standards document. Do not proceed from memory.
The standards define the structural hierarchy (L0-L3+), visual channels
(fill color, border style, arrow weight), dependency diagram rules (section
3.1), and legend requirements (section 2.5).

Key rules to internalize before proceeding:

- Fill color encodes subsystem ownership. Same subsystem = same color.
- Solid border = internal (defined within the diagram's scope).
- Dashed border = external reference (defined outside the diagram's scope).
- Arrows point from dependent to dependency (A --> B means "A depends on B").
- All arrows are thin and uniform (no weight distinction unless hardware
  boundary crossings are relevant to the dependency structure).
- Cycle edges may be highlighted in red with increased stroke width.
- One level of containment only -- never nest containers.

## Step 2: Read the palette

Look for a `palette.yaml` file in the package's parent `docs/` directory
(or the nearest ancestor `docs/` directory). Read the full file. Extract:

- `subsystems.<key>.stroke` -- border and container stroke color.
- `subsystems.<key>.node_fill` -- node fill (darker tint).
- `subsystems.<key>.container_fill` -- container background (lighter tint).
- `external.stroke`, `external.node_fill` -- for entities with no palette
  entry.

If no `palette.yaml` exists, ask the user before inventing colors. Never
proceed with made-up hex values.

## Step 3: Determine scope and classify nodes

Read the BUILD file at `package-path`. The level determines scope:

### L1 (targets within a package)

- **Scope**: One package (one BUILD file).
- **Internal nodes**: Non-test `cc_library` / `cuda_library` targets in this
  BUILD. Solid borders. Fill = owning subsystem's `node_fill`.
- **External nodes**: Dependency packages referenced in `deps` from other
  packages. Collapse all targets from the same external package into one
  node. Dashed borders. Fill = that package's owning subsystem's `node_fill`.
- **Container**: This package. Style with owning subsystem's
  `container_fill`.
- **Testonly targets**: Omit by default. Include only when `--include-test`
  is specified. When omitted, note this in the legend.

### L2 (packages within a subsystem)

- **Scope**: One subsystem containing the package being documented.
- **Identify the owning subsystem**: Match the package path against the
  palette.yaml `subsystems` keys. The longest matching prefix wins.
- **Internal nodes**: All packages within the owning subsystem that appear
  as this package's direct deps, plus the package itself. Solid borders.
  Fill = owning subsystem's `node_fill`.
- **External nodes**: Dependency packages from other subsystems. Dashed
  borders. Fill = each external package's owning subsystem's `node_fill`,
  stroke = that subsystem's `stroke`. If a subsystem has no palette entry,
  use the `external` palette entry.
- **Container**: The owning subsystem. Style with its `container_fill`.
- **Edges**: Show only edges originating from the documented package's BUILD
  deps, not all intra-subsystem edges (those belong in the subsystem's own
  diagram or arch-assessment).

### L3 (subsystems)

- **Scope**: Full system.
- **Internal nodes**: All subsystems. Solid borders. Each colored by its own
  `node_fill` / `stroke`.
- **External nodes**: None (everything is in scope at L3).
- **Container**: None.
- **Edges**: An edge from A to B exists when any target in A depends on any
  target in B. Collapse to one edge per direction. Detect bidirectional
  pairs and render with `<-->`.

## Step 4: Extract edges from BUILD files

Parse the `deps` lists from the BUILD file(s) within scope.

- Resolve Bazel labels (`//pkg:target`) to `(package, target)` pairs.
- Filter out external deps (`@...`) unless they are in scope (L3 only).
- Filter out test-only deps unless `--include-test`.
- At L1: edges between targets. At L2: edges between packages. At L3:
  edges between subsystems.
- Deduplicate: one edge per (source, destination) pair at the diagram's
  level.

## Step 5: Detect cycles (L2 and L3 only)

Run Tarjan's strongly connected components algorithm on the extracted
adjacency graph.

- If cycles exist, mark the cycle edges for red highlighting.
- At L1, skip cycle detection (cycles within a single package are
  structurally unlikely in Bazel).

## Step 6: Assemble the mermaid source

Use `flowchart TD` (not `graph TD` -- `graph` is legacy and lacks full
subgraph support).

### Node declarations

- **Subgraph IDs**: Use `<name>_sub` suffix (e.g., `behavior_sub`).
- **Subgraph labels**: The subsystem or package display name in brackets.
- **Node IDs**: Replace hyphens with underscores (mermaid restriction).
- **Multi-word labels**: Use `<br>` for line breaks in display names.
- Internal nodes go inside the subgraph. External nodes go outside.

### Edge declarations

- All edges use thin arrows (`-->`).
- Bidirectional edges (L3 only) use `<-->`.
- Do not use arrow labels. Dependency diagrams are unlabeled.

### Style declarations

For each subsystem present in the diagram, emit a `classDef`:

```
classDef <name> fill:<node_fill>,stroke:<stroke>,stroke-width:2px
```

For external reference classDefs, add dashed border:

```
classDef <name>_ext fill:<node_fill>,stroke:<stroke>,stroke-width:1px,stroke-dasharray:5 5
```

For containers:

```
style <subgraph_id> fill:<container_fill>,stroke:<stroke>,stroke-width:2px
```

For cycle edges (if any):

```
linkStyle <indices> stroke:#c62828,stroke-width:3px
```

### Mermaid technical constraints

- **`direction` inside subgraphs is unreliable.** Mermaid ignores
  `direction LR` (or any direction override) inside a subgraph when any
  of that subgraph's nodes has an edge crossing the subgraph boundary.
  Do not rely on `direction` for layout control.
- **Do not use `~~~` invisible links for ordering.** In `flowchart TD`,
  `~~~` creates rank offsets (vertical displacement), not horizontal
  ordering.
- **Use long arrows (`--->`) to control vertical rank spacing.** Each
  extra dash adds one rank of spacing. This is the reliable way to
  force vertical ordering between subgraphs.

## Step 7: Write the legend

Every diagram gets a `> [!NOTE]` block immediately after the closing
` ``` `. The legend must state:

- The diagram level (L1, L2, or L3) and what the scope is.
- What the container represents (or "no container" for L3).
- What solid vs dashed borders mean.
- The subsystem colors used, by name (e.g., "behavior (blue),
  capabilities (yellow)").
- That arrows point from dependent to dependency.
- If all deps are test-only, state this.
- If test-only targets were omitted, state this.
- If cycle edges are highlighted, explain the red styling.

**Legends must not reference internal tooling.** Do not mention
`palette.yaml`, arcane, or any generation pipeline. The legend is for
readers of the diagram -- it explains what visual encodings mean, not how
they were produced.

## Step 8: Pre-commit checklist

Before committing, verify every item. A "no" on any item means going back
to the relevant step.

- [ ] Every node corresponds to a real BUILD target (L1), package (L2), or
      subsystem (L3).
- [ ] No entity from outside the scope has a solid border.
- [ ] Every edge is traceable to a `deps` entry in a BUILD file.
- [ ] Node colors match `palette.yaml` exactly (`node_fill` and `stroke`
      from the owning subsystem's entry).
- [ ] Container uses `container_fill` (lighter tint), not `node_fill`.
- [ ] External references use their own subsystem's palette colors with
      dashed borders, not the scoped subsystem's colors.
- [ ] Entities with no palette entry use the `external` palette key's
      colors.
- [ ] `flowchart TD`, not `graph TD`.
- [ ] Legend is present, accurate, and does not reference internal tooling.
- [ ] Diagram renders without mermaid parse errors.
- [ ] Cycle edges (if any) are highlighted in red.

## Rules

- **Never invent colors.** Read `palette.yaml` before writing any
  `classDef`. If no palette exists, ask the user. Proceeding with
  made-up hex values is the most common diagram error.
- **`flowchart TD`, not `graph TD`.** `graph` is legacy mermaid syntax
  and lacks full subgraph support.
- **Container is the owning subsystem (L2) or package (L1).** Never
  use a higher-level grouping (e.g., the top-level project) as the
  container. At L3, there is no container.
- **External references use their own subsystem's colors.** Not the
  scoped subsystem's colors, not uniform gray. Gray is only for
  entities with no palette entry (the `external` key).
- **Solid = internal, dashed = external.** This is the single most
  important visual distinction. Verify it for every node before
  committing.
- **Legends must not reference internal tooling.** Do not mention
  `palette.yaml`, arcane, or any generation pipeline. The legend
  explains visual semantics to the reader, not provenance to the
  author.
- **One level of containment only.** Never nest subgraphs inside
  subgraphs.
- **Read the standards first.** Do not proceed from memory. The
  visual vocabulary standards are the normative reference.
