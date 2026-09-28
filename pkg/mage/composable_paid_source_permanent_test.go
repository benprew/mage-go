package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidSourcePermanentIdentityAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindPermanent("paid source")
	source := NewCreature("Offering", "", 2, 2, WithKeyword(core.Lifelink), WithAbility(Activate(
		Costs(SacrificeMatchingIncludingSourceCostWithPermanent(IsCreature, "Sacrifice a creature", paid)),
		DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(2)),
	)))
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
	g.PutOnBattlefield(source, b.PlayerID())
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 18 || a.Life() != 20 {
		t.Fatalf("copy or clone lost old source: clone A=%d B=%d live A=%d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life(), a.Life())
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 18 {
		t.Fatalf("old source result A=%d B=%d", a.Life(), b.Life())
	}
}

func TestPaidSourcePermanentRollbackAndDeclaration(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindPermanent("paid source")
	source := NewCreature("Offering", "", 2, 2)
	g.PutOnBattlefield(source, a.PlayerID())
	cost := SacrificeMatchingIncludingSourceCostWithPermanent(IsCreature, "Sacrifice a creature", paid)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(source.ID())
	if err := tx.Commit(); err == nil || len(tx.paidPermanents) != 0 {
		t.Fatal("phased source paid a stale cost")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	second, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{
		SacrificeMatchingIncludingSourceCostWithPermanent(IsCreature, "Sacrifice a creature", paid),
	}})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := second.Commit(); err == nil || len(second.paidPermanents) != 0 {
		t.Fatal("returned source paid a stale cost")
	}
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{cost, ""},
		{SacrificeMatchingIncludingSourceCostWithPermanent(IsCreature, "", paid), "text"},
		{SacrificeMatchingIncludingSourceCostWithPermanent(IsCreature, "Sacrifice a creature", nil), "non-nil"},
	} {
		card := NewCreature("Definition", "", 1, 1, WithAbility(Activate(Costs(tc.cost), DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	missing := NewCreature("Missing", "", 1, 1, WithAbility(Activate(Costs(), DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(1)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer validation = %v", err)
	}
	resolutionCost := NewCreature("Resolution", "", 1, 1, WithAbility(Activate(Costs(),
		IfPaid(You, Costs(SacrificeMatchingIncludingSourceCostWithPermanent(IsCreature, "Sacrifice a creature", paid)),
			DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(1))))))
	if err := ValidateCardDefinition(resolutionCost.Name(), resolutionCost); err == nil || !strings.Contains(err.Error(), "activated ability cost") {
		t.Fatalf("resolution cost validation = %v", err)
	}
}
