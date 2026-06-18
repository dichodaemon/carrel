---
name: create-dataflow-diagram
description: >
  Create a standards-compliant data flow diagram for a package or component.
  Reads source code to determine scope, entities, dispatch order, and arrow
  semantics. Produces a mermaid diagram with correct stereotypes, palette, and
  legend. Usage: /create-dataflow-diagram <package-path> [--level=context|detail]
---

# Create Data Flow Diagram

Create a data flow diagram that complies with the Visual Vocabulary Standards.
The diagram is built by reading source code — never from memory or paraphrased
descriptions. Every node name, arrow direction, arrow weight, and output label
is verified against the actual implementation before the diagram is committed.

## Trigger

User says "create data flow diagram", "draw data flow", "add a data flow
diagram", or invokes `/create-dataflow-diagram`.

## Arguments

- **package-path** (required): Path to the package or component to diagram.
  This determines the scope boundary — what is internal vs. external.
- **--level** (optional): `context` or `detail` (default: `detail`).

## Step 1: Read the standards

Read the Visual Vocabulary Standards document. Do not proceed from memory.
The standards define entity model, stereotypes, visual channels, arrow
semantics, containment rules, and legend requirements.

## Step 2: Determine scope boundary

Read the package's BUILD file. The scope boundary is defined by what this
BUILD file contains:

- **Internal**: Every target defined in this BUILD file. These get solid
  borders.
- **External**: Every target referenced in `deps` from other packages. These
  get dashed borders.

This is the single source of truth for scope. Never include entities the
package does not reference in its BUILD file. Never give a solid border to
a component from another package.

## Step 3: Inventory entities

Read every public header (`.h`) in the package. For each exported symbol,
classify it:

| Source construct | Diagram entity | Label format |
|---|---|---|
| Class with state + methods | **Component** | `ClassName` (plain) |
| Free function or static method | **Operation** `«function»` | `FunctionName` |
| Instance method in the data flow | **Operation** `«method»` | `ClassName::MethodName` |
| `__global__` kernel function | **Operation** `«kernel»` | `KernelName` |

Rules:
- Use the actual symbol name from the code. Never invent generic names like
  `EvaluateKernel` or `Output`.
- If the package exports two kernels, show both — do not merge them into a
  generic node.
- For context-level diagrams, nodes are subsystems (not individual symbols).
  Skip this step and use the subsystem names from the palette file instead.

## Step 4: Trace call ownership

Read the implementation files (`.cc`, `.cu`) to determine **who calls what**.
This step prevents the most common diagram error: drawing direct
operation-to-operation arrows when a component dispatches both.

**Core rule: for functions, methods, and kernels, every incoming arrow must
come from the caller.** If `Initialize` calls both `BuildConfig` and
`BuildCoordinator`, the incoming arrows to both operations originate at
`Initialize` — not at each other, even if one produces data the other
consumes. The data dependency is real, but the call relationship determines
arrow routing.

For each operation identified in Step 3:
1. Find the call site in the implementation.
2. Identify the calling function or component.
3. Record: `Caller X dispatches Operation Y`.

The resulting call map determines arrow routing:
- Arrows go **from the caller to each operation** it dispatches.
- When an operation produces data that a subsequent operation consumes, and
  a third entity orchestrates both, route the data **back through the
  orchestrator**: `Operation A → Orchestrator → Operation B`. Do not draw
  `Operation A → Operation B` unless A literally invokes B in the code.

## Step 5: Classify arrows

For each data flow edge in the diagram, read the implementation to determine
its weight:

- **Thin** (`-->`): Data stays in the same memory space. Host-to-host,
  device-to-device, or a kernel launch where input/output buffers are already
  device-resident.
- **Thick** (`==>`): A hardware boundary crossing. Search for literal
  `cudaMemcpy` (or equivalent) calls in the package's implementation and
  check the direction parameter:
  - `cudaMemcpyHostToDevice` → thick arrow **into** the package or component.
  - `cudaMemcpyDeviceToHost` → thick arrow **out of** the package or component.

Rules:
- If the package does not call `cudaMemcpy` at all, all arrows are thin.
  State this explicitly in the legend.
- If the **caller** does the H2D and passes device pointers to this package,
  the arrows at the package boundary are thin — the transfer happens outside
  this package's scope.
- Thick arrows should be rare. A diagram dense with thick arrows is a flag.
- Never guess. If you cannot find a `cudaMemcpy` call in the implementation
  for a given edge, the arrow is thin.

## Step 6: Verify completeness

For every kernel, function, or method that appears as a node in the diagram,
re-read its **full signature** from the header file. Verify:

- Every input parameter is represented by an incoming arrow or is fed from the
  owning component.
- Every output parameter or return value is represented by an outgoing arrow.
- Arrow labels match the actual type names or parameter names.

If any output is missing, add it. This prevents the "forgot `argmin_timesteps`"
class of errors.

## Step 7: Determine containment

- **Detail level**: One container per subsystem. The package belongs to one
  subsystem. External references sit outside all containers.
- **Context level**: No containers. Nodes are subsystems.
- Never nest containers. One level only.

## Step 8: Resolve palette

Look for a `palette.yaml` file in the package's parent `docs/` directory (or
the nearest ancestor `docs/` directory). Read the subsystem entry to get
`stroke`, `node_fill`, and `container_fill` colors.

- Internal nodes: `fill:<node_fill>,stroke:<stroke>,stroke-width:2px`
- Containers: `fill:<container_fill>,stroke:<stroke>,stroke-width:2px`
- External references: use the external reference's own subsystem colors from
  the palette with dashed borders. If the external entity has no entry in the
  palette, use the `external` key's colors.

If no `palette.yaml` exists, ask the user before inventing colors.

## Step 9: Write the mermaid source

Assemble the diagram. Use `flowchart TD` (not `graph TD` — `graph` is
legacy and lacks full subgraph support).

Technical requirements:

- **Stereotypes**: Unicode guillemets `«»` (U+00AB, U+00BB), never `<<`/`>>`.
- **Stereotype placement**: On its own line above the name, using `<br>`:
  `"«kernel»<br>KernelName"`.
- **Methods**: `ClassName::MethodName` format.
- **Invisible sinks**: For output arrows that exit the diagram, use sink nodes
  with `classDef sink fill:#0000,stroke:#0000,color:#0000` and label `[ ]`.
- **Subgraph IDs**: Use `<name>_sub` suffix (e.g., `planning_sub`).
- **Subgraph labels**: The subsystem's display name in brackets
  (e.g., `[Planning]`).
- **`direction` inside subgraphs is unreliable.** Mermaid ignores
  `direction LR` (or any direction override) inside a subgraph when any
  of that subgraph's nodes has an edge crossing the subgraph boundary.
  The subgraph silently inherits the parent graph's direction instead.
  Since data flow diagrams almost always have cross-boundary edges, do
  not rely on `direction` for layout control. In `flowchart TD`,
  unconnected sibling nodes within a subgraph already arrange
  side-by-side, which is usually the desired behavior.
- **Do not use `~~~` invisible links for ordering.** In `flowchart TD`,
  `~~~` creates rank offsets (vertical displacement), not horizontal
  ordering.
- **Use long arrows (`--->`) to control vertical rank spacing.** When
  two edges from the same node place their targets at the same rank
  (causing peer subgraphs to align horizontally instead of vertically),
  add extra dashes to the edge that should reach a deeper rank. Each
  extra dash adds one rank of spacing. This is the reliable way to
  force vertical ordering between subgraphs.

## Step 10: Render and verify

Render the diagram to verify it parses without errors. Visually inspect the
layout for:

- Arrows pointing in the correct direction.
- Labels readable and not clipped.
- Containers enclosing the correct nodes.
- No orphaned nodes.

## Step 11: Write the legend

Every diagram gets a `> [!NOTE]` block immediately after the closing
` ``` `. The legend must state:

- What the container represents (e.g., "scoped to this package").
- What solid vs. dashed borders mean.
- What thick arrows mean (or "all arrows are thin" and why).
- Any conditional paths (e.g., "extraction kernel runs only for
  TrajectoryPoint* input").
- The stereotype meanings used in this diagram.

## Step 12: Pre-commit checklist

Before committing, verify every item. A "no" on any item means going back to
the relevant step.

- [ ] Every node name matches an actual symbol in the codebase (Step 3).
- [ ] No component from outside the package appears with a solid border
      (Step 2).
- [ ] Every incoming arrow to an operation (function/method/kernel) originates
      at the caller, not at a sibling operation (Step 4).
- [ ] Thick arrows appear only where this package's code calls `cudaMemcpy`
      with H2D/D2H (Step 5).
- [ ] Every kernel/function output is accounted for in outgoing arrows
      (Step 6).
- [ ] Diagram renders without mermaid parse errors (Step 10).
- [ ] Legend is present and accurate (Step 11).
- [ ] Palette colors match `palette.yaml` (Step 8).
