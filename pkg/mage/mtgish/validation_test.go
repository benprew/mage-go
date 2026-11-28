package mtgish

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"uuid"
)

const validRecord = `{"_OracleCard":"Card","Name":"Test","Typeline":{"Cardtypes":["Creature"]},"ManaCost":[{"_ManaSymbol":"ManaCostGeneric","args":0}],"CardPT":{"Power":0,"Toughness":1}}`

func requireDiagnostic(t *testing.T, err error, code DiagnosticCode, path, family, tag string) *CompileError {
	t.Helper()
	var diagnostic *CompileError
	if !errors.As(err, &diagnostic) {
		t.Fatalf("expected CompileError, got %T: %v", err, err)
	}
	if diagnostic.Code != code || diagnostic.Path != path || diagnostic.Family != family || diagnostic.Tag != tag {
		t.Fatalf("unexpected diagnostic: %+v; want %s at %q (%s %s)", diagnostic, code, path, family, tag)
	}
	if diagnostic.Message == "" || diagnostic.Source != "" || diagnostic.Record != 0 {
		t.Fatalf("unexpected diagnostic context: %+v", diagnostic)
	}
	return diagnostic
}

func TestParseStrictStructure(t *testing.T) {
	for _, tt := range []struct {
		name, input, path string
	}{
		{"root null", `null`, ""},
		{"root array", `[]`, ""},
		{"root string", `"Card"`, ""},
		{"empty", ``, ""},
		{"duplicate root", strings.Replace(validRecord, `"Name":"Test"`, `"Name":"Test","Name":"Other"`, 1), "/Name"},
		{"escaped duplicate", strings.Replace(validRecord, `"Name":"Test"`, `"Name":"Test","\u004eame":"Other"`, 1), "/Name"},
		{"duplicate nested", strings.Replace(validRecord, `"Power":0`, `"Power":0,"Power":2`, 1), "/CardPT/Power"},
		{"duplicate rule child", strings.TrimSuffix(validRecord, "}") + `,"Rules":[{"_Rule":"Unknown","args":{"a/b~c":1,"a/b~c":2}}]}`, "/Rules/0/args/a~1b~0c"},
		{"trailing object", validRecord + `{}`, ""},
		{"trailing null", validRecord + ` null`, ""},
		{"trailing syntax", validRecord + ` ?`, ""},
		{"truncated member", `{"_OracleCard":"Card","Name":`, "/Name"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			card, err := ParseMtgishJSON([]byte(tt.input))
			if !reflect.DeepEqual(card, MtgishCard{}) {
				t.Fatalf("returned partial card: %+v", card)
			}
			requireDiagnostic(t, err, InvalidInput, tt.path, "", "")
		})
	}
}

func TestParseStrictMetadata(t *testing.T) {
	for _, tt := range []struct {
		name, old, replacement, path string
	}{
		{"missing name", `"Name":"Test",`, ``, "/Name"},
		{"empty name", `"Name":"Test"`, `"Name":""`, "/Name"},
		{"null name", `"Name":"Test"`, `"Name":null`, "/Name"},
		{"wrong name type", `"Name":"Test"`, `"Name":4`, "/Name"},
		{"wrong case", `"Name"`, `"name"`, "/name"},
		{"unknown root", `"Name":"Test"`, `"Name":"Test","Unknown":1`, "/Unknown"},
		{"missing typeline", `"Typeline":{"Cardtypes":["Creature"]},`, ``, "/Typeline"},
		{"null typeline", `"Typeline":{"Cardtypes":["Creature"]}`, `"Typeline":null`, "/Typeline"},
		{"missing types", `"Cardtypes":["Creature"]`, ``, "/Typeline/Cardtypes"},
		{"null types", `["Creature"]`, `null`, "/Typeline/Cardtypes"},
		{"empty types", `["Creature"]`, `[]`, "/Typeline/Cardtypes"},
		{"null type", `["Creature"]`, `[null]`, "/Typeline/Cardtypes/0"},
		{"empty type", `["Creature"]`, `[""]`, "/Typeline/Cardtypes/0"},
		{"type case", `"Cardtypes"`, `"CardTypes"`, "/Typeline/CardTypes"},
		{"unknown typeline", `"Cardtypes":["Creature"]`, `"Cardtypes":["Creature"],"Restriction":true`, "/Typeline/Restriction"},
		{"null subtypes", `"Cardtypes":["Creature"]`, `"Cardtypes":["Creature"],"Subtypes":null`, "/Typeline/Subtypes"},
		{"null supertype", `"Cardtypes":["Creature"]`, `"Cardtypes":["Creature"],"Supertypes":[null]`, "/Typeline/Supertypes/0"},
		{"null PT", `"CardPT":{"Power":0,"Toughness":1}`, `"CardPT":null`, "/CardPT"},
		{"missing creature PT", `,"CardPT":{"Power":0,"Toughness":1}`, ``, "/CardPT"},
		{"missing power", `"Power":0,`, ``, "/CardPT/Power"},
		{"null power", `"Power":0`, `"Power":null`, "/CardPT/Power"},
		{"missing toughness", `,"Toughness":1`, ``, "/CardPT/Toughness"},
		{"null toughness", `"Toughness":1`, `"Toughness":null`, "/CardPT/Toughness"},
		{"fractional power", `"Power":0`, `"Power":1.5`, "/CardPT/Power"},
		{"large toughness", `"Toughness":1`, `"Toughness":2147483648`, "/CardPT/Toughness"},
		{"PT case", `"Power"`, `"power"`, "/CardPT/power"},
		{"unknown PT", `"Power":0`, `"Power":0,"Variable":true`, "/CardPT/Variable"},
		{"null cost", `[{"_ManaSymbol":"ManaCostGeneric","args":0}]`, `null`, "/ManaCost"},
		{"empty cost", `[{"_ManaSymbol":"ManaCostGeneric","args":0}]`, `[]`, "/ManaCost"},
		{"null symbol", `{"_ManaSymbol":"ManaCostGeneric","args":0}`, `null`, "/ManaCost/0"},
		{"mana case", `"_ManaSymbol"`, `"_manasymbol"`, "/ManaCost/0/_manasymbol"},
		{"mana argument case", `"args"`, `"Args"`, "/ManaCost/0/Args"},
		{"null rules", `"Name":"Test"`, `"Name":"Test","Rules":null`, "/Rules"},
		{"null rule", `"Name":"Test"`, `"Name":"Test","Rules":[null]`, "/Rules/0"},
		{"null colors", `"Name":"Test"`, `"Name":"Test","ColorIndicator":null`, "/ColorIndicator"},
		{"null color", `"Name":"Test"`, `"Name":"Test","ColorIndicator":[null]`, "/ColorIndicator/0"},
		{"null loyalty", `"Name":"Test"`, `"Name":"Test","Loyalty":null`, "/Loyalty"},
		{"null defense", `"Name":"Test"`, `"Name":"Test","Defense":null`, "/Defense"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.Replace(validRecord, tt.old, tt.replacement, 1)
			if input == validRecord {
				t.Fatal("test did not change the input")
			}
			card, err := ParseMtgishJSON([]byte(input))
			if !reflect.DeepEqual(card, MtgishCard{}) {
				t.Fatalf("returned partial card: %+v", card)
			}
			family, tag := "", ""
			switch tt.name {
			case "mana case":
				family = "_ManaSymbol"
			case "mana argument case":
				family, tag = "_ManaSymbol", "ManaCostGeneric"
			}
			requireDiagnostic(t, err, InvalidInput, tt.path, family, tag)
		})
	}
}

func TestParseDispatchesCardFormFirst(t *testing.T) {
	_, err := ParseMtgishJSON([]byte(`{"_OracleCard":"Split","Name":4,"Cards":[{"_OracleCard":"Card"}]}`))
	requireDiagnostic(t, err, UnsupportedConstruct, "/_OracleCard", "_OracleCard", "Split")
}

func TestJSONContainerDepth(t *testing.T) {
	for _, tt := range []struct {
		depth int
		code  DiagnosticCode
	}{
		{128, UnsupportedConstruct},
		{129, ResourceLimit},
	} {
		t.Run(fmt.Sprint(tt.depth), func(t *testing.T) {
			rule := `{"_Rule":"Unknown","args":` + strings.Repeat("[", tt.depth-3) + `0` + strings.Repeat("]", tt.depth-3) + `}`
			input := strings.TrimSuffix(validRecord, "}") + `,"Rules":[` + rule + `]}`
			card, err := ParseMtgishJSON([]byte(input))
			if err == nil {
				_, err = Compile(card)
			}
			var diagnostic *CompileError
			if !errors.As(err, &diagnostic) || diagnostic.Code != tt.code {
				t.Fatalf("depth %d: got %v, want %s", tt.depth, err, tt.code)
			}
			card = sourceCard(t, `[]`)
			card.Rules = []json.RawMessage{json.RawMessage(rule)}
			if def, err := Compile(card); def != nil || !errors.As(err, &diagnostic) || diagnostic.Code != tt.code {
				t.Fatalf("programmatic depth %d: got %v, %v, want %s", tt.depth, def, err, tt.code)
			}
		})
	}
}

func TestKnownMetadataBlockers(t *testing.T) {
	for _, tt := range []struct {
		field, value string
	}{
		{"Loyalty", `{"_LoyaltyNumber":"Integer","args":3}`},
		{"Defense", `3`},
	} {
		t.Run(tt.field, func(t *testing.T) {
			input := strings.TrimSuffix(validRecord, "}") + `,"` + tt.field + `":` + tt.value + `}`
			card, err := ParseMtgishJSON([]byte(input))
			if err != nil {
				t.Fatal(err)
			}
			def, err := Compile(card)
			if def != nil {
				t.Fatal("returned a partial definition")
			}
			requireDiagnostic(t, err, EngineGap, "/"+tt.field, "", "")
		})
	}
}

func TestUnknownManaTag(t *testing.T) {
	for _, args := range []string{"", `,"args":null`, `,"args":[1,2]`} {
		t.Run(args, func(t *testing.T) {
			input := strings.Replace(validRecord, `{"_ManaSymbol":"ManaCostGeneric","args":0}`, `{"_ManaSymbol":"FutureMana"`+args+`}`, 1)
			_, err := ParseMtgishJSON([]byte(input))
			requireDiagnostic(t, err, UnsupportedConstruct, "/ManaCost/0", "_ManaSymbol", "FutureMana")
		})
	}
}

func TestParsePrintedPTBounds(t *testing.T) {
	input := strings.Replace(validRecord, `"Power":0,"Toughness":1`, `"Power":-2147483648,"Toughness":2147483647`, 1)
	card, err := ParseMtgishJSON([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	def, err := Compile(card)
	if err != nil || def.Power() != math.MinInt32 || def.Toughness() != math.MaxInt32 {
		t.Fatalf("changed valid printed values: %v, %v", def, err)
	}
}

func TestParseErrorLocation(t *testing.T) {
	_, err := ParseMtgishJSON([]byte(`{"_OracleCard":"Card","Name":?}`))
	diagnostic := requireDiagnostic(t, err, InvalidInput, "/Name", "", "")
	if !strings.Contains(diagnostic.Message, "byte offset") {
		t.Fatalf("lost syntax offset: %v", err)
	}
	diagnostic.Source, diagnostic.Record = "cards.jsonl", 7
	wrapped := fmt.Errorf("load failed: %w", diagnostic)
	var located *CompileError
	if !errors.As(wrapped, &located) || located.Source != "cards.jsonl" || located.Record != 7 || !strings.Contains(wrapped.Error(), "cards.jsonl:7") {
		t.Fatalf("lost structured location: %v", wrapped)
	}
}

func TestStandaloneManaSymbolValidation(t *testing.T) {
	for _, input := range []string{
		`{"_ManaSymbol":"ManaCostR","_ManaSymbol":"ManaCostG"}`,
		`{"_ManaSymbol":"ManaCostR","\u005fManaSymbol":"ManaCostG"}`,
		`{"_ManaSymbol":"ManaCostR","args":null}`,
		`{"_ManaSymbol":"ManaCostR","args":0}`,
		`{"_ManaSymbol":"ManaCostR","Args":null}`,
		`{"_ManaSymbol":"ManaCostR","_Rule":"Flying"}`,
		`{"_ManaSymbol":"ManaCostR"} {}`,
	} {
		t.Run(input, func(t *testing.T) {
			symbol := MtgishManaSymbol{Kind: "ManaCostU"}
			err := symbol.UnmarshalJSON([]byte(input))
			var diagnostic *CompileError
			if !errors.As(err, &diagnostic) || diagnostic.Code != InvalidInput || symbol.Kind != "ManaCostU" {
				t.Fatalf("accepted invalid symbol or changed the destination: %+v, %v", symbol, err)
			}
		})
	}
}

func TestCompileValidatesCurrentMetadata(t *testing.T) {
	for _, tt := range []struct {
		name, path string
		code       DiagnosticCode
		change     func(*MtgishCard)
	}{
		{"name", "/Name", InvalidInput, func(c *MtgishCard) { c.Name = "" }},
		{"types", "/Typeline/Cardtypes", InvalidInput, func(c *MtgishCard) { c.Typeline.Cardtypes = nil }},
		{"creature PT", "/CardPT", InvalidInput, func(c *MtgishCard) { c.CardPT = nil }},
		{"power", "/CardPT/Power", InvalidInput, func(c *MtgishCard) { c.CardPT.Power = nil }},
		{"toughness", "/CardPT/Toughness", InvalidInput, func(c *MtgishCard) { c.CardPT.Toughness = nil }},
		{"cost", "/ManaCost", EngineGap, func(c *MtgishCard) { c.ManaCost = nil }},
		{"empty cost", "/ManaCost", InvalidInput, func(c *MtgishCard) { c.ManaCost = []MtgishManaSymbol{} }},
		{"loyalty", "/Loyalty", EngineGap, func(c *MtgishCard) { c.Loyalty = json.RawMessage(`{"_LoyaltyNumber":"Integer","args":3}`) }},
		{"empty loyalty", "/Loyalty", InvalidInput, func(c *MtgishCard) { c.Loyalty = json.RawMessage{} }},
		{"defense", "/Defense", EngineGap, func(c *MtgishCard) { c.Defense = json.RawMessage(`3`) }},
		{"empty defense", "/Defense", InvalidInput, func(c *MtgishCard) { c.Defense = json.RawMessage{} }},
		{"battle", "/Typeline/Cardtypes/0", EngineGap, func(c *MtgishCard) { c.Typeline.Cardtypes = []string{"Battle"} }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			card, err := ParseMtgishJSON([]byte(validRecord))
			if err != nil {
				t.Fatal(err)
			}
			tt.change(&card)
			def, err := Compile(card)
			if def != nil {
				t.Fatal("returned partial definition")
			}
			requireDiagnostic(t, err, tt.code, tt.path, "", "")
		})
	}

	card, err := ParseMtgishJSON([]byte(validRecord))
	if err != nil {
		t.Fatal(err)
	}
	card.Name = "Changed"
	*card.CardPT.Power = -1
	def, err := Compile(card)
	if err != nil || def.Name() != "Changed" || def.Power() != -1 {
		t.Fatalf("did not compile the current fields: %v, %v", def, err)
	}
}

func TestManaCostPresence(t *testing.T) {
	for _, tt := range []struct {
		name, input string
		wantCost    bool
		code        DiagnosticCode
	}{
		{"land without cost", `{"_OracleCard":"Card","Name":"Forest","Typeline":{"Cardtypes":["Land"],"Subtypes":["Forest"]}}`, false, ""},
		{"nonland without cost", `{"_OracleCard":"Card","Name":"No Cost","Typeline":{"Cardtypes":["Sorcery"]}}`, false, EngineGap},
		{"explicit zero", validRecord, true, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			card, err := ParseMtgishJSON([]byte(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			if (card.ManaCost != nil) != tt.wantCost {
				t.Fatal("lost cost presence")
			}
			def, err := Compile(card)
			if tt.code != "" {
				requireDiagnostic(t, err, tt.code, "/ManaCost", "", "")
				if def != nil {
					t.Fatal("published a spell without a cost")
				}
			} else if err != nil || def == nil {
				t.Fatalf("failed to compile: %v", err)
			}
		})
	}
}

func TestCompileStrictRuleEnvelopes(t *testing.T) {
	for _, tt := range []struct {
		rule, suffix, family, tag string
		code                      DiagnosticCode
	}{
		{`{"_Rule":"Unknown","args":{"_Action":"Unknown"}}`, "", "_Rule", "Unknown", UnsupportedConstruct},
		{`{"_Action":"Flying"}`, "/_Action", "_Rule", "", InvalidInput},
		{`{"_Rule":"Flying","_Action":"Flying"}`, "/_Action", "_Rule", "Flying", InvalidInput},
		{`{"_rule":"Flying"}`, "/_rule", "_Rule", "", InvalidInput},
		{`{"_Rule":"Flying","Args":null}`, "/Args", "_Rule", "Flying", InvalidInput},
		{`{"_Rule":"Flying","args":null}`, "/args", "_Rule", "Flying", InvalidInput},
		{`{"_Rule":"Flying","args":[]}`, "/args", "_Rule", "Flying", InvalidInput},
		{`{"_Rule":"Flying","condition":false}`, "/condition", "_Rule", "Flying", InvalidInput},
		{`{"_Rule":"Flying","_Rule":"Haste"}`, "/_Rule", "", "", InvalidInput},
		{`{"_Rule":null}`, "/_Rule", "_Rule", "", InvalidInput},
		{`{"_Rule":3}`, "/_Rule", "_Rule", "", InvalidInput},
		{`{"_Rule":""}`, "/_Rule", "_Rule", "", InvalidInput},
		{`{}`, "/_Rule", "_Rule", "", InvalidInput},
		{`null`, "", "", "", InvalidInput},
		{`[]`, "", "", "", InvalidInput},
		{`"Flying"`, "", "", "", InvalidInput},
		{`{"_Rule":"Flying"} {}`, "", "", "", InvalidInput},
	} {
		t.Run(tt.rule, func(t *testing.T) {
			card := sourceCard(t, `[]`)
			card.Rules = []json.RawMessage{json.RawMessage(`{"_Rule":"Flying"}`), json.RawMessage(tt.rule)}
			def, err := Compile(card)
			if def != nil {
				t.Fatal("returned partial definition")
			}
			diagnostic := requireDiagnostic(t, err, tt.code, "/Rules/1"+tt.suffix, tt.family, tt.tag)
			if diagnostic.Card != card.Name {
				t.Fatalf("lost card context: %+v", diagnostic)
			}
		})
	}
}

func TestManaArithmeticBounds(t *testing.T) {
	maxAmount, one := int32(math.MaxInt32), int32(1)
	for _, tt := range []struct {
		name    string
		symbols []MtgishManaSymbol
		bad     bool
	}{
		{"maximum", []MtgishManaSymbol{{Kind: "ManaCostGeneric", Amount: &maxAmount}}, false},
		{"generic sum", []MtgishManaSymbol{{Kind: "ManaCostGeneric", Amount: &maxAmount}, {Kind: "ManaCostGeneric", Amount: &one}}, true},
		{"colored sum", []MtgishManaSymbol{{Kind: "ManaCostGeneric", Amount: &maxAmount}, {Kind: "ManaCostR"}}, true},
		{"colorless sum", []MtgishManaSymbol{{Kind: "ManaCostGeneric", Amount: &maxAmount}, {Kind: "ManaCostC"}}, true},
		{"hybrid sum", []MtgishManaSymbol{{Kind: "ManaCostGeneric", Amount: &maxAmount}, {Kind: "ManaCostWU"}}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			card := sourceCard(t, `[]`)
			card.ManaCost = tt.symbols
			def, err := Compile(card)
			if tt.bad {
				if def != nil {
					t.Fatal("published an overflowing mana cost")
				}
				requireDiagnostic(t, err, InvalidInput, "/ManaCost/1", "_ManaSymbol", tt.symbols[1].Kind)
			} else if err != nil || def.ManaCost().Generic != math.MaxInt32 {
				t.Fatalf("rejected in-range cost: %v", err)
			}
		})
	}
}

func TestCompiledMetadataIsolation(t *testing.T) {
	card := sourceCard(t, `[{"_Rule":"Flying"}]`)
	def, err := Compile(card)
	if err != nil {
		t.Fatal(err)
	}
	*card.CardPT.Power = -2
	card.Typeline.Cardtypes[0] = "Land"
	card.Rules[0][0] = '['
	if instance := def.NewInstance(uuid.New()); instance.Power() != 2 || def.Types()[0].String() != "Creature" || len(instance.Abilities()) != 1 {
		t.Fatal("source mutation changed a published definition")
	}
}
