package mtgish

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestParseMtgishJSON(t *testing.T) {
	data, err := os.ReadFile("../testdata/mtgish/lightning_bolt.json")
	if err != nil {
		t.Fatal(err)
	}
	card, err := ParseMtgishJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if card.Kind != "Card" || card.Name != "Lightning Bolt" || len(card.Typeline.Subtypes) != 0 || card.Typeline.Cardtypes[0] != "Instant" {
		t.Fatalf("unexpected card: %+v", card)
	}
	if len(card.ManaCost) != 1 || card.ManaCost[0].Kind != "ManaCostR" || card.ManaCost[0].Amount != nil {
		t.Fatalf("unexpected mana: %+v", card.ManaCost)
	}
	var rule map[string]any
	if err := json.Unmarshal(card.Rules[0], &rule); err != nil {
		t.Fatal(err)
	}
	actions := rule["args"].(map[string]any)["args"].([]any)
	damage := actions[1].(map[string]any)["args"].([]any)[0].(map[string]any)
	args := damage["args"].([]any)
	if damage["_Action"] != "SpellDealsDamage" || args[1].(map[string]any)["args"] != float64(3) || args[2].(map[string]any)["_DamageRecipient"] != "Ref_AnyTarget" {
		t.Fatalf("unexpected damage action: %v", damage)
	}
	var original, preserved any
	if err := json.Unmarshal(data, &original); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(card.Raw, &preserved); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, preserved) {
		t.Fatal("card data was lost")
	}
}

func TestMtgishCardToCardDefinitionSpec(t *testing.T) {
	data := []byte(`{"_OracleCard":"Card","Name":"Test creature",
		"Typeline":{"Supertypes":["Legendary","Snow"],"Cardtypes":["Artifact","Creature"],"Subtypes":["Golem"]},
		"ManaCost":[{"_ManaSymbol":"ManaCostGeneric","args":3},{"_ManaSymbol":"ManaCostWU"},{"_ManaSymbol":"ManaCostX"},{"_ManaSymbol":"ManaCostX"}],
		"CardPT":{"Power":2,"Toughness":4},"ColorIndicator":["Green"]}`)
	source, err := ParseMtgishJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	card, err := source.ToCardDefinitionSpec()
	if err != nil {
		t.Fatal(err)
	}
	if card.Name != source.Name || card.Power != 2 || card.Toughness != 4 {
		t.Fatalf("unexpected card: %+v", card)
	}
	if !reflect.DeepEqual(card.TypeLine, TypeLine{CardTypes: []core.CardType{core.TypeArtifact, core.TypeCreature}, SuperTypes: []core.SuperType{core.SuperLegendary, core.SuperSnow}, SubTypes: []string{"Golem"}}) {
		t.Fatalf("unexpected type line: %+v", card)
	}
	wantCost := core.ManaCost{Generic: 3, HasX: true, XCount: 2, Hybrid: []core.HybridSymbol{{A: core.White, B: core.Blue}}}
	if !reflect.DeepEqual(card.ManaCost, wantCost) || !reflect.DeepEqual(card.Colors, []core.Color{core.Green}) {
		t.Fatalf("unexpected cost or colors: %+v, %v", card.ManaCost, card.Colors)
	}
	other, err := source.ToCardDefinitionSpec()
	if err != nil {
		t.Fatal(err)
	}
	other.TypeLine.SubTypes[0] = "Human"
	other.TypeLine.SuperTypes[0] = core.SuperBasic
	other.TypeLine.CardTypes[0] = core.TypeLand
	other.ManaCost.Hybrid[0].A = core.Red
	other.Colors[0] = core.Red
	if !reflect.DeepEqual(card.TypeLine.SubTypes, []string{"Golem"}) || card.TypeLine.SuperTypes[0] != core.SuperLegendary || card.TypeLine.CardTypes[0] != core.TypeArtifact || !reflect.DeepEqual(card.ManaCost, wantCost) || !reflect.DeepEqual(card.Colors, []core.Color{core.Green}) {
		t.Fatal("conversions share mutable data")
	}
	source.Typeline.Subtypes[0] = "Elf"
	source.ColorIndicator[0] = "Red"
	if !reflect.DeepEqual(card.TypeLine.SubTypes, []string{"Golem"}) || !reflect.DeepEqual(card.Colors, []core.Color{core.Green}) {
		t.Fatal("source change affected the converted card")
	}
}

func TestMtgishCardToCardDefinitionSpecPreservesRulesForCompiler(t *testing.T) {
	data, err := os.ReadFile("../testdata/mtgish/lightning_bolt.json")
	if err != nil {
		t.Fatal(err)
	}
	source, err := ParseMtgishJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	rules := append([]byte(nil), source.Rules[0]...)
	spec, err := source.ToCardDefinitionSpec()
	if err != nil {
		t.Fatal(err)
	}
	if spec.Name != "Lightning Bolt" || !reflect.DeepEqual(spec.ManaCost, core.ManaCost{Red: 1}) || len(spec.Abilities) != 0 {
		t.Fatalf("unexpected metadata spec: %+v", spec)
	}
	if len(source.Rules) != 1 || !reflect.DeepEqual([]byte(source.Rules[0]), rules) {
		t.Fatal("conversion changed the source rules")
	}
}

func TestMtgishCardToCardDefinitionSpecMana(t *testing.T) {
	tests := []struct {
		kinds []string
		want  core.ManaCost
	}{
		{nil, core.ManaCost{}},
		{[]string{"ManaCostW", "ManaCostU", "ManaCostU", "ManaCostB", "ManaCostR", "ManaCostG", "ManaCostC"}, core.ManaCost{White: 1, Blue: 2, Black: 1, Red: 1, Green: 1, Colorless: 1}},
		{[]string{"ManaCostWU", "ManaCostUB", "ManaCostBR", "ManaCostRG", "ManaCostGW", "ManaCostWB", "ManaCostUR", "ManaCostBG", "ManaCostRW", "ManaCostGU"}, core.ManaCost{Hybrid: []core.HybridSymbol{
			{A: core.White, B: core.Blue}, {A: core.Blue, B: core.Black}, {A: core.Black, B: core.Red},
			{A: core.Red, B: core.Green}, {A: core.Green, B: core.White}, {A: core.White, B: core.Black},
			{A: core.Blue, B: core.Red}, {A: core.Black, B: core.Green}, {A: core.Red, B: core.White}, {A: core.Green, B: core.Blue},
		}}},
	}
	for _, tt := range tests {
		source := MtgishCard{Kind: "Card", Name: "Test", Typeline: MtgishTypeline{Cardtypes: []string{"Instant"}}}
		for _, kind := range tt.kinds {
			source.ManaCost = append(source.ManaCost, MtgishManaSymbol{Kind: kind})
		}
		card, err := source.ToCardDefinitionSpec()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(card.ManaCost, tt.want) {
			t.Fatalf("mana = %+v, want %+v", card.ManaCost, tt.want)
		}
	}
}

func TestMtgishCardToCardDefinitionSpecManaFromJSON(t *testing.T) {
	tests := []struct {
		name string
		mana string
		want core.ManaCost
	}{
		{
			name: "no mana cost",
			mana: `[]`,
			want: core.ManaCost{},
		},
		{
			name: "zero generic",
			mana: `[{"_ManaSymbol":"ManaCostGeneric","args":0}]`,
			want: core.ManaCost{},
		},
		{
			name: "three generic",
			mana: `[{"_ManaSymbol":"ManaCostGeneric","args":3}]`,
			want: core.ManaCost{Generic: 3},
		},
		{
			name: "generic amounts add",
			mana: `[{"_ManaSymbol":"ManaCostGeneric","args":2},{"_ManaSymbol":"ManaCostGeneric","args":3}]`,
			want: core.ManaCost{Generic: 5},
		},
		{
			name: "Lightning Bolt R",
			mana: `[{"_ManaSymbol":"ManaCostR"}]`,
			want: core.ManaCost{Red: 1},
		},
		{
			name: "Meddling Mage W U",
			mana: `[{"_ManaSymbol":"ManaCostW"},{"_ManaSymbol":"ManaCostU"}]`,
			want: core.ManaCost{White: 1, Blue: 1},
		},
		{
			name: "one W or U hybrid",
			mana: `[{"_ManaSymbol":"ManaCostWU"}]`,
			want: core.ManaCost{Hybrid: []core.HybridSymbol{{A: core.White, B: core.Blue}}},
		},
		{
			name: "Azorius Guildmage two W or U hybrids",
			mana: `[{"_ManaSymbol":"ManaCostWU"},{"_ManaSymbol":"ManaCostWU"}]`,
			want: core.ManaCost{Hybrid: []core.HybridSymbol{{A: core.White, B: core.Blue}, {A: core.White, B: core.Blue}}},
		},
		{
			name: "Supreme Verdict 1 W W U",
			mana: `[{"_ManaSymbol":"ManaCostGeneric","args":1},{"_ManaSymbol":"ManaCostW"},{"_ManaSymbol":"ManaCostW"},{"_ManaSymbol":"ManaCostU"}]`,
			want: core.ManaCost{Generic: 1, White: 2, Blue: 1},
		},
		{
			name: "Fireball X R",
			mana: `[{"_ManaSymbol":"ManaCostX"},{"_ManaSymbol":"ManaCostR"}]`,
			want: core.ManaCost{HasX: true, XCount: 1, Red: 1},
		},
		{
			name: "Walking Ballista X X",
			mana: `[{"_ManaSymbol":"ManaCostX"},{"_ManaSymbol":"ManaCostX"}]`,
			want: core.ManaCost{HasX: true, XCount: 2},
		},
		{
			name: "Sphinx's Revelation X W U U",
			mana: `[{"_ManaSymbol":"ManaCostX"},{"_ManaSymbol":"ManaCostW"},{"_ManaSymbol":"ManaCostU"},{"_ManaSymbol":"ManaCostU"}]`,
			want: core.ManaCost{HasX: true, XCount: 1, White: 1, Blue: 2},
		},
		{
			name: "generic and required colorless",
			mana: `[{"_ManaSymbol":"ManaCostGeneric","args":2},{"_ManaSymbol":"ManaCostC"},{"_ManaSymbol":"ManaCostC"}]`,
			want: core.ManaCost{Generic: 2, Colorless: 2},
		},
		{
			name: "generic colored and hybrid together",
			mana: `[{"_ManaSymbol":"ManaCostGeneric","args":2},{"_ManaSymbol":"ManaCostW"},{"_ManaSymbol":"ManaCostWU"}]`,
			want: core.ManaCost{Generic: 2, White: 1, Hybrid: []core.HybridSymbol{{A: core.White, B: core.Blue}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(`{"_OracleCard":"Card","Name":"Test","Typeline":{"Cardtypes":["Instant"]},"ManaCost":` + tt.mana + `}`)
			source, err := ParseMtgishJSON(input)
			if err != nil {
				t.Fatal(err)
			}
			card, err := source.ToCardDefinitionSpec()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(card.ManaCost, tt.want) {
				t.Fatalf("mana = %+v, want %+v", card.ManaCost, tt.want)
			}
		})
	}
}

func TestMtgishCardToCardDefinitionSpecUnsupportedManaFromJSON(t *testing.T) {
	tests := []struct {
		name string
		kind string
	}{
		{"snow", "ManaCostS"},
		{"white Phyrexian", "ManaCostWP"},
		{"blue Phyrexian", "ManaCostUP"},
		{"black Phyrexian", "ManaCostBP"},
		{"red Phyrexian", "ManaCostRP"},
		{"green Phyrexian", "ManaCostGP"},
		{"hybrid Phyrexian", "ManaCostRWP"},
		{"two generic hybrid", "ManaCost2W"},
		{"colorless hybrid", "ManaCostCW"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(`{"_OracleCard":"Card","Name":"Test","Typeline":{"Cardtypes":["Instant"]},
				"ManaCost":[{"_ManaSymbol":"ManaCostGeneric","args":2},{"_ManaSymbol":"` + tt.kind + `"},{"_ManaSymbol":"ManaCostU"}]}`)
			source, err := ParseMtgishJSON(input)
			if err != nil {
				t.Fatalf("valid mtgish mana failed to parse: %v", err)
			}
			card, err := source.ToCardDefinitionSpec()
			if err == nil || !reflect.DeepEqual(card, CardDefinitionSpec{}) {
				t.Fatalf("unsupported cost produced a spec: %+v, %v", card, err)
			}
		})
	}
}

func TestMtgishCardToCardDefinitionSpecErrors(t *testing.T) {
	for _, kind := range []string{"ManaCostS", "ManaCostUP", "ManaCostRWP", "ManaCost2W", "ManaCostCW", "Unknown", "ManaCostGeneric"} {
		source := MtgishCard{Kind: "Card", Name: "Test", Typeline: MtgishTypeline{Cardtypes: []string{"Instant"}}, ManaCost: []MtgishManaSymbol{{Kind: kind}}}
		if card, err := source.ToCardDefinitionSpec(); err == nil || !reflect.DeepEqual(card, CardDefinitionSpec{}) {
			t.Fatalf("converted unsupported mana %s: %+v, %v", kind, card, err)
		}
	}
	for _, input := range []string{
		`{"_OracleCard":"Split","Cards":[]}`,
		`{"_OracleCard":"Card","Name":"Test","Typeline":{"Cardtypes":["Battle"]}}`,
		`{"_OracleCard":"Card","Name":"Test","Typeline":{"Cardtypes":["Creature"],"Supertypes":["Unknown"]}}`,
		`{"_OracleCard":"Card","Name":"Test","Typeline":{"Cardtypes":["Creature"]},"ColorIndicator":["Unknown"]}`,
		`{"_OracleCard":"Card","Name":"Test"}`,
	} {
		source, err := ParseMtgishJSON([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if card, err := source.ToCardDefinitionSpec(); err == nil || !reflect.DeepEqual(card, CardDefinitionSpec{}) {
			t.Fatalf("converted unsupported card %s: %+v, %v", input, card, err)
		}
	}
}

func TestParseMtgishJSONManaSymbols(t *testing.T) {
	kinds := []string{
		"ManaCostW", "ManaCostU", "ManaCostB", "ManaCostR", "ManaCostG",
		"ManaCostC", "ManaCostS", "ManaCostX",
		"ManaCostWP", "ManaCostUP", "ManaCostBP", "ManaCostRP", "ManaCostGP",
		"ManaCostRWP", "ManaCostRGP", "ManaCostGWP", "ManaCostGUP",
		"ManaCost2W", "ManaCost2U", "ManaCost2B", "ManaCost2R", "ManaCost2G",
		"ManaCostCW", "ManaCostCU", "ManaCostCB", "ManaCostCR", "ManaCostCG",
		"ManaCostWU", "ManaCostUB", "ManaCostBR", "ManaCostRG", "ManaCostGW",
		"ManaCostWB", "ManaCostUR", "ManaCostBG", "ManaCostRW", "ManaCostGU",
	}
	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			input := []byte(`{"_OracleCard":"Card","Name":"Test","ManaCost":[{"_ManaSymbol":"` + kind + `"}]}`)
			card, err := ParseMtgishJSON(input)
			if err != nil {
				t.Fatal(err)
			}
			if len(card.ManaCost) != 1 || card.ManaCost[0].Kind != kind || card.ManaCost[0].Amount != nil {
				t.Fatalf("unexpected mana: %+v", card.ManaCost)
			}
			encoded, err := json.Marshal(card.ManaCost)
			if err != nil {
				t.Fatal(err)
			}
			want := `[{"_ManaSymbol":"` + kind + `"}]`
			if string(encoded) != want {
				t.Fatalf("encoded mana = %s, want %s", encoded, want)
			}
		})
	}
}

func TestParseMtgishJSONMixedManaCost(t *testing.T) {
	input := []byte(`{"_OracleCard":"Card","Name":"Test","ManaCost":[
		{"_ManaSymbol":"ManaCostGeneric","args":0},
		{"_ManaSymbol":"ManaCostGeneric","args":3},
		{"_ManaSymbol":"ManaCostX"},{"_ManaSymbol":"ManaCostX"},
		{"_ManaSymbol":"ManaCostWU"},{"_ManaSymbol":"ManaCostW"},{"_ManaSymbol":"ManaCostU"},
		{"_ManaSymbol":"ManaCostS"},{"_ManaSymbol":"ManaCostUP"}
	]}`)
	card, err := ParseMtgishJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	zero, three := int32(0), int32(3)
	want := []MtgishManaSymbol{
		{Kind: "ManaCostGeneric", Amount: &zero},
		{Kind: "ManaCostGeneric", Amount: &three},
		{Kind: "ManaCostX"}, {Kind: "ManaCostX"},
		{Kind: "ManaCostWU"}, {Kind: "ManaCostW"}, {Kind: "ManaCostU"},
		{Kind: "ManaCostS"}, {Kind: "ManaCostUP"},
	}
	if !reflect.DeepEqual(card.ManaCost, want) {
		t.Fatalf("mana = %+v, want %+v", card.ManaCost, want)
	}
	encoded, err := json.Marshal(card.ManaCost)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []MtgishManaSymbol
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, want) {
		t.Fatalf("mana changed after encoding: %+v", decoded)
	}
}

func TestParseMtgishJSONInvalidMana(t *testing.T) {
	for _, symbol := range []string{
		`null`, `{}`, `"ManaCostX"`,
		`{"_ManaSymbol":"Unknown"}`,
		`{"_ManaSymbol":"ManaCostGeneric"}`,
		`{"_ManaSymbol":"ManaCostGeneric","args":null}`,
		`{"_ManaSymbol":"ManaCostGeneric","args":-1}`,
		`{"_ManaSymbol":"ManaCostGeneric","args":1.5}`,
		`{"_ManaSymbol":"ManaCostGeneric","args":"3"}`,
		`{"_ManaSymbol":"ManaCostGeneric","args":2147483648}`,
		`{"_ManaSymbol":"ManaCostX","args":3}`,
		`{"_ManaSymbol":"ManaCostWU","args":[]}`,
	} {
		t.Run(symbol, func(t *testing.T) {
			input := []byte(`{"_OracleCard":"Card","Name":"Test","ManaCost":[` + symbol + `]}`)
			if _, err := ParseMtgishJSON(input); err == nil {
				t.Fatalf("accepted invalid mana: %s", symbol)
			}
		})
	}
}

func TestParseMtgishJSONErrors(t *testing.T) {
	for _, input := range []string{"Card(Name: \"Bolt\")", "{}", "null", `{"_OracleCard":"Card"}`, `{"_OracleCard":"Card","Name":"Bolt"} {}`} {
		if _, err := ParseMtgishJSON([]byte(input)); err == nil {
			t.Errorf("accepted invalid input: %s", input)
		}
	}
}

func TestParseMtgishJSONPreservesSplitCard(t *testing.T) {
	data := []byte(`{"_OracleCard":"Split","Cards":[{"Name":"Fire","Rules":[]},{"Name":"Ice","Rules":[]}]}`)
	card, err := ParseMtgishJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	data[0] = '['
	if card.Kind != "Split" || card.Raw[0] != '{' || !json.Valid(card.Raw) {
		t.Fatalf("split card was not preserved: %+v", card)
	}
}
