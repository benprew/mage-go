package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidSacrificeTotalPowerActivationCopyAndClone(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	total := BindNumber("total power")
	source := NewArtifact("Offering", "", WithAbility(Activate(Costs(SacrificeNMatchingCostWithTotalPower(2, IsCreature, "Sacrifice two creatures", total)), GainLifeFor(You, FromNumber(total)))))
	g.PutOnBattlefield(source, a.PlayerID())
	small := NewCreature("Small", "", 1, 1)
	large := NewCreature("Large", "", 3, 3)
	g.PutOnBattlefield(small, a.PlayerID())
	g.PutOnBattlefield(large, a.PlayerID())
	g.MutablePermanent(large.ID()).AddCounter(core.P1P1, 1)
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.choices != 2 {
		t.Fatalf("choices = %d", a.choices)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 25 || clone.GetPlayer(b.PlayerID()).Life() != 25 {
		t.Fatal("clone lost paid total")
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 25 || b.Life() != 25 {
		t.Fatal("stack copy lost paid total")
	}
}

func TestPaidSacrificeTotalPowerRollback(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	first := NewCreature("First", "", 2, 2)
	second := NewCreature("Second", "", 3, 3)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	total := BindNumber("total")
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{SacrificeNMatchingCostWithTotalPower(2, IsCreature, "Sacrifice two", total)}})
	if err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(second.ID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || g.FindPermanent(first.ID()) == nil {
		t.Fatal("phased choice paid or captured")
	}
	g.PhaseIn([]uuid.UUID{second.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
	g.PutOnBattlefield(second, a.PlayerID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || g.FindPermanent(first.ID()) == nil {
		t.Fatal("returned choice paid or captured")
	}
}

func TestPaidSacrificeTotalPowerDefinition(t *testing.T) {
	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"valid", SacrificeNMatchingCostWithTotalPower(2, IsCreature, "Sacrifice two", BindNumber("total")), ""},
		{"nil", SacrificeNMatchingCostWithTotalPower(2, IsCreature, "Sacrifice two", nil), "non-nil"},
		{"unnamed", SacrificeNMatchingCostWithTotalPower(2, IsCreature, "Sacrifice two", BindNumber("")), "name"},
		{"count", SacrificeNMatchingCostWithTotalPower(0, IsCreature, "Sacrifice", BindNumber("total")), "positive"},
		{"text", SacrificeNMatchingCostWithTotalPower(2, IsCreature, "", BindNumber("total")), "text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

func TestPaidSacrificeTotalPowerPendingActivations(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	total := BindNumber("total")
	target := DeclarePlayerTarget("recipient")
	source := NewArtifact("Offering", "", WithAbility(Activate(Costs(SacrificeNMatchingCostWithTotalPower(2, IsCreature, "Sacrifice two", total)), GainLifeFor(target, FromNumber(total)), WithTargets(target))))
	g.PutOnBattlefield(source, a.PlayerID())
	for i := 1; i <= 4; i++ {
		g.PutOnBattlefield(NewCreature("Creature", "", i, 1), a.PlayerID())
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 27 || b.Life() != 23 {
		t.Fatalf("life totals = %d, %d", a.Life(), b.Life())
	}
}

func TestPaidSacrificeTotalPowerSpellAndScope(t *testing.T) {
	total := BindNumber("total")
	spell := NewInstant("Offering", "", WithAbility(Spell(GainLifeFor(You, FromNumber(total)))), WithAdditionalCost(SacrificeNMatchingCostWithTotalPower(2, IsCreature, "Sacrifice two", total)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		SacrificeNMatchingCostWithTotalPower(1, IsCreature, "Sacrifice one", total),
		SacrificeNMatchingCostWithTotalPower(1, IsCreature, "Sacrifice another", total),
	), GainLifeFor(You, FromNumber(total)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate validation = %v", err)
	}
	branch := NewArtifact("Branch", "", WithAbility(Activate(Costs(), IfPaid(You,
		Costs(SacrificeNMatchingCostWithTotalPower(1, IsCreature, "Sacrifice one", total)),
		GainLifeFor(You, FromNumber(total))))))
	if err := ValidateCardDefinition(branch.Name(), branch); err == nil || !strings.Contains(err.Error(), "activated ability cost") {
		t.Fatalf("branch validation = %v", err)
	}
}
