# RON player reference contract

The compiler has a typed player reference plan for the existing single-target
action context. This contract does not establish multiple groups or modes.

Supported declarations and reads:

- `Targeted([TargetPlayer(AnyPlayer)], ActionList([...]))` declares one player.
- `PlayerAction(Ref_TargetPlayer, DrawNumberCards(Integer(n)))` reads that player.
- `PlayerAction(Ref_TargetPlayer, DrawNumberCards(ValueX))` reads that player
  and the announced spell X. Activation X still requires a payment adapter.
- `PlayerAction(You, DrawNumberCards(...))` reads the stored resolving controller.
- Direct `DrawNumberCards(...)` also reads the stored resolving controller.

These forms use the existing spell, activation, and supported trigger action
builders. The player declaration belongs to that action scope. A different
rule cannot read it. Wrong reference types, undeclared references, and tuple
forms of a symbol reference fail compilation with source positions. Existing
nested and multiple target declarations remain rejected.

The immutable `playerTargetPlan` supplies declaration identity. Each action
build creates a fresh `mage.PlayerTargetBinding`. Its `instanceBindings` map
connects reference plans to this instance's declaration. No selected UUID or
other game state belongs to the compiled plan. `mage.WithTargets` supplies the
engine declaration and `mage.DrawFor` reads it. Engine definition validation
can therefore detect an absent target declaration, rather than treating the
read as an implicit first target.

The engine keeps chosen players in the resolving object's context. Spell copies
can choose a different player without changing the original. Game clones keep
independent draws and target choices. Source departure does not erase an
activation's player target. Draws occur during resolution; announcement and
payment still use the engine's normal target and cost services.

Real RON evidence covers Ancestral Recall and Braingeyser. Synthetic evidence
covers sibling `You` actions, separate factory calls, declaration removal, invalid
scopes, an activation whose source leaves before resolution, and a targeted
entry trigger. Trigger constructors receive the completed action sequence so
engine validation sees no nil placeholder. Real Onulet and Su-Chi death-trigger
tests verify this prerequisite repair. The contract
uses CR 608.2b/c/n for legality, instruction order, and final graveyard movement;
CR 113.7a permits an ability to resolve after its source leaves.

Player-scoped life actions, explicit `Player`/`Permanent` wrappers, controller
and owner queries, event references, loop players, modes, and multiple groups
remain outside this slice. This contract adds no new engine rules or production
registry. Permanent, spell, and union-target reads still need further typed-plan
migration. Compilation counts do not establish full card behavior coverage.
