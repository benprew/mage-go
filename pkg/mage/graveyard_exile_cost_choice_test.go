package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type graveyardExileChoicePlayer struct {
	*BasePlayer
	choices []uuid.UUID
	seen    [][]uuid.UUID
	mode    int
	modes   int
}

func (p *graveyardExileChoicePlayer) ChooseMode([]string, string) int {
	p.modes++
	return p.mode
}

func (p *graveyardExileChoicePlayer) ChooseCardFromGraveyard(cards []Card, _ string, _ GameReader) Card {
	ids := make([]uuid.UUID, 0, len(cards))
	for _, card := range cards {
		ids = append(ids, card.ID())
	}
	p.seen = append(p.seen, ids)
	if len(p.choices) == 0 {
		return nil
	}
	id := p.choices[0]
	p.choices = p.choices[1:]
	for _, card := range cards {
		if card.ID() == id {
			return card
		}
	}
	return nil
}

func TestGraveyardExileCostLocksChosenCards(t *testing.T) {
	a := &graveyardExileChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first, second, third := NewLand("First"), NewLand("Second"), NewLand("Third")
	for _, card := range []Card{first, second, third} {
		card.SetOwner(a.PlayerID())
		a.AddToGraveyard(card)
	}
	a.choices = []uuid.UUID{third.ID(), first.ID()}
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{ExileFromGraveyardCost(2)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.seen) != 2 || len(a.seen[0]) != 3 || len(a.seen[1]) != 2 {
		t.Fatalf("choices offered = %v", a.seen)
	}
	clone := g.Clone()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if g.FindExiledCard(third.ID()) == nil || g.FindExiledCard(first.ID()) == nil || g.FindExiledCard(second.ID()) != nil {
		t.Fatal("payment exiled cards other than the locked choices")
	}
	if len(g.GetPlayer(a.PlayerID()).Graveyard()) != 1 || len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 3 {
		t.Fatal("payment or clone changed the wrong graveyard")
	}
}

func TestGraveyardExileCostRejectsReturnedVisitBeforeOtherPayment(t *testing.T) {
	a := &graveyardExileChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	for _, card := range []Card{first, second} {
		card.SetOwner(a.PlayerID())
		a.AddToGraveyard(card)
	}
	a.choices = []uuid.UUID{second.ID()}
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{ExileFromGraveyardCost(1), LifePayCost(2)}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g.MoveFromGraveyard(a.PlayerID(), second.ID(), core.ZoneHand); !ok {
		t.Fatal("chosen card did not leave")
	}
	a.RemoveFromHand(second.ID())
	a.AddToGraveyard(second)
	if err := tx.Commit(); err == nil {
		t.Fatal("returned card paid a stale cost")
	}
	if a.Life() != 20 || len(g.GetExile()) != 0 || len(a.Graveyard()) != 2 {
		t.Fatal("failed payment changed live resources")
	}
}

func TestGraveyardExileCostRejectsInvalidChoiceAndCount(t *testing.T) {
	a := &graveyardExileChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	first.SetOwner(a.PlayerID())
	second.SetOwner(a.PlayerID())
	a.AddToGraveyard(first)
	a.AddToGraveyard(second)
	a.choices = []uuid.UUID{uuid.New()}
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{ExileFromGraveyardCost(1)}}); err == nil {
		t.Fatal("invalid choice paid the cost")
	}
	if len(a.Graveyard()) != 2 || len(g.GetExile()) != 0 {
		t.Fatal("invalid choice changed live zones")
	}
	card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(ExileFromGraveyardCost(-1)), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("negative cost count passed registration")
	}
	wrapped := NewArtifact("Wrapped", "", WithAbility(Activate(Costs(EitherCost(ExileFromGraveyardCost(-1), LifePayCost(1))), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(wrapped.Name(), wrapped); err == nil {
		t.Fatal("negative count inside a cost choice passed registration")
	}
}

func TestGraveyardExileCostInEitherCostKeepsModeAndCards(t *testing.T) {
	a := &graveyardExileChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	for _, card := range []Card{first, second} {
		card.SetOwner(a.PlayerID())
		a.AddToGraveyard(card)
	}
	a.ManaPool().Add(core.White, 2)
	a.mode = 0
	a.choices = []uuid.UUID{second.ID()}
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
		EitherCost(ExileFromGraveyardCost(1), ManaCostOf("{1}")),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if a.modes != 1 || g.FindExiledCard(second.ID()) == nil || g.FindExiledCard(first.ID()) != nil || a.ManaPool().TotalMana() != 2 {
		t.Fatalf("mode was not locked: choices=%d, exile=%v, mana=%d", a.modes, g.GetExile(), a.ManaPool().TotalMana())
	}
}
