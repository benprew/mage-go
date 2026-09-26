package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPaidSacrificeManaValuePendingActivations(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("paid mana value")
	target := DeclarePlayerTarget("recipient")
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeMatchingCostWithManaValue(IsArtifact, "Sacrifice an artifact", paid)),
		GainLifeFor(target, FromNumber(paid)), WithTargets(target),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Small", "{2}"), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Large", "{5}"), a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 30 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatalf("clone life: A=%d B=%d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life())
	}
	g.ResolveStack()
	if a.Life() != 30 || b.Life() != 22 || a.choices != 2 {
		t.Fatalf("live life: A=%d B=%d, choices=%d", a.Life(), b.Life(), a.choices)
	}
}

func TestPaidSacrificeManaValueValidationAndRollback(t *testing.T) {
	paid := BindNumber("paid")
	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"valid", SacrificeMatchingCostWithManaValue(IsArtifact, "Sacrifice an artifact", paid), ""},
		{"nil", SacrificeMatchingCostWithManaValue(IsArtifact, "Sacrifice an artifact", nil), "non-nil"},
		{"empty", SacrificeMatchingCostWithManaValue(IsArtifact, "Sacrifice an artifact", BindNumber("")), "name"},
		{"no text", SacrificeMatchingCostWithManaValue(IsArtifact, "", paid), "text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, FromNumber(paid)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	artifact := NewArtifact("Payment", "{3}")
	g.PutOnBattlefield(artifact, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: a.PlayerID(), Costs: []Cost{SacrificeMatchingCostWithManaValue(IsArtifact, "Sacrifice an artifact", paid)},
	})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(artifact.ID()))
	g.PutOnBattlefield(artifact, a.PlayerID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || g.FindPermanent(artifact.ID()) == nil {
		t.Fatal("returned artifact produced a payment result")
	}
}

func TestPaidSacrificeManaValueSpell(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("paid mana value")
	card := NewInstant("Offering", "", WithAbility(Spell(GainLifeFor(You, FromNumber(paid)))),
		WithAdditionalCost(SacrificeMatchingCostWithManaValue(IsArtifact, "Sacrifice an artifact", paid)))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err == nil {
		t.Fatal("spell cast without payment")
	}
	if len(a.Hand()) != 1 || !g.stack.IsEmpty() || a.Life() != 20 {
		t.Fatal("failed cast changed state")
	}
	g.PutOnBattlefield(NewArtifact("Payment", "{4}"), a.PlayerID())
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatalf("life = %d, want 24", a.Life())
	}
}
