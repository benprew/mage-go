package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidRevealCardKeepsOneHandVisitPerActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("revealed")
	source := NewArtifact("Watcher", "", WithAbility(Activate(
		Costs(RevealFromHandCostWithCard(IsCreatureCard, "Reveal a creature", paid)),
		IfCardHasType(FromCard(paid), core.TypeCreature, GainLifeFor(You, Fixed(2))))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	land := NewCreature("Creature", "", 1, 1)
	a.AddToHand(land)
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.PublicReveals()) != 2 {
		t.Fatal("reveal costs were not made public on commit")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 || clone.GetPlayer(b.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("reveal binding did not stay in each stack frame")
	}
	if _, ok := a.RemoveFromHand(land.ID()); !ok {
		t.Fatal("revealed card left hand")
	}
	a.AddToHand(land)
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("a returned hand card kept an old reveal binding")
	}
}

func TestPaidRevealCardRegistrationAndRollback(t *testing.T) {
	paid := BindCard("revealed")
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{RevealFromHandCostWithCard(IsCreatureCard, "Reveal a creature", paid), ""},
		{RevealFromHandCostWithCard(IsCreatureCard, "Reveal a creature", nil), "non-nil"},
		{RevealFromHandCostWithCard(IsCreatureCard, "Reveal a creature", BindCard("")), "name"},
		{RevealFromHandCostWithCard(IsCreatureCard, "", paid), "text"},
	} {
		card := NewArtifact("Watcher", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	land := NewCreature("Creature", "", 1, 1)
	a.AddToHand(land)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
		RevealFromHandCostWithCard(IsCreatureCard, "Reveal a creature", paid), LifePayCost(25),
	}})
	if err == nil || tx != nil || a.Life() != 20 || len(a.Hand()) != 1 || len(g.PublicReveals()) != 0 {
		t.Fatalf("failed total payment changed game: %v", err)
	}
	missing := NewArtifact("Missing", "", WithAbility(Activate(Costs(), IfCardHasType(FromCard(paid), core.TypeCreature, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing binding: %v", err)
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		RevealFromHandCostWithCard(IsCreatureCard, "Reveal one", paid),
		RevealFromHandCostWithCard(IsCreatureCard, "Reveal two", paid)), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate binding: %v", err)
	}
}

func TestPaidRevealCardRejectsStalePayment(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindCard("revealed")
	card := NewCreature("Creature", "", 1, 1)
	a.AddToHand(card)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
		RevealFromHandCostWithCard(IsCreatureCard, "Reveal a creature", paid),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := a.RemoveFromHand(card.ID()); !ok {
		t.Fatal("card did not leave hand")
	}
	a.AddToHand(card)
	if err := tx.Commit(); err == nil || len(tx.paidCards) != 0 || a.handVisit(card.ID()) == uuid.Nil {
		t.Fatalf("stale payment committed: %v", err)
	}
}
