package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedUntapOperands(t *testing.T) {
	shapes := map[string]ResolvingEffect{
		"single":      SkipNextUntap(This),
		"counted":     SkipNextUntaps(This, 2),
		"zero":        SkipNextUntaps(This, 0),
		"captured":    SkipNextUntapOf(This, You),
		"player":      SkipNextUntapOf(AttachedPermanent, ControllerOfWithLKI(This)),
		"singleton":   SkipNextUntapEach(SingletonPermanent(This)),
		"auras":       SkipNextUntapEach(AttachedAuras(This)),
		"attachments": SkipNextUntapEach(AttachmentsOn(SingletonPermanent(This))),
		"blockers":    SkipNextUntapEach(BlockersOf(This)),
		"blocked":     SkipNextUntapEach(BlockedAttackersOf(This)),
		"combat":      SkipNextUntapEach(BlockingOrBlockedBy(This)),
		"history":     SkipNextUntapEach(BlockedByThisTurn(This)),
		"excluded":    SkipNextUntapEach(WithoutPermanent(MatchingPermanents(IsCreature), This)),
		"lowest":      SkipNextUntapEach(LowestPower(SingletonPermanent(This))),
		"controller":  SkipNextUntapEach(PermanentsControlledBy(ControllerOfWithLKI(This), AnyPermanent)),
		"owner":       SkipNextUntapEach(PermanentsOwnedBy(OwnerOf(This), AnyPermanent)),
		"opponents":   SkipNextUntapEach(PermanentsControlledByPlayers(OpponentsOf(ControllerOfWithLKI(This)), AnyPermanent)),
		"limit":       SkipNextUntapEach(ManaValueAtMost(MatchingPermanents(IsCreature), PowerOf(This))),
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
					t.Fatal("untap restriction lost exact source declaration")
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

func TestWaitingDelayedUntapPurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	reads := 0
	filter := NewPermanentFilter("must not run in query", func(*Permanent, *Game) bool { reads++; return false })
	body := SkipNextUntapEach(FilterPermanentSet(UnionPermanents(SingletonPermanent(This), SingletonPermanent(This)), filter))
	if err := ApplyEffect(g, AtNextModes(EachEndStep, ModeOf("first", body), ModeOf("second", SkipNextUntap(This))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || g.turns.HasSkipNextUntap(p.ID()) {
		t.Fatal("query installed a restriction or lost empty/mode declaration")
	}
	g.triggers.SetDelayed(nil)
	for _, body := range []ResolvingEffect{SkipNextUntap(AttachedPermanent), SkipNextUntapEach(MatchingPermanents(filter)), SkipNextUntapEach(waitingMultiEmptySet{&reads}), AtNext(EachEndStep, SkipNextUntap(This))} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 {
		t.Fatal("query inferred implicit, opaque or future source")
	}
	for _, body := range []ResolvingEffect{SkipNextUntap(nil), SkipNextUntaps(This, -1), SkipNextUntapOf(This, nil), SkipNextUntapEach(nil), SkipNextUntapEach(SingletonPermanent(nil))} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid untap declaration accepted")
		}
	}
}
