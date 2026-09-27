package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidExileCardActivationAndVisit(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindCard("exiled card")
	source := NewArtifact("Synthetic", "", WithAbility(Activate(
		Costs(ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile a creature card", paid)),
		IfCardHasType(FromCard(paid), core.TypeCreature, GainLifeFor(You, Fixed(2))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	card := NewCreature("Payment", "{2}", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 || a.Life() != 20 {
		t.Fatal("clone did not keep the paid card binding")
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatal("paid card was absent at resolution")
	}
}

func TestPaidExileCardSpellAndFailedCost(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindCard("exiled card")
	spell := NewInstant("Synthetic", "", WithAbility(Spell(
		IfCardHasType(FromCard(paid), core.TypeCreature, GainLifeFor(You, Fixed(2))))),
		WithAdditionalCost(ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile a creature card", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	card := NewCreature("Payment", "", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("spell did not read paid card")
	}
	source := NewArtifact("Failure", "", WithAbility(Activate(Costs(
		ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile", paid), LifePayCost(25)),
		GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(source, a.PlayerID())
	second := NewCreature("Second", "", 1, 1)
	second.SetOwner(a.PlayerID())
	a.AddToGraveyard(second)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("unpayable cost committed")
	}
	if g.FindExiledCard(second.ID()) != nil || a.Life() != 22 || len(g.StackObjects()) != 0 {
		t.Fatal("failed payment changed live state")
	}
}

func TestPaidExileCardVisitAndRollback(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindCard("exiled card")
	card := NewCreature("Payment", "", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
		ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile", paid),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g.MoveFromGraveyard(a.PlayerID(), card.ID(), core.ZoneHand); !ok {
		t.Fatal("card did not leave graveyard")
	}
	a.AddToGraveyard(card)
	if err := tx.Commit(); err == nil || len(tx.paidCards) != 0 || len(g.GetExile()) != 0 {
		t.Fatalf("stale cost committed: %v", err)
	}
	tx, err = g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
		ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile", paid),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	ctx := &EffectContext{Game: g}
	g.resolution.cardBindings = tx.paidCards
	if got, ok := FromCard(paid).resolveCard(ctx); !ok || got.ID() != card.ID() {
		t.Fatal("paid exile visit was absent")
	}
	if _, ok := g.RemoveFromExile(card.ID()); !ok {
		t.Fatal("exiled card was absent")
	}
	g.ExileCard(card, card.ID())
	if _, ok := FromCard(paid).resolveCard(ctx); ok {
		t.Fatal("paid binding followed a new exile visit")
	}
}

func TestPaidExileCardDeclaration(t *testing.T) {
	paid := BindCard("paid")
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile", paid), ""},
		{ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile", nil), "non-nil"},
		{ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile", BindCard("")), "name"},
		{ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "", paid), "text"},
	} {
		card := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(tc.cost), IfCardHasType(FromCard(paid), core.TypeCreature, GainLifeFor(You, Fixed(1))))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "One", paid),
		ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Two", paid)),
		IfCardHasType(FromCard(paid), core.TypeCreature, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate binding: %v", err)
	}
}
