package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDelayedDepartureReplacedDestruction(t *testing.T) {
	for _, batch := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
		card := NewCreature("Watched", "", 2, 3)
		card.SetOwner(b.PlayerID())
		watched := g.PutOnBattlefield(card, b.PlayerID())
		target := DeclarePermanentTarget("watched", IsCreature)
		g.resolution.targetSpecs = []Target{target.targetDeclaration}
		for _, effect := range []ResolvingEffect{
			WhenLeavesThisTurn(target, GainLifeFor(You, Fixed(2))),
			WhenDiesThisTurn(target, GainLifeFor(You, Fixed(8))),
		} {
			if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{watched.ID()}); err != nil {
				t.Fatal(err)
			}
		}
		g.AddExileIfWouldGoToGraveyardThisTurn(watched.ID(), source.ID())
		clone := g.Clone()
		for _, game := range []*Game{g, clone} {
			if batch {
				game.DestroyPermanents([]*Permanent{game.FindPermanent(watched.ID()), game.FindPermanent(source.ID())})
			} else {
				game.DestroyPermanent(game.FindPermanent(watched.ID()))
			}
			if game.StackSize() != 0 {
				t.Fatal("replacement skipped pending trigger timing")
			}
			game.PutTriggersOnStack()
			if game.StackSize() != 1 {
				t.Fatalf("departure trigger count=%d want=1", game.StackSize())
			}
			game.ResolveStack()
			if game.GetPlayer(a.PlayerID()).Life() != 22 || len(game.GetPlayer(b.PlayerID()).Graveyard()) != 0 || game.trackers.Turn.CreatureDeaths() != 0 {
				t.Fatal("exile was treated as death")
			}
			exiled := game.FindExiledCard(watched.ID())
			if exiled == nil || exiled.ExiledBy != source.ID() {
				t.Fatal("replacement lost exile source")
			}
			returned, ok := game.RemoveFromExile(watched.ID())
			if !ok {
				t.Fatal("missing exiled card")
			}
			game.PutOnBattlefield(returned, b.PlayerID())
			game.ExilePermanent(game.FindPermanent(watched.ID()))
			game.PutTriggersOnStack()
			if game.StackSize() != 0 {
				t.Fatal("subscription followed returned incarnation")
			}
			game.RunStepWithPriority(core.Cleanup)
			if len(game.DelayedTriggers()) != 0 {
				t.Fatal("subscription survived cleanup")
			}
		}
	}
}
