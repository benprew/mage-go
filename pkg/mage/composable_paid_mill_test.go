package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPaidMillCountUsesCommittedAmountAndFrames(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	for range 6 {
		card := NewLand("Library card")
		card.SetOwner(a.PlayerID())
		a.AddToLibrary(card)
	}
	paid := BindNumber("milled")
	source := NewArtifact("Mill source", "", WithAbility(Activate(Costs(MillCostInto(2, paid)), GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	modifier := NewCreature("Modifier", "", 1, 1)
	g.PutOnBattlefield(modifier, a.PlayerID())
	g.AddMillReplacement(modifier.ID(), func(_ *Game, _ uuid.UUID, n int) int { return n + 1 })
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(modifier.ID()))
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 1 {
		t.Fatalf("library = %d", len(a.Library()))
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 28 {
		t.Fatalf("clone life = %d", got)
	}
	if a.Life() != 20 {
		t.Fatal("clone changed original life")
	}
	g.ResolveStack()
	if a.Life() != 28 {
		t.Fatalf("life = %d", a.Life())
	}
}

func TestPaidMillCountRollbackAndDeclaration(t *testing.T) {
	paid := BindNumber("milled")
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{MillCostInto(2, paid), ""},
		{MillCostInto(0, paid), "positive"},
		{MillCostInto(2, nil), "non-nil"},
		{MillCostInto(2, BindNumber("")), "name"},
		{EitherCost(MillCostInto(2, paid), LifePayCost(1)), "direct cost"},
	} {
		card := NewArtifact("Test", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(MillCostInto(1, paid), MillCostInto(1, paid)), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil {
		t.Fatal("duplicate producer accepted")
	}
	optional := NewArtifact("Optional", "", WithAbility(Activate(Costs(), IfPaid(You, Costs(MillCostInto(1, paid)), GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(optional.Name(), optional); err == nil {
		t.Fatal("resolution-time capture accepted")
	}
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	for range 2 {
		card := NewLand("Library card")
		card.SetOwner(a.PlayerID())
		a.AddToLibrary(card)
	}
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), MillCostInto(2, paid)}})
	if err != nil {
		t.Fatal(err)
	}
	a.SetLibrary(a.Library()[:1])
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || a.Life() != 20 || len(a.Library()) != 1 {
		t.Fatalf("rollback: %v, %v", err, tx.paidNumbers)
	}
}

func TestPaidMillCountZeroAndSpellCopy(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("milled")
	spell := NewInstant("Mill spell", "", WithAbility(Spell(GainLifeFor(You, FromNumber(paid)))), WithAdditionalCost(MillCostInto(2, paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	modifier := NewCreature("Modifier", "", 1, 1)
	g.PutOnBattlefield(modifier, a.PlayerID())
	g.AddMillReplacement(modifier.ID(), func(_ *Game, _ uuid.UUID, _ int) int { return 0 })
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	if n := g.StackObjects()[0].capturedNumbers[paid]; n != 0 {
		t.Fatalf("captured count = %d", n)
	}
	g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatalf("zero payment changed life: %d", a.Life())
	}
}
