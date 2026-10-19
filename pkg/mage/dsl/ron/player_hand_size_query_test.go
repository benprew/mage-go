package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONPlayerHandSizeQueryStormSeeker(t *testing.T) {
	for _, state := range []string{"normal", "empty", "self", "prevention", "clone", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			b.AddToHand(mage.NewInstant("First", ""))
			a.AddToHand(mage.NewInstant("Caster", ""))
			target := b.PlayerID()
			if state == "self" {
				target = a.PlayerID()
			}
			obj := castRONSpell(t, g, a, repositoryCard(t, "LEG", "Storm Seeker"), target)
			if state == "copy" {
				obj = g.CopySpellOnStack(obj.Card.ID(), b.PlayerID(), true)
				if obj == nil || obj.Targets[0] != a.PlayerID() {
					t.Fatal("copy lost retargeting")
				}
			}
			instructions, ok := mage.InspectSingularDamageInstructions(mage.Sequence(obj.Effects...))
			if !ok || len(instructions) != 1 {
				t.Fatal("missing native singular instruction")
			}
			if _, ok := mage.InspectPlayerHandSize(instructions[0].Amount); !ok {
				t.Fatal("missing native hand-size amount")
			}
			if state == "empty" {
				b.RemoveFromHand(b.Hand()[0].ID())
			}
			if state == "prevention" {
				g.AddPreventionShield(b.PlayerID(), 1)
			}
			for range 2 {
				if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
					t.Fatal("hand-size damage is unknown")
				}
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("inspection dealt damage")
			}
			if state != "empty" {
				b.AddToHand(mage.NewInstant("Later", ""))
			}
			if state == "clone" {
				clone := g.Clone()
				clone.GetPlayer(b.PlayerID()).AddToHand(mage.NewInstant("Clone only", ""))
				if certain, known := clone.InspectSpellDestruction(clone.StackPeek(), nil); certain || !known {
					t.Fatal("clone lost amount")
				}
				clone.ResolveTopOfStack()
				if clone.GetPlayer(b.PlayerID()).Life() != 17 || b.Life() != 20 {
					t.Fatal("clone read wrong hand")
				}
			}
			g.ResolveTopOfStack()
			wantA, wantB := 20, 18
			if state == "self" || state == "copy" {
				wantA, wantB = 19, 20
			}
			if state == "empty" {
				wantB = 20
			}
			if state == "prevention" {
				wantB = 19
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life %d/%d want %d/%d", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}

func TestRONPlayerHandSizeQueryBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([SpellDealsDamage(ThisSpell,TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer),Player(Ref_TargetPlayer))])))`)
	g, a, b := ronGame()
	a.AddToHand(mage.NewInstant("A", ""))
	for range 3 {
		b.AddToHand(mage.NewInstant("B", ""))
	}
	first := castRONSpell(t, g, a, factory(), a.PlayerID())
	second := castRONSpell(t, g, a, factory(), b.PlayerID())
	for _, obj := range []*mage.StackObject{first, second} {
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatal("factory binding lost")
		}
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 17 {
		t.Fatal("factories crossed player bindings")
	}
}

func TestRONPlayerHandSizeQueryInvalid(t *testing.T) {
	for _, amount := range []string{`TheNumberOfCardsInPlayersHand()`, `TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer,You)`, `TheNumberOfCardsInPlayersHand(Ref_TargetPlayer2)`, `TheNumberOfCardsInPlayersHand(ActionPlayer)`, `TheNumberOfCardsInPlayersHand(Ref_TargetPermanent)`} {
		_, err := Compile(nameScopeNode(t, `Card(Name:"Bad hand",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([SpellDealsDamage(ThisSpell,`+amount+`,Player(Ref_TargetPlayer))])))])`))
		if err == nil {
			t.Fatalf("accepted %s", amount)
		}
	}
}

func TestRONPlayerHandSizeQueryOrdered(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([SpellDealsDamage(ThisSpell,TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer),Player(Ref_TargetPlayer)),DestroyEachPermanent(IsCardtype(Land))])))`)
	for _, phase := range []bool{false, true} {
		g, a, b := ronGame()
		p := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
		obj := castRONSpell(t, g, a, factory(), b.PlayerID())
		if phase {
			g.PhaseOut(p.ID())
		}
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || known {
			t.Fatal("empty hand damage lost state barrier")
		}
		g.ResolveTopOfStack()
		if phase {
			if g.FindPermanentIncludingPhased(p.ID()) == nil {
				t.Fatal("phased target destroyed")
			}
		} else if g.FindPermanent(p.ID()) != nil {
			t.Fatal("later destruction did not execute")
		}
		if b.Life() != 20 {
			t.Fatal("empty hand dealt damage")
		}
	}
}
