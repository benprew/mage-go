package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidXCountersActivationFrames(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("removed")
	source := NewArtifact("Counter source", "", WithAbility(Activate(
		Costs(RemoveXCountersFromSourceCostInto(core.Charge, paid)),
		GainLifeFor(You, FromNumber(paid)),
	)))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(core.Charge, 5)
	for _, x := range []int{0, 2, 3} {
		g.SetXValue(x)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	if got := g.FindPermanent(source.ID()).Counters[core.Charge]; got != 0 {
		t.Fatalf("remaining counters = %d", got)
	}
	g.CopyStackObjectDirect(g.StackObjects()[1], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 27 {
		t.Fatalf("clone life = %d", got)
	}
	if got := a.Life(); got != 20 {
		t.Fatalf("original life = %d", got)
	}
	g.ResolveStack()
	if got := a.Life(); got != 27 {
		t.Fatalf("life = %d", got)
	}
}

func TestPaidXCountersRollbackAndDeclaration(t *testing.T) {
	paid := BindNumber("removed")
	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"valid", RemoveXCountersFromSourceCostInto(core.Charge, paid), ""},
		{"nil", RemoveXCountersFromSourceCostInto(core.Charge, nil), "non-nil"},
		{"unnamed", RemoveXCountersFromSourceCostInto(core.Charge, BindNumber("")), "name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Test", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(RemoveXCountersFromSourceCostInto(core.Charge, paid), RemoveXCountersFromSourceCostInto(core.Charge, paid)), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil {
		t.Fatal("duplicate binding accepted")
	}
	spell := NewInstant("Spell", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)))), WithAdditionalCost(RemoveXCountersFromSourceCostInto(core.Charge, paid)))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil {
		t.Fatal("battlefield cost accepted on spell")
	}
	missing := NewArtifact("Missing", "", WithAbility(Activate(Costs(), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil {
		t.Fatal("unproduced binding accepted")
	}
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Source", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(core.Charge, 2)
	resolved := g.prepareActionCosts([]Cost{RemoveXCountersFromSourceCostInto(core.Charge, paid)}, nil, 2)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: append([]Cost{LifePayCost(2)}, resolved...)})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(core.Charge, 2)
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || a.Life() != 20 {
		t.Fatalf("commit = %v, capture = %v", err, tx.paidNumbers)
	}
}

func TestPaidXCountersLockedAmountAndPhasing(t *testing.T) {
	for _, phase := range []bool{false, true} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		paid := BindNumber("paid")
		source := NewArtifact("Source", "")
		g.PutOnBattlefield(source, a.PlayerID())
		g.MutablePermanent(source.ID()).AddCounter(core.Charge, 2)
		costs := g.prepareActionCosts([]Cost{RemoveXCountersFromSourceCostInto(core.Charge, paid)}, nil, 2)
		tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: costs})
		if err != nil {
			t.Fatal(err)
		}
		g.SetXValue(0)
		if phase {
			g.PhaseOut(source.ID())
		}
		err = tx.Commit()
		if phase {
			if err == nil || len(tx.paidNumbers) != 0 {
				t.Fatalf("phased payment = %v, capture = %v", err, tx.paidNumbers)
			}
		} else if err != nil || tx.paidNumbers[paid] != 2 || g.FindPermanent(source.ID()).Counters[core.Charge] != 0 {
			t.Fatalf("locked payment = %v, capture = %v", err, tx.paidNumbers)
		}
	}
}

func TestPaidXCountersRejectsUnpayableX(t *testing.T) {
	for _, x := range []int{-1, 3} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		paid := BindNumber("paid")
		source := NewArtifact("Source", "", WithAbility(Activate(
			Costs(RemoveXCountersFromSourceCostInto(core.Charge, paid)),
			GainLifeFor(You, FromNumber(paid)),
		)))
		g.PutOnBattlefield(source, a.PlayerID())
		g.MutablePermanent(source.ID()).AddCounter(core.Charge, 2)
		g.SetXValue(x)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
			t.Fatalf("X=%d was payable", x)
		}
		if len(g.StackObjects()) != 0 || a.Life() != 20 || g.FindPermanent(source.ID()).Counters[core.Charge] != 2 {
			t.Fatalf("X=%d changed game state", x)
		}
	}
}
