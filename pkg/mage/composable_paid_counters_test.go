package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type counterChoicePlayer struct {
	*BasePlayer
	choice, calls int
}

func (p *counterChoicePlayer) ChooseNumber(minimum, maximum int, _ string) int {
	p.calls++
	return p.choice
}

func TestPaidCounterChoiceActivation(t *testing.T) {
	a := &counterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: 2}
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid counters")
	source := NewArtifact("Battery", "", WithAbility(Activate(
		Costs(RemoveAnyCountersCost(core.Charge, paid)),
		GainLifeFor(You, FromNumber(paid)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(core.Charge, 3)
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.calls != 1 || g.FindPermanent(source.ID()).Counters[core.Charge] != 1 {
		t.Fatal("payment did not use one locked choice")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 || a.Life() != 20 {
		t.Fatal("copy or clone lost payment value")
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatalf("life = %d", a.Life())
	}
}

func TestPaidCounterChoiceRollbackAndValidation(t *testing.T) {
	a := &counterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: 2}
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	source := NewArtifact("Battery", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(core.Charge, 3)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{RemoveAnyCountersCost(core.Charge, paid)}})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(core.Charge, 3)
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || g.FindPermanent(source.ID()).Counters[core.Charge] != 3 {
		t.Fatal("returned source paid old cost")
	}
	phasedSource := NewArtifact("Phasing Battery", "")
	g.PutOnBattlefield(phasedSource, a.PlayerID())
	g.MutablePermanent(phasedSource.ID()).AddCounter(core.Charge, 3)
	phasedTx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: phasedSource.ID(), Controller: a.PlayerID(), Costs: []Cost{RemoveAnyCountersCost(core.Charge, paid)}})
	if err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(phasedSource.ID())
	if err := phasedTx.Commit(); err == nil || len(phasedTx.paidNumbers) != 0 {
		t.Fatal("phased-out source paid cost")
	}

	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"nil", RemoveAnyCountersCost(core.Charge, nil), "non-nil"},
		{"unnamed", RemoveAnyCountersCost(core.Charge, BindNumber("")), "name"},
		{"valid", RemoveAnyCountersCost(core.Charge, paid), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Test", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, FromNumber(paid)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

func TestPaidCounterChoiceZeroAndInvalid(t *testing.T) {
	for _, tc := range []struct {
		name         string
		choice       int
		wantLife     int
		wantCounters int
		wantError    bool
	}{
		{"zero", 0, 20, 3, false},
		{"negative", -1, 20, 3, true},
		{"too many", 4, 20, 3, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &counterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: tc.choice}
			g := NewGame(a, NewBasePlayer("B"))
			paid := BindNumber("paid")
			source := NewArtifact("Battery", "", WithAbility(Activate(Costs(RemoveAnyCountersCost(core.Charge, paid)), GainLifeFor(You, FromNumber(paid)))))
			g.PutOnBattlefield(source, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(core.Charge, 3)
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if (err != nil) != tc.wantError {
				t.Fatalf("activation error = %v", err)
			}
			g.ResolveStack()
			if a.Life() != tc.wantLife || int(g.FindPermanent(source.ID()).Counters[core.Charge]) != tc.wantCounters {
				t.Fatal("invalid or zero payment changed state")
			}
		})
	}
}
