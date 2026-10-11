package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBattlefieldAbsenceBatchAndReplacement(t *testing.T) {
	for _, scenario := range []string{"batch", "cancel", "exile"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			own := g.PutOnBattlefield(NewLand("Own land"), a.PlayerID())
			foreign := g.PutOnBattlefield(NewLand("Foreign land"), b.PlayerID())
			card := NewEnchantment("Observer", "", WithAbility(WhenState(BattlefieldHasNone(IsLand), Sacrifice(This))))
			card.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(NewEnchantment("Other", "", WithAbility(WhenState(BattlefieldHasNone(IsLand), Sacrifice(This)))), b.PlayerID())
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("condition ignored matching lands")
			}
			g.BouncePermanentToHand(own)
			if g.FindPermanent(foreign.ID()) == nil {
				t.Fatal("test moved foreign land")
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("foreign land did not prevent trigger")
			}
			if err := ReturnEachToHand(MatchingPermanents(IsLand)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 2 {
				t.Fatal("batch did not queue independent absence triggers")
			}
			if scenario == "cancel" {
				replacement := &testPrioritizedReplacement{matchFunc: func(action Action) bool { _, ok := action.(*SacrificePermanentAction); return ok }, replaceFunc: func(Action, *Game) Action { return nil }}
				g.effects.AddReplacement(replacement)
				g.ResolveTopOfStack()
				if g.FindPermanent(other.ID()) == nil {
					t.Fatal("canceled sacrifice moved subject")
				}
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 2 {
					t.Fatal("resolved trigger did not rearm")
				}
				g.effects.replacements = nil
			}
			if scenario == "exile" {
				g.AddExileIfWouldGoToGraveyardThisTurn(source.ID(), other.ID())
			}
			g.AddRegenerationShield(source.ID())
			g.MutablePermanent(source.ID()).GrantBaseAttr(core.Indestructible)
			clone := g.Clone()
			g.PutOnBattlefield(NewLand("New foreign land"), b.PlayerID())
			g.ResolveStack()
			if g.FindPermanent(source.ID()) != nil || g.FindPermanent(other.ID()) != nil {
				t.Fatal("sacrifice rechecked absence or used destruction")
			}
			if scenario == "exile" {
				if g.FindExiledCard(source.ID()) == nil || len(b.Graveyard()) != 1 {
					t.Fatal("replacement or owner destination lost")
				}
			} else if len(b.Graveyard()) != 2 {
				t.Fatal("sacrifice lost owner destination")
			}
			if clone.FindPermanent(source.ID()) == nil {
				t.Fatal("clone shared departure")
			}
			clone.ResolveStack()
			if clone.FindPermanent(source.ID()) != nil || clone.FindPermanent(other.ID()) != nil {
				t.Fatal("clone lost pending facts")
			}
		})
	}
}
