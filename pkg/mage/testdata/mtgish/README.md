# mtgish JSON input

Use `data/mtgish.lines.json` from the mtgish checkout directly. This file has
one tagged JSON card per line. Pass each record to `mtgish.ParseMtgishJSON`.
Mage-go does not need a converter or Rust to read this data.

`MtgishCard.Raw` keeps the full record, including card faces and optional
fields. `Rules` keeps nested values as `json.RawMessage`. `ManaCost` contains
typed `MtgishManaSymbol` entries. Each entry has a `Kind` with the upstream
symbol tag and an optional `Amount` for generic mana. Generic zero keeps an
explicit amount. X, hybrid, snow, and Phyrexian symbols keep their own tags;
repeated symbols remain separate entries. Invalid symbols or arguments cause
an import error.
The importer reads card data; it does not compile rules into engine abilities.

Pass the parsed card to `mtgish.Compile` from `pkg/mage/mtgish` to compile its
rules into an immutable `mage.CardDefinition`. The compiler currently supports
Flying, FirstStrike, DoubleStrike, Deathtouch, Defender, Flash, Haste, Hexproof,
Indestructible, Lifelink, Menace, Reach, Shroud, Trample, Vigilance, Banding, and
Fear. Keywords use the same immutable ability specifications as executable rules.
Instances expose them through `card.Abilities()` and the normal engine keyword
attributes. Land subtypes supply their intrinsic mana abilities. `definition.NewInstance(owner)` creates
a playable instance. Rules with arguments, other rules, and unknown card
fields return errors. A failure returns no partial definition.

Call `card.ToCardDefinitionSpec()` to copy the name, type line, supported mana
cost, printed power and toughness, and color indicator into a
`mage.CardDefinitionSpec`. Conversion does not create a card instance or ID.
The spec contains metadata only; a rules compiler must complete and validate
it before publication through `mage.NewCardDefinition`. Source rules remain
available in `MtgishCard.Rules` and `Raw`.

The conversion supports generic, colored, colorless, X, and two-color hybrid
mana. It returns an error for costs that the engine cannot represent, including
snow, Phyrexian, two-generic hybrid, and colorless hybrid mana. Unknown types
and card forms with multiple faces also return an error.

The Lightning Bolt fixture comes from the top-level card named `Lightning Bolt`
in `~/src/mtgish/data/mtgish.lines.json`. To refresh it from the mage-go root:

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
