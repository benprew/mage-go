---
name: implement-engine
description: "Implement a new engine feature in pkg/mage/ to support card implementations. Adds the feature, writes engine-level tests, updates doc.go, and verifies all existing tests still pass. Use when card agents identify an engine gap."
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent, AskUserQuestion
argument-hint: <description of needed engine feature>
---

# Engine Implementation Skill

You are implementing engine features in `pkg/mage/` for the mage-go MTG rules engine. You ONLY modify files in `pkg/mage/`. You do NOT implement cards or modify files in `cards/`.

## Arguments

Engine feature request: $ARGUMENTS

## Scope Boundary

**You MUST only edit files in `pkg/mage/` and `pkg/mage/core/`.**

Do NOT edit:
- `cards/` — card implementations are handled by `/implement-card`
- `cmd/` — CLI tools
- `internal/` — TUI layer
- `web/` — browser UI

If the feature request includes card-level wiring (e.g., "implement Rampage on card X"), implement only the engine-level mechanic (the Rampage keyword, attr, and layer/combat logic). Return the API surface so the card agent can wire it up.

## Step 1: Load Context

Read the engine API reference to understand existing patterns:

```
Read pkg/mage/doc.go
```

Read the rules index for the relevant mechanic:

```
Read docs/comprehensive-rules-index.md
```

Then read the specific comprehensive rules section for the mechanic being implemented. Use `Read offset=LINE limit=N` — never read the whole rules file.

## Step 2: Understand the Request

The request comes from a card-authoring agent or the `/implement-set` orchestrator. It describes what the engine needs but doesn't yet support. Parse out:

1. **What mechanic or capability is needed** — e.g., "Rampage N", "phasing", "new replacement effect for X"
2. **Which cards need it** — context for how the feature will be used
3. **Oracle text driving the need** — the exact rules language that must be supported

## Step 3: Research Existing Engine Code

Search `pkg/mage/` for related patterns. Understand:

- How similar mechanics are implemented (e.g., if adding a new keyword, see how existing keywords work)
- Which files need changes (attrs in `core/`, effects in `effect.go`, combat logic in `combat.go`, etc.)
- Whether this is a new subsystem or an extension of an existing one

Use an Agent with subagent_type=Explore if the search is broad.

Check XMage for how they implemented the mechanic:

```
Search ~/mage/Mage/src/main/java/mage/ for the mechanic
```

## Step 4: Write Engine Tests First (RED)

Write tests in the appropriate `_test.go` file in `pkg/mage/`. These tests should:

- Verify the mechanic works in isolation
- Use `gametest.TestGame` to set up game states that exercise the feature
- Cover edge cases from the comprehensive rules
- Test interaction with existing mechanics where relevant

Register any test-only cards inline using `sync.Once` (see `pkg/mage/mechanics_test.go` for the pattern).

Run the tests to confirm they fail:

```bash
go test ./pkg/mage/ -run TestNewFeature -v
```

## Step 5: Implement the Feature (GREEN)

Implement the minimum engine changes needed. Common patterns:

### New Keyword/Attr
1. Add the `Attr` constant in `pkg/mage/core/attrs.go`
2. Add keyword registration in `pkg/mage/keywords.go` if it's a keyword ability
3. Add combat/layer/SBA logic where needed
4. Add any helper constructors (e.g., `WithRampage(n)` card option)

### New Effect
1. Add the effect implementation in the appropriate file
2. Add a constructor function
3. Document in `doc.go`

### New Replacement Effect
1. Add the replacement type in `pkg/mage/replacement.go`
2. Add the action type in `pkg/mage/action.go` if needed
3. Wire into `ApplyReplacements` pipeline
4. Add *Game proxy method if cards need a simple API

### New Triggered Ability Pattern
1. Add the event type in `pkg/mage/events.go` if needed
2. Add a convenience constructor
3. Document in `doc.go`

Run tests after implementation:

```bash
go test ./pkg/mage/ -run TestNewFeature -v
```

## Step 6: Verify No Regressions

Run ALL engine tests:

```bash
go test ./pkg/mage/ -v
```

Then run ALL tests including cards:

```bash
go test ./...
```

Fix any regressions. Engine changes must not break existing cards.

## Step 7: Update doc.go

Add documentation for the new feature in `pkg/mage/doc.go`. Follow the existing format:
- Add the new constructor/effect/keyword to the appropriate section
- Include a brief usage example showing how card code would use it
- Keep it concise — doc.go is a reference, not a tutorial

## Step 8: Report Back

Output a structured summary of what was added:

```
## Engine Feature: [name]

### New API Surface
- `WithRampage(n int)` — card option for Rampage N
- `AttrRampage` — attr constant

### Files Changed
- pkg/mage/core/attrs.go — new attr
- pkg/mage/combat.go — rampage damage calculation
- pkg/mage/keywords.go — keyword registration
- pkg/mage/doc.go — documentation

### Usage in Card Code
mage.Register("Craw Giant", func() mage.Card {
    return mage.NewCreature(6, 4,
        mage.WithCost("3RRRR"),
        mage.WithKeyword(core.Trample),
        mage.WithRampage(2),
    )
})

### Tests Added
- TestRampageDamageMultipleBlockers
- TestRampageWithTrample
```

This summary is what the card agent or orchestrator uses to continue implementation.

## Important Rules

1. **Stay in `pkg/mage/`.** Never edit card files. Your job is the engine, not cards.
2. **TDD is mandatory.** Engine tests first, implementation second.
3. **No regressions.** `go test ./...` must pass before and after.
4. **Update doc.go.** Every new public API must be documented.
5. **Follow comprehensive rules.** The MTG rulebook is the authority on how mechanics work. Look up the rules before implementing.
6. **Minimal changes.** Implement only what's needed for the requested feature. Don't refactor unrelated code.
7. **Consult XMage.** For complex mechanics, check how XMage handles edge cases.
8. **Report the API surface.** The card agent needs to know exactly what constructors, attrs, and functions are available.
