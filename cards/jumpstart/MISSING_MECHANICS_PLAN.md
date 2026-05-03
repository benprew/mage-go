# JMP Missing-Mechanics Implementation Plan

Survey of `cards/jumpstart/` turned up ~32 cards with `// XXX:` engine gaps and 4 broken tests
(`// FIXME:` / `t.Skip`). The plan below groups them by the composable primitive they need,
ordered so foundational primitives land first and downstream cards become trivial.

Guiding principle: **prefer adding small composable predicates / event types / replacement
shapes over per-card specials**. Where possible, reuse the existing
`TriggerConditionData` predicate algebra (`AndTriggerCond` / `OrTriggerCond` / `NotTriggerCond`)
and the existing replacement-action interception list (`DamageToPlayerAction`,
`DamageToCreatureAction`, `DestroyPermanentAction`, `LifeGainAction`, `DrawCardAction`,
`AddCountersAction`).

### Comp Rules anchors (referenced throughout)

- **CR 119.5** — starting life total is a per-player constant set at game start.
- **CR 601.2f** — total cost of a spell is calculated *after* targets and modes are chosen,
  then locked in; modifiers may add or subtract.
- **CR 603.1f** — a triggered ability can be modal; the mode is chosen as the ability goes
  on the stack.
- **CR 603.2c** — "Whenever one or more …" abilities trigger once for the *batch* of events,
  not once per event.
- **CR 603.3d** — targets for a triggered ability are chosen as it's put on the stack.
- **CR 603.6c / 603.10** — leaves-the-battlefield triggers use last-known information from
  immediately before the source left; they fire from the battlefield, not the destination zone.
- **CR 603.7** — delayed triggered abilities trigger when their condition is met, regardless
  of where their source is.
- **CR 608.2g** — when a spell or ability resolves, it uses values that were set when it was
  put on the stack (when those values are referenced by the effect).
- **CR 614.1c** — self-replacement effects, including "as ~ enters" and "~ enters with N
  counters", are applied once during the ETB process; multiple such effects combine
  (CR 614.5 / 122.1d for counter combination).
- **CR 615** — prevention effects are distinct from replacement effects. "Prevent all
  noncombat damage" is a prevention shield, not a replacement.
- **CR 701.8** — discard fires once per card discarded; multi-card discard produces multiple
  events.
- **CR 707** — face-down spells / permanents and the privacy rules ("look at" vs reveal).

---

## Tier 0 — Bug fixes in existing primitives

These are not new mechanics, they are *broken* primitives blocking real cards.

### 0.1 ETB-trigger event multiplication (Champion of Lambholt)
- `creatures_part3_test.go:539` — "trigger fires 3x on a single creature ETB".
- Likely cause: the `EvtZoneChange` for ETB fires multiple times along the
  hand→stack→battlefield path, or the trigger predicate matches on more than the
  final transition.
- Fix: in `pkg/mage/triggered.go` confirm `WheneverPermanentEntersBattlefieldTrigger`
  filters on `evt.To == ZoneBattlefield && evt.From != ZoneBattlefield`, and that
  ETB events are emitted exactly once per zone transition. Add a regression test
  using a minimal "counter on ETB" trigger.
- Unblocks: Champion of Lambholt; many other ETB triggers silently get extra fires.

### 0.2 Attack trigger does not fire (Hellrider)
- `creatures_part3_test.go:319` — "whenever a creature you control attacks" trigger
  does not fire on declared attackers.
- Likely cause: today's "you control attacks" trigger only fires on the source's own
  `EvtDeclaredAttacker`, not on every controlled attacker.
- Fix: add/repair `WheneverCreatureYouControlAttacksTrigger(effect, optional)` on
  `EvtDeclaredAttacker` with `EventSourceControlledByController`. Resolution must
  consult the attack assignment (`g.AttackAssignments[evt.SourceID]`) to determine
  "the player or planeswalker it's attacking" per CR 506.4 — not Hellrider's own
  source ID.
- Unblocks: Hellrider, future "whenever a creature you control attacks" cards.

### 0.3 Discard trigger fires too often (Fell Specter)
- `creatures_part2_test.go:624` — "got 16, want 18" actually means under-fires (in
  test harness `Want` is canonical). Re-read the test before fixing; symptom is
  `WheneverOpponentDiscardsTrigger` count mismatch.
- Fix: per CR 701.8, each card discarded is its own event; ensure `EvtDiscard` fires
  exactly once per card and that the trigger's effect resolves once per event (not
  once per card-in-hand and not aggregated across a multi-discard cost). Distinct from
  the "one or more" batching pattern in 1.7 — that's an aggregation *consumer*; this
  is making sure the *producer* emits the right cardinality.
- Unblocks: Fell Specter, any per-discard counters / drains.

### 0.4 `AddETBAdditionalCounters` no-op when base counters absent (Oona's Blackguard)
- `creatures.go:2918`. Engine's "additional ETB counters" replacement augments an
  existing `AddCountersAction`; if the entering creature has none, the replacement
  never fires.
- Fix: model "enters with N counters" as a CR 614.1c self-replacement that *always*
  runs on the ETB process — not as an augmenter of a queued `AddCountersAction`. Per
  CR 614.5 / 122.1d, multiple such effects combine into a single application. The
  practical change: `AddETBAdditionalCounters` should always synthesize an
  `AddCountersAction` if none exists, then merge counts.
- Unblocks: Oona's Blackguard, Winding Constrictor edge cases, future Hardened
  Scales-style cards interacting with creatures that have no native counters.

---

## Tier 1 — Composable primitives (foundational, unblock many cards)

These each add one orthogonal capability. They're ordered by dependency depth.

### 1.1 Triggered abilities while-in-zone (graveyard, exile, hand)
- New: extend the trigger registry to key on `(sourceID, activeZone)` rather than
  implicit battlefield. Add a `Zone` field to trigger registration and check it
  inside the dispatch loop.
- Note the CR 603.6c / 603.10 distinction: a *dies* / *leaves-battlefield* trigger
  (e.g. Living Lightning) is **not** a graveyard trigger — it fires from the
  battlefield using last-known information. Don't use 1.1 for those; fix them via
  0.1 / proper LTB plumbing.
- 1.1 is for abilities whose Oracle text functions specifically while the source is
  in graveyard/exile/hand (Pia Nalaar's "at the beginning of your end step, if … ,
  you may pay {R}; if you do, return ~ from your graveyard"; flashback-style
  registrations; Squee-style hand triggers).
- Convenience constructors:
  - `WhileInZoneTrigger(zone, evtType, effect, conds...)`
  - `BeginningOfYourEndStepFromGraveyard(effect, optional)` (composes 1.1 + existing
    `EvtEndStep` + `EventPlayerIsController`)
- Unblocks: Pia Nalaar (`creatures.go:4144`).

### 1.2 Delayed triggered abilities (CR 603.7)
- New primitive in `pkg/mage/triggered.go`:
  `RegisterDelayedTrigger(g, sourceID, controllerID, evtType, expiresAt, effect, conds...)`
  with `expiresAt` defaulting to `EndOfTurn`.
- Per CR 603.7a, a delayed triggered ability fires when its condition occurs and
  is *not* gated on its source's current zone — this is the key distinction from
  1.1. Cleanup happens on the configured `expiresAt` (typically `EvtCleanup`); a
  one-shot delayed trigger that has fired is not re-armed.
- Multiple instances of the same delayed trigger (e.g. one per creature targeted by
  Hunter's Insight) coexist independently — registrations are not de-duplicated.
- Unblocks: Hunter's Insight (`spells.go:977`), and any "this turn, when ~ does X,
  …" effect.

### 1.3 Cast-time spell context capture (CR 608.2g)
- Several cards reference state "as you cast this spell"; CR 608.2g says the
  resolver uses values fixed when the spell went on the stack. Today the resolver
  re-queries live state and gets the wrong answer when state has changed between
  cast and resolution.
- Add `Spell.CastContext` populated at cast-on-stack time:
  - `ControlledTypesAtCast map[CardType]bool` / `ControlledSubtypesAtCast map[SubType]bool`
  - `ManaSpentByColor [6]int` (extend whatever's already partially tracked)
  - `RevealedAtCast []*Card` (for additional-cost reveals like Draconic Roar)
- Effects then read `ResolvingSpell.CastContext` instead of querying live state.
- Unblocks: Draconic Roar (`spells.go:529`), Chamber Sentry's "colors of mana spent"
  (`creatures.go:6059`), and any future "as you cast" rider.

### 1.4 Cost calculation after targets (align with CR 601.2f)
- CR 601.2f says total cost is calculated *after* modes and targets are chosen,
  then locked in. Today the engine evaluates cost-reduction predicates before
  targets, so target-conditional reducers (Outmumber's "costs {2} less if it
  targets a Dinosaur you control") never apply.
- Approach: collapse cost calculation to a single post-target pass. Static
  reducers and target-conditional reducers both consult the chosen targets;
  the locked-in total is what the player pays.
- New helper: `WithTargetConditionalCostReduction(amount, predicate func(targets, g) bool)`.
- Unblocks: Outmumber (`spells.go:1592`), and any future target-conditional
  cost reducer.

### 1.5 Modal triggered abilities — mode then targets, both at stack time
- Per CR 603.1f the mode is chosen as the trigger goes on the stack; per CR 603.3d
  targets are chosen at the same time. The bug today is that the engine gathers
  targets for *all* modes up-front, then asks for mode — violating CR 603.3d's
  "targets only for the chosen mode".
- Fix: extend `TriggeredAbility` with `Modes []Mode` where each `Mode` carries its
  own target spec. At stack-placement time, prompt for mode first, then run target
  selection for that mode only.
- Unblocks: Entomber Exarch (`creatures.go:2218`), Primeval Bounty's modal pattern
  (`enchantments.go:693`), and any future modal trigger.

### 1.6 Per-player, per-turn state flags
- New: `Game.PlayerTurnFlags map[PlayerID]map[FlagKey]any` cleared on `EvtCleanup`.
  Suggested keys:
  - `FlagCastSpellThisTurn` (bool)
  - `FlagAttackedWithCreatureThisTurn` (bool)
  - `FlagFirstCardDrawnThisTurn` (bool — set on first `EvtCardDrawn` per player per turn)
- Dispatcher hooks set the flag; trigger conditions / restriction predicates read it.
- Composable predicates to add: `EventPlayerCastSpellThisTurn`,
  `EventPlayerAttackedThisTurn`, `IsFirstCardDrawnThisTurnByPlayer`.
- Unblocks: Angelic Arbiter (`creatures.go:136`), Zurzoth (first-card-drawn portion;
  `creatures.go:4672`), and future "this turn" / "first time each turn" effects.

### 1.7 "One or more …" event aggregation (CR 603.2c)
- CR 603.2c: a "Whenever one or more …" ability looks at the whole event batch
  and triggers exactly once for that batch. The aggregated objects are visible to
  the effect (e.g. "draw a card" once, but for-each effects use the count).
- Approach: add `BatchedTrigger(evtType, bucketKey func(evt) string, effect, conds...)`
  that collects matching events emitted during the same dispatch tick (or until
  the next state-based-action / priority pass) and fires once with the aggregated
  set bound to the effect closure as `evt.Batch []Event`.
- Bucket key supports the "fight or become blocked" pattern: one bucket per
  ability instance per dispatch window.
- Unblocks: Neyith of the Dire Hunt (`creatures.go:5279`), Path of Bravery
  (`enchantments.go:653`), and future "one or more" cards. Inniaz remains deferred
  (multiplayer). Soul of the Harvest's closure-capture bug is largely independent
  and tracked under 3.7.

### 1.8 Random selection primitive
- Engine already has deterministic test choices; add `Game.RandomChoice([]T) T`
  hooked through `Game.Rand` (seedable for tests). Use it for:
  - Random discard (Zurzoth's second clause; `creatures.go:4672`)
  - Reveal-random / Squee-style (already partially via existing
    `reveal_random_test.go`; verify the helper is exposed as a primitive)
- Unblocks: Zurzoth (random-discard clause), Bottomless Pit-style.

---

## Tier 2 — Mid-level mechanics (compose Tier 1 primitives)

### 2.1 Sacrifice-attached + simultaneous token creation
- Aura "enchanted creature's controller sacrifices it and you create a …" needs the
  sacrifice to resolve and *the same effect* to create the token. The sacrifice is
  performed by the *enchanted creature's controller* (typically an opponent), while
  the token is created under the *aura controller*.
- Add helper: `SacrificeAttachedThenCreateToken(tokenSpec)` that pulls the
  attached permanent's controller from the attach link, queues the sacrifice
  against that player, and creates the token under the aura's controller in the
  same effect (one resolution, one stack item).
- Unblocks: Parasitic Implant (`enchantments.go:644`).

### 2.2 Aura attaches to a freshly-created token (Ajani's Chosen)
- Composition: `CreateToken` returns the new permanent ID; pass it to
  `AttachAura(auraID, targetID)`. Engine has both halves.
- Add helper: `OnEnchantmentETB_AttachIfAura(tokenSpec)`.
- Unblocks: Ajani's Chosen (`creatures.go:61`).

### 2.3 Search library + put-tapped onto battlefield
- Add `PutOntoBattlefieldTapped` option to existing
  `SearchLibraryAndPutOntoBattlefield`. Most search effects already pipe through a
  single helper — this is a flag.
- Unblocks: Path to Exile (`spells.go:1352`).

### 2.4 Bottom-of-library in chosen order
- Add `PutOnBottomInChosenOrder(player, []CardID)` alongside the existing
  random-order helper. Plumbs a per-card ordering choice through the choice system.
- Unblocks: Commune with Dinosaurs (`spells.go:389`), and many future "in any order"
  effects (scry/surveil derivatives, fetchland-style shuffles do not need this).

### 2.5 Play-from-top-of-library + library-top-revealed
- Two distinct features that often appear together:
  - `Game.AddRevealedTopCardEffect(player)` — toggle that makes the top card visible
    to all players (CR 401).
  - `Game.AddPlayLandsFromZone(player, zone)` — extends the legal-zones list for
    land plays (currently `ZoneHand` only).
- Unblocks: Oracle of Mul Daya (`creatures.go:5293`).
- Composes with 2.6.

### 2.6 As-enters-choose-a-subtype + upkeep top-card reveal/draw
- "As ~ enters, choose a creature type" is a CR 614.1c self-replacement: the
  choice is made as part of the ETB process, before the permanent is on the
  battlefield. Add `WithEntersWithChosenSubtype(category SubtypeCategory)` which
  stores the choice on the permanent's attrs as `ChosenSubtype`.
- "At the beginning of your upkeep, look at the top card of your library; if it's
  …, you may reveal and draw it" composes 2.5's reveal helper with a normal upkeep
  trigger that reads `ChosenSubtype` from attrs.
- Unblocks: Herald's Horn (`artifacts.go:170`).

### 2.7 Starting life total tracking (CR 119.5)
- CR 119.5 defines starting life total as a per-player constant set at game start
  (default 20, modifiable by format / Vanguard). Add `Player.StartingLife int`
  set during game initialization. Add predicate
  `LifeGreaterOrEqualToStarting(playerID)`.
- Unblocks: Path of Bravery's first clause (`enchantments.go:653`); the second
  clause uses 1.7 (one-or-more attack aggregation).

### 2.8 Prevent all noncombat damage (CR 615 prevention, not CR 614 replacement)
- This is a *prevention* effect (CR 615), distinct from a replacement effect.
  Today's `DamageToPlayerAction` / `DamageToCreatureAction` interception covers
  both flavors mechanically, but the data flow needs to surface a prevention
  predicate that runs ahead of replacements and short-circuits the action.
- New shape: `AddDamagePreventionRule(predicate func(evt DamageEvent, g *Game) bool)`
  returning an unregister handle. For Blessed Sanctuary, the predicate is
  `!evt.IsCombat && (evt.Target == controller || g.IsControlledBy(evt.Target, controller))`.
- Distinguishing 614 vs 615 matters for stacking interactions (multiple shields,
  "if damage would be dealt, instead …" replacements, etc.); see CR 616 for order.
- Unblocks: Blessed Sanctuary (`enchantments.go:116`).

### 2.9 Combat restriction: "must be blocked this combat if able"
- Already adjacent: `MustBeBlockedIfAble` exists for static "must be blocked" but
  per-combat one-shot variants (Bullwhip, Neyith) need the same predicate scoped
  to a single combat phase.
- Add `Game.AddOneCombatRestriction(restriction)` cleared on `EvtEndOfCombat`.
- Unblocks: Neyith's third clause (`creatures.go:5279`).

### 2.10 Activated-ability suppression by controller
- New continuous effect at Layer 6 (Ability):
  `SuppressActivatedAbilities(filter CardFilter)` removes ability legality from
  matching permanents. Filter for "creatures opponents control".
- Unblocks: Linvala (`creatures.go:590`).

---

## Tier 3 — Higher-friction mechanics (defer or scope down)

### 3.1 Face-down exile + privacy (CR 707, CR 408)
- Gonti, Lord of Luxury (`creatures.go:2483`). Requires (a) face-down exile
  (CR 707 — face-down spells/permanents and the rule that face-down cards in
  a public zone have no characteristics visible to opponents); (b) "look at"
  privacy (CR 408 — only the indicated player learns the card's identity);
  (c) "may cast from exile, any mana type" alternate cast path that reads the
  exile-zone owner / face-down state correctly.
- **Scope:** large (visibility model, alternate-cost casting, zone-tagged
  castability). Defer until at least three cards demand it.

### 3.2 Loyalty counters / planeswalker primitive
- Settle the Score (`spells.go:1618`). Today only the exile half is implemented.
- **Scope:** introducing planeswalkers is a sizable subsystem (loyalty counter
  type, redirect-damage rules CR 306.7, attack-planeswalker option). Defer; for
  Settle the Score in the meantime, leave the `// XXX:` and ship the exile half.

### 3.3 Cast-without-paying-many-spells from exile (Etali, Primal Storm)
- `creatures.go:3746`. The plumbing problem is that
  `CastCardFromZoneWithoutPaying` validates targets/cost the same as a normal
  cast and short-circuits when illegal. Extend the helper with a
  `permitNoLegalTargets` flag (CR 117.4 — "can't cast" if no legal targets;
  Etali specifies "you may", so each individual cast still must be legal).
- Achievable but invasive. Schedule after 1.3 (cast-time context) lands so we
  don't double-touch the cast pipeline.

### 3.4 Card-level alternate-cost registration ("you may cast … by paying …")
- `creatures.go:3063`. Add `Card.AlternateCosts []AlternateCost` consulted by
  the cast-spell action enumerator. Each `AlternateCost` carries a zone, a
  cost expression, and an optional condition.
- Once this exists, Gargoyle Castle, Foundry Inspector-class flashback variants,
  and Etali sub-effects all simplify.

### 3.5 Different-names tracking + draw-from-empty replacement (Ormos)
- `creatures.go:1304`. Two pieces:
  - "Different names among …" — a small helper `CountDistinctNames([]CardID)`.
  - Draw-replacement when library empty — a new `DrawCardAction` replacement
    branch that fires only when `len(library) == 0`.
- Both small, but Ormos also wants its activated ability with split sub-modes,
  so wait until 1.5 (modal) lands before wiring.

### 3.6 Multiplayer "player to your right"
- Inniaz (`creatures.go:1138`). 2-player engine; explicitly **out of scope**.
  Leave the `// XXX:` note and skip.

### 3.7 Selvala / Soul of the Harvest closure capture
- `creatures.go:5486`, `5608`. Both stem from trigger-condition closures that
  capture state at registration. Once 0.1 (ETB de-duplication) and 1.7 (batching)
  land, revisit; the proper fix is to read the closed-over filter through the
  game state at fire time, not at registration.

---

## Implementation order (recommended)

1. **Tier 0 bug fixes** — quickest wins, unblock the four broken tests.
2. **1.1 graveyard triggers**, **1.2 delayed triggers**, **1.6 per-player flags** —
   independent foundations; can land in parallel.
3. **1.3 cast-time context**, **1.4 post-target cost reduction**, **1.5 modal
   triggers** — together they make spell resolution composable.
4. **1.7 batched events**, **1.8 random** — small additions on top.
5. **Tier 2** — each is a 1–2-day card-package wave that consumes Tier 1
   primitives. Order doesn't matter much; pick by which cards you want first.
6. **Tier 3** — schedule per-card based on demand. 3.6 (multiplayer) is permanently
   out.

## Card → primitive map (quick reference)

| Card | File:line | Needs |
|---|---|---|
| Champion of Lambholt | creatures.go (test 539) | 0.1 |
| Hellrider | creatures.go (test 319) | 0.2 |
| Fell Specter | creatures.go (test 624) | 0.3 |
| Oona's Blackguard | creatures.go:2918 | 0.4, 1.7-ish trigger |
| Living Lightning | creatures.go (test 373) | 0.1 (this is on-BF) |
| Living Lightning | creatures.go (test 373) | 0.1 (LTB trigger via CR 603.6c, *not* 1.1) |
| Pia Nalaar | creatures.go:4144 | 1.1 (true while-in-graveyard) |
| Hunter's Insight | spells.go:977 | 1.2 |
| Draconic Roar | spells.go:529 | 1.3 |
| Chamber Sentry | creatures.go:6059 | 1.3, 0.4 |
| Outmumber | spells.go:1592 | 1.4 |
| Entomber Exarch | creatures.go:2218 | 1.5 |
| Angelic Arbiter | creatures.go:136 | 1.6 |
| Zurzoth | creatures.go:4672 | 1.6, 1.8 |
| Neyith of the Dire Hunt | creatures.go:5279 | 1.7, 2.9, may-cost |
| Path of Bravery | enchantments.go:653 | 1.7, 2.7 |
| Soul of the Harvest | creatures.go:5608 | 3.7 (closure-capture; revisit after 0.1) |
| Selvala, Heart of the Wilds | creatures.go:5486 | 3.7 (entering-creature controller binding) |
| Parasitic Implant | enchantments.go:644 | 2.1 |
| Ajani's Chosen | creatures.go:61 | 2.2 |
| Path to Exile | spells.go:1352 | 2.3 |
| Commune with Dinosaurs | spells.go:389 | 2.4 |
| Oracle of Mul Daya | creatures.go:5293 | 2.5 |
| Herald's Horn | artifacts.go:170 | 2.5, 2.6 |
| Blessed Sanctuary | enchantments.go:116 | 2.8 |
| Linvala | creatures.go:590 | 2.10 |
| Towering Titan | creatures.go:5711 | new "ETB with X = sum-toughness" replacement |
| Thirst for Knowledge | spells.go:1730 | branching discard cost (small, card-local) |
| Primeval Bounty | enchantments.go:693 | composes 1.5 + landfall + cast-spell triggers |
| Gonti, Lord of Luxury | creatures.go:2483 | 3.1 (defer) |
| Settle the Score | spells.go:1618 | 3.2 (defer) |
| Etali, Primal Storm | creatures.go:3746 | 3.3 (after 1.3) |
| Gargoyle-class alt-cost | creatures.go:3063 | 3.4 |
| Ormos | creatures.go:1304 | 3.5 (after 1.5) |
| Inniaz | creatures.go:1138 | 3.6 (out of scope) |

