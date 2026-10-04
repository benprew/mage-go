package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONOtherStaticCards(t *testing.T) {
	for _, tc := range []struct {
		name             string
		power, toughness int
		keyword          core.Attr
	}{
		{"Kobold Drill Sergeant", 0, 2, core.Trample},
		{"Kobold Overlord", 0, 1, core.FirstStrike},
		{"Kobold Taskmaster", 1, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			put := func(name string, controller mage.Player) *mage.Permanent {
				card := repositoryCard(t, "LEG", name)
				card.SetOwner(b.PlayerID())
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				return g.PutOnBattlefield(card, controller.PlayerID())
			}
			source := put(tc.name, a)
			own := put("Crimson Kobolds", a)
			opposing := put("Crimson Kobolds", b)
			nonKobold := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
			check := func(game *mage.Game, id *mage.Permanent, active bool) {
				t.Helper()
				p := game.FindPermanent(id.ID())
				wantP, wantT := 0, 1
				if active {
					wantP, wantT = tc.power, tc.toughness
				}
				if p.CurrentPower(game) != wantP || p.CurrentToughness(game) != wantT {
					t.Fatalf("Kobold P/T = %d/%d, want %d/%d", p.CurrentPower(game), p.CurrentToughness(game), wantP, wantT)
				}
				if tc.keyword != 0 && p.HasKeyword(tc.keyword) != active {
					t.Fatal("keyword membership mismatch")
				}
			}
			check(g, own, true)
			check(g, opposing, false)
			if nonKobold.CurrentPower(g) != 1 || nonKobold.CurrentToughness(g) != 1 || (tc.keyword != 0 && nonKobold.HasKeyword(tc.keyword)) {
				t.Fatal("non-Kobold received effect")
			}
			src := g.FindPermanent(source.ID())
			if src.CurrentPower(g) != 1 || src.CurrentToughness(g) != 2 || (tc.name == "Kobold Drill Sergeant" && src.HasKeyword(core.Trample)) {
				t.Fatal("source received its own effect")
			}
			if tc.name == "Kobold Overlord" && !src.HasKeyword(core.FirstStrike) {
				t.Fatal("printed first strike missing")
			}
			clone := g.Clone()
			clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
			check(clone, own, false)
			check(g, own, true)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			check(g, own, false)
			check(g, opposing, true)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: own.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			check(g, own, true)
			phased := g.PhaseOut(source.ID())
			g.ApplyContinuousEffects()
			check(g, own, false)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			check(g, own, true)
			phased = g.PhaseOut(own.ID())
			g.ApplyContinuousEffects()
			if g.FindPermanent(own.ID()) != nil {
				t.Fatal("phased recipient remained visible")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			check(g, own, true)
			remover := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			check(g, own, false)
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			check(g, own, true)
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			check(g, own, false)
			returned := g.PutOnBattlefield(source.Card, b.PlayerID())
			check(g, own, true)
			if returned.CurrentPower(g) != 1 || returned.CurrentToughness(g) != 2 {
				t.Fatal("returned source received its own boost")
			}
		})
	}
}

func TestRONOtherStaticInstances(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"Kobold Drill Sergeant", "Kobold Overlord", "Kobold Taskmaster"} {
		t.Run(name, func(t *testing.T) {
			g, a, _ := ronGame()
			var factory mage.CardFactory
			for _, node := range nodes {
				if node.Fields["Name"].String == name {
					factory, err = Compile(node)
					if err != nil {
						t.Fatal(err)
					}
					break
				}
			}
			if factory == nil {
				t.Fatal("missing RON card")
			}
			first := g.PutOnBattlefield(factory(), a.PlayerID())
			second := g.PutOnBattlefield(factory(), a.PlayerID())
			target := g.PutOnBattlefield(repositoryCard(t, "LEG", "Crimson Kobolds"), a.PlayerID())
			switch name {
			case "Kobold Drill Sergeant":
				if first.CurrentToughness(g) != 3 || second.CurrentToughness(g) != 3 || target.CurrentToughness(g) != 3 || !first.HasKeyword(core.Trample) || !second.HasKeyword(core.Trample) {
					t.Fatal("source identities or accumulated boosts incorrect")
				}
			case "Kobold Taskmaster":
				if first.CurrentPower(g) != 2 || second.CurrentPower(g) != 2 || target.CurrentPower(g) != 2 {
					t.Fatal("source identities or accumulated boosts incorrect")
				}
			case "Kobold Overlord":
				if !target.HasKeyword(core.FirstStrike) {
					t.Fatal("first strike not granted")
				}
			}
			g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
			if g.FindPermanent(second.ID()).CurrentPower(g) != 1 || g.FindPermanent(second.ID()).CurrentToughness(g) != 2 {
				t.Fatal("remaining source retained another instance's boost")
			}
			if name == "Kobold Drill Sergeant" && g.FindPermanent(second.ID()).HasKeyword(core.Trample) {
				t.Fatal("remaining source retained granted trample")
			}
			g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
			if g.FindPermanent(target.ID()).CurrentPower(g) != 0 || g.FindPermanent(target.ID()).CurrentToughness(g) != 1 || g.FindPermanent(target.ID()).HasKeyword(core.FirstStrike) || g.FindPermanent(target.ID()).HasKeyword(core.Trample) {
				t.Fatal("departed grants remained")
			}
		})
	}
}

func TestRONOtherStaticGlobalAndNested(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name: "Global source", ColorIndicator: [Red], Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []), Rules: [EachPermanentLayerEffect(And([IsCardtype(Creature), And([Other(ThisPermanent), Other(ThisPermanent), Or([IsColor(Red), IsColor(Green)])])]), [AdjustPT(1, 2), AddAbility([Flying])])], CardPT: (Power: 1, Toughness: 1))`), "global-other.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	own := g.PutOnBattlefield(repositoryCard(t, "LEG", "Crimson Kobolds"), a.PlayerID())
	opposing := g.PutOnBattlefield(repositoryCard(t, "LEG", "Crimson Kobolds"), b.PlayerID())
	for _, p := range []*mage.Permanent{own, opposing} {
		if p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 3 || !p.HasKeyword(core.Flying) {
			t.Fatal("global source exclusion changed ordinary disjunction membership")
		}
	}
	if source.CurrentPower(g) != 1 || source.CurrentToughness(g) != 1 || source.HasKeyword(core.Flying) {
		t.Fatal("global source received its own grant")
	}
}

func TestRONOtherStaticRejectsInvalidDeclarations(t *testing.T) {
	for _, filter := range []string{
		`Other`, `Other()`, `Other(ThisPermanent, ThisPermanent)`, `Other(ThisPermanent())`,
		`Other(HostPermanent)`, `Other(Ref_TargetPermanent)`, `Other(Trigger_ThatPermanent)`, `Other(You)`,
		`Other(SinglePermanent(ThisPermanent))`, `Or([Other(ThisPermanent), IsColor(Red)])`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: []), Rules: [EachPermanentLayerEffect(And([IsCardtype(Creature), %s]), [AdjustPT(1, 1)])])`, filter)
		nodes, err := Parse(strings.NewReader(src), "invalid-other.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid declaration: %s", filter)
		}
	}
	for _, rule := range []string{
		`EachPermanentLayerEffect(Other(ThisPermanent), [AdjustPTX(PlusX, Zero, Integer(1))])`,
		`Activated(TapPermanent(ThisPermanent), Targeted([TargetPermanent(And([IsCardtype(Creature), Other(ThisPermanent())]))], ActionList([TapPermanent(Ref_TargetPermanent)])))`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [%s])`, rule)
		nodes, err := Parse(strings.NewReader(src), "invalid-other.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted unsupported context: %s", rule)
		}
	}
}

func TestRONOtherStaticInventoryRole(t *testing.T) {
	src := `Card(Name: "Inventory", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: []), Rules: [EachPermanentLayerEffect(And([IsCardtype(Creature), Other(ThisPermanent)]), [AdjustPT(1, 1)])])`
	report, err := InspectFS(fstest.MapFS{"other.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	seen := false
	for _, entry := range report.Signatures {
		if entry.Name == "ThisPermanent" {
			seen = true
			if entry.Role != "permanent_reference" {
				t.Fatalf("source operand role = %s", entry.Role)
			}
		}
	}
	if !seen {
		t.Fatal("source operand missing from inventory")
	}
}
