# Target-untap action slice

Status: complete within later action slice 2, with a bounded random-choice
adapter from slice 3. Other action and choice families remain open. Rules
reference: repository Comprehensive Rules, effective February 27, 2026;
CR 400.7, 603.3d, 608.2b, 701.26, and 702.26.

## Removed exports and execution contract

| Removed export | Canonical replacement | Existing executor |
| --- | --- | --- |
| `mage.UntapTarget`, `dsl.UntapTarget` | `dsl.Untap(target)` with `TargetPermanent` and `WithTargets`, or `Untap(From(saved))` for a random choice | `UntapPermanent` resolves the typed reference and calls `Game.UntapPermanent` |
| `mage.UntapTargetStep`, `dsl.UntapTargetStep` | `ForEach(ControlledPermanents(IsLand), land, Untap(From(land)))` | Existing typed loop and `Game.UntapPermanent` |

The private positional executor is removed. Target declarations retain filters
and stack-time choice. Resolution checks legality before creating the typed
frame. Partial illegality does not shift the other bindings. Absent, returned,
and phased-out subjects do nothing. A completed phasing cycle keeps identity.
No source LKI is required to untap a surviving target. Target-source legality
continues to use the existing targeting path.

Untap instructions retain benefit metadata and expose their subjects to
inspection and validation. Reset now derives both mass and benefit metadata
from its typed loop; its old loop exposed only mass metadata. Each instruction
uses the existing stun replacement and untap event path. Definitions contain
no selected objects and can be shared by original spells, copies, and clones.

## Caller and clause parity

| Caller and file | Migrated expression and preserved clause | Evidence |
| --- | --- | --- |
| Ley Druid, `cards/limited/creatures.go` | `Activate(Costs(TapCost(This)), Untap(land), WithTargets(land))`; any target land | `TestLeyDruidTypedUntap`: opponent land, rejected creature, tap cost |
| Jandor's Saddlebags, `cards/arabian/artifacts.go` | Typed creature target; {3} and tap costs | `TestJandorsSaddlebags` |
| Disharmony, `cards/legends/spells.go` | Typed attacking-creature target; untap before removal from combat and control effect | `TestDisharmony`: control and untap assertions |
| Reset, `cards/legends/spells.go` | Typed loop over controlled lands; untap each saved member | `TestResetTypedLandLoop`: own land, excluded opposing land and own creature |
| Act of Treason, `cards/jumpstart/spells.go` | Typed creature target; untap after control change and before haste | `TestActOfTreason`: previously tapped creature is untapped and gains haste |
| Initiate's Companion, `cards/jumpstart/creatures.go` | Typed creature-or-land target on the existing combat-damage trigger | `TestInitiatesCompanionTypedUntap`: combat versus noncombat damage |
| Whimsy untap action, `cards/astral/spells.go` | Random saved member of tapped artifacts, creatures, or lands; `Untap(From(chosen))` | `TestWhimsyRandomUntapMembership`, existing Whimsy sequence and empty-choice tests |
| Ward fixture, `pkg/mage/keyword_ward_test.go` | Typed creature target in the friendly untap spell | `TestWard_DoesNotTriggerOnOwnSpell` |

Every migrated production expression compiles through `dsl`. Disharmony and
Act of Treason retain their unrelated positional control/combat/keyword effects
in the same composite body. The legacy event predicate for Initiate's Companion
is unchanged. `GenericTriggered.WithTargets` installs typed declarations through
the existing target option adapter; it does not introduce a new event pattern.

Disharmony and Reset retain their documented casting-timing gaps. This slice
preserves their existing untap behavior and does not certify those restrictions.
Reset retains sequential execution; this migration does not add a simultaneous
mass-untap action. Direct engine imports for unrelated families remain in all
six migrated production files.

## Random choice contract

`ChooseRandomPermanent(subjects, binding)` is a resolving instruction. It gets
current members once, selects through `Game.RandIntn`, and captures the selected
incarnation. It is a non-target choice at resolution, not a target declaration
or a random predicate. `MatchingPermanents` excludes phased-out objects. An
empty set clears a previous binding without consuming randomness. A nonempty
set consumes one random value, including a single-member set, as the previous
random executor did. The same saved-reference lifetime rules apply after choice.

The existing sequence frame owns writes. The definition visitor records the
producer and subjects, rejects nil inputs and reads before assignment, and
keeps choice metadata separate from the following untap action. Game clones
have independent random queues and saved frames. Other legacy random wrappers
remain for their callers.

## Clone defect corrected before migration

`cloneStackObject` omitted `targetIncarnations`. The new
`TestComposableTargetUntapPartialAndLifetime/return` failed because a cloned
spell untapped a returned target while still resolving for its other legal
target. The clone now retains the immutable incarnation map, consistent with
its stack ownership contract. Copy retargeting already makes its own map.
No new target, action, replacement, or clone subsystem was added.

## Focused evidence

- `TestComposableTargetUntapPartialAndLifetime`: partial legality, absence,
  returned targets, phasing, completed phasing cycles, and clone isolation.
- `TestComposableTargetUntapCopyRetarget`: a copy can untap a new target while
  the original keeps its target.
- `TestComposableLegacyTriggerTypedUntap`: typed declaration validation and
  pending trigger placement and resolution through the legacy builder.
- `TestComposableRandomUntapBinding`: deterministic selection, repeated
  resolution, empty-set random consumption, and clone isolation.
- `TestComposableRandomUntapValidationAndAbsence`: producer ordering, nil
  inputs, undeclared targets, and clearing a previous saved value.
- `TestTypedTargetUntapComposition`: public spell, activation, trigger, mode,
  saved choice, and structural benefit metadata.
- Compile-fail fixtures reject player/collection untap subjects and singular
  subjects where random selection requires a collection.
- Existing source-untap tests continue to cover stun counters, untap events,
  control changes, source lifetime, and nil-subject validation.

## Verification

Verification results are recorded in the implementation inventory.
