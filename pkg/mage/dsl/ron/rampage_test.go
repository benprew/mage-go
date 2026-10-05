package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

var ronRampageCards = []struct {
	name string
	n    int
}{
	{"Aerathi Berserker", 3}, {"Chromium", 2}, {"Craw Giant", 2}, {"Frost Giant", 2}, {"Hunding Gjornersen", 1}, {"Marhault Elsdragon", 1}, {"Wolverine Pack", 2},
}

func TestRONRampageRealCards(t *testing.T) {
	for _, tc := range ronRampageCards {
		for _, count := range []int{0, 1, 2, 3} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, count), func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEG", tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				power, toughness := p.CurrentPower(g), p.CurrentToughness(g)
				if tc.name == "Chromium" && !p.HasKeyword(core.Flying) {
					t.Fatal("missing flying")
				}
				if tc.name == "Craw Giant" && !p.HasKeyword(core.Trample) {
					t.Fatal("missing trample")
				}
				g.GetCombat().AddAttacker(p.ID(), b.PlayerID())

				for range count {
					g.PutOnBattlefieldBlocking(mage.NewCreature("Blocker", "", 1, 1, mage.WithKeyword(core.Flying)), b.PlayerID(), p.ID())
				}

				g.PutTriggersOnStack()
				wantStack := 0
				if count > 0 {
					wantStack = 1
				}
				if g.StackSize() != wantStack || g.FindPermanent(p.ID()).CurrentPower(g) != power {
					t.Fatal("wrong trigger timing")
				}
				clone := g.Clone()
				g.ResolveStack()
				bonus := max(0, count-1) * tc.n
				if g.FindPermanent(p.ID()).CurrentPower(g) != power+bonus || g.FindPermanent(p.ID()).CurrentToughness(g) != toughness+bonus {
					t.Fatal("wrong rampage bonus")
				}
				if clone.FindPermanent(p.ID()).CurrentPower(clone) != power {
					t.Fatal("clone changed")
				}
				clone.ResolveStack()
				if clone.FindPermanent(p.ID()).CurrentPower(clone) != power+bonus {
					t.Fatal("clone lost rampage")
				}
				g.RunStepWithPriority(core.Cleanup)
				if g.FindPermanent(p.ID()).CurrentPower(g) != power {
					t.Fatal("bonus did not expire")
				}
			})
		}
	}
}

func TestRONRampageInvalid(t *testing.T) {
	for _, rule := range []string{`Rampage`, `Rampage()`, `Rampage(1,2)`, `Rampage(-1)`, `Rampage(Integer(2))`, `Rampage(ValueX)`, `Rampage("2")`, `SpellActions(ActionList([Rampage(2)]))`} {
		typ := "Creature"
		if strings.HasPrefix(rule, "SpellActions") {
			typ = "Instant"
		}
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[%s],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[%s])`, typ, rule)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", rule)
		}
	}
}

func TestRONRampageNativeCombat(t *testing.T) {
	for _, name := range []string{"Aerathi Berserker", "Craw Giant", "Chromium"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEG", name)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 1), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 1), b.PlayerID())
			assignments := []mage.BlockAssignment{{BlockerID: first.ID(), AttackerID: p.ID()}, {BlockerID: second.ID(), AttackerID: p.ID()}}
			g.ExecuteBlockers(assignments)
			bonus := 3
			if name == "Craw Giant" {
				bonus = 2
			}
			if name == "Chromium" {
				bonus = 0
			}
			if g.FindPermanent(p.ID()).CurrentPower(g) != c.Power()+bonus {
				t.Fatal("native blocker legality or bonus failed")
			}
			g.ExecuteCombatDamage()
			want := 20
			if name == "Craw Giant" {
				want = 14
			}
			if name == "Chromium" {
				want = 13
			}
			if g.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatal("flying or trample combat damage failed")
			}
		})
	}
}

func TestRONRampageZeroAndInstances(t *testing.T) {
	for _, n := range []int{0, 2} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Instances",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[Rampage(%d),Rampage(%d)])`, n, n)), "instances.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		first, second := factory(), factory()
		if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
			t.Fatal(err)
		}
		if first.Abilities()[0] == second.Abilities()[0] {
			t.Fatal("factory shared ability")
		}
		p := g.PutOnBattlefield(first, a.PlayerID())
		q := g.PutOnBattlefield(second, a.PlayerID())
		g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
		g.GetCombat().AddAttacker(q.ID(), b.PlayerID())
		for range 2 {
			g.PutOnBattlefieldBlocking(mage.NewCreature("Blocker", "", 1, 1), b.PlayerID(), p.ID())
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 2 {
			t.Fatal("instances did not trigger separately")
		}
		base := p.CurrentPower(g)
		g.ResolveStack()
		if g.FindPermanent(p.ID()).CurrentPower(g) != base+2*n || g.FindPermanent(q.ID()).CurrentPower(g) != base {
			t.Fatal("instances shared source")
		}
	}
}

func TestRONRampageInventory(t *testing.T) {
	report, err := InspectFS(fstest.MapFS{"r.ron": {Data: []byte(`Card(Name:"Rampage",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[Rampage(2)])`)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 || !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Kind == "integer" && s.Role == "integer_literal" }) {
		t.Fatal("rampage amount role missing")
	}
}
