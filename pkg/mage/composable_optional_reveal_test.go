package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestOptionalRevealPaymentInstances(t *testing.T) {
	a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	paid, revealed := BindNumber("paid"), BindCard("revealed")
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(
		OptionalRevealFromHandCost(IsCreatureCard, "Reveal a creature", "Reveal?", paid, revealed)),
		Sequence(GainLifeFor(You, FromNumber(paid)),
			IfCardHasType(FromCard(revealed), core.TypeCreature, GainLifeFor(You, Fixed(2)))))))
	g.PutOnBattlefield(source, a.PlayerID())
	card := NewCreature("Creature", "", 1, 1)
	a.AddToHand(card)
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	a.accept = false
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if got := len(g.PublicReveals()); got != 1 {
		t.Fatalf("public reveals = %d", got)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || clone.GetPlayer(b.PlayerID()).Life() != 23 || a.Life() != 20 {
		t.Fatalf("optional reveal values escaped their stack or game instances: clone A=%d B=%d live A=%d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life(), a.Life())
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 23 {
		t.Fatal("optional reveal values changed after clone resolution")
	}
}

func TestOptionalRevealPaymentVisitAndRollback(t *testing.T) {
	a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	paid, revealed := BindNumber("paid"), BindCard("revealed")
	card := NewCreature("Creature", "", 1, 1)
	a.AddToHand(card)
	cost := OptionalRevealFromHandCost(IsCreatureCard, "Reveal a creature", "Reveal?", paid, revealed)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := a.RemoveFromHand(card.ID()); !ok {
		t.Fatal("card did not leave hand")
	}
	a.AddToHand(card)
	if err := tx.Commit(); err == nil || len(tx.paidCards) != 0 || len(g.PublicReveals()) != 0 {
		t.Fatalf("stale optional reveal committed: %v", err)
	}
	if a.Life() != 20 {
		t.Fatal("failed payment changed life")
	}
	failed, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost, LifePayCost(25)}})
	if err == nil || failed != nil || len(g.PublicReveals()) != 0 {
		t.Fatalf("failed total payment changed live state: %v", err)
	}
}

func TestOptionalRevealSpellAndMissingCard(t *testing.T) {
	a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	paid, revealed := BindNumber("paid"), BindCard("revealed")
	makeSpell := func(name string) Card {
		return NewInstant(name, "",
			WithAbility(Spell(Sequence(
				GainLifeFor(You, FromNumber(paid)),
				IfCardHasType(FromCard(revealed), core.TypeCreature, GainLifeFor(You, Fixed(2)))))),
			WithAdditionalCost(OptionalRevealFromHandCost(IsCreatureCard, "Reveal a creature", "Reveal?", paid, revealed)))
	}
	first := makeSpell("First")
	first.SetOwner(a.PlayerID())
	a.AddToHand(first)
	if err := ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), first.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 20 || len(g.PublicReveals()) != 0 {
		t.Fatal("missing matching card paid an optional reveal")
	}
	second := makeSpell("Second")
	second.SetOwner(a.PlayerID())
	a.AddToHand(second)
	a.AddToHand(NewCreature("Creature", "", 1, 1))
	if err := g.CastSpellByName(a.PlayerID(), second.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 23 || len(g.PublicReveals()) != 1 {
		t.Fatalf("spell reveal result: life=%d reveals=%d", a.Life(), len(g.PublicReveals()))
	}
}

func TestOptionalRevealPaymentDeclaration(t *testing.T) {
	paid, revealed := BindNumber("paid"), BindCard("revealed")
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{OptionalRevealFromHandCost(IsCreatureCard, "Reveal", "Reveal?", paid, revealed), ""},
		{OptionalRevealFromHandCost(IsCreatureCard, "Reveal", "Reveal?", nil, revealed), "non-nil"},
		{OptionalRevealFromHandCost(IsCreatureCard, "Reveal", "Reveal?", paid, nil), "non-nil"},
		{OptionalRevealFromHandCost(IsCreatureCard, "", "Reveal?", paid, revealed), "text"},
		{OptionalRevealFromHandCost(IsCreatureCard, "Reveal", "", paid, revealed), "prompt"},
	} {
		card := NewArtifact("Test", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v; want %q", err, tc.want)
		}
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		OptionalRevealFromHandCost(IsCreatureCard, "Reveal", "Reveal?", paid, revealed),
		OptionalRevealFromHandCost(IsCreatureCard, "Reveal", "Reveal?", paid, BindCard("second"))), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate binding = %v", err)
	}
}
