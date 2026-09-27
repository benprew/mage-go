package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExiledGraveyardCostManaValueActivation(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("exiled mana value")
	source := NewArtifact("Synthetic", "", WithAbility(Activate(
		Costs(ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile a creature card", paid)),
		GainLifeFor(You, FromNumber(paid)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	for _, cost := range []string{"{2}", "{5}"} {
		card := NewCreature("Payment", cost, 1, 1)
		card.SetOwner(a.PlayerID())
		a.AddToGraveyard(card)
	}
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 29 || a.Life() != 20 {
		t.Fatalf("clone life = %d, original = %d", clone.GetPlayer(a.PlayerID()).Life(), a.Life())
	}
	g.ResolveStack()
	if a.Life() != 29 {
		t.Fatalf("life = %d", a.Life())
	}
}

func TestExiledGraveyardCostManaValueDeclaration(t *testing.T) {
	paid := BindNumber("paid")
	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"valid", ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile a creature card", paid), ""},
		{"nil", ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile a creature card", nil), "non-nil"},
		{"unnamed", ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile a creature card", BindNumber("")), "name"},
		{"text", ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "", paid), "text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, FromNumber(paid)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile one", paid),
		ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile two", paid)), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate validation = %v", err)
	}
	missing := NewArtifact("Missing", "", WithAbility(Activate(Costs(), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer validation = %v", err)
	}
	branch := NewArtifact("Branch", "", WithAbility(Activate(Costs(), IfPaid(You,
		Costs(ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile", paid)), GainLifeFor(You, FromNumber(paid))))))
	if err := ValidateCardDefinition(branch.Name(), branch); err == nil || !strings.Contains(err.Error(), "activated ability") {
		t.Fatalf("branch validation = %v", err)
	}
}

func TestExiledGraveyardCostManaValueStaleChoiceAndFailedPayment(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	card := NewCreature("Payment", "{4}", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
		ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile", paid),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g.MoveFromGraveyard(a.PlayerID(), card.ID(), core.ZoneHand); !ok {
		t.Fatal("card did not leave graveyard")
	}
	a.AddToGraveyard(card)
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || len(a.Graveyard()) != 1 {
		t.Fatalf("stale payment = %v, captures = %v", err, tx.paidNumbers)
	}
	source := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(
		ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile", paid), LifePayCost(25)),
		GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("unpayable activation succeeded")
	}
	if len(a.Graveyard()) != 1 || a.Life() != 20 || len(g.StackObjects()) != 0 {
		t.Fatal("failed payment changed live state")
	}
}
