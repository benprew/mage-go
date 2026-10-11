package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRequiredTapPaymentIdentity(t *testing.T) {
	for _, state := range []string{"normal", "tapped", "phased", "phase-cycle", "returned", "missing", "opponent"} {
		t.Run(state, func(t *testing.T) {
			a := &typedSacrificePlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 1, 1)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			binding := BindPermanent("subject")
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, binding).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "tapped":
				g.TapPermanent(p)
			case "phased":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "missing", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "opponent":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			paid := state == "normal" || state == "phase-cycle" || state == "opponent"
			effect := RequiredTapOrElse(You, From(binding), GainLifeFor(You, Fixed(2)), GainLifeFor(You, Fixed(3)))
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 23
			if paid {
				want = 22
			}
			if a.Life() != want || a.offers != 0 {
				t.Fatalf("life %d, offers %d", a.Life(), a.offers)
			}
			if state == "returned" && g.FindPermanent(p.ID()).Tapped {
				t.Fatal("returned subject paid")
			}
		})
	}
}

func TestRequiredTapPaymentDefinition(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		RequiredTapOrElse(nil, This, GainLifeFor(You, Fixed(1)), nil),
		RequiredTapOrElse(You, nil, GainLifeFor(You, Fixed(1)), nil),
		RequiredTapOrElse(You, This, nil, nil),
		RequiredTapOrElse(You, From(BindPermanent("unbound")), GainLifeFor(You, Fixed(1)), nil),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(), RequiredTapOrElse(You, This, nil, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
}

func TestRequiredTapPaymentRollback(t *testing.T) {
	for _, state := range []string{"tapped", "phased", "returned", "missing"} {
		t.Run(state, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			card := NewCreature("Subject", "", 1, 1)
			p := g.PutOnBattlefield(card, a.PlayerID())
			cost := &exactPermanentTapCost{subject: capturedPermanent{id: p.ID(), incarnation: p.incarnationID}}
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), cost}})
			if err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 || g.FindPermanent(p.ID()).Tapped {
				t.Fatal("preparation spent resources")
			}
			switch state {
			case "tapped":
				g.TapPermanent(p)
			case "phased":
				g.PhaseOut(p.ID())
			case "returned", "missing":
				g.RemoveFromBattlefield(p)
				if state == "returned" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			}
			if err := tx.Commit(); err == nil {
				t.Fatal("stale subject paid")
			}
			if a.Life() != 20 {
				t.Fatal("failed commit spent life")
			}
		})
	}
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewCreature("Subject", "", 1, 1), a.PlayerID())
	cost := &exactPermanentTapCost{subject: capturedPermanent{id: p.ID(), incarnation: p.incarnationID}}
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), cost, cost}}); err == nil {
		t.Fatal("one tap paid twice")
	}
	if a.Life() != 20 || g.FindPermanent(p.ID()).Tapped {
		t.Fatal("failed validation spent resources")
	}
}

func TestRequiredTapPaymentWaitingAndCopies(t *testing.T) {
	for _, state := range []string{"live", "returned", "departed", "phased", "phase-cycle"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Source", "{U}", WithAbility(Activate(Costs(), AtNext(EachEndStep, RequiredTapOrElse(You, This, GainLifeFor(You, Fixed(2)), GainLifeFor(You, Fixed(3)))))))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch state {
			case "departed", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			for range 2 {
				sources := g.WaitingDelayedPermanentSources()
				if len(sources) != 1 || !sources[0].SameObject(saved) {
					t.Fatal("waiting query lost exact visit")
				}
				if a.Life() != 20 {
					t.Fatal("waiting query executed")
				}
			}
			clone := g.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			want := 26
			if state == "live" || state == "phase-cycle" {
				want = 25
			}
			if a.Life() != want {
				t.Fatalf("life %d want %d", a.Life(), want)
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || len(clone.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("clone shared delay")
			}
			if state == "returned" && g.FindPermanent(p.ID()).Tapped {
				t.Fatal("copy tapped returned source")
			}
		})
	}
}
