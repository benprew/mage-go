package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONOpponentCollectionQuery(t *testing.T) {
	for _, syntax := range []string{
		`ControlledByAPlayer(Opponent)`,
		`And([IsCardtype(Artifact),ControlledByAPlayer(Opponent)])`,
		`Or([And([IsColor(Black),ControlledByAPlayer(Opponent)]),OwnedByAPlayer(SinglePlayer(You))])`,
	} {
		build, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := mage.InspectContextualPermanentCollection(build(instanceBindings{})); !ok {
			t.Fatalf("lost opponent declaration: %s", syntax)
		}
	}
	for _, syntax := range []string{
		`ControlledByAPlayer`, `ControlledByAPlayer()`, `ControlledByAPlayer(Opponent())`,
		`ControlledByAPlayer(Opponent,Opponent)`, `ControlledByAPlayer(SinglePlayer(Opponent))`,
		`And([ControlledByAPlayer(Opponent),Or([])])`,
	} {
		if _, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true}); err == nil {
			t.Fatalf("accepted %s", syntax)
		}
	}
}

func TestRONOpponentCollectionQueryGameplay(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		base      int
	}{
		{"ATQ", "Gaea's Avenger", 1}, {"DRK", "Angry Mob", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := repositoryRecord(t, tc.set, tc.name, tc.set+".ron")
			collection := contextualQueryNode(t, record, "TheNumberOfPermanentsOnTheBattlefield").Args[0]
			build, err := compileDestructionCollection(collection, scope{resolving: true})
			if err != nil {
				t.Fatal(err)
			}
			plan, ok := mage.InspectContextualPermanentCollection(build(instanceBindings{}))
			if !ok {
				t.Fatal("real record lost opponent predicate")
			}
			g, a, b := ronGame()
			var memberCard mage.Card = mage.NewArtifact("Member", "")
			var ownCard mage.Card = mage.NewArtifact("Own", "")
			if tc.name == "Angry Mob" {
				memberCard = mage.NewLand("Member", mage.WithSubTypes("Swamp"))
				ownCard = mage.NewLand("Own", mage.WithSubTypes("Swamp"))
			}
			memberCard.SetOwner(a.PlayerID())
			member := g.PutOnBattlefield(memberCard, b.PlayerID())
			ownCard.SetOwner(b.PlayerID())
			g.PutOnBattlefield(ownCard, a.PlayerID())
			card := repositoryCard(t, tc.set, tc.name)
			castRONSpell(t, g, a, card)
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				game.SetActivePlayerIndex(0)
				game.ApplyContinuousEffects()
				check := func(want int) {
					t.Helper()
					game.ApplyContinuousEffects()
					p := game.FindPermanent(card.ID())
					ctx := &mage.EffectContext{Game: game, Controller: a.PlayerID()}
					visits := plan.Select(ctx)
					if len(visits) != want {
						t.Fatalf("got %v, want %d members", visits, want)
					}
					if p == nil || p.CurrentPower(game) != tc.base+want || p.CurrentToughness(game) != tc.base+want {
						t.Fatal("native PT disagrees with pure collection")
					}
				}
				check(1)
				phased := game.PhaseOut(member.ID())
				check(0)
				game.PhaseIn(phased)
				check(1)
				current := game.FindPermanent(member.ID())
				game.RemoveFromBattlefield(current)
				check(0)
				returned := game.PutOnBattlefield(current.Card, b.PlayerID())
				check(1)
				game.AddControlEffect(mage.ControlEffectSpec{SourceID: card.ID(), TargetID: returned.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				check(0)
			}
		})
	}
}
