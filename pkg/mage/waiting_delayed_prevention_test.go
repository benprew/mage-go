package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedPreventionOperands(t *testing.T) {
	for name, body := range map[string]ResolvingEffect{
		"finite-recipient":    PreventNextDamage(This, Fixed(2)),
		"finite-player":       PreventNextDamage(ControllerOfWithLKI(This), Fixed(2)),
		"finite-amount":       PreventNextDamage(You, PowerOfWithLKI(This)),
		"finite-zero":         PreventNextDamage(You, MultiplyValues(Fixed(0), PowerOfWithLKI(This))),
		"unlimited-recipient": PreventDamageThisTurn(This, PermanentFilter{}),
		"unlimited-player":    PreventDamageThisTurn(ControllerOfWithLKI(This), PermanentFilter{}),
		"next-source":         PreventNextDamageFromPermanent(This, You, PermanentFilter{}),
		"next-player":         PreventNextDamageFromPermanent(AttachedPermanent, OwnerOf(This), PermanentFilter{}),
		"exact-source":        PreventNextDamageFromExactSource(This, You, DamageSourceFilter{}),
		"selected-source":     PreventNextDamageFromSelectedSource(DamageSourceOf(This), You, DamageSourceFilter{}),
		"selected-player":     PreventNextDamageFromSelectedSource(DamageSourceOf(AttachedPermanent), ControllerOfWithLKI(This), DamageSourceFilter{}),
	} {
		for _, state := range []string{"live", "phased", "cycle", "departed", "returned", "control"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Source", "{U}", 3, 20)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				delay := AtNext(EachEndStep, Sequence(body))
				if err := ValidateCardDefinition("Delay", NewArtifact("Delay", "", WithAbility(Activate(Costs(), delay)))); err != nil {
					t.Fatal(err)
				}
				if err := ApplyEffect(g, delay, p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				switch state {
				case "phased":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "departed", "returned":
					g.BouncePermanentToHand(p)
					g.zones.ClearLKI()
					if state == "returned" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, b.PlayerID())
						g.ChangeObjectColor(card.ID(), core.Red)
					}
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				for range 3 {
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(saved) || !got[0].HasColor(core.Blue) {
						t.Fatal("missing exact prevention operand", got)
					}
					if state == "control" && got[0].Controller() != b.PlayerID() {
						t.Fatal("lost current control")
					}
					got[0].card.AddType(core.TypeEnchantment)
					if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
						t.Fatal("query shares mutable facts")
					}
				}
				if len(g.WaitingPreventionPermanentSources()) != 0 || a.Life() != 20 {
					t.Fatal("query installed shield")
				}
				clone := g.Clone()
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
					t.Fatal("waiting lifetime shared")
				}
				g.PutTriggersOnStack()
				g.ResolveStack()
			})
		}
	}
}

func TestWaitingDelayedPreventionPurityAndScope(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewCreature("Source", "", 3, 20), a.PlayerID())
	reads := 0
	amount := AddValues(PowerOfWithLKI(This), ValueAtResolution(waitingTokenCount{&reads}))
	if err := ApplyEffect(g, AtNext(EachEndStep, PreventNextDamage(You, amount)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 {
			t.Fatal("query read amount")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.DealDamageToPlayer(a, 6, p.ID())
	if reads != 1 || a.Life() != 19 {
		t.Fatal("amount not read once at installation", reads, a.Life())
	}
	for _, body := range []ResolvingEffect{
		PreventNextDamage(You, Fixed(1)),
		PreventNextDamage(AttachedPermanent, Fixed(1)),
		PreventDamageThisTurn(You, PermanentFilter{}),
		PreventNextDamageFromPermanent(AttachedPermanent, You, PermanentFilter{}),
		PreventNextDamageFromSelectedSource(DamageSourceOf(AttachedPermanent), You, DamageSourceFilter{}),
		AtNext(EachEndStep, PreventNextDamage(This, Fixed(1))),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("implicit or future source inferred")
	}
}
