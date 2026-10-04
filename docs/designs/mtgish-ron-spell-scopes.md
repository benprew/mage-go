# RON single spell targets and counter actions

Capability: P0-SCOPE-SPELL-FILTERS. Status: complete for the shapes below.

## Selection from fresh evidence

Baseline: 2,445 records, 986 unique names, 844 compiled records, 1,601 rejected
records, 246 compiled unique names. Focused RON tests and `make build` passed.
`go run ./cmd/loadcards -json` rejected the collection with exit 1. This is
collection rejection, not a build failure. Nested AST inventory came from
`go run ./cmd/loadcards -inventory -json`; names below count reprints once.

| Candidate | Distinct affected names | Estimated new full-card compilations | Other blockers | Downstream benefit | Estimated implementation and tests |
| --- | ---: | ---: | --- | --- | --- |
| Spell targets and filters (selected) | 28 | 5: Artifact Blast, Deathgrip, Flash Counter, Lifeforce, Remove Soul | Modes, copy/text actions, optional payment, delayed triggers, controller/target-history and mana-value filters | Typed spell identity for controller/value readers, counter modes and spell copies | Small adapter and binding change; medium legality/copy/clone tests |
| Temporary layer references | 107 | 0; 35 already compile | Dynamic P/T, durations, rule effects, event refs, decisions, costs and filters | Typed subjects for 35 accepted cards and later dynamic boosts, removal and type changes | Small adapter change; medium identity/attachment/cleanup tests |
| Player life actions | 48 | 1: Stream of Life | Modes/prevention for Healing Salve and Alabaster Potion; controller snapshots for Swords and Crumble; damage results, events and payment scopes elsewhere | Explicit players for modal life, event life and saved-controller actions | Small adapter change; medium player/X/copy/legality tests |

All 28 complete spell-target records were read, including nested forms after
first errors. The competing life records were also read. The layer inventory
and complete records show that reference lowering alone does not accept another
card. The selected slice offers the largest credible immediate gain for similar
effort and supplies a reusable typed identity needed by multiple later cards.
The phase order and previous ledger suggestion did not fix this selection.

### Temporary layer references names

Arcades Sabboth, Ashnod's Battle Gear, Atog, Battering Ram, Berserk, Blessing, Blood Lust, Carrion Ants, Cave People, Coral Helm, Cyclopean Tomb, Deep Spawn, Dragon Engine, Dragon Whelp, Dwarven Lieutenant, Dwarven Soldier, Electric Eel, Elven Fortress, Elven Lyre, Emerald Dragonfly, Erhnam Djinn, Fallen Angel, Fire Drake, Firebreathing, Flying Carpet, Frozen Shade, Gabriel Angelfire, Gaea's Liege, Ghosts of the Damned, Giant Growth, Giant Shark, Giant Slug, Glyph of Delusion, Glyph of Destruction, Goblin Balloon Brigade, Goblin Flotilla, Goblin Kites, Goblin Wizard, Granite Gargoyle, Great Defender, Halfdane, Hammerheim, Helm of Chatzuk, Holy Armor, Homarid Warrior, Howl from Beyond, Icatian Infantry, Icatian Lieutenant, Icatian Priest, Icatian Scout, Island of Wak-Wak, Jade Statue, Johan, Jump, Killer Bees, Kry Shield, Lesser Werewolf, Mishra's Factory, Murk Dwellers, Nalathni Dragon, Orcish Captain, Orcish Veteran, Order of Leitbur, Order of the Ebon Hand, Osai Vultures, Pavel Maliki, Pendelhaven, Pixie Queen, Pradesh Gypsies, Radjan Spirit, Rapid Fire, Righteousness, River Merfolk, Runesword, Sandals of Abdallah, Scarwood Hag, Shelkin Brownie, Shivan Dragon, Singing Tree, Sorceress Queen, Spirit Shield, Spitting Slug, Staff of Zegon, Stone Giant, Subdue, Svyelunite Priest, Tawnos's Weaponry, Thallid Devourer, Tolaria, Transmutation, Tuknir Deathlock, Urborg, Urza's Avenger, Vaevictis Asmadi, Vampire Bats, Vodalian Knights, Vodalian War Machine, Wall of Caltrops, Wall of Fire, Wall of Opposition, Wall of Water, Wall of Wonder, War Barge, Wormwood Treefolk, Wyluli Wolf, Xenic Poltergeist, Zelyon Sword.

### Player life actions names

Alabaster Potion, Axelrod Gunnarson, Balm of Restoration, Crumble, Crystal Rod, Dark Heart of the Wood, Delif's Cone, Diamond Valley, Divine Offering, Drain Life, El-Hajjâj, Elvish Farmer, Farmstead, Fasting, Fountain of Youth, Glyph of Life, Grave Robbers, Healing Salve, Iron Star, Ivory Cup, Ivory Tower, Lich, Life Chisel, Lifeblood, Lifetap, Living Artifact, Merchant Ship, Nafs Asp, Onulet, Personal Incarnation, Powerleech, Psychic Purge, Relic Bind, Reverse Damage, Reverse Polarity, Shahrazad, Simulacrum, Sol'kanar the Swamp King, Soul Net, Spirit Link, Spiritual Sanctuary, Stream of Life, Swords to Plowshares, Syphon Soul, Tablet of Epityr, Throne of Bone, Urza's Chalice, Wooden Sphere.

### Spell targets and filters names

Artifact Blast, Avoid Fate, Blue Elemental Blast, Chaoslace, Counterspell, Deathgrip, Deathlace, Equinox, Flash Counter, Force Spike, Fork, Goblin Artisans, Lifeforce, Lifelace, Magical Hack, Mana Drain, Power Sink, Purelace, Red Elemental Blast, Reflecting Mirror, Remove Soul, Reverberation, Ring of Immortals, Sleight of Mind, Spell Blast, Thoughtlace, Thrull Wizard, Vodalian Mage.

## Contract and reuse boundary

- Actions: `CounterSpell(symbol Ref_TargetSpell)` in `ActionList/1`, under one
  `Targeted([TargetSpell(filter)], actions)` declaration, in spells, activations,
  and supported triggers. The reference is local to that declaration and rule.
- Filters: symbol `AnySpell`; `IsCardtype(symbol)/1`; `IsColor(symbol)/1`;
  recursive `And([filters])/1` and `Or([filters])/1` with nonempty lists.
  Card types and colors use the existing validated metadata vocabulary.
- Only stack spells match. Abilities, absent objects, incompatible references,
  numbered references, malformed operands, empty conjunctions/disjunctions,
  undeclared reads and cross-rule reads fail declaration or target validation.
- Each factory builds its own `DeclareSpellTarget` and typed `Counter` read.
  The immutable plan identity indexes the per-instance binding map. The engine
  normalizes declared targets and supplies each resolution frame's choices.
- Filters read the spell card through `CardFilter`, `HasColorCardFilter`,
  `AndCardFilter`, and `NewCardFilter`. RON owns recursive grammar lowering.
  The engine owns stack eligibility, announcement/resolution legality, target
  capture, copies, clones, costs, uncounterability and owner graveyard movement.
- No engine contract is missing, and no engine implementation changes are needed.
  No fallback registry, card-name branch, new global state, or DSL removal exists.

Dependencies: P0-SCOPE-PLAYER/PERMANENT per-instance action lowering,
`DeclareSpellTarget`, `StackObjectRef`, `Counter`, and engine target normalization.
CR 608.2b/c and 701.6a/b require legality rechecks, ordered resolution, owner
movement and no refund. CR 113.7a preserves a queued ability after its source
leaves or phases. CR 707.10 supplies copied spell identity. A stack spell has no
battlefield incarnation or attachment; permanent LKI, target phasing,
simultaneous batches and replacement ordering introduce no new contract here.
The existing counter service handles an uncounterable target. Old Go callers
remain intact, including the legacy source-ID counter service.

## Affected card outcomes

| Unique card | Outcome or remaining blocker |
| --- | --- |
| Artifact Blast | Newly compiled and gameplay tested |
| Avoid Fate | Still rejected: `unsupported spell target filter (IsEnchantmentType)` |
| Blue Elemental Blast | Still rejected: `unsupported action list (Modal_ChooseOne)` |
| Chaoslace | Still rejected: `unsupported action list (Targeted_Modal)` |
| Counterspell | Remains compiled; Counterspell gameplay regression |
| Deathgrip | Newly compiled and gameplay tested |
| Deathlace | Still rejected: `unsupported action list (Targeted_Modal)` |
| Equinox | Still rejected: `unsupported static ability grant (Activated)` |
| Flash Counter | Newly compiled and gameplay tested |
| Force Spike | Still rejected: `unsupported action (PlayerMayCost)` |
| Fork | Still rejected: `unsupported action (CopySpellAndMayChooseNewTargetsWithEffects)` |
| Goblin Artisans | Still rejected: `unsupported spell target filter (ControlledByAPlayer)` |
| Lifeforce | Newly compiled and gameplay tested |
| Lifelace | Still rejected: `unsupported action list (Targeted_Modal)` |
| Magical Hack | Still rejected: `unsupported action list (Targeted_Modal)` |
| Mana Drain | Still rejected: `unsupported action (CreateFutureTrigger)` |
| Power Sink | Still rejected: `unsupported action (PlayerMayCost)` |
| Purelace | Still rejected: `unsupported action list (Targeted_Modal)` |
| Red Elemental Blast | Still rejected: `unsupported action list (Modal_ChooseOne)` |
| Reflecting Mirror | Still rejected: `unsupported rule (ActivatedWithModifiers)` |
| Remove Soul | Newly compiled and gameplay tested |
| Reverberation | Still rejected: `unsupported action (CreateReplaceWouldDealDamageUntil)` |
| Ring of Immortals | Still rejected: `unsupported spell target filter (IsEnchantmentType)` |
| Sleight of Mind | Still rejected: `unsupported action list (Targeted_Modal)` |
| Spell Blast | Still rejected: `unsupported spell target filter (ManaValueIs)` |
| Thoughtlace | Still rejected: `unsupported action list (Targeted_Modal)` |
| Thrull Wizard | Still rejected: `unsupported action (PlayerMayCost)` |
| Vodalian Mage | Still rejected: `unsupported action (PlayerMayCost)` |

Other first errors advance without full-card acceptance:

- Avoid Fate: `unsupported spell target filter (And)` -> `unsupported spell target filter (IsEnchantmentType)`.
- Fork: `unsupported spell target filter (Or)` -> `unsupported action (CopySpellAndMayChooseNewTargetsWithEffects)`.
- Goblin Artisans: `unsupported spell target filter (And)` -> `unsupported spell target filter (ControlledByAPlayer)`.
- Reverberation: `unsupported spell target filter (IsCardtype)` -> `unsupported action (CreateReplaceWouldDealDamageUntil)`.
- Ring of Immortals: `unsupported spell target filter (And)` -> `unsupported spell target filter (IsEnchantmentType)`.
- Spell Blast: `unsupported spell target filter (And)` -> `unsupported spell target filter (ManaValueIs)`.
- Thrull Wizard: `unsupported spell target filter (IsColor)` -> `unsupported action (PlayerMayCost)`.

## Test mapping and acceptance

All tests are in `pkg/mage/dsl/ron/spell_scope_test.go`.

| Test | Cards and contract evidence |
| --- | --- |
| TestRONSpellTargetTypedDeclarations | Synthetic AnySpell/type/color/And/Or declarations; fresh factory targets; typed reads; engine rejects missing declarations |
| TestRONSpellScopeInvalidDeclarations | Unbound, cross-rule, wrong-type, wrong syntax, numbered/source reads, empty filters, invalid colors/types, unsupported nested operands |
| TestRONRealFilteredCounters | Artifact Blast/Sol Ring, Flash Counter/Jump/Air Elemental, Remove Soul/Air Elemental, Deathgrip/Llanowar Elves, Lifeforce/Frozen Shade, Counterspell/Sol Ring; timing, cost rollback, departed source, clone isolation, owner graveyard |
| TestRONSpellFiltersAndAnnouncement | Synthetic multitype/multicolor spell and nested And/Or; pure repeated queries; wrong type, ability, absent target rejection without state changes; only declared spell countered |
| TestRONSpellCounterResolutionLegality | Deathgrip/Llanowar Elves; changed color, removed spell, new cast of same card, uncounterable spell, phased source |
| TestRONSpellCounterCopyRetargetAndClone | Flash Counter/Jump/Air Elemental; independent copy target frame; original target preserved; clone has original choices |
| TestRONSpellTargetInTrigger | Synthetic RON entry trigger and Sol Ring; target chosen at stack placement; source departure does not stop resolution |

The failing tests were executable before implementation: AnySpell exposed no
typed declaration, all filters failed compilation, and all five real counter
cards failed compilation. The old Counterspell path also rejected stack object
IDs. After implementation, test setup was corrected to inspect normalized target
metadata, use `Stack.FindByID` (the public `FindStackObject` takes a source ID),
and stack two legal instants rather than cast an artifact while another spell
was on the stack. These corrections did not weaken the acceptance assertions.

Gain matches the estimate: five unique cards and 15 reprint records. Previously
accepted records remain accepted. Fresh totals: 859 compiled records, 1,586
rejected records, 251 compiled unique names and 735 rejected unique names.
Gameplay evidence adds five newly executable cards plus a Counterspell regression.
Six counter cards gain direct gameplay tests; Sol Ring, Jump, Llanowar Elves
and Frozen Shade also serve as real spell fixtures. No global behavior-verification
total is claimed. Compilation and these tests do not establish complete gameplay
coverage for the collection.

No RON asset changed. The vocabulary file hashes still match the sources, so
no source snapshot refresh is required. No source defect was corrected. The
Gaea's Liege/Fastbond observations remain open. There is no environment failure.

## Next candidates and removal gates

1. Typed temporary-layer references: replace legacy selectors for 35 accepted
   cards before extending dynamic values, keyword removal or durations.
2. Permanent controller filters and explicit permanent references: multi-card
   access to controlled-target/collection shapes, with timing and identity tests.
3. Optional mana payment result scopes: concrete dependents include Force Spike,
   Thrull Wizard, Vodalian Mage and Power Sink; validate payer/result lifetimes
   before adding adapters.

Explicit life lowering remains useful but has only one immediate full-card gain.
Complete loading/gameplay, embedded startup, Go-card deletion, generator retirement
and whole-repository unused-DSL removal gates remain unmet. This run stops after
P0-SCOPE-SPELL-FILTERS.

## Validation

Focused tests with `-count=1`, engine short tests with `-count=1`, `make test`,
`make build`, `go vet ./...`, `make lint` and `git diff --check` pass. Lint reports
zero issues. Final affected tests and required build/test checks pass after lint.
Both loader commands still exit 1 for the rejected collection: 859 compiled,
1,586 rejected. The ledger records the commands and baseline comparison.
