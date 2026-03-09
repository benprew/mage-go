---
name: validate-set
description: "Validate completeness and correctness of an implemented Magic: The Gathering card set. Checks for stubs, missing tests, Oracle text fidelity, and simplified implementations. Use when the user wants to audit a set's implementation quality."
allowed-tools: Read, Bash, Grep, Glob, Agent, AskUserQuestion, WebFetch
argument-hint: <package-name> [set-code]
---

# Set Validation Skill

You are auditing the implementation quality and completeness of a Magic: The Gathering card set in the mage-go engine.

## Arguments

- `$0` — The card package name (e.g., `legends`, `arabian`, `antiquities`). **Required.**
- `$1` — Scryfall set code (e.g., `LEG`, `ARN`, `ATQ`). Optional — will be auto-detected if omitted.

## Step 0: Resolve Scryfall JSON

The Oracle text audit requires authoritative card data from Scryfall. Resolve the JSON file before doing anything else.

### 0a. Determine the set code

If `$1` was provided, use it. Otherwise, read `data/sets.txt` and look up the package name to get the set code and set name. The file format is:

```
package-name SET_CODE "Set Name"
```

If the package name isn't in `data/sets.txt`, use **AskUserQuestion** to ask the user for the set code.

### 0b. Fetch if missing

Check if the JSON exists:

```bash
ls data/$SET_CODE.json 2>/dev/null
```

If not, fetch it (set `FETCHSET_SKIP_TLS=1` to work around sandbox TLS restrictions):

```bash
FETCHSET_SKIP_TLS=1 go run ./cmd/fetchset -o data/$SET_CODE.json $SET_CODE
```

Verify the fetch succeeded before proceeding.

## Step 1: Gather All Card Data

Read every source file in the set:

```
Read cards/$0/creatures.go
Read cards/$0/artifacts.go
Read cards/$0/enchantments.go
Read cards/$0/spells.go
Read cards/$0/lands.go
```

Read all test files:

```
Glob for cards/$0/*_test.go and read each one
```

Read the Scryfall JSON resolved in Step 0:

```
Read data/$SET_CODE.json
```

## Step 2: Build the Card Inventory

For every card registered in the set, extract:

1. **Card name** (from `Register("Card Name", ...)`)
2. **Oracle text comment** (the `// Oracle:` or comment block above the Register call)
3. **Implementation status**:
   - `STUB` — has `// TODO: implement` marker or the body is just a bare constructor with no abilities
   - `PARTIAL` — has `// XXX:` comments indicating incomplete implementation
   - `COMPLETE` — no TODO/XXX markers and has non-trivial implementation (or is genuinely vanilla)
4. **Has tests** — whether a `TestCardName` function exists in the test files
5. **Test coverage quality** — number of test cases (sub-tests) and whether they cover positive AND negative cases

## Step 3: Oracle Text Fidelity Audit

This is the most critical check. For every card that is not a STUB, compare the implementation against the Oracle text **word by word**. Look for:

### 3a. Missing Abilities
Oracle text describes an ability that has no corresponding implementation code. Examples:
- Oracle says "Flying, first strike" but only Flying is implemented
- Oracle describes a triggered ability that isn't wired up
- Oracle has an activated ability that's missing

### 3b. Simplified or Paraphrased Effects — THE MOST CRITICAL CHECK
The implementation does something *similar* to but not *exactly* what Oracle says. **Simplified implementations are wrong implementations.** This is a rules engine — every condition matters. Flag ALL of these aggressively:
- Oracle says "nontoken creature" but code doesn't filter for nontoken
- Oracle says "each creature" but code only targets one
- Oracle says "can't be regenerated" but the destroy effect doesn't prevent regeneration
- Oracle says "at the beginning of your upkeep" but trigger is on a different step
- Oracle says "you may" (optional) but code makes it mandatory, or vice versa
- Oracle specifies a target restriction (e.g., "target creature with flying") but code uses a broader filter
- Oracle says "another creature" but code doesn't exclude self
- Oracle says "controller" but code affects "owner"
- Oracle says "exile" but code puts in graveyard, or vice versa
- Oracle references a specific zone (hand, library, graveyard) but code uses a different one
- Oracle has a conditional ("if", "unless", "only when") that the code doesn't check
- Oracle limits by color, type, subtype, power, toughness, or CMC but code doesn't filter

### 3c. Hardcoded or Approximate Values
- Damage amounts, life totals, or counters that don't match Oracle
- P/T bonuses that are wrong
- Mana costs on activated abilities that are wrong

### 3d. Missing Conditions or Restrictions
- Oracle has "if" / "unless" / "only if" conditions that aren't checked
- Oracle specifies timing restrictions ("only as a sorcery", "only during combat") that aren't enforced
- Oracle limits targets ("another creature", "creature you don't control") but code doesn't

### 3e. Multiplayer Shortcuts (Acceptable)
"Each opponent" simplified to one opponent is acceptable (2-player engine). Note but don't flag these.

To do this audit effectively, **use the Scryfall JSON** as the source of truth for Oracle text. Do NOT rely only on the comments in the Go code — the comments may have been paraphrased or truncated. For cards where the JSON is unavailable, fetch Oracle text from Scryfall:

```
WebFetch https://api.scryfall.com/cards/named?exact=CARD_NAME
```

## Step 4: Test Coverage Audit

For each non-vanilla, non-stub card, assess test quality:

### Cards that NEED tests (flag if missing):
- Any card with a triggered ability
- Any card with an activated ability (beyond simple mana abilities)
- Any card with a static/continuous effect
- Any card with a spell effect (instants/sorceries)
- Any card with conditional behavior
- Any aura or equipment

### Cards that DON'T need tests:
- Vanilla creatures (no abilities beyond basic keywords like flying, trample)
- Basic lands
- Simple mana rocks that just tap for one color

### Test quality checks:
- Does the test actually exercise the card's ability? (Not just "card exists on battlefield")
- Are there negative cases? (Ability does NOT trigger when conditions aren't met)
- For targeted effects: does a test verify targeting restrictions?
- For conditional effects: are both branches tested?
- Does the test check the right thing? (e.g., testing life total change for a damage spell, not just that the spell resolves)

## Step 5: Run the Tests

```bash
go test ./cards/$0/ -v -count=1
```

Report any test failures — these indicate bugs in existing implementations.

Also run:

```bash
go vet ./cards/$0/
```

## Step 6: Compile the Report

Organize findings into these sections:

---

### Set Validation Report: [Set Name]

#### Summary
- Total cards in set: N
- Fully implemented: N
- Partially implemented (XXX): N
- Stubs (TODO): N
- Test coverage: N of M testable cards have tests

#### Stubs (Not Implemented)
List every card still marked TODO, grouped by card type:
```
Creatures: Card A, Card B
Spells: Card C
Artifacts: Card D
```

#### Partial Implementations (XXX Markers)
For each card with XXX comments:
- Card name
- What's implemented vs. what's missing
- The XXX comment text

#### Oracle Text Violations
For each card where implementation doesn't match Oracle:
- **Card name**
- **Oracle says**: [exact Oracle text for the relevant ability]
- **Implementation does**: [what the code actually does]
- **Issue**: [concise description of the discrepancy]

Group related violations together (e.g., all cards missing "nontoken" checks, all cards with wrong trigger timing).

#### Missing Tests
List every testable card that lacks tests, grouped by what kind of test is needed:
```
Needs triggered ability tests: Card A, Card B
Needs activated ability tests: Card C
Needs spell effect tests: Card D, Card E
```

#### Weak Tests
Cards that have tests but the tests are insufficient:
- Card name
- What's tested
- What's missing (e.g., "no negative case", "doesn't verify targeting restriction")

#### Test Failures
Any tests that are currently failing, with the error output.

#### Engine Feature Gaps
Cards grouped by the engine feature they're missing. This helps prioritize engine work:
```
- [Feature description]: Card A, Card B, Card C
- [Feature description]: Card D
```

---

## Step 7: Suggest Next Steps

Based on the report, suggest a prioritized action plan:

1. **Quick wins**: Cards that are almost complete (just missing a keyword, wrong value, etc.)
2. **Test gaps**: Cards that work but need tests written
3. **Oracle fixes**: Cards where the implementation is subtly wrong
4. **Engine work**: Grouped by feature, with estimated impact (how many cards each feature unblocks)
5. **Hard problems**: Cards that may need significant engine work or user decisions

Use **AskUserQuestion** if you're unsure whether a discrepancy is intentional (e.g., a known engine limitation the user chose to accept) or a bug.

## Important Rules

1. **Scryfall JSON is the source of truth for Oracle text.** Not the comments in the Go code.
2. **Be precise about violations.** Quote the exact Oracle text and the exact code behavior. Don't be vague.
3. **Don't flag multiplayer simplifications.** "Each opponent" → single opponent is expected.
4. **Aggressively flag ALL simplifications.** This is the single most important job of validation. If Oracle says "nontoken" and code doesn't check, that's a real bug — full stop. If Oracle says "another" and code doesn't exclude self, flag it. If Oracle has a condition the code doesn't check, flag it. No simplification is acceptable, no matter how "rarely" it matters. A simplified implementation is a wrong implementation.
5. **Read the actual implementation code**, not just the comments. The code might do something different from what the comment says.
6. **Run the tests.** Don't just check if they exist — verify they pass.
