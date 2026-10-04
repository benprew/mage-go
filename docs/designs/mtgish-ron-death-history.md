# Creature death history counter amounts

Capability P6-DEATH-HISTORY-COUNTERS, 2026-10-04. One compiler adapter slice.

The fresh baseline passed short engine tests and build. Strict loading rejected
1,392 records with 1,053 compiled; semantic inspection agreed. User edits in
`.gitignore` and the execution design are outside this slice.

| Candidate | Distinct affected cards | Estimated complete unlocks | Blockers, downstream benefit and effort |
| --- | --- | ---: | --- |
| Creature death history, selected | Khabál Ghoul; Scavenging Ghoul | 2 | No other blockers in either complete record. Reuses the typed engine death count. Establishes history consumption for later intervening-if work such as Osai Vultures. Small adapter, moderate gameplay tests. |
| Storage-land lifecycle | Bottomless Vault; Dwarven Hold; Hollow Trees; Icatian Store; Sand Silos | 5 | Entry and counter/mana payments work, but optional untap and tapped-source TriggerI remain missing. Two additional contracts with untap decisions and dual-time conditions. Medium implementation, large lifecycle/payment coverage. |
| Saved-sacrifice repeated mana | Priest of Yawgmoth; Sacrifice; Metamorphosis | 1 | Priest requires the repeated-mana saved-value consumer. Sacrifice also needs casting-payment bindings; Metamorphosis adds Plus and restricted mana. Shared downstream spell-payment capability. Small activation adapter with moderate rollback/solver tests; larger spell prerequisites. |

Complete records and nested expressions for every candidate were inspected.
Reprints count once by name. Two complete unlocks from one small adapter give
the best credible benefit for this slice's effort. Osai Vultures was also read;
its TriggerI condition remains a separate missing adapter. No engine prerequisite,
source defect or environment failure was found. No RON source changed.

Contract stated before edits: exact one-argument death-history constructor,
exact `IsCardtype(Creature)` filter, resolving counter-placement amounts. Read
current turn history at resolution through `mage.CreaturesDiedThisTurn()`;
keep source incarnation and counter replacement behavior in `mage.AddCountersTo`.
Reject other types, controller filters, malformed declarations and mixed types.

The compiler now uses private
`compileCounterAmount(Node, scope) (mage.ResolutionValue, error)` from
`compileCounterPlacement`. Legacy integer/X amounts still use
`mage.ValueAtResolution(compileValue(...))`. History is immutable and has no
binding producer or mutable instance state. Factories build separate effects;
source references retain their existing per-instance identity. Supported
counter-placement contexts are spell, activation and trigger action lists.
No general filter-history engine, global last-action slot, fallback, card-name
branch, declaration panic or partial registry was added. The syntax inventory
assigns the argument a permanent_filter role without claiming execution support.

| Card or contract | Tests |
| --- | --- |
| Khabál Ghoul, complete ARN record | TestRONDeathHistoryCards; TestRONDeathHistoryEvents; TestRONDeathHistoryCastingAndInstances |
| Scavenging Ghoul, complete LEA/LEB/2ED/3ED/CED/CEI records | TestRONDeathHistoryCastingAndInstances validates native casting, failed payment, independent sources and clone-specific values for each reprint; TestRONDeathHistoryCards adds counter cost and regeneration |
| Resolution timing, both end steps, before-entry deaths, tokens, simultaneous batch, repeated object exclusion | TestRONDeathHistoryCards; TestRONDeathHistoryCastingAndInstances |
| Returned incarnation, source control, phasing before/after event, trigger copies, clones, cleanup reset | TestRONDeathHistoryCards |
| Sacrifice, exile, bounce, noncreature destruction, regeneration, phased exclusion, animated artifact LKI | TestRONDeathHistoryEvents |
| Synthetic spell and activation history amounts | TestRONDeathHistoryActionContexts |
| Invalid shape/type/filter; syntax roles | TestRONDeathHistoryInvalid; TestRONDeathHistoryInventory |

TDD first exposed a missing owner in a fixture. After fixing ownership, tests
failed on the expected unsupported value and absent filter role. Subsequent
fixture corrections selected Scavenging Ghoul's activation at index 1 and used
DestroyPermanents for phased-subject legality. No gameplay assertion was weakened.
Repository inventory fixture counts changed to the freshly measured gain.
Existing engine death-count, batching, counter replacement, transactional cost,
source identity and cleanup services remain unchanged. CR 700.4, 603.1–3,
513.1–2, 400.7 and 701.19a–c and relevant API/rules contracts were read.
No new replacement ordering or transactional execution contract is claimed.

Actual and estimated gains agree: two unique names, seven records. Compiled
1,053 -> 1,060/2,445; rejected 1,392 -> 1,385. Accepted unique names 346 -> 348;
rejected names 640 -> 638. Zero accepted-record regressions. Strict and semantic
issue arrays agree. Unknown-role occurrences 16,310 -> 16,303. Collection-wide
engine_validated and behavior_verified remain null; seven newly accepted
records have explicit complete-card gameplay and definition evidence. All RON
hashes match the vocabulary snapshot, so a source snapshot refresh is not needed.

Ranked next candidates: (1) storage-land optional untap and tapped-source TriggerI
prerequisites; (2) saved-sacrifice repeated mana with downstream casting-payment
bindings; (3) creature-history intervening-if adapter for Osai Vultures, to defer
while useful multi-card work remains unless selected as a shared prerequisite.
The open Gaea's Liege/Fastbond data concerns remain unchanged. Runtime cutover,
Go-card/generator deletion and DSL removal gates remain closed. No DSL export
was deleted and no removal caller audit is claimed. No second slice began.

Validation on final code: focused RON/loadcards tests with `-count=1`,
`go test ./pkg/mage/... -short -count=1`, `make test`, `make build`,
`go vet ./...`, `make lint` (zero issues), and `git diff --check` all passed.
Automatic lint changes were reviewed; final tests followed final formatting.
Strict and semantic loader commands returned expected exit 1 for the remaining
1,385 records, with identical diagnostics and no accepted-record regressions.
Source-hash audit passed for all twelve files. Collection rejection is not a
build/test failure. Detailed command evidence is in the progress ledger.
