# Gathered untap and combat-prevention slice

Status: complete within later action slice 2, with a bounded prevention adapter
from slice 7 and a controller restriction for typed permanent targets. The full
DSL design remains in progress. Rules reference: repository Comprehensive
Rules, effective February 27, 2026; CR 400.7, 506.4b, 608.2b, 615, 701.26, and
702.26.

## Removed exports and semantic parity

| Removed surface | In-tree callers | Canonical replacement | Existing executor |
| --- | --- | --- | --- |
| `mage.UntapGathered`, `dsl.UntapGathered`, `UntapGatheredData` | Ebony Horse in `cards/arabian/artifacts.go` | `Untap(creature)` with a typed target; `Untap(From(saved))` remains available for saved values | `UntapPermanent` and `Game.UntapPermanent` |
| `mage.PreventCombatDamageToAndByTarget`, `dsl.PreventCombatDamageToAndByTarget` | No production or test callers; DSL export and API documentation only | `PreventCombatDamageToAndBy(subject)` | `Game.PreventCombatDamageToAndBy` and the existing replacement pipeline |

The two obsolete executor types are removed. The direct game method remains
an engine API. String-based `RemoveFromCombatGathered` remains because other
cards still use it. No other tap, attachment, or damage-source family is removed.

## Ebony Horse clause coverage

Oracle text retained above registration:

> {2}, {T}: Untap target attacking creature you control. Prevent all combat
> damage that would be dealt to and dealt by that creature this turn.

| Clause | Expression path | Rules contract | Evidence |
| --- | --- | --- | --- |
| {2}, {T} | `Activate(Costs(Mana("{2}"), TapCost(This)), ...)` | Existing total-cost transaction; failed targeting pays nothing | `TestEbonyHorseRestrictionsAndStun` |
| Target attacking creature you control | `TargetPermanent(..., And(IsCreature, IsAttacking), TargetYouControl())` and `WithTargets` | Select at activation; recheck restrictions at resolution | `TestEbonyHorseRestrictionsAndStun`, `TestTypedControlledTarget` |
| Untap | `Sequence` first child: `Untap(creature)` | Existing untap and stun replacement; the creature stays in combat | `TestEbonyHorseKeepsAttackerInCombat`, `TestEbonyHorseRestrictionsAndStun` |
| Prevent combat damage to and by that creature this turn | Second child: `PreventCombatDamageToAndBy(creature)` | Unlimited combat-only prevention for the captured incarnation until cleanup; independent of successful untap or source survival | Both new Ebony Horse tests, `TestCombatPreventionIncarnation`, existing `TestPreventCombatDamageToAndBy` |

The earlier card implementation removed the creature from combat. That was not
Oracle-equivalent: CR 506.4b keeps an untapped attacker or blocker in combat.
The new card regression failed on this exact distinction before migration. The
old happy-path test is renamed and still checks both creatures survive; it now
also checks the attacker is untapped. The new test asserts combat membership
before damage, prevention in both directions, and noncombat damage afterward.
This is an intentional correction, not preservation of a known approximation.

All new card expressions compile through `dsl`. The file retains its direct
engine import for unrelated abilities and custom costs.

## Target restriction adapter

`TargetYouControl()` is a closed `PermanentTargetOption` accepted by singular
and collection declarations. It adds a controller restriction to the existing
`PermanentTarget.Possible` path. The controller is the stored ability controller,
not a later controller of the source. The same restriction runs during selection
and the normal resolution legality check. The declaration exposes a
`target-controller` child for inspection. The new option does not filter the
subject again between sequential effects.

No existing unrestricted target changes behavior. Nil options are rejected at
construction; unrelated DSL categories cannot be used as target options.

## Prevention contract and separate engine correction

`PreventCombatDamageToAndBy(subject)` resolves one `PermanentRef` when it runs.
An absent, phased-out, or stale subject has no effect. A valid subject installs
the existing combat prevention replacement until end of turn. The replacement
is independent of the resolving source. Both source and recipient combat damage
checks use the subject's captured battlefield incarnation. Control changes and
a completed phasing cycle preserve it; a returned card is a different object.
Noncombat and unrelated combat damage are unaffected. The shield has no amount
to consume. Cleanup removes it through the existing duration path.

The prior replacement stored only a physical card ID. The new
`TestCombatPreventionIncarnation/return` failed because it prevented damage to
and by a returned card. `Game.PreventCombatDamageToAndBy` now captures the
incarnation, and the existing replacement checks it before matching. Installation
with an absent ID does nothing. Its clone copies the captured value; no mutable
state is shared. These changes correct this replacement only and do not close
the separately documented general damage-source LKI gap.

The typed instruction exposes its subject and benefit metadata. The removed,
unused positional constructor reported detriment metadata. Ebony Horse's body
retains its prior benefit summary while exposing both sequential instructions.
There is no new replacement executor or special card callback.

## Composition and validation evidence

- `TestTypedCombatPreventionSubject`: an explicit source subject overrides an
  unrelated positional target; both combat directions and unrelated damage.
- `TestTypedControlledTarget`: own versus opposing target candidates, definition
  validation, and target control changing before resolution.
- `TestCombatPreventionIncarnation`: ordinary prevention, return, phasing cycle,
  control change, expiration, and cloned-game isolation.
- `TestTypedCombatPreventionComposition`: public singular and collection targets,
  sequence, saved loop member, trigger, subject validation, and metadata.
- `combat_prevention_player.go` rejects a player as a permanent subject.
- `permanent_target_cost.go` rejects a cost as a target restriction.
- Existing typed-target, untap, replacement, and registered-definition suites.

Verification results are recorded in the implementation inventory.
