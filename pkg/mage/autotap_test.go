package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"testing"
)

func TestAutoTapForCost_UsesManaPool(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// Put one Forest on the battlefield (produces {G})
	forest := NewLand("Forest", WithManaAbility(Green))
	forest.SetOwner(pid)
	perm := g.PutOnBattlefield(forest, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	// Add {R} to the mana pool directly
	g.players[0].ManaPool().Add(Red, 1)

	// Try to pay {1}{G} — should tap forest for {G} and use pool {R} for generic
	cost := ManaCost{Green: 1, Generic: 1}
	if err := g.AutoTapForCost(pid, cost); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}

	if !perm.Tapped {
		t.Error("expected forest to be tapped")
	}
}

func TestAutoTapForCost_PoolCoversColoredCost(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// No lands — mana pool has exactly {R}{G}
	g.players[0].ManaPool().Add(Red, 1)
	g.players[0].ManaPool().Add(Green, 1)

	// Pay {R}{G} entirely from pool, no tapping needed
	cost := ManaCost{Red: 1, Green: 1}
	if err := g.AutoTapForCost(pid, cost); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}
}

func TestAutoTapForCost_PoolCoversGenericCost(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// Pool has {R}{R}, no lands
	g.players[0].ManaPool().Add(Red, 2)

	// Pay {2} entirely from pool
	cost := ManaCost{Generic: 2}
	if err := g.AutoTapForCost(pid, cost); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}
}

func TestCanAfford_UsesManaPool(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// One Forest on the battlefield
	forest := NewLand("Forest", WithManaAbility(Green))
	forest.SetOwner(pid)
	perm := g.PutOnBattlefield(forest, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	// Without pool mana, can only afford {G} or {1}
	if g.CanAfford(pid, ManaCost{Green: 1, Red: 1}) {
		t.Error("should not afford {G}{R} with only a Forest")
	}

	// Add {R} to pool
	g.players[0].ManaPool().Add(Red, 1)

	// Now can afford {G}{R}
	if !g.CanAfford(pid, ManaCost{Green: 1, Red: 1}) {
		t.Error("should afford {G}{R} with Forest + {R} in pool")
	}
}

func TestCanAfford_PoolCoversEntireCost(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// No lands, just pool mana
	g.players[0].ManaPool().Add(Blue, 2)

	if !g.CanAfford(pid, ManaCost{Blue: 1, Generic: 1}) {
		t.Error("should afford {1}{U} with {U}{U} in pool")
	}
}

func TestManaCostPayment_CanPay_ConsidersUntappedSources(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// One Forest on the battlefield (produces {G})
	forest := NewLand("Forest", WithManaAbility(Green))
	forest.SetOwner(pid)
	perm := g.PutOnBattlefield(forest, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	cost := ManaCostOf("{G}")
	if !cost.CanPay(perm.ID(), pid, g) {
		t.Error("ManaCostPayment.CanPay should consider untapped sources, not just pool")
	}
}

func TestActivateAbilityByIndex_SetsXValue(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()
	g.step = PrecombatMain

	// Create a creature with an activated ability: {0}: gain 1 life
	card := NewCreature("Test Creature", "{G}", 1, 1,
		WithAbility(NewActivatedAbility(GainLife(1), ManaCostOf("{0}"))),
	)
	card.SetOwner(pid)
	perm := g.PutOnBattlefield(card, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	abilityIdx := -1
	for i, a := range perm.RuntimeAbilities {
		if _, ok := UnwrapAbility(a).(ActivatedAbility); ok {
			abilityIdx = i
			break
		}
	}
	if abilityIdx < 0 {
		t.Fatal("no activated ability found")
	}

	// Set CurrentX and activate — the stack object should capture XValue
	g.currentX = 5
	err := g.ActivateAbilityByIndex(pid, perm.ID(), abilityIdx, nil)
	if err != nil {
		t.Fatalf("ActivateAbilityByIndex failed: %v", err)
	}

	obj := g.stack.Peek()
	if obj == nil {
		t.Fatal("expected stack object")
	}
	if obj.XValue != 5 {
		t.Errorf("stack object XValue = %d, want 5", obj.XValue)
	}
}

func TestCanAfford_RespectsManaConversions(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// One Mountain on the battlefield (produces {R})
	mountain := NewLand("Mountain", WithManaAbility(Red))
	mountain.SetOwner(pid)
	perm := g.PutOnBattlefield(mountain, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	// Sunglasses of Urza style: Red → White conversion
	g.players[0].ManaPool().ManaConversions = map[Color]Color{Red: White}

	// Should be able to afford {W} because Mountain produces {R} which converts to {W}
	if !g.CanAfford(pid, ManaCost{White: 1}) {
		t.Error("should afford {W} with Mountain + Red→White conversion")
	}

	// Should NOT be able to afford {R}{W} — only one source, conversion makes it {W} not both
	if g.CanAfford(pid, ManaCost{Red: 1, White: 1}) {
		t.Error("should not afford {R}{W} with only one Mountain")
	}
}

func TestAutoTapForCost_ManaBonusReducesTapping(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// Two Mountains on the battlefield
	for i := 0; i < 2; i++ {
		m := NewLand("Mountain", WithManaAbility(Red))
		m.SetOwner(pid)
		p := g.PutOnBattlefield(m, pid)
		p.RevokeBaseAttr(AttrSummonSick)
	}

	// Add Mana Flare (doubles mana from lands)
	flare := NewEnchantment("Mana Flare", "{2}{R}",
		WithAbility(NewManaFlareAbility(IsLand)),
	)
	flare.SetOwner(pid)
	g.PutOnBattlefield(flare, pid)

	// Pay {2} — with Mana Flare, one Mountain produces {R}{R}, so only 1 should be tapped
	cost := ManaCost{Generic: 2}
	if err := g.AutoTapForCost(pid, cost); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}

	tappedCount := 0
	for _, perm := range g.battlefield {
		if perm.Name() == "Mountain" && perm.Tapped {
			tappedCount++
		}
	}
	if tappedCount != 1 {
		t.Errorf("expected 1 Mountain tapped with Mana Flare, got %d", tappedCount)
	}
}

func TestCanAfford_AccountsForManaBonus(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// One Mountain
	m := NewLand("Mountain", WithManaAbility(Red))
	m.SetOwner(pid)
	p := g.PutOnBattlefield(m, pid)
	p.RevokeBaseAttr(AttrSummonSick)

	// Add Mana Flare
	flare := NewEnchantment("Mana Flare", "{2}{R}",
		WithAbility(NewManaFlareAbility(IsLand)),
	)
	flare.SetOwner(pid)
	g.PutOnBattlefield(flare, pid)

	// Should afford {2} with one Mountain + Mana Flare (produces {R}{R})
	if !g.CanAfford(pid, ManaCost{Generic: 2}) {
		t.Error("should afford {2} with one Mountain + Mana Flare")
	}
}

func TestTapForMana_MultiMana(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// Sol Ring style: tap for 2 colorless
	ring := NewArtifact("Sol Ring", "{1}", WithMultiManaAbility(ManaProduction{Color: Colorless, Amount: 2}))
	ring.SetOwner(pid)
	perm := g.PutOnBattlefield(ring, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	if err := g.TapForMana(pid, perm.ID()); err != nil {
		t.Fatalf("TapForMana failed: %v", err)
	}

	pool := g.players[0].ManaPool()
	if pool.Count(Colorless) != 2 {
		t.Errorf("expected 2 colorless mana, got %d", pool.Count(Colorless))
	}
}

func TestTapForMana_MultiColor(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// Produces {G}{W} at once
	land := NewLand("Dual Land",
		WithMultiManaAbility(ManaProduction{Color: Green, Amount: 1}, ManaProduction{Color: White, Amount: 1}),
	)
	land.SetOwner(pid)
	perm := g.PutOnBattlefield(land, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	if err := g.TapForMana(pid, perm.ID()); err != nil {
		t.Fatalf("TapForMana failed: %v", err)
	}

	pool := g.players[0].ManaPool()
	if pool.Count(Green) != 1 {
		t.Errorf("expected 1 green mana, got %d", pool.Count(Green))
	}
	if pool.Count(White) != 1 {
		t.Errorf("expected 1 white mana, got %d", pool.Count(White))
	}
}

func TestAutoTapForCost_MultiMana(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// Sol Ring style: produces 2 colorless
	ring := NewArtifact("Sol Ring", "{1}", WithMultiManaAbility(ManaProduction{Color: Colorless, Amount: 2}))
	ring.SetOwner(pid)
	perm := g.PutOnBattlefield(ring, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	// Pay {2} — Sol Ring alone should cover it
	cost := ManaCost{Generic: 2}
	if err := g.AutoTapForCost(pid, cost); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}

	if !perm.Tapped {
		t.Error("expected Sol Ring to be tapped")
	}
}

func TestCanAfford_MultiMana(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// Sol Ring style: produces 2 colorless
	ring := NewArtifact("Sol Ring", "{1}", WithMultiManaAbility(ManaProduction{Color: Colorless, Amount: 2}))
	ring.SetOwner(pid)
	perm := g.PutOnBattlefield(ring, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	if !g.CanAfford(pid, ManaCost{Generic: 2}) {
		t.Error("should afford {2} with Sol Ring")
	}

	// Should not afford {3} with just Sol Ring
	if g.CanAfford(pid, ManaCost{Generic: 3}) {
		t.Error("should not afford {3} with only Sol Ring (produces 2)")
	}

	_ = perm
}

func TestProducedAmount(t *testing.T) {
	tests := []struct {
		name string
		ma   *ManaAbility
		want int
	}{
		{"single color", NewManaAbility(Green), 1},
		{"multi amount", NewMultiManaAbility(ManaProduction{Color: Colorless, Amount: 2}), 2},
		{"multi amount 3", NewMultiManaAbility(ManaProduction{Color: Colorless, Amount: 3}), 3},
		{"any color", NewManaAbility(AnyColor), 1},
		{"multi color", NewMultiManaAbility(
			ManaProduction{Color: Green, Amount: 1}, ManaProduction{Color: White, Amount: 1},
		), 2},
		{"multi color varied", NewMultiManaAbility(
			ManaProduction{Color: Colorless, Amount: 2}, ManaProduction{Color: Red, Amount: 1},
		), 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ma.ProducedAmount(); got != tt.want {
				t.Errorf("ProducedAmount() = %d, want %d", got, tt.want)
			}
		})
	}
}
