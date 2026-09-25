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
  - [x] Stabilize the shared token entry path: legacy token constructors copy characteristic slices, `NewToken` owns its type and subtype slices, and rejected battlefield entries do not panic. Token status is set by `NewPermanent` before entry processing. Engine regressions cover input mutation and prohibited land-token entry for normal, attacking, and blocking creation.
  - [x] Give each token created by a legacy token effect its own ability instance and ID. Entry setup now assigns the source and controller on that instance without changing the declaration or another token's ability.
  - [x] Make legacy multi-token creation enter as one group. The shared entry path prepares accepted tokens, registers all of them, then processes entry effects and events. `TestCreatedTokensAreVisibleTogetherToEntryTriggers` verifies that each entry observer sees both tokens.
  - [x] Add immutable `DefineToken` characteristics and `CreateTokenCopies(PlayerRef, ResolutionValue, TokenDefinition)`. The typed effect uses the shared batch entry service, evaluates the count once, and creates no tokens for nonpositive counts. Registration checks missing or conflicting characteristics and isolates nested token ability bindings. Engine tests cover recipient, dynamic and nonpositive counts, exact characteristics, input mutation, independent abilities, clone isolation, and simultaneous entry observation. No cards are migrated in this tracker.
- [x] C1: Temporary control modification. `ChangeController(PermanentRef, PlayerRef)` captures the target incarnation and evaluated controller through the existing `Until` layer path. Registration rejects static, source-bound, and non-end-of-turn installation. Engine tests cover identity, phasing, source departure, timestamps, expiry, control prohibition, copied spells, clones, validation, and AI outcome metadata. Card migration remains outside this tracker’s scope.
- [x] S1: Typed spell target and counter action. `TargetSpell` binds a stack object by its stack ID, and `Counter` uses that binding through the shared counter service. Filters apply at announcement and resolution. Engine tests cover target order, uncounterable spells, removed and recast subjects, copies, retargeting, clones, mode scope, validation, and compile-time category rejection. The legacy counter path still counters abilities for ward. Card migration remains outside this tracker's scope.
- [x] G1: Typed graveyard card target and return to hand. `TargetGraveyardCard` defaults to your graveyard and accepts `InAnyGraveyard` and `GraveyardCardFilter` options. `ReturnCardToHand` moves the referenced card through the shared graveyard movement service to its owner's hand. Graveyard visits now have distinct IDs in the shared target path, including copies and clones. Engine tests cover target order, ownership, scope, filtering, stale visits, validation, and clone isolation. Card migration remains outside this tracker's scope.
- [x] G2: Typed graveyard return to battlefield. `ReturnCardToBattlefield(CardRef, PlayerRef)` resolves the bound graveyard visit and selected controller, then uses the shared graveyard movement and battlefield entry paths. The shared path keeps a land in its graveyard if a rule prohibits entry. Engine tests cover target order, ownership, chosen controller, entry state, entry prohibition, stale visits, clone isolation, validation, and compile-time category rejection. Card migration remains outside this tracker's scope.

## Optional follow-up

- [x] Typed ability removal. `RemoveAbilities(PermanentRef)` and `RemoveAbilitiesEach(PermanentSet)` reuse the Layer 6 removal helper. Temporary sets capture incarnations, static sets update dynamically, and later grants restore abilities. Engine tests cover restrictions, source departure, phasing, clone expiry, and validation. No card changes were made.
