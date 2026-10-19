package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONOrderedDestruction(t *testing.T) {
	for _, name := range []string{"Armageddon", "Wrath of God", "Shatterstorm", "Tsunami"} {
		for _, state := range []string{"normal", "shield", "phase", "clone"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := mage.NewLand("Victim")
				switch name {
				case "Wrath of God":
					card = mage.NewCreature("Victim", "", 2, 2)
				case "Shatterstorm":
					card = mage.NewArtifact("Victim", "")
				case "Tsunami":
					card = mage.NewLand("Victim", mage.WithSubTypes("Island"))
				}
				card.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(card, b.PlayerID())
				obj := castRONSpell(t, g, a, repositoryCard(t, "3ED", name))
				plan, ok := mage.InspectOrderedDestruction(mage.Sequence(obj.Effects...))
				if !ok {
					t.Fatal("real RON action did not retain an ordered declaration")
				}
				if state == "shield" {
					g.AddRegenerationShield(victim.ID())
				}
				if state == "phase" {
					g.PhaseOut(victim.ID())
				}
				g.ApplyContinuousEffects()
				watch, ok := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
				if !ok {
					t.Fatal("watch collection rejected")
				}
				ctx := &mage.EffectContext{Game: g, Controller: a.PlayerID()}
				visits := watch.Select(ctx)
				certain, known := plan.Certain(ctx, visits)
				want := state != "phase" && (state != "shield" || name == "Wrath of God" || name == "Shatterstorm")
				if !known || certain != want || g.StackSize() != 1 {
					t.Fatalf("certainty=%v/%v want=%v", certain, known, want)
				}
				if state == "clone" {
					cloned := g.Clone()
					cloned.PhaseOut(victim.ID())
					if certain, known := plan.Certain(&mage.EffectContext{Game: cloned, Controller: a.PlayerID()}, visits); !known || certain {
						t.Fatal("clone reused original state")
					}
					cloned.ResolveStack()
				}
				g.ResolveStack()
				if (g.FindPermanentIncludingPhased(victim.ID()) == nil) != want {
					t.Fatal("native gameplay disagrees with certainty")
				}
				if state == "shield" && !want && !g.FindPermanent(victim.ID()).Tapped {
					t.Fatal("regeneration did not execute after inspection")
				}
			})
		}
	}
}

func TestRONOrderedDestructionSequence(t *testing.T) {
	for _, state := range []string{"empty-prefix", "changed-prefix", "reverse"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			g.PutOnBattlefield(mage.NewLand("Watched"), b.PlayerID())
			if state != "empty-prefix" {
				g.PutOnBattlefield(mage.NewCreature("Earlier", "", 1, 1), a.PlayerID())
			}
			actions := `DestroyEachPermanent(IsCardtype(Creature)),DestroyEachPermanent(IsCardtype(Land))`
			if state == "reverse" {
				actions = `DestroyEachPermanent(IsCardtype(Land)),DestroyEachPermanent(IsCardtype(Creature))`
			}
			factory, err := Compile(nameScopeNode(t, `Card(Name:"Ordered fixture",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([`+actions+`]))])`))
			if err != nil {
				t.Fatal(err)
			}
			obj := castRONSpell(t, g, a, factory())
			plan, ok := mage.InspectOrderedDestruction(mage.Sequence(obj.Effects...))
			if !ok {
				t.Fatal("lowering lost complete ordered sequence")
			}
			ctx := &mage.EffectContext{Game: g, Controller: a.PlayerID()}
			watch, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.IsLand))
			certain, known := plan.Certain(ctx, watch.Select(ctx))
			if certain != (state != "changed-prefix") || known != (state != "changed-prefix") {
				t.Fatalf("certainty=%v/%v", certain, known)
			}
			g.ResolveStack()
			if len(g.AllBattlefield()) != 0 {
				t.Fatal("ordered native gameplay did not finish both groups")
			}
		})
	}
	for _, action := range []string{
		`DestroyEachPermanent()`, `DestroyEachPermanent(And([]))`,
		`DestroyEachPermanent(IsCardtype(Land),IsCardtype(Creature))`,
		`DestroyEachPermanent(Or([IsCardtype(Land),Unknown]))`,
		`DestroyEachPermanent(ControlledByAPlayer(SinglePlayer(ActionPlayer)))`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([`+action+`]))])`)); err == nil {
			t.Fatalf("accepted invalid action %s", action)
		}
	}
}
