package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONBoundPlayerCollectionRecall(t *testing.T) {
	for _, set := range []string{"3ED", "ATQ"} {
		for _, state := range []string{"normal", "phase", "return", "copy", "clone", "type change"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				owned := mage.NewArtifact("Owned", "")
				owned.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(owned, a.PlayerID())
				foreign := mage.NewArtifact("Foreign", "")
				foreign.SetOwner(a.PlayerID())
				q := g.PutOnBattlefield(foreign, b.PlayerID())
				p.GrantBaseAttr(core.Shroud)
				visits, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
				watched := visits.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
				obj := castRONSpell(t, g, a, repositoryCard(t, set, "Hurkyl's Recall"), b.PlayerID())
				switch state {
				case "phase":
					g.PhaseOut(p.ID())
				case "return":
					g.RemoveFromBattlefield(p)
					p = g.PutOnBattlefield(owned, a.PlayerID())
				case "copy":
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				case "type change":
					p.RevokeBaseAttr(core.AttrIsArtifact)
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, watched); certain || !known {
						t.Fatalf("query=%v/%v", certain, known)
					}
				}
				if state == "clone" {
					clone := g.Clone()
					if certain, known := clone.InspectSpellDestruction(clone.StackPeek(), watched); certain || !known {
						t.Fatal("clone query differs")
					}
					clone.ResolveStack()
					if clone.FindPermanent(p.ID()) != nil || g.FindPermanent(p.ID()) == nil {
						t.Fatal("clone isolation")
					}
				}
				g.ResolveStack()
				absent := state != "phase" && state != "type change"
				if (g.FindPermanentIncludingPhased(p.ID()) == nil) != absent || g.FindPermanent(q.ID()) == nil {
					t.Fatal("owner collection or presence differs")
				}
				found := false
				for _, c := range b.Hand() {
					found = found || c.ID() == owned.ID()
				}
				if found != absent {
					t.Fatal("wrong hand destination")
				}
			})
		}
	}
}

func TestRONBoundPlayerCollectionInvalid(t *testing.T) {
	for _, body := range []string{
		`Targeted([TargetPlayer(AnyPlayer())],ActionList([]))`,
		`Targeted([TargetPlayer()],ActionList([]))`,
		`Targeted([TargetPlayer(AnyPlayer)],ActionList([PutEachPermanentIntoItsOwnersHand(OwnedByAPlayer(SinglePlayer(Ref_TargetPlayer2)))]))`,
		`Targeted([TargetPlayer(AnyPlayer)],ActionList([PutEachPermanentIntoItsOwnersHand(OwnedByAPlayer(SinglePlayer(Ref_TargetPermanent)))]))`,
		`ActionList([PutEachPermanentIntoItsOwnersHand(OwnedByAPlayer(SinglePlayer(Ref_TargetPlayer)))])`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(`+body+`)])`)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONBoundPlayerCollectionLowering(t *testing.T) {
	for _, domain := range []string{"AnyPlayer", "Opponent"} {
		factory, _, err := compileTarget(nameScopeNode(t, `TargetPlayer(`+domain+`)`), scope{})
		if err != nil {
			t.Fatal(err)
		}
		first := factory(instanceBindings{}).(*mage.PlayerTargetBinding)
		second := factory(instanceBindings{}).(*mage.PlayerTargetBinding)
		if first == second {
			t.Fatal("factory shared declaration")
		}
		if _, ok := mage.InspectPlayerTarget(first); !ok {
			t.Fatal("lowering lost native declaration")
		}
		plan := &playerTargetPlan{name: "Ref_TargetPlayer"}
		node := nameScopeNode(t, `PutEachPermanentIntoItsOwnersHand(And([IsCardtype(Artifact),OwnedByAPlayer(SinglePlayer(Ref_TargetPlayer))]))`)
		effect, err := compileDirectReturnPlan(node, scope{resolving: true, playerTarget: plan})
		if err != nil {
			t.Fatal(err)
		}
		declaration := effect.build(instanceBindings{players: map[*playerTargetPlan]*mage.PlayerTargetBinding{plan: first}})
		if _, ok := mage.InspectContextualPermanentCollection(declaration.Subjects); !ok {
			t.Fatal("lowering lost bound owner collection")
		}
	}
}
