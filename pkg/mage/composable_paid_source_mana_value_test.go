package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidSourceSacrificeManaValue(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("paid mana value")
	source := NewCreature("Source", "{3}{U}", 1, 1, WithAbility(Activate(
		Costs(SacrificeMatchingIncludingSourceCostWithManaValue(And(IsCreature, HasColorFilter(core.Blue)), "Sacrifice a blue creature", paid)),
		GainLifeFor(You, FromNumber(paid)),
	)))
	g.PutOnBattlefield(NewCreature("Red creature", "{R}", 1, 1), a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("source was not sacrificed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 {
		t.Fatalf("clone life = %d", clone.GetPlayer(a.PlayerID()).Life())
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatalf("life = %d", a.Life())
	}
}

func TestPaidSourceSacrificeManaValueValidationAndRollback(t *testing.T) {
	paid := BindNumber("paid")
	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"valid", SacrificeMatchingIncludingSourceCostWithManaValue(IsCreature, "Sacrifice a creature", paid), ""},
		{"nil", SacrificeMatchingIncludingSourceCostWithManaValue(IsCreature, "Sacrifice a creature", nil), "non-nil"},
		{"empty", SacrificeMatchingIncludingSourceCostWithManaValue(IsCreature, "Sacrifice a creature", BindNumber("")), "name"},
		{"no text", SacrificeMatchingIncludingSourceCostWithManaValue(IsCreature, "", paid), "text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewCreature("Synthetic", "{2}", 1, 1, WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, FromNumber(paid)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "{2}", 1, 1)
	g.PutOnBattlefield(source, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{SacrificeMatchingIncludingSourceCostWithManaValue(IsCreature, "Sacrifice a creature", paid)},
	})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || g.FindPermanent(source.ID()) == nil {
		t.Fatal("returned source produced a payment result")
	}
}
