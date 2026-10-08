package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedExileOperands(t *testing.T) {
	shapes := map[string]ResolvingEffect{
		"single":      Exile(This),
		"singleton":   ExileEach(SingletonPermanent(This)),
		"auras":       ExileEach(AttachedAuras(This)),
		"attachments": ExileEach(AttachmentsOn(SingletonPermanent(This))),
		"blockers":    ExileEach(BlockersOf(This)),
		"blocked":     ExileEach(BlockedAttackersOf(This)),
		"combat":      ExileEach(BlockingOrBlockedBy(This)),
		"history":     ExileEach(BlockedByThisTurn(This)),
		"excluded":    ExileEach(WithoutPermanent(MatchingPermanents(IsCreature), This)),
		"lowest":      ExileEach(LowestPower(SingletonPermanent(This))),
		"controller":  ExileEach(PermanentsControlledBy(ControllerOfWithLKI(This), AnyPermanent)),
		"owner":       ExileEach(PermanentsOwnedBy(OwnerOf(This), AnyPermanent)),
		"opponents":   ExileEach(PermanentsControlledByPlayers(OpponentsOf(ControllerOfWithLKI(This)), AnyPermanent)),
		"limit":       ExileEach(ManaValueAtMost(MatchingPermanents(IsCreature), PowerOf(This))),
	}
	for name, body := range shapes {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 1, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := ApplyEffect(g, AtNext(EachEndStep, Sequence(body)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.BouncePermanentToHand(p)
			a.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, b.PlayerID())
			g.ChangeObjectColor(returned.ID(), core.Red)
			for range 3 {
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || got[0].HasColor(core.Red) {
					t.Fatal("exile action lost exact source declaration")
				}
				got[0].card.AddType(core.TypeEnchantment)
				if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
					t.Fatal("query shares mutable facts")
				}
			}
			clone := g.Clone()
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("clone shares waiting lifetime")
			}
		})
	}
}

func TestWaitingDelayedExilePurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	reads := 0
	filter := NewPermanentFilter("must not run in query", func(*Permanent, *Game) bool { reads++; return false })
	body := ExileEach(FilterPermanentSet(UnionPermanents(SingletonPermanent(This), SingletonPermanent(This)), filter))
	if err := ApplyEffect(g, AtNextModes(EachEndStep, ModeOf("first", body), ModeOf("second", Exile(This))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || len(g.GetExile()) != 0 {
		t.Fatal("query executed exile or lost empty/mode declaration")
	}
	g.triggers.SetDelayed(nil)
	for _, body := range []ResolvingEffect{Exile(AttachedPermanent), ExileEach(MatchingPermanents(filter)), ExileEach(waitingMultiEmptySet{&reads}), AtNext(EachEndStep, Exile(This))} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 {
		t.Fatal("query inferred implicit, opaque or future source")
	}
	for _, body := range []ResolvingEffect{Exile(nil), ExileEach(nil), ExileEach(SingletonPermanent(nil))} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid exile declaration accepted")
		}
	}
}
