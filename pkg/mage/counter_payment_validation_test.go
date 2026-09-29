package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func counterPaymentDeclarations(ct CounterType) []Cost {
	return []Cost{
		RemoveCountersCost(ct, 1),
		RemoveCountersCostInto(ct, 1, BindNumber("fixed")),
		RemoveXCountersFromSourceCost(ct),
		RemoveXCountersFromSourceCostInto(ct, BindNumber("X")),
		RemoveAnyCountersCost(ct, BindNumber("chosen")),
	}
}

func TestCounterPaymentDeclarationBounds(t *testing.T) {
	for ct := CounterType(-1); ct <= NumCounters+1; ct++ {
		for i, cost := range counterPaymentDeclarations(ct) {
			t.Run(fmt.Sprintf("%d/%d", ct, i), func(t *testing.T) {
				card := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(cost), GainLifeFor(You, Fixed(1)))))
				err := ValidateCardDefinition(card.Name(), card)
				invalid := ct < 0 || ct >= NumCounters
				if (err != nil) != invalid {
					t.Fatalf("counter %d: validation error = %v", ct, err)
				}
			})
		}
	}
	for _, cost := range []Cost{RemoveCountersCost(Charge, -1), RemoveCountersCostInto(Charge, -1, BindNumber("paid"))} {
		card := NewArtifact("Negative", "", WithAbility(Activate(Costs(cost), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("negative fixed count passed registration")
		}
	}
}

func TestCounterPaymentInvalidDirectCalls(t *testing.T) {
	for _, ct := range []CounterType{-1, NumCounters, NumCounters + 1} {
		for i, cost := range counterPaymentDeclarations(ct) {
			t.Run(fmt.Sprintf("%d/%d", ct, i), func(t *testing.T) {
				a := NewBasePlayer("A")
				g := NewGame(a, NewBasePlayer("B"))
				source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
				g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
				for _, x := range []int{0, 1} {
					g.resolution.SetX(x)
					if cost.CanPay(source.ID(), a.PlayerID(), g) {
						t.Fatal("invalid counter type was payable")
					}
					if err := cost.Pay(source.ID(), a.PlayerID(), g); err == nil {
						t.Fatal("invalid counter type was paid")
					}
				}
				if g.FindPermanent(source.ID()).Counters[Charge] != 3 {
					t.Fatal("invalid payment changed counters")
				}
				if cost.CanPay(uuid.Nil, a.PlayerID(), g) {
					t.Fatal("invalid type was payable without source")
				}
			})
		}
	}
}

func TestCounterPaymentInvalidTransactionRollback(t *testing.T) {
	for i, cost := range counterPaymentDeclarations(NumCounters) {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			a := &counterChoicePlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			a.ManaPool().Add(Red, 1)
			g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
			_, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), XValue: 1,
				Costs: []Cost{ManaPayment("{R}"), LifePayCost(2), cost}})
			if err == nil {
				t.Fatal("invalid payment prepared")
			}
			if a.Life() != 20 || a.ManaPool().Count(Red) != 1 || g.FindPermanent(source.ID()).Counters[Charge] != 3 || a.calls != 0 {
				t.Fatal("invalid payment spent resources or requested a choice")
			}
		})
	}
}

func TestCounterPaymentNestedDeclarationBounds(t *testing.T) {
	for _, cost := range []Cost{RemoveCountersCost(NumCounters, 1), RemoveXCountersFromSourceCost(NumCounters), RemoveCountersCost(Charge, -1)} {
		RunDefinitionValidationTable(t, []DefinitionValidationCase{
			{Name: "spell additional", Card: NewInstant("Additional", "", Spell(GainLifeFor(You, Fixed(1))), WithAdditionalCost(cost)), WantErr: true},
			{Name: "optional", Card: NewArtifact("Optional", "", WithAbility(Activate(Costs(OptionalCost(cost, "Pay?")), GainLifeFor(You, Fixed(1))))), WantErr: true},
			{Name: "alternative", Card: NewArtifact("Alternative", "", WithAbility(Activate(Costs(EitherCost(cost, LifePayCost(1))), GainLifeFor(You, Fixed(1))))), WantErr: true},
			{Name: "resolution", Card: NewInstant("Resolution", "", Spell(IfPaid(You, Costs(cost), GainLifeFor(You, Fixed(1))))), WantErr: true},
		})
	}
}
