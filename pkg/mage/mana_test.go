package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAddRestricted_NilContextRejects(t *testing.T) {
	mp := NewManaPool()
	mp.AddRestricted(Colorless, 3, ArtifactSpellsOnly{})

	if mp.CanPay(ManaCost{Generic: 3}, nil) {
		t.Fatal("CanPay with nil context should treat restricted mana as unusable")
	}
}

func TestAddRestricted_NonMatchingSpellRejects(t *testing.T) {
	mp := NewManaPool()
	mp.AddRestricted(Colorless, 3, ArtifactSpellsOnly{})

	creatureCtx := &SpellPaymentContext{IsCreature: true, CardName: "Hill Giant"}
	if mp.CanPay(ManaCost{Generic: 3}, creatureCtx) {
		t.Fatal("ArtifactSpellsOnly mana should not pay for a non-artifact spell")
	}
}

func TestAddRestricted_MatchingSpellPays(t *testing.T) {
	mp := NewManaPool()
	mp.AddRestricted(Colorless, 3, ArtifactSpellsOnly{})

	artifactCtx := &SpellPaymentContext{IsArtifact: true, CardName: "Amulet of Kroog"}
	if !mp.CanPay(ManaCost{Generic: 2}, artifactCtx) {
		t.Fatal("ArtifactSpellsOnly mana should pay for an artifact spell")
	}
	if err := mp.Pay(ManaCost{Generic: 2}, artifactCtx); err != nil {
		t.Fatalf("Pay failed: %v", err)
	}
	if got := mp.TotalMana(); got != 1 {
		t.Errorf("after paying 2 of 3, want 1 remaining, got %d", got)
	}
}

func TestAddRestricted_RestrictedFirstPreference(t *testing.T) {
	// Mixed pool: 3 unrestricted + 2 ArtifactSpellsOnly colorless.
	// When paying for an artifact, restricted mana should be drained first
	// so it isn't wasted at end-of-step.
	mp := NewManaPool()
	mp.Add(Colorless, 3)
	mp.AddRestricted(Colorless, 2, ArtifactSpellsOnly{})

	artifactCtx := &SpellPaymentContext{IsArtifact: true, CardName: "Amulet of Kroog"}
	if err := mp.Pay(ManaCost{Generic: 2}, artifactCtx); err != nil {
		t.Fatalf("Pay failed: %v", err)
	}

	// Expect: all 2 restricted spent, all 3 unrestricted remain.
	restricted, unrestricted := 0, 0
	for _, m := range mp.SnapshotPool() {
		if m.Restriction != nil {
			restricted++
		} else {
			unrestricted++
		}
	}
	if restricted != 0 {
		t.Errorf("expected 0 restricted remaining, got %d", restricted)
	}
	if unrestricted != 3 {
		t.Errorf("expected 3 unrestricted remaining, got %d", unrestricted)
	}
}

func TestAddRestricted_SnapshotRoundTrip(t *testing.T) {
	mp := NewManaPool()
	mp.AddRestricted(Colorless, 2, ArtifactSpellsOnly{})
	mp.Add(Red, 1)

	snap := mp.SnapshotPool()
	mp.Clear()
	mp.RestorePool(snap)

	artifactCtx := &SpellPaymentContext{IsArtifact: true, CardName: "X"}
	if !mp.CanPay(ManaCost{Generic: 2}, artifactCtx) {
		t.Fatal("restricted mana should round-trip through snapshot/restore")
	}

	creatureCtx := &SpellPaymentContext{IsCreature: true, CardName: "Y"}
	if mp.CanPay(ManaCost{Generic: 2}, creatureCtx) {
		t.Fatal("restriction predicate should round-trip through snapshot/restore")
	}
}

func TestAddRestricted_CreatureSpellsOnly(t *testing.T) {
	mp := NewManaPool()
	mp.AddRestricted(Green, 3, CreatureSpellsOnly{})

	creatureCtx := &SpellPaymentContext{IsCreature: true, CardName: "Bear"}
	if !mp.CanPay(ManaCost{Green: 1, Generic: 2}, creatureCtx) {
		t.Fatal("CreatureSpellsOnly mana should pay for a creature spell")
	}

	artifactCtx := &SpellPaymentContext{IsArtifact: true, CardName: "Amulet"}
	if mp.CanPay(ManaCost{Generic: 2}, artifactCtx) {
		t.Fatal("CreatureSpellsOnly mana should not pay for an artifact spell")
	}
}

func TestManaPoolPayment_BacktracksAcrossOverlappingHybridSymbols(t *testing.T) {
	mp := NewManaPool()
	mp.Add(White, 1)
	mp.Add(Blue, 1)
	cost := ManaCost{Hybrid: []HybridSymbol{
		{A: White, B: Blue},
		{A: White, B: Black},
	}}

	if !mp.CanPay(cost, nil) {
		t.Fatal("{W}{U} can pay {W/U}{W/B} by using blue, then white")
	}
	if err := mp.Pay(cost, nil); err != nil {
		t.Fatalf("Pay rejected the allocation accepted by CanPay: %v", err)
	}
	if got := mp.TotalMana(); got != 0 {
		t.Fatalf("payment left %d mana, want 0", got)
	}
}

func TestManaPoolPayment_ConversionCanPayHybridSymbol(t *testing.T) {
	mp := NewManaPool()
	mp.Add(Red, 1)
	mp.ManaConversions = map[Color]Color{Red: White}
	cost := ManaCost{Hybrid: []HybridSymbol{{A: White, B: Blue}}}

	if !mp.CanPay(cost, nil) {
		t.Fatal("red mana spendable as white should pay {W/U}")
	}
	mp.ResetLastDrained()
	if err := mp.Pay(cost, nil); err != nil {
		t.Fatalf("Pay rejected converted hybrid payment: %v", err)
	}
	if got := mp.LastDrainedColors[Red]; got != 1 {
		t.Fatalf("payment drained %d red mana, want 1", got)
	}
}
