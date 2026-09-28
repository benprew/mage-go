package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidGraveyardExileCardsActivationAndVisits(t *testing.T) {
	a := &graveyardExileChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	for _, card := range []Card{first, second} {
		card.SetOwner(a.PlayerID())
		a.AddToGraveyard(card)
	}
	a.choices = []uuid.UUID{second.ID(), first.ID()}
	paid := BindCards("exiled cards")
	source := NewArtifact("Synthetic", "", WithAbility(Activate(
		Costs(ExileFromGraveyardCostWithCards(2, paid)),
		GainLifeFor(You, CountCards(FromCards(paid))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	frame := cloneCardsBindings(g.StackObjects()[0].capturedCardSets)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.resolution.cardsBindings = frame
	if got := CountCards(FromCards(paid)).resolveValue(ctx); got != 2 {
		t.Fatalf("paid cards = %d", got)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("clone lost paid exile visits")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("activation lost paid exile visits")
	}
	if _, ok := g.RemoveFromExile(first.ID()); !ok {
		t.Fatal("first card not exiled")
	}
	g.ExileCard(first, source.ID())
	g.resolution.cardsBindings = frame
	if got := CountCards(FromCards(paid)).resolveValue(ctx); got != 1 {
		t.Fatalf("returned visit counted: %d", got)
	}
}

func TestPaidGraveyardExileCardsDeclaration(t *testing.T) {
	paid := BindCards("paid")
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(ExileFromGraveyardCostWithCards(2, paid)), GainLifeFor(You, CountCards(FromCards(paid))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{ExileFromGraveyardCostWithCards(1, nil), "non-nil"},
		{ExileFromGraveyardCostWithCards(1, BindCards("")), "name"},
		{ExileFromGraveyardCostWithCards(-1, paid), "negative"},
		{EitherCost(ExileFromGraveyardCostWithCards(1, paid), LifePayCost(1)), "direct cost"},
	} {
		card := NewArtifact("Bad", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		ExileFromGraveyardCostWithCards(0, paid), ExileFromGraveyardCostWithCards(0, paid)),
		GainLifeFor(You, CountCards(FromCards(paid))))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate validation = %v", err)
	}
	unbound := NewInstant("Unbound", "", Spell(GainLifeFor(You, CountCards(FromCards(paid)))))
	if err := ValidateCardDefinition(unbound.Name(), unbound); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("unbound validation = %v", err)
	}
}

func TestPaidGraveyardExileCardsSpellCopyAndRollback(t *testing.T) {
	a := &graveyardExileChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	for _, card := range []Card{first, second} {
		card.SetOwner(a.PlayerID())
		a.AddToGraveyard(card)
	}
	paid := BindCards("paid")
	spell := NewInstant("Synthetic", "", WithAbility(Spell(GainLifeFor(You, CountCards(FromCards(paid))))),
		WithAdditionalCost(ExileFromGraveyardCostWithCards(2, paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.choices = []uuid.UUID{first.ID(), second.ID()}
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	copiedSpell := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
	if copiedSpell == nil {
		t.Fatal("copy absent")
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatalf("copied spell life = %d", a.Life())
	}

	third := NewLand("Third")
	third.SetOwner(a.PlayerID())
	a.AddToGraveyard(third)
	a.choices = []uuid.UUID{third.ID()}
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
		ExileFromGraveyardCostWithCards(1, paid), LifePayCost(2),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g.MoveFromGraveyard(a.PlayerID(), third.ID(), core.ZoneHand); !ok {
		t.Fatal("card did not leave")
	}
	a.RemoveFromHand(third.ID())
	a.AddToGraveyard(third)
	if err := tx.Commit(); err == nil || len(tx.paidCardSets) != 0 || a.Life() != 24 || g.FindExiledCard(third.ID()) != nil {
		t.Fatalf("stale payment changed state: %v", err)
	}
}
