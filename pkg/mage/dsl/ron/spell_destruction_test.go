package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSpellDestruction(t *testing.T) {
	for _, name := range []string{"Stone Rain", "Shatter", "Disenchant"} {
		for _, state := range []string{"normal", "shield", "shroud", "phase", "return", "clone", "changed type"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := mage.NewLand("Victim")
				if name != "Stone Rain" {
					card = mage.NewArtifact("Victim", "")
				}
				card.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(card, b.PlayerID())
				watch, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
				visits := watch.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
				obj := castRONSpell(t, g, a, repositoryCard(t, "3ED", name), p.ID())
				want := state == "normal" || state == "clone" || state == "changed type" && name == "Disenchant"
				switch state {
				case "shield":
					g.AddRegenerationShield(p.ID())
				case "shroud":
					g.FindPermanent(p.ID()).GrantBaseAttr(core.Shroud)
				case "phase":
					g.PhaseOut(p.ID())
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
					g.PutOnBattlefield(card, b.PlayerID())
				case "changed type":
					current := g.FindPermanent(p.ID())
					current.RevokeBaseAttr(core.AttrIsLand)
					current.RevokeBaseAttr(core.AttrIsArtifact)
					current.GrantBaseAttr(core.AttrIsEnchantment)
				}
				g.ApplyContinuousEffects()
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, visits); !known || certain != want {
						t.Fatalf("certainty=%v/%v want=%v", certain, known, want)
					}
				}
				if state == "clone" {
					clone := g.Clone()
					clone.PhaseOut(p.ID())
					if certain, known := clone.InspectSpellDestruction(clone.StackPeek(), visits); !known || certain {
						t.Fatal("clone read original state")
					}
					clone.ResolveStack()
				}
				g.ResolveStack()
				if (g.FindPermanentIncludingPhased(p.ID()) == nil) != want {
					t.Fatal("native gameplay differs")
				}
				if state == "shield" && !g.FindPermanent(p.ID()).Tapped {
					t.Fatal("query consumed regeneration shield")
				}
			})
		}
	}
}

func TestRONSpellDestructionInvalid(t *testing.T) {
	for _, action := range []string{`DestroyPermanent()`, `DestroyPermanent(Ref_TargetPermanent_2)`, `DestroyPermanent(You)`, `DestroyPermanent(Ref_TargetPermanent,Ref_TargetPermanent)`, `DestroyPermanentNoRegen(Ref_TargetPermanent_2)`} {
		_, err := Compile(nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Land))],ActionList([`+action+`])))])`))
		if err == nil {
			t.Fatalf("accepted invalid action %s", action)
		}
	}
}

func TestRONSpellDestructionBindings(t *testing.T) {
	for _, action := range []string{"DestroyPermanent", "DestroyPermanentNoRegen"} {
		t.Run(action, func(t *testing.T) {
			factory, err := Compile(nameScopeNode(t, `Card(Name:"Bound destruction",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Land)),TargetPermanent(IsCardtype(Land))],ActionList([`+action+`(Ref_TargetPermanent1),`+action+`(Ref_TargetPermanent2)])))])`))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			first := g.PutOnBattlefield(mage.NewLand("First"), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewLand("Second"), b.PlayerID())
			watch, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
			visits := watch.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
			one := castRONSpell(t, g, a, factory(), first.ID(), second.ID())
			two := castRONSpell(t, g, a, factory(), second.ID(), first.ID())
			g.FindPermanent(first.ID()).GrantBaseAttr(core.Shroud)
			for _, obj := range []*mage.StackObject{one, two} {
				if certain, known := g.InspectSpellDestruction(obj, visits); !known || !certain {
					t.Fatal("numbered or factory binding lost")
				}
			}
			g.AddRegenerationShield(second.ID())
			for _, obj := range []*mage.StackObject{one, two} {
				want := action == "DestroyPermanentNoRegen"
				wantKnown := want || obj == one
				if certain, known := g.InspectSpellDestruction(obj, visits); known != wantKnown || certain != want {
					t.Fatalf("regeneration or order changed: certainty=%v/%v want=%v/%v", certain, known, want, wantKnown)
				}
			}
			g.ResolveStack()
			if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) != nil {
				t.Fatal("independent native spells resolved incorrectly")
			}
		})
	}
}
