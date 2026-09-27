package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type paidLandChoicePlayer struct{ *BasePlayer }

func (p *paidLandChoicePlayer) ChooseCardsFromHand(_ int, _ string, _ GameReader) []Card {
	for _, card := range p.Hand() {
		if card.HasType(core.TypeLand) {
			return []Card{card}
		}
	}
	return nil
}

func TestPaidDiscardCardActivationAndClone(t *testing.T) {
	a, b := &paidLandChoicePlayer{NewBasePlayer("A")}, NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("discarded")
	source := NewArtifact("Edge", "", WithAbility(Activate(Costs(DiscardCardCostInto(paid)),
		IfCardHasType(FromCard(paid), core.TypeLand, GainLifeFor(You, Fixed(2))))))
	g.PutOnBattlefield(source, a.PlayerID())
	a.AddToHand(NewLand("Paid land"))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("paid discard capture escaped cloned resolution")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("paid land type was not available to the activation")
	}
}

func TestPaidDiscardCardDefinition(t *testing.T) {
	paid := BindCard("discarded")
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{DiscardCardCostInto(paid), ""},
		{DiscardCardCostInto(nil), "non-nil"},
	} {
		card := NewArtifact("Edge", "", WithAbility(Activate(Costs(tc.cost), IfCardHasType(FromCard(paid), core.TypeLand, GainLifeFor(You, Fixed(2))))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	missing := NewArtifact("Missing", "", WithAbility(Activate(Costs(), IfCardHasType(FromCard(paid), core.TypeLand, GainLifeFor(You, Fixed(2))))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer: %v", err)
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(DiscardCardCostInto(paid), DiscardCardCostInto(paid)), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate producer: %v", err)
	}
}

func TestPaidDiscardCardReplacementAndRollback(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindCard("discarded")
	land := NewLand("Paid land")
	a.AddToHand(land)
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{DiscardCardCostInto(paid)}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 1 || tx.paidCards[paid].visit != uuid.Nil {
		t.Fatal("hidden replacement exposed the discarded card")
	}

	b := NewBasePlayer("C")
	other := NewGame(b, NewBasePlayer("D"))
	b.AddToHand(NewLand("Paid land"))
	other.effects.AddReplacement(&preventCostDiscardReplacement{})
	if _, err := other.prepareActionPaymentTransaction(actionPaymentSpec{Controller: b.PlayerID(), Costs: []Cost{DiscardCardCostInto(paid)}}); err == nil {
		t.Fatal("prevented discard paid the cost")
	}
	if len(b.Hand()) != 1 || len(b.Graveyard()) != 0 {
		t.Fatal("failed payment changed live zones")
	}
}

func TestPaidDiscardCardSpellCopyAndVisit(t *testing.T) {
	a, b := &paidLandChoicePlayer{NewBasePlayer("A")}, NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("discarded")
	spell := NewInstant("Discard spell", "", WithAbility(Spell(
		IfCardHasType(FromCard(paid), core.TypeLand, GainLifeFor(You, Fixed(2))))),
		WithAdditionalCost(DiscardCardCostInto(paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	land := NewLand("Paid land")
	a.AddToHand(land)
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("stack copy did not keep its paid card")
	}

	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.resolution.cardBindings = make(map[*CardBinding]capturedCard)
	g.resolution.cardBindings[paid] = capturedCard{id: land.ID(), visit: g.graveyardVisit(land.ID()), zone: core.ZoneGraveyard}
	if err := ReturnCardToHand(FromCard(paid)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(paid).resolveCard(ctx); ok {
		t.Fatal("paid card followed a later zone visit")
	}
}
