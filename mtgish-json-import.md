# Compile mtgish JSON into playable cards

## Overview

Compile `mtgish` tagged JSON into shared, immutable `mage.CardDefinition`
values. Lower rules to existing `mage.Ability`, `mage.Target`, `mage.Cost`,
and `mage.Effect` specifications. Keep JSON decoding outside game execution
and reuse the engine's events, announcements, resolution, and cloning.

A card is available for play only when its complete definition compiles.
Unsupported rules, metadata, or card forms produce structured diagnostics
and no definition. A supported outer tag does not make all its children
supported. Compilation establishes fidelity to the supplied record; checked
fixtures and behavior tests must also verify that record against Oracle text.

## Current implementation and remaining work

The repository already contains the initial compiler and engine primitives.
This design extends them; it does not introduce a second import path.

| Area | Current implementation | Remaining work |
| --- | --- | --- |
| Definitions and registry | [CardDefinition](pkg/mage/card_definition.go) and atomic [Registry.RegisterDefinitions](pkg/mage/registry.go) | Preserve their publication and instance-isolation contracts |
| Parser | [schema.go](pkg/mage/mtgish/schema.go) and [validation.go](pkg/mage/mtgish/validation.go) enforce exact fields, duplicate-key and depth checks, required members, and mana-cost presence without `Raw` | Bounded JSONL input in phase 5 |
| Compiler | [compiler.go](pkg/mage/mtgish/compiler.go) and [rules.go](pkg/mage/mtgish/rules.go) support 17 argument-free keywords, intrinsic basic-land-type mana abilities, and the phase 3 spell and activation rules | Temporary P/T effects and entry triggers in phase 4 |
| Diagnostics | [diagnostics.go](pkg/mage/mtgish/diagnostics.go) supplies structured errors with JSON Pointer paths and capability codes | Attach file and record context in phase 5 |
| Triggers and filters | [TriggerSpec](pkg/mage/trigger_spec.go), context-aware [ObjectFilter](pkg/mage/object_filter.go), and shared [ObjectView/LKIView](pkg/mage/lki.go) exist | Lower supported source tags and test event-specific semantics |
| Targets and references | [TargetPermanent](pkg/mage/target.go) accepts object filters; [TargetGroup](pkg/mage/effect_references.go), `FirstTarget`, and `VarObject` return `Selector` | Bind source references to the correct target group and event role |
| Mana and values | `ActionDefinition.Type()` applies activated mana-ability criteria; `StaticInt` permits safe static inspection | Preserve classification, payment-planner, and AI behavior in compiled abilities |
| Batch import | Registry publication is atomic | JSONL input, read-only inventory, and an explicit batch loader |

Phases 1 through 3 are implemented. Parser and compiler tests cover the strict
input contract, including programmatic changes to parsed cards. P/T fields
use pointers to distinguish missing values from zero. Metadata conversion
rejects absent nonland mana costs with `engine_gap` before it creates a spec.
Lightning Bolt, Prodigal Sorcerer, Llanowar Elves, and Dark Ritual compile
and have behavior tests. The next phase covers Giant Growth and Soul Warden.

## Input and coverage baseline

The full input is external to this repository. The reviewed snapshot is
`~/src/mtgish/data/mtgish.lines.json`, from mtgish revision
`153451e44d88baabfeb9902092032c328f4d5416`, with SHA-256:

```text
f2513283c1ffd32ab954bdc10a4bd78a495f297f532c68b8c69d6f05f1e95d06
```

It contains 33,529 records, including 32,186 with `_OracleCard: "Card"`.
The remaining forms are initially unsupported. The longest line is 7,096
bytes including its line ending. Fixtures in
[pkg/mage/testdata/mtgish](pkg/mage/testdata/mtgish/README.md) are sufficient
for ordinary tests; tests must not require the sibling checkout or a download.

Top-level rule occurrences count only direct `Rules` entries, without
descending into card faces or nested rules:

| Rule | All records | `_OracleCard: "Card"` records |
| --- | ---: | ---: |
| `TriggerA` | 13,811 | 13,312 |
| `Activated` | 8,868 | 8,804 |
| `SpellActions` | 6,903 | 6,902 |
| `AsPermanentEnters` | 1,562 | 1,512 |
| `PermanentLayerEffect` | 1,531 | 1,529 |
| `EachPermanentLayerEffect` | 1,248 | 1,189 |
| `TriggerI` | 1,230 | 1,195 |

These counts guide priorities; they are not playable-card coverage.
Coverage reports must identify the input hash and count complete cards.

## Goals and supported boundary

- Produce fresh `*mage.Card` instances that use existing execution, cloning,
  AI evaluation, and WASM builds.
- Preserve every supported condition, restriction, quantity, source, and
  timing rule. Preserve rule order and repeated abilities.
- Identify blockers by source file, physical record number, card, JSON path,
  and semantic reason.
- Extend support in tested subsets. Reject all constructs outside those
  subsets; do not infer behavior from card names or Oracle prose.

The first rule subsets cover the six examples below, existing keywords, and
intrinsic basic-land-type mana abilities. Large rule families in the coverage
table are priorities for later work, not a promise to support every variant.

The engine currently has no Battle support. `AnyTarget` therefore operates
in the engine's existing game domain of creatures, planeswalkers, and players.
This is an explicit game-wide boundary, not full current-Magic targeting
support. Battle objects must remain unavailable to those games. If that
boundary cannot be enforced, reject `AnyTarget` with `engine_gap` until the
target and damage paths support Battles.

## Compiler architecture

```text
JSONL reader with bounded record size
    -> ParseMtgishJSON: JSON structure and supported card schema
    -> recursive lowering: strict tagged nodes and lexical bindings
    -> complete CardDefinitionSpec
    -> mage.NewCardDefinition
    -> staged definitions
    -> one Registry.RegisterDefinitions call
    -> fresh instances via CardDefinition.NewInstance
```

Inventory uses the same parser and lowerers but never publishes definitions.
Both parser and compiler live in `pkg/mage/mtgish`, which imports `mage` and
`core`. `pkg/mage` must not import `mtgish` or retain source-format nodes.

### Strict parsing and validation

Parsing has a structural pass and a typed decoding pass. Recursive lowering
then decodes nested `json.RawMessage` values. Do not call this single-pass
parsing or rely on one struct decoder to validate nested rule objects.

1. Walk `json.Decoder` tokens to reject duplicate keys at every object depth,
   including keys that are equal after JSON escape decoding. Require exactly
   one non-null root object and EOF after it. Reject trailing JSON values.
2. Dispatch on `_OracleCard` before decoding form-specific fields. Initially
   accept only `Card`. Report other forms as `unsupported_construct`; do not
   decode a face from an unsupported form as a standalone playable card.
3. Decode the supported card schema with `DisallowUnknownFields()`. Enforce
   exact field spelling in the structural/schema checks: Go's struct decoder
   alone also accepts case-insensitive field matches. Apply field checks to
   `Typeline`, `CardPT`, mana symbols, and every decoded tagged node.
   Recognize known but unsupported fields such as `Loyalty` and `Defense`
   sufficiently to report a capability blocker instead of ignoring them.
4. Validate required members explicitly after decoding. Require a nonempty
   `Name`, a `Typeline` object, and a nonempty `Cardtypes` list. Use
   `Power *int32` and `Toughness *int32`; if `CardPT` is present, require both
   pointers to be non-nil. A creature requires `CardPT`. Reject explicit null
   for these objects and fields. Negative printed P/T values are valid.
5. Preserve legal omission: vanilla cards can omit `Rules`; lands can omit
   `ManaCost`. Reject explicit null arrays. `ManaCost: []` is not the chosen
   encoding for either an absent cost or `{0}` and is rejected. The reviewed
   snapshot uses omission for no cost and an explicit generic-zero symbol
   for `{0}`. Bound integer conversion and arithmetic for 32-bit WASM too.
6. Validate programmatically constructed `MtgishCard` values in `Compile`
   using the same semantic checks. Input bytes and exported struct fields
   must not become competing sources of truth. Remove `MtgishCard.Raw` and
   `validateFields` after strict parsing replaces them; update the existing
   tests and fixture README with that change.

`DisallowUnknownFields()` does not inspect `json.RawMessage` contents and
does not reject missing members, null pointers, or duplicate keys. Each
tag-specific lowerer must check its complete input before emitting a spec.

### Mana cost presence is an engine capability boundary

An absent cost is unpayable, whereas `{0}` can be paid (CR 202.1b and 118.6).
Keeping `ManaCost == nil` in the parsed card does not preserve this distinction
after conversion to the current value-typed `CardDefinitionSpec.ManaCost`.

Initially, allow absent costs on lands, which are played rather than cast.
Return `engine_gap` for a nonland with no mana cost before producing a
definition. Do not publish Ancestral Vision or a synthetic no-cost spell as
a free spell, even when its other rules compile.

Before enabling such nonlands, add an explicit mana-cost-presence property to
the engine definition and carry it through copies, clones, and casting checks.
Tests must distinguish normal casting, alternative costs, and casting without
paying a mana cost. A cost-presence flag must not reject a permitted cast
without paying the mana cost merely because the printed cost is absent.

### Tagged nodes

Use an internal envelope such as:

```go
type taggedNode struct {
    Family string
    Tag    string
    Args   json.RawMessage
}
```

Decode it with the expected family, such as `_Rule`, `_Actions`, or `_Cost`.
A node must have exactly one family discriminator, a nonempty string tag,
and only the fields allowed for that tag. Do not pick an arbitrary key from
a map or accept a tag from another family.

Validate argument presence, JSON type, tuple length, and list members for
each supported tag. No-argument tags reject `args`, including `args: null`.
A one-argument tag can contain a scalar, object, or list as specified by the
upstream schema; do not assume every `args` is a tuple. Unknown tags produce
`unsupported_construct`; malformed supported tags produce `invalid_input`.

Use the pinned upstream `typescript_types/oracle_cards.d.ts` and grammar to
check argument shapes. The source snapshot and checked fixtures define the
input contract until a later schema revision is explicitly reviewed.

### Recursive lowering and lexical bindings

Pass a value-typed scope through `compileRule`, `compileActions`,
`compileAction`, `compileTarget`, `compileCost`, `compileTrigger`,
`compileObjectFilter`, and `compileValue`:

```go
type compileScope struct {
    path         string
    abilityKind  mage.AbilityType
    targetGroups []targetGroupBinding
    trigger      *triggerBinding
}
```

The scope describes the source ability kind; mana classification occurs
after the activated ability is assembled. Bindings store immutable metadata,
not game objects or UUIDs. Copy a binding slice before extending it so child
or sibling calls cannot change another scope through a shared backing array.

- Start a fresh ability scope for each independent ability. Targets and
  trigger references cannot leak between top-level rules.
- Initially accept at most one `SpellActions` rule per card. Multiple spell
  sections belong to one spell announcement and need coordinated target
  indices and ordered resolution. Reject that composition until assembly
  across sections is tested; the reviewed input contains eight such cards.
- `Targeted(targets, actions)` allocates target groups in declaration order
  for that ability and exposes bindings only within its child actions.
  Phase 3 supports one wrapper at the ability boundary with one `AnyTarget`
  group. Nested targeting, conditional targeting, and multiple groups
  require further lowering.
- Map each supported `Ref_*` tag explicitly to a group index and its expected
  kind and cardinality. `Ref_AnyTarget` can select a player or damageable
  permanent; `Ref_TargetPermanent` must select a permanent. The shared
  engine `Selector` interface does not enforce this distinction.
- Reject unbound or incompatible references. Do not resolve ambiguous or
  unsupported numbered-reference conventions by choosing the first target.
- `ThisSpell` is valid in a spell context; `ThisPermanent` identifies the
  source permanent. Trigger references use a per-trigger role mapping to
  `EventSource`, `EventTarget`, `EventPlayer`, or `EventAmount`, not declared
  targets or the legacy flat-target fallback.
- Distinguish a resolving ability's controller (`You`) from an object's
  current controller or owner. Read the appropriate runtime context; never
  capture a controller or source ID while compiling.

`compileRule` contributes to a local definition builder. It is not restricted
to returning one `Ability`: future rules can supply static effects, cast
targets, additional costs, or other definition fields. Publish nothing from
that builder until all rules and metadata pass. Reuse the same lowerers for
inventory and compilation so their support decisions cannot diverge.

### Immutability and execution

`NewCardDefinition` copies containers; it does not recursively freeze every
interface value or callback. Finish all ability, target, cost, and effect
construction before publication. Copy retained slices and maps, and capture
only immutable values in callbacks. Do not retain raw JSON for interpretation
during execution, mutable builders, `*Game`, `*Card`, or `EffectContext`.

Targets are selected at announcement, costs are paid at activation/casting,
and actions run in source order at resolution. Use `TargetGroups` to retain
empty optional groups and illegal-target slots. A later illegal target must
not shift another reference to a different group. Effects that refer to a
permanent must preserve incarnation checks when it leaves and returns.

Keep intrinsic mana abilities from basic land subtypes even when `Rules` is
absent. Do not treat every omitted rule list as an error or add intrinsic
mana abilities to nonland objects that merely share a subtype string.

### Package layout

| File | Responsibility |
| --- | --- |
| `schema.go` | Card schema, required metadata, and metadata conversion |
| `validation.go` | Structural walk, depth limits, exact field checks, and JSON decoding helpers |
| `compiler.go` | `Compile`, tagged envelopes, and definition assembly |
| `diagnostics.go` | Diagnostic codes, source locations, and `CompileError` |
| `rules.go` | Ability scope and lowerers for actions, targets, costs, and values |
| `inventory.go` | JSONL reader, inventory aggregation, and staged batch loader |

## Engine contracts to preserve

Read [pkg/mage/doc.go](pkg/mage/doc.go), especially "Import Compiler
Primitives", before extending the compiler or engine.

### Triggers, filters, and last-known information

Reuse `TriggerSpec.MatchesEventType`,
`TriggerSpec.MatchesEvent(*GameEvent, AbilityContext, GameReader)`, and
`NewTriggeredAbility(spec, parts...)`. Existing constructors cover permanent
entry, death, departure, attacks, spell casts, upkeep, end step, and `TriggerOr`.
Their existence does not enable the corresponding source tag automatically.

`ObjectFilter.Match(object, game, abilityContext)` supports relative source
and controller filters. Compose `IsCardType`, `ObjectHasSubtype`,
`ObjectHasSuperType`, `OtherThan(SourceObject())`, `ObjectControlledBy`,
`ObjectOwnedBy`, `ObjectAnd`, `ObjectOr`, and `ObjectNot`. Existing `ControlledBy`,
`OwnedBy`, and `HasSubType` helpers return `PermanentFilter`; keep these API
distinctions explicit.

`ObjectView` provides shared `View*` accessors and `LKIView` embeds it.
`LookupObject` currently supplies live battlefield state or battlefield LKI;
it is not a universal lookup for every zone. Spell-cast filters need the
spell view and caster binding already used by `WhenPlayerCastsSpell`.

Entry predicates inspect state after entry. Death/departure predicates and
their ability controllers need state before departure, including simultaneous
departures (CR 603.6c and 603.10). A generic live-first lookup alone does not
prove event-time correctness across repeated zone changes. Test source death,
simultaneous deaths, and leave/return before enabling those source triggers.

Do not conflate these distinct constructs:

- `_Trigger: "If"` has arguments **`[Condition, Trigger]`**. It constrains
  the event, such as "during your turn"; it is not an intervening-if rule.
- `_Rule: "TriggerI"` has arguments **`[Trigger, Condition, Actions]`**.
  Its intervening condition must pass both when the ability would trigger
  and when it resolves (CR 603.4).
- A conditional action inside `ActionList` is checked during resolution.

Reject these conditional constructs until their particular check timing is
implemented. `TriggerOr` matches an event once even if multiple children
match; distinct events can still create distinct trigger instances.

### Targets, mana abilities, and static inspection

- Lower permanent predicates through `TargetPermanent(ObjectFilter, ...)`.
  Keep the `CreatureTarget` compatibility wrapper because AI code still
  inspects it. Use `TargetsOnlyCreatures` for conservative static inspection.
  A `[]PermanentFilter` cannot expand into `TargetPermanent`'s `...any`.
- Retain normal targeting legality at announcement and resolution, including
  controller restrictions, protection, shroud, hexproof, and incarnation.
  Do not use a filter to replace the target system's legality checks.
- Reuse `NewActivated` and `ActionDefinition.Type()` for CR 605.1a: no targets,
  possible mana production, and not a loyalty ability. Additional effects do
  not disqualify a mana ability. Mana abilities without a tap cost also
  resolve immediately. Dark Ritual remains a spell and uses the stack.
- Keep mana classification separate from automatic payment planning.
  `ManaProducer`, `ManaProductionEffect`, and the planner's accepted profile
  must describe effects the planner can execute in full. Decline automatic
  planning when it cannot execute every effect; preserve manual activation.
  Do not hide mana production in an opaque effect that is misclassified.
- Triggered mana abilities have separate CR 605.1b/605.4 requirements. Reject
  them until immediate-resolution behavior is supported by the lowering;
  do not put every mana-producing trigger on the stack.
- Use `StaticInt` for static text and AI properties. Never call a dynamic
  `ValueSource.Resolve` with a nil game to inspect a compiled ability.

## Lowering examples

The first four examples have checked fixtures and behavior tests. Giant
Growth and Soul Warden describe the planned phase 4 lowering. Compact forms
omit family markers for readability. Use the pinned checkout's
`show_card "<Card Name>"` or `show_card --ron "<Card Name>"` to inspect
source definitions.

### Lightning Bolt

```text
SpellActions(Targeted([AnyTarget], ActionList([SpellDealsDamage(ThisSpell, Integer(3), Ref_AnyTarget)])))
```

```go
mage.NewSpell(
    mage.WithTarget(mage.TargetDamageAnyTarget()),
    mage.DealDamageFrom(mage.SourceObject(), mage.TargetGroup(0), mage.Fixed(3)),
)
```

### Prodigal Sorcerer

```text
Activated(TapPermanent(ThisPermanent), Targeted([AnyTarget], ActionList([PermanentDealsDamage(ThisPermanent, Integer(1), Ref_AnyTarget)])))
```

```go
mage.NewActivated(
    mage.Tap(),
    mage.WithTarget(mage.TargetDamageAnyTarget()),
    mage.DealDamageFrom(mage.SourceObject(), mage.TargetGroup(0), mage.Fixed(1)),
)
```

Only `TapPermanent(ThisPermanent)` lowers to `Tap()` here. Costs that tap
some other permanent require a different supported cost specification.

### Llanowar Elves

```text
Activated(TapPermanent(ThisPermanent), ActionList([AddMana(ManaProduceG)]))
```

```go
mage.NewActivated(mage.Tap(), mage.AddMana(core.Green, 1))
```

### Dark Ritual

```text
SpellActions(ActionList([AddMana(And([ManaProduceB, ManaProduceB, ManaProduceB]))]))
```

```go
mage.NewSpell(mage.AddMana(core.Black, 3))
```

This `And` combines fixed mana production. It does not establish support for
arbitrary mana choices, spending restrictions, or other `And` families.

### Giant Growth

```text
SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [AdjustPT(3, 3)], UntilEndOfTurn)])))
```

```go
mage.NewSpell(
    mage.WithTarget(mage.TargetPermanent(mage.IsCardType(core.TypeCreature))),
    mage.Boost(mage.Fixed(3), mage.Fixed(3)).
        Targeting(mage.TargetGroup(0)).
        Until(core.EndOfTurn),
)
```

This is a resolving temporary layer-7 P/T adjustment. It does not implement
top-level `PermanentLayerEffect`, `EachPermanentLayerEffect`, P/T-setting
effects, or characteristic-defining abilities.

### Soul Warden

```text
TriggerA(WhenAPermanentEntersTheBattlefield(And([Other(ThisPermanent), IsCardtype(Creature)])), ActionList([GainLife(Integer(1))]))
```

```go
mage.NewTriggeredAbility(
    mage.WhenPermanentEntersBattlefield(mage.ObjectAnd(
        mage.OtherThan(mage.SourceObject()),
        mage.IsCardType(core.TypeCreature),
    )),
    mage.GainLifeAmount(mage.Fixed(1)),
)
```

## Event mapping and support gates

Event names are not sufficient evidence of support. Validate the emitted
payload, observation time, and trigger count for each source tag. Only the
Soul Warden entry subset is required by the first trigger milestone.

| Source `_Trigger` tag | Engine event and required checks |
| --- | --- |
| `WhenAPermanentEntersTheBattlefield` | `EvtZoneChange`, `ToZone: ZoneBattlefield`; filter the entered permanent after entry |
| `WhenAPermanentDies` | `EvtZoneChange`, battlefield to graveyard; use departure state and controller |
| `WhenAPermanentLeavesTheBattlefield` | `EvtZoneChange`, from battlefield; use departure state and controller |
| `WhenACreatureAttacks` | `EvtDeclaredAttacker`; filter the declared creature, not a creature merely put onto the battlefield attacking |
| `WhenACreatureBlocks` | Emitted `EvtDeclaredBlocker` with `Flag == true`, once per blocker; do not subscribe to both block event enums or count every blocker/attacker pair |
| `WhenAPlayerCastsASpell` | `EvtSpellCast`; bind the caster and spell, not an arbitrary permanent lookup |
| `AtTheBeginningOfAPlayersUpkeep` | `EvtUpkeep`; match the active player |
| `AtTheBeginningOfAPlayersEndStep` | `EvtEndStep`; match the active player |
| `WhenACreatureDealsCombatDamageToAPlayer` | `EvtDamageDealt`; require combat damage (`Flag`), positive dealt amount, creature source, and player recipient in `TargetID`; do not assume `PlayerID` identifies that recipient |
| `WhenAPlayerGainsLife` | `EvtLifeGained`; match the gaining player and actual gained amount |
| `WhenAPermanentBecomesTapped` | `EvtTapped`; distinguish this from `EvtTappedForMana` |
| `Or` | Union of child event specifications, one trigger instance per matching event |

Counter-change, cycling, turn-face-up, and transform triggers remain
`engine_gap` until their operations and payloads are implemented and tested.
Existing events with no compiler mapping remain unsupported too.

## Diagnostics

Use a structured error across parsing, lowering, and loading:

```go
type CompileError struct {
    Source  string
    Record  int
    Card    string
    Path    string
    Code    DiagnosticCode
    Family  string
    Tag     string
    Message string
}
```

`Source` is the caller-supplied path or label. `Record` is a one-based physical
JSONL line number; zero means no record location was supplied. `Path` is a
JSON Pointer with zero-based array indices, such as
`/Rules/0/args/args/1/args/0`; the empty string refers to the root. Escape
pointer keys correctly. Card and tag can be empty when decoding fails before
they are known. Keep syntax byte offsets in the message when available.

| Code | Meaning |
| --- | --- |
| `invalid_input` | Invalid JSON, duplicate/unknown fields, wrong argument shape, missing required data, or an invalid binding |
| `unsupported_construct` | A source form, tag, or valid composition has no supported lowering |
| `engine_gap` | Faithful lowering needs an engine behavior or representation that is not available |
| `registration_conflict` | A name conflicts within the batch or with the destination registry |
| `resource_limit` | A configured size or nesting limit was exceeded |
| `io_error` | Reading the input failed |

`Compile(card)` keeps its existing signature and returns a nil definition
on every failure. Return the first blocker in deterministic metadata/rule
order; within a rule, visit tuple/list children in source order. Callers must
be able to use `errors.As` without parsing an error string. The JSONL reader
attaches source and record context to errors from parser and compiler.

Inventory reports metadata blockers and, when the card structure permits,
up to one blocker per top-level rule by invoking the same lowerers with fresh
scopes. Structural failures that prevent safe traversal produce a record
diagnostic instead. Do not label this an exhaustive list of all blockers:
another unsupported child can become visible after the first is implemented.
Never count successfully compiled rules on a blocked card as playable cards.

## JSONL reading, inventory, and publication

Use a maximum JSON payload of 1 MiB per line and a maximum JSON container
depth of 128, with the root container at depth one. Enforce the depth limit
in the structural walk before recursive lowering. For `bufio.Scanner`, allow
line-ending overhead in its buffer limit and separately check payload size;
the scanner's token limit is not an exact JSON-payload limit.

Accept LF, CRLF, and a final record without a newline. Reject blank lines
as `invalid_input` while retaining physical line numbering. Always check
`Scanner.Err()`. An oversized token or read failure stops the stream and
produces a diagnostic; do not publish the successfully read prefix.

Provide two explicit operations:

- **Inventory** reads all recoverable records and returns coverage and
  blocker reports without touching any registry. Malformed complete lines
  do not prevent checking later lines; a scanner/read failure is terminal.
- **Load batch** accepts a curated input expected to compile in full. Parse
  and compile all records into temporary definitions, then call the supplied
  registry's `RegisterDefinitions` once. Any parse, compile, read, or conflict
  error leaves that registry unchanged. Do not offer implicit skip/overwrite
  behavior or mutate `mage.DefaultRegistry` during package initialization.

The full dataset is appropriate for inventory and will initially fail strict
loading. A supported subset must be selected explicitly before loading.
Name conflicts include existing legacy factories and diacritic-folded names,
as checked by the registry. Use a fresh registry for tests and isolated
imports; do not silently replace an existing Go card implementation.

The proposed `cmd/mtgish-inventory` takes an explicit input path and emits
stable JSON with the input SHA-256, schema revision, compiler build revision,
total records, compilable/blocked records, diagnostic counts, and blockers
grouped by code/family/tag with examples. Sort groups and examples explicitly;
do not rely on Go map order. Count each record once for card coverage. Report
rule and diagnostic counts separately. A complete inventory has exit status
zero even when it contains blocked cards. Terminal read, scanner-size, or
output failures have nonzero exit status and an incomplete report status
when a report can be written. A depth-limit failure blocks its record but
does not prevent reading later lines. Report the full input hash only when
the complete stream was read; never label a prefix hash as the input hash.

## Implementation order and acceptance

Write failing tests before each implementation change. Keep fixtures small
and checked in, record their source revision, and verify exact Oracle text.

| Phase | Work | Acceptance condition |
| --- | --- | --- |
| 1. Preserve existing foundation | Reuse definitions, registry, triggers, filters, targets, selectors, mana qualification, and `StaticInt` | Existing importer and primitive tests pass; no duplicate API or runtime JSON path |
| 2. Strict input and diagnostics | Structural validation, supported metadata schema, envelopes, required P/T, cost-presence gate, and structured errors | Reject duplicate keys at any depth, multiple values, wrong field case, null/missing members, malformed arguments, and unsupported metadata; preserve vanilla cards, lands, and `{0}`; no partial definition |
| 3. Spells and activated abilities | `SpellActions`, `Activated`, one `Targeted` wrapper, `ActionList`, self tap cost, fixed `AddMana`, `SpellDealsDamage`, and `PermanentDealsDamage` | Lightning Bolt, Prodigal Sorcerer, Llanowar Elves, and Dark Ritual fixtures compile and pass behavior tests through ordinary engine APIs |
| 4. Temporary P/T and entry triggers | Supported permanent filters, `CreatePermanentLayerEffectUntil` with fixed `AdjustPT` until end of turn, `TriggerA`, permanent-entry events, and `GainLife` | Giant Growth and Soul Warden fixtures compile; source exclusion, controller binding, duration, and instance/clone isolation pass |
| 5. Inventory and batch loading | Bounded JSONL reader, deterministic inventory, explicit staged loader, and CLI | Reproducible report on the pinned external snapshot; late parse/compile/read errors and exact/folded name conflicts leave the destination registry unchanged |

Focused semantic checks must cover:

- A supported rule followed by an unsupported rule fails the complete card;
  supported outer tags with unsupported children also fail.
- Wrong family, extra tuple arguments, null list members, unbound references,
  cross-ability target references, and unsupported multiple `SpellActions`
  sections produce exact diagnostic paths.
- Damage to each supported target kind, an illegal target at resolution,
  source removal before an activated ability resolves, tap restrictions and
  summoning sickness, immediate mana activation, and Dark Ritual using the
  stack preserve normal engine behavior.
- Giant Growth expires at cleanup and does not affect a returned incarnation.
  Soul Warden excludes itself, sees other players' creatures, and retains
  the trigger controller if control changes before resolution.
- Multiple instances and `Game.Clone` branches share definitions while their
  targets, counters, activation state, and resolving effects remain isolated.
- JSONL boundary sizes, depth limits, CRLF, a missing final newline, blank
  lines, malformed middle records, read errors, and late conflicts have the
  specified inventory and loading behavior.

Run focused tests during development, then `make test`, `make build`, and
`make lint` for implementation changes. Confirm the WASM compile path too;
parser/compiler tests must not depend on native-only tools or the full corpus.

## Deferred capabilities

Reject these until their complete semantics have implementations and tests:

- Battle cards and Battle targeting/damage, subject to the game-domain
  boundary above.
- Nonland cards without a mana cost, until the engine retains cost presence.
- Snow, Phyrexian, two-generic hybrid, and colorless hybrid mana costs.
- Loyalty/defense metadata, loyalty abilities, and triggered mana abilities.
- Counter-change, cycling, turn-face-up, and transform events.
- All card forms other than `_OracleCard: "Card"`, including split, flip,
  adventure, prepared/omen, meld, room, double-faced, and nontraditional forms.
- General `AsPermanentEnters` replacement effects, characteristic-defining
  abilities, and top-level static layer effects.
- Conditional triggers/actions, intervening-if abilities, modal or distributed
  targets, variable quantities, and other rule variants outside the tested
  subsets, even when related engine helpers exist.

New capability work must expand this boundary explicitly. It must never make
an unsupported card appear complete by dropping one of its rules or fields.
