package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type optionalPaymentPlayer struct {
	*BasePlayer
	accept bool
}

func (p *optionalPaymentPlayer) ChooseMayAbility(string) bool { return p.accept }

func TestOptionalPaymentResultInstances(t *testing.T) {
	a, b := &acceptingMayPlayer{NewBasePlayer("A")}, NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("paid")
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(
		OptionalCostInto(LifePayCost(2), "Pay two life?", paid)),
		GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 {
		t.Fatalf("payment life = %d", a.Life())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 19 || clone.GetPlayer(b.PlayerID()).Life() != 21 || a.Life() != 18 {
		t.Fatal("optional result did not stay in its stack and game instances")
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 21 {
		t.Fatal("optional result escaped its payment instance")
	}
}

func TestOptionalPaymentResultDeclinedAndFailure(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(
		OptionalCostInto(LifePayCost(25), "Pay life?", paid)),
		GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatalf("declined or impossible payment changed life: %d", a.Life())
	}
	if err := ValidateCardDefinition("Nil", NewArtifact("Nil", "", WithAbility(Activate(Costs(
		OptionalCostInto(LifePayCost(1), "Pay?", nil)), GainLifeFor(You, Fixed(1)))))); err == nil || !strings.Contains(err.Error(), "non-nil") {
		t.Fatalf("nil binding validation = %v", err)
	}
	if err := ValidateCardDefinition("Wrong", NewArtifact("Wrong", "", WithAbility(Activate(Costs(
		OptionalCostInto(LifePayCost(1), "Pay?", paid)), IfCardHasType(FromCard(BindCard("card")), core.TypeCreature, GainLifeFor(You, Fixed(1))))))); err == nil {
		t.Fatal("unbound card was accepted")
	}
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{OptionalCostInto(LifePayCost(1), "", paid), "prompt"},
		{OptionalCostInto(LifePayCost(1), "Pay?", BindNumber("")), "name"},
		{OptionalCostInto(RevealFromHandCostWithCard(IsCreatureCard, "Reveal", BindCard("card")), "Reveal?", paid), "captured"},
		{OptionalCostInto(EitherCost(LifePayCost(1), LifePayCost(2)), "Pay?", paid), "direct"},
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v; want %s", err, tc.want)
		}
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		OptionalCostInto(LifePayCost(1), "First?", paid),
		OptionalCostInto(LifePayCost(1), "Second?", paid)), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate binding = %v", err)
	}
}

func TestOptionalPaymentResultPendingAndRollback(t *testing.T) {
	a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(
		OptionalCostInto(LifePayCost(2), "Pay life?", paid)),
		GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	a.accept = false
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 19 {
		t.Fatalf("pending optional outcomes = %d", a.Life())
	}
	a.accept = true
	failed := NewArtifact("Failed", "", WithAbility(Activate(Costs(
		OptionalCostInto(LifePayCost(2), "Pay life?", paid), LifePayCost(25)),
		GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(failed, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), failed.ID(), 0, nil); err == nil {
		t.Fatal("unpayable total cost was accepted")
	}
	if a.Life() != 19 || len(g.StackObjects()) != 0 {
		t.Fatal("failed total cost changed live payment state")
	}
}

func TestOptionalPaymentResultSpellAdditionalCost(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	spell := NewInstant("Optional", "", WithAbility(Spell(GainLifeFor(You, FromNumber(paid)))),
		WithAdditionalCost(OptionalCostInto(LifePayCost(2), "Pay life?", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 19 {
		t.Fatalf("spell payment result = %d", a.Life())
	}
}
