package mtgish

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func sourceCard(t *testing.T, rules string) mage.MtgishCard {
	t.Helper()
	card, err := mage.ParseMtgishJSON([]byte(`{"_OracleCard":"Card","Name":"Keyword Creature","Typeline":{"Cardtypes":["Creature"]},"CardPT":{"Power":2,"Toughness":2},"Rules":` + rules + `}`))
	if err != nil {
		t.Fatal(err)
	}
	return card
}

func TestCompileKeywords(t *testing.T) {
	keywords := map[string]core.Attr{
		"Flying": core.Flying, "FirstStrike": core.FirstStrike, "DoubleStrike": core.DoubleStrike,
		"Deathtouch": core.Deathtouch, "Defender": core.Defender, "Flash": core.Flash,
		"Haste": core.Haste, "Hexproof": core.Hexproof, "Indestructible": core.Indestructible,
		"Lifelink": core.Lifelink, "Menace": core.Menace, "Reach": core.Reach,
		"Shroud": core.Shroud, "Trample": core.Trample, "Vigilance": core.Vigilance,
		"Banding": core.Banding, "Fear": core.Fear,
	}
	for tag, want := range keywords {
		t.Run(tag, func(t *testing.T) {
			def, err := Compile(sourceCard(t, `[{"_Rule":"`+tag+`"}]`))
			if err != nil {
				t.Fatal(err)
			}
			card := def.NewInstance(uuid.New())
			if got := card.Abilities(); len(got) != 1 || mage.AbilityAttribute(got[0]) != want {
				t.Fatalf("unexpected compiled abilities: %v", got)
			}
			if !mage.NewPermanent(card, card.Owner()).HasKeyword(want) {
				t.Fatalf("instance lacks %s", want)
			}
		})
	}
}

func TestCompileKeywordOrder(t *testing.T) {
	def, err := Compile(sourceCard(t, `[{"_Rule":"Flying"},{"_Rule":"FirstStrike"},{"_Rule":"Flying"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if got := def.NewInstance(uuid.New()).Abilities(); len(got) != 3 || mage.AbilityAttribute(got[0]) != core.Flying || mage.AbilityAttribute(got[1]) != core.FirstStrike || mage.AbilityAttribute(got[2]) != core.Flying {
		t.Fatalf("lost source order or repeated abilities: %v", got)
	}
}

func TestCompiledKeywordsInCombat(t *testing.T) {
	tg := gametest.NewTestGame(t)
	makePermanent := func(rules string) *mage.Permanent {
		t.Helper()
		def, err := Compile(sourceCard(t, rules))
		if err != nil {
			t.Fatal(err)
		}
		card := def.NewInstance(uuid.New())
		return mage.NewPermanent(card, card.Owner())
	}
	attacker := makePermanent(`[{"_Rule":"Flying"},{"_Rule":"FirstStrike"}]`)
	ground := makePermanent(`[]`)
	reach := makePermanent(`[{"_Rule":"Reach"}]`)
	if mage.CanBlock(ground, attacker, tg.Game) || !mage.CanBlock(reach, attacker, tg.Game) {
		t.Fatal("compiled flying did not apply blocking restrictions")
	}
	combat := mage.NewCombat()
	if !combat.DealsDamageInStep(attacker, true) || combat.DealsDamageInStep(attacker, false) {
		t.Fatal("compiled first strike did not select the correct damage step")
	}
	double := makePermanent(`[{"_Rule":"DoubleStrike"}]`)
	if !combat.DealsDamageInStep(double, true) || !combat.DealsDamageInStep(double, false) {
		t.Fatal("compiled double strike did not apply in both damage steps")
	}
}

func TestCompileRejectsIncompleteRules(t *testing.T) {
	for _, rule := range []string{
		`{"_Rule":"Ward","args":2}`, `{"_Rule":"Protection"}`, `{"_Rule":"Landwalk"}`,
		`{"_Rule":"Flying","args":null}`, `{"_Rule":"Flying","args":[]}`,
		`{"_Rule":"Flying","condition":false}`, `{"_Rule":"Unknown"}`,
		`{}`, `null`, `[]`, `"Flying"`, `{"_Rule":3}`,
	} {
		t.Run(rule, func(t *testing.T) {
			def, err := Compile(sourceCard(t, `[{"_Rule":"Flying"},`+rule+`]`))
			if def != nil || err == nil || !strings.Contains(err.Error(), "Keyword Creature") || !strings.Contains(err.Error(), "Rules[1]") {
				t.Fatalf("expected card and rule path with no partial definition, got %v, %v", def, err)
			}
		})
	}
}

func TestCompileMetadataValidation(t *testing.T) {
	card := sourceCard(t, `[]`)
	def, err := Compile(card)
	if err != nil || len(def.NewInstance(uuid.New()).Abilities()) != 0 || def.Power() != 2 || def.Toughness() != 2 {
		t.Fatalf("unexpected vanilla definition: %v, %v", def, err)
	}
	card.Kind = "SplitCard"
	if def, err := Compile(card); def != nil || err == nil {
		t.Fatal("accepted unsupported card form")
	}
	card = sourceCard(t, `[]`)
	for _, raw := range []string{
		`{"_OracleCard":"Card","StartingLoyalty":3}`,
		`{"_OracleCard":"Card","Typeline":{"Restriction":true}}`,
		`{"_OracleCard":"Card","CardPT":{"Power":2,"Toughness":2,"Variable":true}}`,
		`null`,
	} {
		card.Raw = json.RawMessage(raw)
		if def, err := Compile(card); def != nil || err == nil {
			t.Fatalf("accepted unsupported source data %s: %v, %v", raw, def, err)
		}
	}
}

func TestCompileRejectsLightningBolt(t *testing.T) {
	data, err := os.ReadFile("../testdata/mtgish/lightning_bolt.json")
	if err != nil {
		t.Fatal(err)
	}
	card, err := mage.ParseMtgishJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if def, err := Compile(card); def != nil || err == nil || !strings.Contains(err.Error(), "SpellActions") {
		t.Fatalf("accepted uncompiled spell actions: %v, %v", def, err)
	}
}
