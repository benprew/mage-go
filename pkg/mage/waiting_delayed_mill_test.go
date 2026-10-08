package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedMillOperands(t *testing.T) {
	for name, body := range map[string]ResolvingEffect{
		"controller":  Mill(ControllerOfWithLKI(This), Fixed(0)),
		"owner":       Mill(OwnerOf(This), Fixed(0)),
		"amount":      Mill(You, PowerOfWithLKI(This)),
		"zero amount": Mill(You, SubtractValues(PowerOfWithLKI(This), PowerOfWithLKI(This))),
	} {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "{U}", 2, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := ApplyEffect(g, AtNext(EachEndStep, Sequence(body)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check := func() {
				t.Helper()
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || !got[0].HasColor(core.Blue) {
					t.Fatal("mill lost exact operand", got)
				}
				got[0].card.AddType(core.TypeEnchantment)
				if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
					t.Fatal("query shared facts")
				}
			}
			check()
			phased := g.PhaseOut(p.ID())
			check()
			g.PhaseIn(phased)
			check()
			g.BouncePermanentToHand(p)
			g.zones.ClearLKI()
			check()
			a.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, b.PlayerID())
			g.ChangeObjectColor(card.ID(), core.Red)
			check()
			clone := g.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("mill wait lifetime shared")
			}
		})
	}
}

func TestWaitingDelayedMillPurityAndScope(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 4), a.PlayerID())
	for range 4 {
		a.AddToLibrary(NewLand("Land"))
	}
	reads := 0
	body := Mill(ControllerOfWithLKI(This), AddValues(PowerOfWithLKI(This), ValueAtResolution(waitingTokenCount{&reads})))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || len(a.Library()) != 4 {
			t.Fatal("query evaluated mill")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if reads != 1 || len(a.Library()) != 0 || len(a.Graveyard()) != 4 || len(b.Graveyard()) != 0 {
		t.Fatal("mill timing or player", reads, len(a.Library()))
	}
	for _, body := range []ResolvingEffect{
		Mill(You, Fixed(0)), Mill(You, PowerOfWithLKI(AttachedPermanent)),
		AtNext(EachEndStep, Mill(ControllerOfWithLKI(This), Fixed(0))),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("inferred implicit or future reads")
	}
	for _, body := range []ResolvingEffect{Mill(nil, Fixed(1)), Mill(You, nil), Mill(You, Fixed(-1))} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid mill accepted")
		}
	}
}
