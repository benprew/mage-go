---
name: implement-set
description: "Implement an entire Magic: The Gathering set (or resume implementation of a partially-complete set). Fetches card data, generates stubs, analyzes cards, batches them by complexity and shared mechanics, then implements each batch using /implement-card. Use when the user wants to implement a full set or continue work on one."
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent, AskUserQuestion, Skill
argument-hint: <set-code> [set-name] [package-name]
---

# Set Implementation Skill

You are implementing an entire Magic: The Gathering expansion set for the mage-go engine.

## Arguments

- `$0` — Scryfall set code (e.g., `DRK`, `FEM`, `ICE`). **Required.**
- `$1` — Human-readable set name (e.g., `"The Dark"`, `"Fallen Empires"`). If omitted, look it up from the fetched JSON or ask the user.
- `$2` — Go package name (e.g., `thedark`, `fallenempires`). If omitted, derive from the set name (lowercase, no spaces).

## Step 1: Fetch Card Data

Resolve the set code. If `$0` was provided as a Scryfall set code (2-3 uppercase letters), use it directly. If it looks like a package name, read `data/sets.txt` and look up the package name to get the set code. The file format is:

```
package-name SET_CODE "Set Name"
```

Check if the JSON already exists in `data/`:

```bash
ls data/$SET_CODE.json 2>/dev/null
```

If not, fetch it (set `FETCHSET_SKIP_TLS=1` to work around sandbox TLS restrictions):

```bash
FETCHSET_SKIP_TLS=1 go run ./cmd/fetchset -o data/$SET_CODE.json $SET_CODE
```

## Step 2: Assess Existing State

Check if the card package directory already exists:

```bash
ls cards/$PACKAGE_NAME/ 2>/dev/null
```

There are three possible states:

### State A — No directory exists (fresh set)
Generate stubs:

```bash
go run ./cmd/genset "$SET_NAME" data/$0.json cards/$PACKAGE_NAME/
```

Verify the generated files compile:

```bash
go build ./cards/$PACKAGE_NAME/
```

Proceed to Step 3.

### State B — Directory exists with only stubs (generated but no implementation work)
Check if the files are all stubs by looking for any non-TODO implementations. If they're all stubs, skip generation and proceed to Step 3.

### State C — Directory exists with partial implementation (work already done)
The set has had prior implementation work. Check for a **design doc** first — these are manually edited and marked-up validation reports that take priority over a fresh `/validate-set` run:

```bash
ls active-design-docs/set-"$SET_NAME".md active-design-docs/set-$PACKAGE_NAME.md active-design-docs/set-$SET_CODE.md 2>/dev/null
```

If a design doc exists, read it and use it as the source of truth for what's done, what needs work, and any notes or priorities the user has annotated. Skip running `/validate-set`.

If no design doc exists, **run `/validate-set`** to discover the current state:

```
/validate-set $PACKAGE_NAME
```

Either way, you now have a report (design doc or validation output) that tells you:
- Which cards are fully implemented and correct
- Which cards are partially implemented (XXX markers) and what's missing
- Which cards are still stubs (TODO markers)
- Which implementations have Oracle text fidelity issues
- Which cards have tests and which don't
- Any test failures or user-annotated notes

Use this report to drive Step 4. Cards already fully implemented and validated are **done** — skip them entirely. Cards with Oracle violations or missing tests need fixes, not reimplementation. Only stubs and cards flagged as needing engine work go through the normal tier analysis.

## Step 3: Read and Load Context

Read the engine API reference and rules index:

```
Read pkg/mage/doc.go
Read docs/comprehensive-rules-index.md
```

The full MTG comprehensive rules are at `docs/comprehensive-rules.md` (~9200 lines). **Do not read the whole file.** Use the index to find relevant sections by line number, then read with `Read offset=LINE limit=N`. Consult the comp rules when categorizing cards that use keywords or mechanics whose exact behavior affects tier placement.

Read all card files to understand the full card list and existing implementations:

```
Read cards/$PACKAGE_NAME/creatures.go
Read cards/$PACKAGE_NAME/artifacts.go
Read cards/$PACKAGE_NAME/enchantments.go
Read cards/$PACKAGE_NAME/spells.go
Read cards/$PACKAGE_NAME/lands.go
```

Also read the fetched JSON to get full Oracle text for every card:

```
Read data/$0.json
```

## Step 4: Analyze and Categorize Cards

**If resuming from a validation report (State C)**, start from the report's findings. Categorize only the remaining work:

- **Already complete**: Cards that passed validation with correct Oracle text and adequate tests. **Skip these.**
- **Oracle fixes needed**: Cards that are implemented but have fidelity issues from the validation report. These are typically quick fixes — group them as a "fix" batch.
- **Missing tests**: Cards that work but lack test coverage. Group as a "test backfill" batch.
- **Partial implementations (XXX)**: Cards with known gaps. Categorize the missing parts into the tiers below.
- **Stubs (TODO)**: Cards not yet implemented. Categorize into tiers below.

**If starting fresh (State A/B)**, categorize every card into one of these tiers:

### Tier 1 — Vanilla / Basic Keywords Only
Cards whose Oracle text is fully covered by existing constructors with no custom logic:
- Vanilla creatures (no abilities)
- Creatures with only basic keywords (flying, first strike, trample, etc.)
- Basic lands, simple mana-producing lands
- Simple "tap for mana" artifacts

These can be implemented in bulk — they just need the right `WithKeyword()` options added to the stub. They generally don't need tests.

### Tier 2 — Standard Effects (No Engine Work)
Cards that use existing engine effects but need wiring:
- Targeted damage/destruction spells
- Pump effects (+N/+N until end of turn)
- Card draw, discard, life gain/loss
- Simple triggered abilities (ETB, dies, upkeep)
- Auras with standard enchant effects
- Activated abilities with standard costs and effects

These need tests and implementation but no engine changes.

### Tier 3 — Complex but Supported
Cards requiring custom `FuncEffect` or `FuncContinuousEffect` logic but using existing engine infrastructure:
- Cards with complex targeting conditions
- Multi-step effects
- Conditional triggers
- Cards that care about game state (e.g., creature count, color)

### Tier 4 — Requires Engine Work
Cards that need new engine features:
- New keyword mechanics (e.g., a set keyword like Rampage, Bushido)
- New event types or trigger conditions
- New cost types
- Effects the engine doesn't support yet

Group these by the shared engine feature they need.

### Tier 5 — Likely Cannot Implement
Cards with mechanics the engine fundamentally lacks support for (e.g., complex replacement effects, subgames). These should be flagged with AskUserQuestion for the user to decide.

## Step 5: Present the Analysis

Present the categorization to the user as a summary. If resuming a partial set, lead with the current state:

```
## Set Analysis: [Set Name] ([N] total cards)

### Already Complete ([N] cards)
[list card names — only shown when resuming a partial set]

### Needs Fixes ([N] cards)
- Oracle violations: Card A (missing nontoken check), Card B (wrong trigger timing)
- Missing tests: Card C, Card D
[only shown when resuming a partial set]

### Tier 1 — Vanilla/Keywords ([N] cards)
[list card names]

### Tier 2 — Standard Effects ([N] cards)
[list with brief description of what each does]

### Tier 3 — Complex ([N] cards)
[list with brief description]

### Tier 4 — Engine Work Required ([N] cards)
- [Engine Feature A]: Card1, Card2, Card3
- [Engine Feature B]: Card4, Card5

### Tier 5 — Likely Out of Scope ([N] cards)
[list with explanation of what's missing]
```

Use **AskUserQuestion** to confirm the plan with the user before proceeding. Ask if they want to:
- Implement all tiers, or skip certain tiers
- Adjust any categorizations
- Prioritize certain cards or mechanics

## Step 6: Create Implementation Batches

Group cards into batches for `/implement-card`. Batching rules:

**When resuming a partial set, handle fixes first:**
0. **Oracle fixes**: Group by type of fix (e.g., all "missing nontoken check" cards together, all "wrong trigger" cards together). These are small targeted edits, not reimplementations.
0. **Test backfill**: Group by card type or mechanic. Write tests for cards that already work but lack coverage.

**Then proceed with remaining unimplemented cards:**
1. **Tier 1 cards**: One large batch (or split by card type if >15 cards). These are fast.
2. **Tier 2 cards**: Group by similar mechanic (e.g., all "deal N damage" spells together, all ETB creatures together). 3-6 cards per batch.
3. **Tier 3 cards**: 1-3 cards per batch, grouped by shared complexity type.
4. **Tier 4 cards**: Group by the engine feature they share. The engine feature gets built once, then all cards using it are implemented together.
5. **Tier 5 cards**: Handle last, one at a time, with user confirmation.

## Step 7: Execute Batches

Work through batches in tier order (1 → 2 → 3 → 4 → 5).

For each batch, use the `/implement-card` skill:

```
/implement-card Card Name 1, Card Name 2, Card Name 3
```

**Parallelization guidance:**
- Tier 1 batches can run in parallel using background agents since they don't interact
- Tier 2 batches that touch different files (e.g., creatures vs. spells) can run in parallel
- Tier 3+ batches should generally run sequentially since they may need engine changes that affect each other
- Tier 4 batches MUST run sequentially — each may change the engine in ways that affect subsequent batches

After each batch completes:
- Verify `go test ./...` passes
- Verify `go build ./...` passes
- Commit the batch (the /implement-card skill handles this)

## Step 8: Final Verification

After all batches are complete:

```bash
go test ./...
go build ./...
go vet ./...
```

Count remaining TODO/XXX markers:

```
Grep for "TODO: implement" in cards/$PACKAGE_NAME/
Grep for "XXX:" in cards/$PACKAGE_NAME/
```

Present a final summary to the user:
- Total cards implemented
- Cards with partial implementations (XXX markers)
- Cards left as stubs (TODO markers)
- Engine features added
- Test count

## Important Rules

1. **Always ask before proceeding past Step 5.** The user must approve the plan.
2. **NEVER simplify card implementations.** Every condition, restriction, and edge case in Oracle text must be implemented exactly. Do not drop "nontoken" checks, skip targeting restrictions, approximate complex effects, or cut any corners. A simplified implementation is a wrong implementation — this is a rules engine and correctness is the entire point. If something can't be fully implemented, mark it `// XXX:` and ask the user. Never silently simplify.
3. **Commit after each batch**, not at the end. This keeps commits focused and reviewable.
4. **Engine changes get their own verification.** Always run `go test ./pkg/mage/...` after engine changes.
5. **Don't skip cards silently.** Every card in the set should be either implemented, partially implemented with XXX, or explicitly confirmed as out-of-scope by the user.
6. **Use AskUserQuestion liberally.** When in doubt about categorization, scope, or implementation approach, ask.
7. **Basic lands that are reprints** will be auto-skipped by genset (it detects already-registered names). Don't worry about these.
