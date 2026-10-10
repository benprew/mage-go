package ron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type globalGrantSpellCase struct {
	subtype             string
	keyword             core.Attr
	power, toughness    int
	controlled, exclude bool
	land                core.BasicLandType
}

func globalGrantSpellCases() map[string]globalGrantSpellCase {
	return map[string]globalGrantSpellCase{
		"Goblin King":           {subtype: "Goblin", keyword: core.Mountainwalk, power: 1, toughness: 1, exclude: true, land: core.MountainLandType},
		"Lord of Atlantis":      {subtype: "Merfolk", keyword: core.Islandwalk, power: 1, toughness: 1, exclude: true, land: core.IslandLandType},
		"Fortified Area":        {subtype: "Wall", keyword: core.Banding, power: 1, controlled: true},
		"Kobold Drill Sergeant": {subtype: "Kobold", keyword: core.Trample, toughness: 1, controlled: true, exclude: true},
		"Concordant Crossroads": {keyword: core.Haste},
		"Goblin War Drums":      {keyword: core.Menace, controlled: true},
		"Hidden Path":           {keyword: core.Forestwalk, land: core.ForestLandType},
	}
}

func TestRONTextGlobalGrantSpellRecords(t *testing.T) {
	cases := globalGrantSpellCases()
	names := map[string]bool{}
	records := 0
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			name := node.Fields["Name"].String
			tc, ok := cases[name]
			eligible := globalGrantSpellTextRules(node.Fields["Rules"].Args)
			if eligible != ok {
				t.Fatalf("unmapped global grant context: %s", name)
			}
			if !ok {
				continue
			}
			t.Run(filepath.Base(file)+"/"+name, func(t *testing.T) {
				factory, err := Compile(node)
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				g.SetStep(core.PrecombatMain)
				card := factory()
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
					a.ManaPool().Add(color, 10)
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				paid := a.ManaPool().TotalMana()
				change, _ := mage.ReplaceColorWord(core.Green, core.Blue)
				keyword := tc.keyword
				if tc.land != 0 {
					change, _ = mage.ReplaceBasicLandWord(tc.land, core.SwampLandType)
					keyword = core.Swampwalk
				}
				if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != paid || g.FindPermanent(card.ID()) != nil || g.StackPeek().Card != card {
					t.Fatal("revision changed payment, timing or physical identity")
				}
				g.ResolveStack()
				source := g.FindPermanent(card.ID())
				if source == nil || source.Card.Owner() != a.PlayerID() {
					t.Fatal("entry lost physical identity or owner")
				}
				for _, player := range []mage.Player{a, b} {
					probe := mage.NewCreature("Recipient", "{G}", 1, 1, mage.WithSubTypes(tc.subtype))
					p := g.PutOnBattlefield(probe, player.PlayerID())
					want := !tc.controlled || player == a
					power, toughness := 1, 1
					if want {
						power += tc.power
						toughness += tc.toughness
					}
					staticPTCheck(t, g, p, power, toughness)
					if g.FindPermanent(p.ID()).HasKeyword(keyword) != want {
						t.Fatal("entry lost revised grant or controller selection")
					}
					if tc.land != 0 && g.FindPermanent(p.ID()).HasKeyword(tc.keyword) {
						t.Fatal("printed land word survived revision")
					}
				}
				if tc.exclude && g.FindPermanent(source.ID()).HasKeyword(keyword) {
					t.Fatal("source exclusion lost")
				}
			})
			names[name] = true
			records++
		}
	}
	if len(names) != 7 || records != 17 {
		t.Fatalf("unmapped records: %v / %d", names, records)
	}
}

func TestRONTextGlobalGrantSpellBoundaries(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Zombie Master"}, {"LEG", "Kobold Overlord"}} {
		g, a, _ := ronGame()
		g.SetStep(core.PrecombatMain)
		card := repositoryCard(t, tc.set, tc.name)
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green} {
			a.ManaPool().Add(color, 10)
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		before := g.StackPeek()
		change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.ForestLandType)
		if err := g.RevisePrintedSpellText(before.ID, change); err == nil {
			t.Fatal("unsupported sibling context accepted", tc.name)
		}
		if g.StackPeek() != before {
			t.Fatal("rejection changed stack")
		}
	}
	for _, rule := range []string{
		`EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([])])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([Landwalk(IsLandType(Bog))])])`,
		`EachPermanentLayerEffect(Ref_TargetPermanent,[AddAbility([Flying])])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AdjustPT(ValueX,1),AddAbility([Flying])])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
}
