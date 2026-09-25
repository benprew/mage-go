# Composable DSL Primitive Reuse Implementation

This tracker records engine-only slices for the primitive reuse plan. Card migrations are outside the scope of the current work.

## F1: Typed fight

- [x] Add `Fight(PermanentRef, PermanentRef)` as a resolving effect that reuses `fightPermanents` and preserves the enclosing resolution context.
- [x] Inspect and validate both references, including nil and undeclared target bindings; reject player and collection arguments at compile time.
- [x] Cover absent or changed subjects, battlefield incarnation identity, same-creature damage, and copy/clone isolation in engine tests.
- [x] Correct the shared helper's self-fight damage to twice the creature's power (CR 701.14c).
- [x] Audit simultaneous fight damage and damage-trigger observation in the shared damage service. Fight now prepares both damage actions through replacement and prevention, applies both results, then sends damage events and callbacks. `TestFightDamageObserversSeeBothResults` covers the observer state.
- [ ] Migrate Arena and verify its target and tap behavior. No card changes were made in this slice.

## Remaining work packages

- [ ] T1: Immutable token definitions and typed creation.
- [x] C1: Temporary control modification. `ChangeController(PermanentRef, PlayerRef)` captures the target incarnation and evaluated controller through the existing `Until` layer path. Registration rejects static, source-bound, and non-end-of-turn installation. Engine tests cover identity, phasing, source departure, timestamps, expiry, control prohibition, copied spells, clones, validation, and AI outcome metadata. Card migration remains outside this tracker’s scope.
- [ ] S1: Typed spell target and counter action.
- [ ] G1: Typed graveyard card target and return to hand.
- [ ] G2: Typed graveyard return to battlefield.

## Optional follow-up

- [x] Typed ability removal. `RemoveAbilities(PermanentRef)` and `RemoveAbilitiesEach(PermanentSet)` reuse the Layer 6 removal helper. Temporary sets capture incarnations, static sets update dynamically, and later grants restore abilities. Engine tests cover restrictions, source departure, phasing, clone expiry, and validation. No card changes were made.
