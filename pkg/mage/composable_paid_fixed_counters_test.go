package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestFixedCounterPaymentCaptureAndCopies(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	source := NewArtifact("Battery", "", WithAbility(Activate(
		Costs(RemoveCountersCostInto(core.Charge, 2, paid)),
		GainLifeFor(You, FromNumber(paid)),
	)))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(core.Charge, 3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if got := g.FindPermanent(source.ID()).Counters[core.Charge]; got != 1 {
		t.Fatalf("counters = %d", got)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 24 {
		t.Fatalf("clone life = %d", got)
	}
	if got := a.Life(); got != 20 {
		t.Fatalf("original life before resolution = %d", got)
	}
	g.ResolveStack()
	if got := a.Life(); got != 24 {
		t.Fatalf("original life = %d", got)
	}
}

func TestFixedCounterPaymentRejectsChangedSource(t *testing.T) {
	for _, change := range []string{"return", "phase", "short"} {
		t.Run(change, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			paid := BindNumber("paid")
			source := NewArtifact("Battery", "")
			g.PutOnBattlefield(source, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(core.Charge, 2)
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), RemoveCountersCostInto(core.Charge, 2, paid)}})
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(source, a.PlayerID())
				g.MutablePermanent(source.ID()).AddCounter(core.Charge, 2)
			case "phase":
				g.PhaseOut(source.ID())
			case "short":
				g.MutablePermanent(source.ID()).RemoveCounter(core.Charge, 1)
			}
			if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || a.Life() != 20 {
				t.Fatalf("commit = %v, captures = %v", err, tx.paidNumbers)
			}
		})
	}
}

func TestFixedCounterPaymentDeclaration(t *testing.T) {
	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"valid", RemoveCountersCostInto(core.Charge, 1, BindNumber("paid")), ""},
		{"nil", RemoveCountersCostInto(core.Charge, 1, nil), "non-nil"},
		{"negative", RemoveCountersCostInto(core.Charge, -1, BindNumber("paid")), "negative"},
		{"unnamed", RemoveCountersCostInto(core.Charge, 1, BindNumber("")), "name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Test", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	paid := BindNumber("paid")
	card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(RemoveCountersCostInto(core.Charge, 1, paid), RemoveCountersCostInto(core.Charge, 1, paid)), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("duplicate producer accepted")
	}
	missing := NewArtifact("Missing", "", WithAbility(Activate(Costs(), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil {
		t.Fatal("unproduced binding accepted")
	}
	spell := NewInstant("Spell", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)))), WithAdditionalCost(RemoveCountersCostInto(core.Charge, 1, paid)))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil {
		t.Fatal("battlefield counter payment accepted for a spell")
	}
}

func TestFixedCounterPaymentZeroAndInterleavedActivations(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	firstPaid := BindNumber("first")
	secondPaid := BindNumber("second")
	first := NewArtifact("First", "", WithAbility(Activate(Costs(RemoveCountersCostInto(core.Charge, 0, firstPaid)), GainLifeFor(You, FromNumber(firstPaid)))))
	second := NewArtifact("Second", "", WithAbility(Activate(Costs(RemoveCountersCostInto(core.Charge, 2, secondPaid)), GainLifeFor(You, FromNumber(secondPaid)))))
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.MutablePermanent(second.ID()).AddCounter(core.Charge, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if got := a.Life(); got != 22 {
		t.Fatalf("life = %d", got)
	}
}
