package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestMillCostOptionalPayment(t *testing.T) {
	for _, count := range []int{0, 1, 2, 3} {
		a := &acceptingMayPlayer{NewBasePlayer("A")}
		g := NewGame(a, NewBasePlayer("B"))
		for range count {
			card := NewLand("Library card")
			card.SetOwner(a.PlayerID())
			a.AddToLibrary(card)
		}
		if err := ApplyEffect(g, UnlessPaid(You, Costs(MillCost(2)), GainLifeFor(You, Fixed(3))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if want := max(0, count-2); count >= 2 && len(a.Library()) != want {
			t.Fatalf("%d cards: library = %d, want %d", count, len(a.Library()), want)
		}
		if count < 2 && len(a.Library()) != count {
			t.Fatalf("%d cards: failed cost changed library", count)
		}
		wantLife := 20
		if count < 2 {
			wantLife = 23
		}
		if a.Life() != wantLife {
			t.Fatalf("%d cards: life = %d, want %d", count, a.Life(), wantLife)
		}
	}
}

func TestMillCostDeclinedPayment(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	for range 2 {
		card := NewLand("Library card")
		card.SetOwner(a.PlayerID())
		a.AddToLibrary(card)
	}
	if err := ApplyEffect(g, UnlessPaid(You, Costs(MillCost(2)), GainLifeFor(You, Fixed(3))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 2 || a.Life() != 23 {
		t.Fatal("declined cost changed library or missed unpaid branch")
	}
}

func TestMillCostReplacementAndRollback(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	for range 3 {
		card := NewLand("Library card")
		card.SetOwner(a.PlayerID())
		a.AddToLibrary(card)
	}
	source := NewCreature("Modifier", "", 1, 1)
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddMillReplacement(source.ID(), func(_ *Game, _ uuid.UUID, n int) int { return n * 2 })
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), MillCost(2)}}); err == nil {
		t.Fatal("replacement increased cost beyond available library")
	}
	if a.Life() != 20 || len(a.Library()) != 3 {
		t.Fatal("failed preparation changed live state")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), MillCost(2)}})
	if err != nil {
		t.Fatal(err)
	}
	a.SetLibrary(a.Library()[:1])
	if err := tx.Commit(); err == nil {
		t.Fatal("stale library paid cost")
	}
	if a.Life() != 20 || len(a.Library()) != 1 {
		t.Fatal("failed commit changed live state")
	}
}

func TestMillCostReplacementChangeAndClone(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	for range 4 {
		card := NewLand("Library card")
		card.SetOwner(a.PlayerID())
		a.AddToLibrary(card)
	}
	source := NewCreature("Modifier", "", 1, 1)
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddMillReplacement(source.ID(), func(_ *Game, _ uuid.UUID, n int) int { return n * 2 })
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{MillCost(2)}})
	if err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.GetPlayer(a.PlayerID()).SetLibrary(clone.GetPlayer(a.PlayerID()).Library()[:2])
	if len(a.Library()) != 4 {
		t.Fatal("clone changed original library")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	if err := tx.Commit(); err == nil {
		t.Fatal("changed replacement paid stale locked cost")
	}
	if len(a.Library()) != 4 {
		t.Fatal("failed commit milled cards")
	}
	if err := ApplyEffect(g, UnlessPaid(You, Costs(MillCost(2)), GainLifeFor(You, Fixed(3))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 2 || a.Life() != 20 {
		t.Fatal("new payment did not use current mill rule")
	}
}

func TestMillCostZeroReplacement(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	source := NewCreature("Modifier", "", 1, 1)
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddMillReplacement(source.ID(), func(_ *Game, _ uuid.UUID, _ int) int { return 0 })
	if err := ApplyEffect(g, UnlessPaid(You, Costs(MillCost(2)), GainLifeFor(You, Fixed(3))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 {
		t.Fatal("replacement made zero-card payment impossible")
	}
}

func TestMillCostMultipleComponentsRecheck(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	for range 4 {
		card := NewLand("Library card")
		card.SetOwner(a.PlayerID())
		a.AddToLibrary(card)
	}
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), MillCost(2), MillCost(2)}})
	if err != nil {
		t.Fatal(err)
	}
	a.SetLibrary(a.Library()[:3])
	if err := tx.Commit(); err == nil {
		t.Fatal("multiple mill components paid with too few cards")
	}
	if a.Life() != 20 || len(a.Library()) != 3 {
		t.Fatal("failed payment changed resources")
	}
}

func TestMillCostDeclaration(t *testing.T) {
	valid := NewCreature("Valid", "", 1, 1, WithAbility(At(YourUpkeep, UnlessPaid(You, Costs(MillCost(2)), Sacrifice(This)))))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{-1, 0} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(At(YourUpkeep, UnlessPaid(You, Costs(MillCost(n)), Sacrifice(This)))))
		if err := ValidateCardDefinition("Invalid", card); err == nil || !strings.Contains(err.Error(), "positive") {
			t.Fatalf("count %d: validation = %v", n, err)
		}
	}
}
