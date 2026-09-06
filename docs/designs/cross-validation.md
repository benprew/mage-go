# Cross-Validation: mage-go vs XMage — Focused on Known Gaps

## Context

Validate mage-go's rules engine correctness using XMage (at `../xmage`) as a reference oracle. The user has identified specific engine gaps and one confirmed bug. The work covers:
1. **Fix CR 509.1h bug** — blocked creature deals damage to player after blocker dies mid-combat
2. **Write turn-structure tests** covering the known gaps, using XMage behavior as the expected baseline
3. **Install Java/Maven** to run XMage tests for reference validation

## Step 1: Install Java + Maven

```bash
brew install openjdk maven
```

## Step 2: Fix CR 509.1h Bug — Blocked Creature Remains Blocked

**The bug:** In `ResolveDamage()` at `combat.go:258`, `len(group.BlockerIDs) == 0` is used to determine unblocked status. When a blocker dies mid-combat (e.g., killed by a spell between declare blockers and combat damage), its ID is removed from `BlockerIDs` by `RemoveFromCombat()`. The attacker then incorrectly deals damage to the defending player.

**The rule:** CR 509.1h — "A creature remains blocked even if all the creatures blocking it are removed from combat." CR 510.1c — if no creatures are currently blocking it, it assigns no combat damage.

**XMage reference:** `CombatGroup.java` has a `boolean blocked` field, set to `true` in `addBlockerToGroup()` (line 588), never reset when blockers are removed. At damage time (line 253): `if (!blocked || hasTrample(attacker))`.

**Fix in `pkg/mage/combat.go`:**

1. Add `Blocked bool` to `CombatGroup` struct (line 17-21)
2. Set `Blocked = true` in `AddBlocker()` (line 46-53)
3. In `ResolveDamage()` (line 258), change `len(group.BlockerIDs) == 0` to `!group.Blocked`
4. For unblocked-but-was-blocked case (Blocked=true, no remaining blockers): assign no damage unless trample
5. Update `doBandedAttackDamage()` (line 430) similarly
6. Update trigger conditions in `trigger_condition_data.go` (~line 239-254): `SourceIsBlockedAttacker` and `SourceIsUnblockedAttacker` should use `group.Blocked`

**Test (TDD):**
```go
func TestBlockedCreatureRemainsBlocked(t *testing.T) {
    // Attacker blocked by a creature, blocker killed by Lightning Bolt before damage
    // Attacker should deal NO damage to defending player (CR 509.1h + 510.1c)
}
```

## Step 3: Write Turn-Structure Tests

Create test files organized by turn phase. Each test documents a specific CR rule. Tests that expose engine gaps should be skipped with `t.Skip("engine gap: ...")` so they compile and document the gap without failing CI.

### Files to create:
- `pkg/mage/turn_test.go` — comprehensive turn structure tests

### Tests covering known gaps:

| CR Rule | Test | Status |
|---------|------|--------|
| **509.1h** | Blocked creature stays blocked after blocker removed | **Bug fix** |
| 500.5 | Mana pool empties at each step (not just cleanup) | Gap: `t.Skip` |
| 500.11 | Skip-step/phase/turn effects | Gap: `t.Skip` |
| 500.12 | Nothing observable between steps | Gap: `t.Skip` |
| 503.2 | Extra upkeep step effects | Gap: `t.Skip` |
| 505.1a | Skip combat phase effects | Gap: `t.Skip` |
| 506.4a | Runtime "can't attack" instant-speed effects | Gap: `t.Skip` |
| 506.5 | "Attacks alone" / "blocks alone" selectors | Gap: `t.Skip` |
| 508.2a | Color override for attacking restrictions | Gap: `t.Skip` |
| 510.1c | Multi-blocker damage assignment DSL | Gap: `t.Skip` |
| 514.3 | Priority denial in cleanup observable | Gap: `t.Skip` |
| 514.3a | EvtCleanup event | Gap: `t.Skip` |

### XMage validation for each test:
For each test, write a matching XMage JUnit test in `../xmage/Mage.Tests/src/test/java/org/mage/test/crossval/` that validates the expected behavior. Run XMage tests to confirm expected values before hardcoding them in Go tests.

## Step 4: XMage Reference Test Class

**File:** `../xmage/Mage.Tests/src/test/java/org/mage/test/crossval/TurnStructureTest.java`

Extends `CardTestPlayerBase`. Each method mirrors a Go test scenario with the same setup/actions/assertions. Run with:
```bash
cd ../xmage && mvn test -pl Mage.Tests -Dtest=TurnStructureTest
```

## Key Files

| File | Action |
|------|--------|
| `pkg/mage/combat.go` | Add `Blocked` field, fix `ResolveDamage()` |
| `pkg/mage/trigger_condition_data.go` | Update blocked/unblocked checks |
| `pkg/mage/turn_test.go` | New: turn-structure gap tests |
| `pkg/mage/combat_test.go` | New: CR 509.1h regression test |
| `../xmage/.../crossval/TurnStructureTest.java` | New: XMage reference tests |

## Verification

1. `go test ./pkg/mage/ -run TestBlockedCreature` — confirms 509.1h fix
2. `go test ./pkg/mage/...` — no regressions
3. `go test ./cards/...` — no regressions in card tests
4. XMage tests pass with `mvn test -pl Mage.Tests -Dtest=TurnStructureTest`
5. Go skip-tests document gaps clearly for future implementation
