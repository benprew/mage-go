package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidDistinctDiscardCardsActivationCopiesAndVisits(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCards("paid")
	source := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(DiscardCardsWithDifferentNamesCostWithCards(2, paid)), GainLifeFor(You, CountCards(FromCards(paid))))))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	first, second := NewLand("First"), NewLand("Second")
	a.AddToHand(first)
	a.AddToHand(second)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	frame := cloneCardsBindings(g.StackPeek().capturedCardSets)
	if len(frame[paid]) != 2 {
		t.Fatalf("paid visits = %v", frame[paid])
	}
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	clone := g.Clone()
	if _, ok := clone.MoveFromGraveyard(a.PlayerID(), first.ID(), ZoneHand); !ok {
		t.Fatal("clone card absent")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("clone or copy frame lost isolation")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("copies did not retain the paid set")
	}
	if _, ok := g.MoveFromGraveyard(a.PlayerID(), first.ID(), ZoneHand); !ok {
		t.Fatal("card absent")
	}
	a.RemoveFromHand(first.ID())
	a.AddToGraveyard(first)
	g.resolution.cardsBindings = frame
	if got := CountCards(FromCards(paid)).resolveValue(&EffectContext{Game: g, Controller: a.PlayerID()}); got != 1 {
		t.Fatalf("returned graveyard visit counted: %d", got)
	}
}

func TestPaidDistinctDiscardCardsSpellAndIndependentPayments(t *testing.T) {
	a := &discardChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindCards("paid")
	cost := DiscardCardsWithDifferentNamesCostWithCards(2, paid)
	spell := NewInstant("Synthetic", "", WithAbility(Spell(GainLifeFor(You, CountCards(FromCards(paid))))), WithAdditionalCost(cost))
	a.AddToHand(spell)
	for _, name := range []string{"First", "Second"} {
		a.AddToHand(NewLand(name))
	}
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	old := cloneCardsBindings(g.StackPeek().capturedCardSets)
	for _, name := range []string{"Third", "Fourth"} {
		a.AddToHand(NewLand(name))
	}
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(tx.paidCardSets[paid]) != 2 || tx.paidCardSets[paid][0].id == old[paid][0].id {
		t.Fatal("reused declaration shared payment choices")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("later payment replaced spell capture")
	}
}

func TestPaidDistinctDiscardCardsDefinition(t *testing.T) {
	paid := BindCards("paid")
	valid := func(costs ...Cost) Card {
		return NewArtifact("Synthetic", "", WithAbility(Activate(Costs(costs...), GainLifeFor(You, CountCards(FromCards(paid))))))
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: valid(DiscardCardsWithDifferentNamesCostWithCards(2, paid))},
		{Name: "zero", Card: valid(DiscardCardsWithDifferentNamesCostWithCards(0, paid))},
		{Name: "nil", Card: valid(DiscardCardsWithDifferentNamesCostWithCards(2, nil)), WantErrSubstr: "non-nil"},
		{Name: "unnamed", Card: valid(DiscardCardsWithDifferentNamesCostWithCards(2, BindCards(""))), WantErrSubstr: "name"},
		{Name: "negative", Card: valid(DiscardCardsWithDifferentNamesCostWithCards(-1, paid)), WantErrSubstr: "nonnegative"},
		{Name: "duplicate", Card: valid(DiscardCardsWithDifferentNamesCostWithCards(1, paid), DiscardCardsWithDifferentNamesCostWithCards(1, paid)), WantErrSubstr: "duplicate"},
		{Name: "unbound", Card: valid(), WantErrSubstr: "producer"},
		{Name: "optional", Card: valid(OptionalCost(DiscardCardsWithDifferentNamesCostWithCards(2, paid), "Pay?")), WantErrSubstr: "direct cost"},
		{Name: "nested optional", Card: valid(OptionalCost(OptionalCost(DiscardCardsWithDifferentNamesCostWithCards(2, paid), "Pay?"), "Pay?")), WantErrSubstr: "direct cost"},
		{Name: "alternative", Card: valid(EitherCost(DiscardCardsWithDifferentNamesCostWithCards(2, paid), LifePayCost(2))), WantErrSubstr: "direct cost"},
		{Name: "resolution payment", Card: NewInstant("Bad", "", Spell(IfPaid(You, Costs(DiscardCardsWithDifferentNamesCostWithCards(2, paid)), GainLifeFor(You, Fixed(1))))), WantErrSubstr: "activated ability cost or spell additional cost"},
	})
}

func TestPaidDistinctDiscardCardsInterleavedTransactions(t *testing.T) {
	a := NewBasePlayer("A")
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCards("paid")
	cost := DiscardCardsWithDifferentNamesCostWithCards(2, paid)
	for _, p := range []*BasePlayer{a, b} {
		p.AddToHand(NewLand("First"))
		p.AddToHand(NewLand("Second"))
	}
	first, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: b.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	if err = second.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = first.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(first.paidCardSets[paid]) != 2 || len(second.paidCardSets[paid]) != 2 || first.paidCardSets[paid][0].id == second.paidCardSets[paid][0].id {
		t.Fatal("interleaved payments shared captures")
	}
	first.paidCardSets[paid][0] = capturedCard{}
	if second.paidCardSets[paid][0].visit == uuid.Nil {
		t.Fatal("transaction arrays shared storage")
	}
}

func TestPaidDistinctDiscardCardsConstraintsAndRollback(t *testing.T) {
	for _, mode := range []string{"same name", "returned", "prevented", "hidden", "zero"} {
		t.Run(mode, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			paid := BindCards("paid")
			first, second := NewLand("First"), NewLand("Second")
			if mode == "same name" {
				second = NewLand("First")
			}
			a.AddToHand(first)
			a.AddToHand(second)
			switch mode {
			case "prevented":
				g.effects.AddReplacement(&preventOneCostDiscardReplacement{card: second.ID()})
			case "hidden":
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
			}
			n := 2
			if mode == "zero" {
				n = 0
			}
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), DiscardCardsWithDifferentNamesCostWithCards(n, paid)}})
			if mode == "returned" {
				if err != nil {
					t.Fatal(err)
				}
				a.RemoveFromHand(first.ID())
				a.AddToHand(first)
			}
			if err == nil {
				err = tx.Commit()
			}
			if mode == "zero" {
				if err != nil || a.Life() != 18 || len(tx.paidCardSets[paid]) != 0 {
					t.Fatalf("zero capture: tx=%+v err=%v", tx, err)
				}
			} else if err == nil || a.Life() != 20 {
				t.Fatal("invalid payment spent life or succeeded")
			}
			if len(a.Hand()) != 2 || len(a.Graveyard()) != 0 || len(a.Library()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 {
				t.Fatal("failed or empty payment moved cards or published events")
			}
			if mode != "zero" && tx != nil && len(tx.paidCardSets) != 0 {
				t.Fatal("failed payment published a capture")
			}
		})
	}
}
