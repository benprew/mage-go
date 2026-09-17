# Source-exit duration slice

Status: complete. This is the source-exit duration family within later
continuous-effect slice 4. Other durations, modifications, dependency handling,
and compound-effect identity remain open in the implementation inventory.

Rules reference: repository Comprehensive Rules, effective February 27, 2026,
CR 400.7 (object identity), 611.2a–d (resolved continuous effects), 613 (layers),
305.7 (basic land types), and 702.26d–f (phasing).

## Contract and engine ownership

`dsl.UntilSourceLeaves(dsl.This, modifications...)` installs the existing typed
layer modifications through the existing effect manager. It is a resolving
instruction, not a static rule. The source argument accepts only `This`.
The existing `Until` constructor still handles its supported fixed durations.

The adapter captures the ability's source incarnation, all affected subjects,
and values at resolution. All modifications capture before any are installed.
An absent source or a different incarnation does not start the effect. The
source can change controller or phase out without ending this duration,
including when it phases out before resolution. This is an "until it leaves"
duration, not a "for as long as" state condition. Captured subjects retain their
identity through phasing but reject a zone round trip. Later entrants are not
added, and values are not recalculated on later layer passes.

Installation uses `EffectManager.Add`; application uses the existing typed
layer executors. `Game.RemoveFromBattlefield` and `EffectManager.Remove` remove
all layers on source exit. No new duration enumeration, zone system, expiry
observer, or mutable shared latch was added. Game clones have independent
installed-effect lists and retain immutable captured values.

Inspection includes the source and modification children, the duration text,
and the existing modification metadata and legacy properties. Registration
checks modification lists, binding scope, nil values, and the permanent-source
requirement. The interface rejects resolving instructions as modifications,
players as sources, and installed instructions as static rules.

## Removed export and caller parity

| Removed export | Canonical replacement | Existing executor |
| --- | --- | --- |
| `BecomesBasicLandTargetUntilSourceLeaves` in `mage` and `dsl` | `UntilSourceLeaves(This, SetSubtypeFamily(land, SubtypeLand, "Forest"))` | `applySetSubtypes` in layer 4; effect manager installation and source cleanup |

All callers of the removed export have migrated:

| Caller | Target, timing, and lifetime | Evidence |
| --- | --- | --- |
| Gaea's Liege in `cards/fourthedition/creatures.go` | Typed land target; tap cost; capture at ability resolution; Forest replacement lasts until the original Liege leaves | `TestGaeasLiege`, `TestGaeasLiegeSourceDuration` |
| Source-phasing regression in `pkg/mage/basic_land_type_effect_test.go` | Typed land target, same layer-4 operation; source phasing preserves duration | `TestComposableSourceDurationLandSourcePhasing` |
| Target-phasing regression in the same file | Typed land target; captured target survives phasing | `TestComposableSourceDurationLandTargetPhasing` |

The card's complete Oracle comment matches `data/4ED.json`. Its unrelated
power/toughness static ability remains in its existing implementation and keeps
its direct engine import. Basic land replacement still removes printed rules
abilities, replaces land subtypes, and grants intrinsic Forest mana. No source
LKI is needed: the duration checks identity, not source characteristics.
The new instruction retains beneficial outcome metadata and also exposes the
layer operation and binding through inspection.

A failing card regression demonstrated that the old callback still changed the
land if the Liege had left or returned before resolution. The new adapter does
nothing in those cases. Existing target legality already rejects a land that
leaves and returns before resolution. These are separate source and target
checks, both exercised by the migrated card test.

A separate failing engine regression showed that direct `ApplyEffect` calls
lost the incarnation of an already phased-out source. Its fallback capture now
uses `FindPermanentIncludingPhased`. Battlefield subject selectors continue to
exclude phased-out permanents; this change only retains source identity.

## Behavioral and composition evidence

| Contract | Regression |
| --- | --- |
| PT, color, keyword layers share expiry; source phasing and clone isolation | `TestComposableSourceDurationLifetime` |
| Missing, returned, or phased source before installation | `TestComposableSourceDurationBeforeResolution` |
| Real pending/stack path; stored controller after control change; returned source | `TestComposableSourceDurationTriggerContext` |
| Direct resolution retains phased source identity | `TestComposableSourceDurationDirectPhasedContext` |
| Captured membership and values; later entrants; phased and returned subjects | `TestComposableSourceDurationCapturedMembers` |
| Empty/nil modifications, missing saved producer, spell-only source | `TestComposableSourceDurationValidation` |
| Saved and target bindings, sequence, loop, mode, activation, trigger, metadata | `TestTypedSourceDurationComposition` |
| Category rejection through the public DSL | `source_duration_player.go`, `source_duration_resolving.go`, `source_duration_static.go` |
| Land restriction, Forest mana, source-exit restoration | `TestGaeasLiege` |
| Source absence/return/phasing and target return before resolution | `TestGaeasLiegeSourceDuration` |

The general `BecomesBasicLandTargetEffect` remains because other durations and
callers still use it. The new duration does not implement control-dependent,
tap-dependent, next-turn, or other conditional lifetimes.

## Verification for the source-exit duration slice

- Focused duration, card, public composition, and compile-fail tests passed.
- `go test ./cards/fourthedition -count=1` passed.
- `go test ./pkg/mage/... -short -count=1` passed.
- `go test ./... -short -count=1` passed, including all card packages and
  registered-definition validation.
- `go build ./...` and `go vet ./...` passed.
- `make lint` passed with zero issues. Its unrelated import formatting change
  in Secrets of Strixhaven was reverted.
- `git diff --check` passed.

Long AI simulations were not rerun; this slice changes no AI executor.
