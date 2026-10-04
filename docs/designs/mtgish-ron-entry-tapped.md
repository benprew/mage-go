# P8-ENTRY-TAPPED: one supported entry context

Run date: 2026-10-04. This run implements one compiler adapter and its semantic
inventory roles. No engine service, RON source, production wiring, Go card
constructor, generator or DSL export changes.

## Selection evidence

Baseline working tree: user changes in `.gitignore` and
`docs/designs/mtgish-ron-execution.md`; both are preserved and excluded from the
slice. No unrelated untracked files were present. Fresh short engine tests and
build passed. Strict and semantic loaders both exited 1 for collection rejection:
2,445 records, 986 unique names, 1,042 compiled, 1,403 rejected; 340 accepted
unique names and 646 rejected names. This is not a build failure.

The positioned AST inventory was read below unsupported parents. Its
`AsPermanentEnters` tuple shape groups lists by kind, so the parent signature
alone cannot distinguish tapped entry from choices or counters. Nested
`EntersTapped` signatures under `AsPermanentEnters[1]` identify the 15 affected
unique names below. Complete RON records, including all other abilities and
nested instructions, were inspected for all three candidates. Reprints count
once in the candidate comparison.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers | Downstream capabilities | Effort |
| --- | --- | ---: | --- | --- | --- |
| Tapped self entry, selected | Bottomless Vault, Bronze Tablet, Dwarven Hold, Dwarven Ruins, Ebon Stronghold, Havenwood Battleground, Hollow Trees, Icatian Store, Leviathan, Nevinyrral's Disk, Ruins of Trokair, Sand Silos, Svyelunite Temple, Sword of the Ages, Time Vault | 6 | Other nine need untap rules, TriggerI, plural payments, ante/ownership or turn replacements | Five storage-land lifecycles; further entry contexts retain explicit roles | Small adapter and inventory change; moderate six-card gameplay coverage |
| Creature death history | Khabál Ghoul, Scavenging Ghoul | 2 | None for these two exact records; broader filtered histories need contracts | Shared history values for future end-step and intervening-if consumers | Small adapter to CreaturesDiedThisTurn; moderate LKI/token/turn-reset tests |
| Saved-sacrifice repeated mana | Priest of Yawgmoth, Sacrifice, Metamorphosis | 1 | Sacrifice needs casting-payment scope; Metamorphosis also Plus and restricted mana | Shared paid-value consumer for two additional spells | Small activation consumer; moderate saved-result/rollback/solver tests, larger casting extension |

Tapped entry wins on credible complete-card gain per effort and supplies a
concrete shared prerequisite for the storage lands. No missing engine contract
was found for the selected context. Entry copying and optional untap are separate
capabilities; this slice does not partly compile those rules.

The selection and acceptance contract were stated before edits: only permanent
self entry with a singleton tapped operation; validate declarations; prove entry
state before observers, re-entry, phasing and clone isolation; execute both mana
abilities on all five sacrifice lands; execute Disk casting, activation rollback,
and its destruction union through shared services.

## Exact context and reuse boundary

Private signature: `compileTappedEntry(n Node) (optionFactory, error)`.
`compileRule` dispatches only after its existing permanent/non-spell check.
Accepted shape: `AsPermanentEnters/2`, symbol `ThisPermanent`, list of exactly
one symbol `EntersTapped`. An immutable option factory returns
`mage.WithKeyword(core.EntersTapped)` per card instance. There is no mutable
binding, game-time declaration check or new player/target scope in this form.

Invalid arity, non-list or empty bodies, additional operations, wrong/unbound
subjects, tuple-shaped symbols, spell declarations and use of the entry symbol
as a general action/keyword fail compilation. Inventory operands have roles
`permanent_reference` and `entry_operation`, including when the parent is
unsupported; syntax support remains `unverified`.

Engine reuse: entry preparation, card attribute seeds, permanent state,
card/game cloning, source incarnation references, phase-in/out, normal untap,
land timing, mana solver metadata, immediate mana abilities, transactional
activation/casting costs, owner-directed sacrifice, simultaneous destruction,
regeneration and indestructible checks, typed event conditions and stack
resolution. No fallback, name-specific production branch, global last-action
slot, declaration panic or partial production registry was added.

Read contracts: `pkg/mage/doc.go`, relevant DSL/rules documentation; CR
400.6–7, 614.1c–d/4/12, 702.26b–d, 701.21a and 602.2/5. Existing engine batch
and mana contracts remain unchanged. General competing entry replacements,
entry choices/copies/counters and copied stack abilities gain no new support.

## Card-to-test mapping

| Complete record / contract | Evidence |
| --- | --- |
| Dwarven Ruins, Ebon Stronghold, Havenwood Battleground, Ruins of Trokair, Svyelunite Temple | `TestRONEntryTappedLands`: complete FEM records, engine definition validation, native land play, tapped entry, solver exclusion while tapped, both abilities, immediate correct color/amount/controller, tap and sacrifice payments, unauthorized activation, clone isolation |
| All five sacrifice lands | `TestRONEntryTappedNativeUntapAndOwner`: actual untap steps for both players, normal untap, controller mana versus owner graveyard, sacrifice bypasses regeneration |
| Nevinyrral's Disk | `TestRONEntryTappedDisk`: complete LEB record, definition validation, native casting and cost rollback, entry after resolution, tap/mana activation rollback, pending destruction after source departure, both players' permanents, all three types, union uniqueness, regeneration, indestructible, phased/land exclusions, simultaneous pre-change death-observer state and clone resolution |
| All six newly accepted cards | `TestRONEntryTappedLifecycle`: direct entry under another controller, untap, phasing without re-entry, zone return with renewed tapped entry, independent instances and cloned games |
| Entry observer timing | `TestRONEntryTappedEventTiming`: synthetic RON entry/ETB ability plus engine observer fixtures; tapped-state condition at ETB, entry causes no tap transition, subsequent tap does |
| Card cloning | `TestRONEntryTappedCardCopy`: copied RON Disk retains entry declaration and does not copy untapped battlefield status |
| Invalid declarations | `TestRONEntryTappedInvalid` |
| Semantic inventory | `TestRONEntryTappedInventory`, refreshed whole-collection inventory fixtures |

TDD first encountered a fixture API typo, then failed for the expected unsupported
`AsPermanentEnters` rule and missing operand roles. An initial native land-play
fixture placed another owner's card in hand; it was corrected to use the proper
owner. A separate direct-entry test proves distinct owner/controller sacrifice.
The batch observer fixture first assumed post-change state; inspection of
`withZoneChangeBatch` and the leave-event LKI contract corrected it to require
the complete pre-change battlefield, with no intermediate removal. The
observer also retains its original creature-death filter when it adds this
condition. No gameplay expectation was weakened. Lint reported a test variable named after
the builtin `copy`; it was renamed. Automatic import formatting was reviewed.

## Fresh results and remaining gaps

Estimated and actual gains agree: six unique names, eleven records (Disk has six
reprints; the five lands have one record each). Compiled 1,042 -> **1,053/2,445**;
rejected 1,403 -> **1,392**. Accepted names 340 -> **346**; rejected names 646 ->
**640**. Previously accepted records have zero regressions. Strict and inspection
issue arrays match. AST occurrences/signatures stay **72,938/8,528**;
unknown-role occurrences fall **16,588 -> 16,310**. Inventory classification of
entry operands is broader than this adapter's execution support.

All twelve source hashes match the vocabulary snapshot. No source defect was
corrected, so that syntax snapshot is unchanged. Gaea's Liege/Fastbond concerns
remain open. Collection-wide `engine_validated` and `behavior_verified` remain
null. Six complete records gain explicit gameplay/definition evidence;
compilation counts do not establish collection-wide gameplay completion.

| Still rejected affected names | Remaining complete-record gaps |
| --- | --- |
| Bottomless Vault, Dwarven Hold, Hollow Trees, Icatian Store, Sand Silos | PermanentRuleEffect/MayChooseNotToUntapDuringUntap and tapped-source TriggerI; charging and optional untap contexts |
| Bronze Tablet | DeckConstruction/ante, owner filter, plural exile results, optional life payment and ownership exchange |
| Leviathan | DoesntUntapDuringControllersUntap, plural Island payments, optional payment and attack-cost rule |
| Sword of the Ages | SacrificeAnyNumberOfPermanents, collection payment results, total power and linked graveyard exile |
| Time Vault | DoesntUntapDuringControllersUntap, rule If, turn replacement/skip payment and extra turn |

Leading first diagnostics after the slice: PermanentRuleEffect 121, If 65,
CastEffect 54, ChooseADamageSource 46, Targeted_Modal 42, HostController 35,
ControlledByAPlayer 29, TriggerI 28, TargetGraveyardCard 26. Changed first errors
expose remaining rules; they are not extra unlocked cards.

Validation command results are recorded in the progress ledger. Production
cutover/removal gates remain closed. No DSL export was removed and no deletion
caller audit is claimed. Ranked next work to reassess: (1) death-history value
for Khabál Ghoul/Scavenging Ghoul, (2) optional untap and tapped-source TriggerI
for the five storage lands, (3) saved-sacrifice repeated mana. No second slice
began.
