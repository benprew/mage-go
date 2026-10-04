# Basic creature-token creation

Slice P5-TOKEN-CREATION, 2026-10-04. Completed adapter work; engine rules and
production startup are unchanged.

## Selection evidence

Fresh strict and semantic loaders began at 1,021/2,445 compiled records,
1,424 rejected records, 324 accepted unique names, and 662 rejected names.
The AST report includes nested forms under rejected parents. Complete records
were read for each name below. Reprints count once.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers and downstream benefit | Effort |
| --- | --- | ---: | --- | --- |
| Basic creature tokens, named tokens and counts | Boris Devilboon, Bottle of Suleiman, Breeding Pit, Dance of Many, Elvish Farmer, Goblin Warrens, Hazezon Tamar, Homarid Spawning Bed, Icatian Town, Master of the Hunt, Night Soil, Rukh Egg, Serpent Generator, Stangg, Tetravus, Thallid, Thallid Devourer, The Hive | 6 | Remaining cards need coin choices, end-step/delayed events, plural sacrifice, paid-object values, graveyard costs, token abilities, copying or linked results. Shared token declarations and batch creation serve all 18 names. | Small adapter; medium gameplay, metadata, batch, copy and rejection tests |
| Sacrifice characteristics | Diamond Valley, Life Chisel, Priest of Yawgmoth, Homarid Spawning Bed | 2 | Priest needs dynamic mana; Bed needs tokens and mana-value results. Typed payment snapshots enable all four. | Medium adapter; substantial payment, LKI and copy tests |
| Variable optional mana | In the Eye of Chaos, Invoke Prejudice | 1 | Invoke needs Opponent and color-sharing spell filters. Event mana-value payment serves both. | Medium adapter; substantial payment and event tests |

Tokens won because their existing engine contract provides the largest credible
full-card gain for the effort. The selection included named tokens and fixed
counts because The Hive, Boris Devilboon and Icatian Town need those forms and
share the same declaration and execution service. First-error frequency did
not determine the estimate. Counter-cost and sacrifice-filter prerequisites
from earlier slices are already available.

## Supported contract

`compileTokenCreation(Node, scope) (effectFactory, error)` supports
`CreateTokens/2` as a resolving action in the existing spell, activation and
supported trigger contexts, including existing targeted action lists. Argument
0 is a list with exactly one instruction; argument 1 is an empty modifier list.
An instruction is `TokenWithPT/6`, `NamedTokenWithPT/7`, or
`NumberTokens/2` wrapping either declaration. Counts use the existing
`Integer/1` literal or in-scope announced spell `ValueX` through
`ValueAtResolution`. Nonpositive counts create no tokens.

`compileCreatureToken(Node) (mage.TokenDefinition, error)` accepts:

- `PT/2` with literal signed power and toughness.
- Symbol `Colorless`, or `Colors/1` with a list of distinct colored symbols.
  `Colors([])` also produces colorless tokens.
- An empty supertype list.
- A nonempty list of distinct permanent card types that includes Creature.
- `SubtypesList/1` with distinct symbol subtypes. The existing engine
  `ValidateSubtypeFamily(SubtypeCreature, ...)` rejects known noncreature
  subtypes; it permits custom creature subtype names as its API specifies.
- A list of distinct symbol keywords from the compiler's existing keyword map.
- For `NamedTokenWithPT`, a nonempty string name before those six operands.

Unnamed tokens require subtypes. Their name is the joined subtypes followed by
` Token`, per local CR 111.4. Named tokens retain the exact supplied name. Token
characteristics have no mana cost or abilities inherited from the creating card.
Unsupported token abilities, supertypes, copy definitions, linked results,
multiple instructions, dynamic payment results and creation modifiers are errors.
Cost and static contexts do not accept resolving token creation.

RON owns syntax, scope, diagnostics and token-name derivation. The immutable
engine token declaration can be shared. Each factory builds fresh card and action
instances; no chosen targets or game state are held in that declaration. Lowering
uses `DefineToken`, `TokenPT`, `TokenTypes`, `TokenColors`, `TokenSubtypes`,
`TokenKeywords`, `CreateTokenCopies(You, ..., token)` and `ValueAtResolution`.
Empty optional options are omitted because nil slice options fail engine
validation. No engine prerequisite or implementation change was needed.

The engine owns payment transactions, resolving controller, announced X, stack
copies, source/event LKI, target legality, phasing, clone isolation and token
batches. CR 111.2–8, 400.7, 602.2 and 608.2b apply. Tokens enter in one engine
batch and all are visible to entry observers. The slice adds no replacement
operation, token-count replacement, token ability or entry-counter declaration;
replacement-order contracts remain in their existing services. No new LKI
characteristic query is introduced. Source departure does not cancel creation.

Inventory supplies `token_instruction`, `token_definition`, `token_pt`,
`token_colors`, `token_color`, `token_supertype`, `token_card_type`,
`token_subtypes`, `token_subtype`, `token_keyword`, `token_name` and
`token_creation_modifier` roles under known contracts. Shapes remain unverified
syntax observations. Unsupported parents retain unknown descendant roles.

## Acceptance and evidence

Gameplay tests first failed on `unsupported action (CreateTokens)`. Inventory
then failed on the absent token roles. Initial test fixture compile errors were
corrected to actual engine APIs before checking the expected adapter failure.
The first implementation revealed nil optional token options; lowering now omits
them. Boris's fixture now advances through untap before its tap-cost activation.
Token mana-value checks use CMC; unknown subtype rejection was corrected to a
known land subtype because the engine intentionally permits custom subtypes.
No gameplay contract or existing test was weakened.

| Cards or contract | Tests |
| --- | --- |
| Thallid, Thallid Devourer, Elvish Farmer; complete rules | TestRONTokenCreationRealCards: own/opponent upkeep, delayed counter placement, insufficient/exact spore payment, token characteristics, Saproling sacrifice for life or bonus and cleanup |
| The Hive, Boris Devilboon; complete rules | TestRONTokenCreationRealCards: mana/tap costs, resolution timing, explicit name, colors, artifact type and flying |
| Icatian Town; complete rules | TestRONTokenCreationTownBatch: cast/payment, four simultaneous entries observed together, Citizen characteristics and spell graveyard |
| Thallid and shared copied creation | TestRONTokenCreationCopiesAndSource: source departure/return, phasing/control change, independent copied controller/owner, pending clone isolation |
| Count and payment contracts | TestRONTokenCreationCountsAndRollback: fixed/zero/negative/X counts, cost-free copy retains X, no inherited mana or abilities, independent factory identity, failed casting and mana/tap activation rollback |
| Trigger, legality and token lifecycle | TestRONTokenCreationTriggerAndLegality: pending trigger retains controller after return, all-illegal targeted ability fizzles, token phase cycle and destruction |
| Declarations and semantic inventory | TestRONTokenCreationInvalid, TestRONTokenCreationDeclarationBoundaries, TestRONTokenCreationInventory: operand kinds, lists, types, colors, P/T, keywords, names, unbound X, modifiers, unsupported copies, static/cost contexts and nested roles |

Actual gain matches the estimate: six unique names and eleven records. The Hive
has six identical reprints; the other five names have one record each. Strict
loader ends at **1,032 compiled, 1,413 rejected**, with **330 accepted names and
656 rejected names**. No formerly accepted record or name regressed. All six
newly accepted cards have complete-rule RON gameplay tests above; those tests do
not imply collection-wide gameplay coverage. Global engine-validation and
behavior-verification totals remain null.

AST totals remain 72,938 node occurrences and 8,528 signatures; unknown-role
occurrences fall from 17,251 to 16,921. Strict and inventory issues agree. All
12 source hashes are unchanged; no source defect was corrected and the syntax
snapshot does not require refresh. Gaea's Liege/Fastbond concerns remain open.

See the progress ledger for final commands and commit readiness. Cutover and
removal gates remain closed. No Go cards, generators, DSL functions or current
callers were removed. Only this slice was implemented.
