package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestTapAttachedPermanentCostPayment(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	land := NewLand("Host")
	host := g.PutOnBattlefield(land, a.PlayerID())
	aura := NewAura("Source", "", WithCastTarget(TargetLand()))
	source := g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(source.ID(), host.ID())
	cost := TapAttachedPermanentCost()
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(host.ID()).Tapped || g.FindPermanent(source.ID()).Tapped {
		t.Fatal("payment did not tap only the attached permanent")
	}
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}}); err == nil {
		t.Fatal("already tapped host paid again")
	}
}

func TestTapAttachedPermanentCostStaleAttachmentAndRollback(t *testing.T) {
	for _, change := range []string{"detach", "reattach", "phase", "return", "tap", "source return"} {
		t.Run(change, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			host := g.PutOnBattlefield(NewLand("Host"), a.PlayerID())
			other := g.PutOnBattlefield(NewLand("Other"), a.PlayerID())
			auraCard := NewAura("Source", "", WithCastTarget(TargetLand()))
			source := g.PutOnBattlefield(auraCard, a.PlayerID())
			g.Attach(source.ID(), host.ID())
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{LifePayCost(2), TapAttachedPermanentCost()}})
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "detach":
				g.MutablePermanent(source.ID()).AttachedTo = other.ID()
			case "reattach":
				g.Attach(source.ID(), other.ID())
			case "phase":
				g.PhaseOut(host.ID())
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				g.PutOnBattlefield(host.Card, a.PlayerID())
			case "tap":
				g.TapPermanent(g.FindPermanent(host.ID()))
			case "source return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(auraCard, a.PlayerID())
			}
			clone := g.Clone()
			if err := tx.Commit(); err == nil {
				t.Fatal("stale attached cost paid")
			}
			if a.Life() != 20 || g.FindPermanent(other.ID()).Tapped || clone.FindPermanent(other.ID()).Tapped {
				t.Fatal("failed payment changed live state or clone")
			}
		})
	}
}

func TestTapAttachedPermanentCostDeclaration(t *testing.T) {
	valid := NewAura("Valid", "", WithAbility(Activate(Costs(TapAttachedPermanentCost()), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	invalid := NewInstant("Invalid", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)))), WithAdditionalCost(TapAttachedPermanentCost()))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("invalid cost validation = %v", err)
	}
}

func TestTapAttachedPermanentCostAbsentAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	host := g.PutOnBattlefield(NewLand("Host"), a.PlayerID())
	source := g.PutOnBattlefield(NewAura("Source", "", WithCastTarget(TargetLand())), a.PlayerID())
	cost := TapAttachedPermanentCost()
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}}); err == nil {
		t.Fatal("unattached source paid")
	}
	g.Attach(source.ID(), host.ID())
	g.PhaseOut(host.ID())
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}}); err == nil {
		t.Fatal("phased host paid")
	}
	g.PhaseIn([]uuid.UUID{host.ID()})
	source = g.PutOnBattlefield(NewAura("Replacement source", "", WithCastTarget(TargetLand())), a.PlayerID())
	g.Attach(source.ID(), host.ID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(host.ID()).Tapped || clone.FindPermanent(host.ID()).Tapped {
		t.Fatal("payment changed cloned host")
	}
}
