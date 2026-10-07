package ron

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONManaActivationPaymentPrism(t *testing.T) {
	g, a, _ := ronGame()
	prism := g.PutOnBattlefield(repositoryCard(t, "LEA", "Celestial Prism"), a.PlayerID())
	a.ManaPool().Add(core.White, 1)
	a.ManaPool().Add(core.Red, 1)
	a.ManaPool().Add(core.Green, 1)
	spell := mage.NewInstant("Payment probe", "{W}{U}", mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1)))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if !g.CanAfford(a.PlayerID(), spell.ManaCost(), nil) {
		t.Fatal("prism payment must preserve white")
	}
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(prism.ID()).Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 1 {
		t.Fatal("planned payment was not executed")
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("spell did not resolve")
	}
}

func TestRONManaActivationPaymentColoredSources(t *testing.T) {
	for _, tc := range []struct {
		set, name       string
		payment, output core.Color
		amount          int
	}{{"DRK", "Apprentice Wizard", core.Blue, core.Colorless, 3}, {"LEG", "Fire Sprites", core.Green, core.Red, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 3)
			a.ManaPool().Add(core.Green, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			a.ManaPool().Clear()
			source := g.FindPermanent(card.ID())
			a.ManaPool().Add(tc.payment, 1)
			cost := core.ManaCost{Generic: tc.amount}
			if tc.output == core.Red {
				cost = core.ManaCost{Red: 1}
			}
			if g.CanAfford(a.PlayerID(), cost, nil) && tc.output == core.Red {
				t.Fatal("summoning-sick source planned")
			}
			g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
			a.ManaPool().Clear()
			if g.CanAfford(a.PlayerID(), cost, nil) {
				t.Fatal("unfunded costed source planned")
			}
			a.ManaPool().Add(tc.payment, 1)
			if !g.CanAfford(a.PlayerID(), cost, nil) {
				t.Fatal("funded colored source excluded")
			}
			clone := g.Clone()
			if err := g.AutoTapForCost(a.PlayerID(), cost); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().Count(tc.output) != tc.amount || !g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
				t.Fatal("colored payment or immediate production")
			}
			if clone.FindPermanent(source.ID()).Tapped || clone.GetPlayer(a.PlayerID()).ManaPool().Count(tc.payment) != 1 {
				t.Fatal("query or activation changed clone")
			}
			if tc.name == "Fire Sprites" && !g.FindPermanent(source.ID()).HasKeyword(core.Flying) {
				t.Fatal("lost flying")
			}
		})
	}
}

func TestRONManaActivationPaymentRollback(t *testing.T) {
	g, a, _ := ronGame()
	prism := g.PutOnBattlefield(repositoryCard(t, "LEA", "Celestial Prism"), a.PlayerID())
	a.ManaPool().Add(core.White, 1)
	a.ManaPool().Add(core.Red, 1)
	a.ManaPool().Add(core.Green, 1)
	spell := mage.NewInstant("Unpayable", "{W}{U}", mage.WithAdditionalCost(mage.LifePayCost(21)), mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1)))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil {
		t.Fatal("unpaid life cost accepted")
	}
	if a.Life() != 20 || a.ManaPool().Count(core.White) != 1 || a.ManaPool().Count(core.Red) != 1 || a.ManaPool().Count(core.Green) != 1 || g.FindPermanent(prism.ID()).Tapped || g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("failed total payment committed mana activation")
	}
}

func TestRONManaActivationPaymentInvalid(t *testing.T) {
	for _, rule := range []string{
		`Activated(And([PayMana([ManaCostGeneric(-1)]),TapPermanent(ThisPermanent)]),ActionList([AddMana(AnyManaColor)]))`,
		`Activated(And([PayMana([ManaCostGeneric(2)]),TapPermanent(Ref_TargetPermanent)]),ActionList([AddMana(AnyManaColor)]))`,
		`Activated(And([PayMana([ManaCostGeneric(2)]),TapPermanent(ThisPermanent)]),ActionList([AddMana(AnyManaColor,Integer(2))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid mana declaration accepted", rule)
		}
	}
}

func TestRONManaActivationPaymentStrictLoad(t *testing.T) {
	files := fstest.MapFS{}
	counts := map[string]int{}
	sourceFiles, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range sourceFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(bytes.NewReader(data), filepath.Base(file))
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(data), "\n")
		var records []string
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if name != "Celestial Prism" && name != "Apprentice Wizard" && name != "Fire Sprites" {
				continue
			}
			counts[name]++
			records = append(records, lines[node.Position.Line-1])
		}
		if len(records) > 0 {
			files[filepath.Base(file)] = &fstest.MapFile{Data: []byte(strings.Join(records, "\n"))}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 3 || counts["Celestial Prism"] != 6 || counts["Apprentice Wizard"] != 1 || counts["Fire Sprites"] != 1 {
		t.Fatal("wrong canonical records", counts)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 8 || report.UniqueCards != 3 || len(report.Issues) != 0 {
		t.Fatal("fixture inventory", report.Issues)
	}
	for _, name := range []string{"Celestial Prism", "Apprentice Wizard", "Fire Sprites"} {
		card, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := mage.ValidateCardDefinition(name, card); err != nil {
			t.Fatal(err)
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 3 {
		t.Fatal("invalid collection changed registry")
	}
}
