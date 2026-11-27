# mtgish JSON input

Use `data/mtgish.lines.json` from the mtgish checkout directly. This file has
one tagged JSON card per line. Pass each record to `mtgish.ParseMtgishJSON`.
Mage-go does not need a converter or Rust to read this data.

The parser accepts one non-null `_OracleCard: "Card"` object. It rejects
duplicate keys at every depth, trailing values, unknown fields, incorrect
field spelling, and JSON container depth above 128. Other card forms return
`unsupported_construct` before their fields are decoded. `MtgishCard.Raw`
has been removed; the exported fields are the compiler's source of truth.

`Rules` keeps nested values as `json.RawMessage`. `ManaCost` contains
typed `MtgishManaSymbol` entries. Each entry has a `Kind` with the upstream
symbol tag and an optional `Amount` for generic mana. Generic zero keeps an
explicit amount. X, hybrid, snow, and Phyrexian symbols keep their own tags;
repeated symbols remain separate entries. Invalid symbols or arguments cause
an import error. A card requires a name and a type line with at least one
card type. A creature also requires printed power and toughness. The P/T
fields use `*int32` so a missing value differs from a printed zero. Negative
printed values are accepted. Optional arrays may be omitted but must not be
null. Rules remain source data until compilation.

An omitted mana cost differs from an explicit generic-zero symbol. An empty
mana-cost array is invalid. Lands may omit their cost. A nonland without a
cost returns `engine_gap`; the engine cannot yet preserve this distinction
when it casts the card. Mana totals must fit a signed 32-bit integer.

Pass the parsed card to `mtgish.Compile` from `pkg/mage/mtgish` to compile its
rules into an immutable `mage.CardDefinition`. The compiler currently supports
Flying, FirstStrike, DoubleStrike, Deathtouch, Defender, Flash, Haste, Hexproof,
Indestructible, Lifelink, Menace, Reach, Shroud, Trample, Vigilance, Banding, and
Fear. Keywords use the same immutable ability specifications as executable rules.
Instances expose them through `card.Abilities()` and the normal engine keyword
attributes. Land subtypes supply their intrinsic mana abilities. `definition.NewInstance(owner)` creates
a playable instance. Rules with arguments, other rules, and unknown card
fields return errors. Loyalty and defense metadata return `engine_gap`.
A failure returns no partial definition.

Parser, metadata, and compiler failures return `*mtgish.CompileError`. Use
`errors.As` to read the card name, JSON Pointer path, diagnostic code, family,
tag, and message. Source and record are empty and zero until a caller supplies
them. A record number identifies a physical line and starts at one. The
compiler also validates cards that callers construct or change in Go.

Call `card.ToCardDefinitionSpec()` to copy the name, type line, supported mana
cost, printed power and toughness, and color indicator into a
`mage.CardDefinitionSpec`. Conversion does not create a card instance or ID.
The spec contains metadata only; a rules compiler must complete and validate
it before publication through `mage.NewCardDefinition`. Source rules remain
available in `MtgishCard.Rules`.

The conversion supports generic, colored, colorless, X, and two-color hybrid
mana. It returns an error for costs that the engine cannot represent, including
snow, Phyrexian, two-generic hybrid, and colorless hybrid mana. Unknown types
and card forms with multiple faces also return an error.

The Lightning Bolt fixture comes from the top-level card named `Lightning Bolt`
in `~/src/mtgish/data/mtgish.lines.json`, at revision
`153451e44d88baabfeb9902092032c328f4d5416`. Its Oracle text is
"Lightning Bolt deals 3 damage to any target." The source snapshot has SHA-256
`f2513283c1ffd32ab954bdc10a4bd78a495f297f532c68b8c69d6f05f1e95d06`.
Tests use the checked fixture and do not require that checkout. To refresh it
from the mage-go root:

```sh
jq -s 'map(select(._OracleCard == "Card" and .Name == "Lightning Bolt")) | if length == 1 then .[0] else error("expected one Lightning Bolt record") end' \
  "$HOME/src/mtgish/data/mtgish.lines.json" \
  > pkg/mage/testdata/mtgish/lightning_bolt.json
```

The upstream checkout includes the JSON output. To regenerate that output in
mtgish, use its `just rust_build_all` and `just convert_to_json` recipes.
Its current `export_sets` script exports RON; use the full JSON file as the
input to mage-go instead.

Run the importer tests with:

```sh
go test ./pkg/mage/mtgish -run TestParseMtgishJSON
```

Validated against mtgish revision `153451e44d88baabfeb9902092032c328f4d5416`.
