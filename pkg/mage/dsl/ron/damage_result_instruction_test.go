package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDamageResultInstructionDrainLife(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, state := range []string{"normal", "partial", "prevented", "phase", "cycle", "returned", "copy", "zero"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, set, "Drain Life")
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				victim := g.PutOnBattlefield(mage.NewCreature("victim", "", 2, 3), b.PlayerID())
				x, want := 5, 3
				if state == "zero" {
					x, want = 0, 0
				}
				a.ManaPool().Add(core.Black, x+2)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, x); err != nil {
					t.Fatal(err)
				}
				switch state {
				case "partial":
					g.AddPreventionShield(victim.ID(), 4)
					want = 1
				case "prevented":
					g.AddPreventionShield(victim.ID(), 5)
					want = 0
				case "phase":
					g.PhaseOut(victim.ID())
					want = 0
				case "cycle":
					g.PhaseIn(g.PhaseOut(victim.ID()))
				case "returned":
					g.BouncePermanentToHand(victim)
					b.RemoveFromHand(victim.Card.ID())
					g.PutOnBattlefield(victim.Card, b.PlayerID())
					want = 0
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				for _, game := range []*mage.Game{g, g.Clone()} {
					for range 2 {
						certain, known := game.InspectSpellDestruction(game.StackPeek(), nil)
						if certain || !known {
							t.Fatalf("damage-result query=%v/%v", certain, known)
						}
					}
				}
				if a.Life() != 20 {
					t.Fatal("query gained life")
				}
				clone := g.Clone()
				clone.ResolveTopOfStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 20+want || a.Life() != 20 {
					t.Fatal("cloned result crossed games")
				}
				g.ResolveTopOfStack()
				if a.Life() != 20+want {
					t.Fatalf("gain=%d want=%d", a.Life()-20, want)
				}
			})
		}
	}
}

func TestRONDamageResultInstructionInvalid(t *testing.T) {
	for _, body := range []string{
		`GainLife(TheAmountOfDamageDealtThisWay)`,
		`SpellDealsDamage(ThisSpell,Integer(2),Player(You)),GainLife(TheAmountOfDamageDealtThisWay())`,
		`SpellDealsDamage(ThisSpell,Integer(2),EachPlayer(AnyPlayer)),GainLife(TheClampedAmountOfDamageDealtThisWay)`,
		`SpellDealsDamage(ThisSpell,Integer(2),Ref_AnyTarget),GainLife(TheClampedAmountOfDamageDealtThisWay)`,
	} {
		source := `Card(Name:"Invalid result",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([` + body + `]))])`
		if _, err := Compile(nameScopeNode(t, source)); err == nil {
			t.Fatal("invalid result accepted", body)
		}
	}
}

func TestRONDamageResultInstructionTotal(t *testing.T) {
	for _, recipient := range []string{`Player(You)`, `EachPermanent(IsCardtype(Creature))`} {
		for _, prevention := range []int{0, 1, 2} {
			source := `Card(Name:"Native total",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([SpellDealsDamage(ThisSpell,Integer(2),` + recipient + `),GainLife(TheAmountOfDamageDealtThisWay)]))])`
			factory, err := Compile(nameScopeNode(t, source))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			p := g.PutOnBattlefield(mage.NewCreature("one", "", 1, 8), b.PlayerID())
			q := g.PutOnBattlefield(mage.NewCreature("two", "", 1, 8), b.PlayerID())
			g.AddPreventionShield(a.PlayerID(), prevention)
			g.AddPreventionShield(p.ID(), prevention)
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			certain, known := g.InspectSpellDestruction(g.StackPeek(), nil)
			if certain || !known {
				t.Fatalf("total query %v/%v", certain, known)
			}
			if a.Life() != 20 || g.FindPermanent(p.ID()).Damage != 0 {
				t.Fatal("query ran damage")
			}
			clone := g.Clone()
			clone.ResolveTopOfStack()
			g.ResolveTopOfStack()
			want := 20
			if recipient != `Player(You)` {
				want += 4 - prevention
			}
			if a.Life() != want || clone.GetPlayer(a.PlayerID()).Life() != want {
				t.Fatal("wrong saved total")
			}
			if recipient != `Player(You)` && (g.FindPermanent(p.ID()).Damage != 2-prevention || g.FindPermanent(q.ID()).Damage != 2) {
				t.Fatal("wrong simultaneous damage")
			}
		}
	}
}
