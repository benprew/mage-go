package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"testing"
)

func TestAutoTapForCost_UsesManaPool(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.Players[0].PlayerID()

	// Put one Forest on the battlefield (produces {G})
	forest := NewLand("Forest", WithManaAbility(Green))
	forest.SetOwner(pid)
	perm := g.PutOnBattlefield(forest, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	// Add {R} to the mana pool directly
	g.Players[0].ManaPool().Add(Red, 1)

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
	pid := g.Players[0].PlayerID()

	// No lands — mana pool has exactly {R}{G}
	g.Players[0].ManaPool().Add(Red, 1)
	g.Players[0].ManaPool().Add(Green, 1)

	// Pay {R}{G} entirely from pool, no tapping needed
	cost := ManaCost{Red: 1, Green: 1}
	if err := g.AutoTapForCost(pid, cost); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}
}

func TestAutoTapForCost_PoolCoversGenericCost(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.Players[0].PlayerID()

	// Pool has {R}{R}, no lands
	g.Players[0].ManaPool().Add(Red, 2)

	// Pay {2} entirely from pool
	cost := ManaCost{Generic: 2}
	if err := g.AutoTapForCost(pid, cost); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}
}

func TestCanAfford_UsesManaPool(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.Players[0].PlayerID()

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
	g.Players[0].ManaPool().Add(Red, 1)

	// Now can afford {G}{R}
	if !g.CanAfford(pid, ManaCost{Green: 1, Red: 1}) {
		t.Error("should afford {G}{R} with Forest + {R} in pool")
	}
}

func TestCanAfford_PoolCoversEntireCost(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.Players[0].PlayerID()

	// No lands, just pool mana
	g.Players[0].ManaPool().Add(Blue, 2)

	if !g.CanAfford(pid, ManaCost{Blue: 1, Generic: 1}) {
		t.Error("should afford {1}{U} with {U}{U} in pool")
	}
}

func TestManaCostPayment_CanPay_ConsidersUntappedSources(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.Players[0].PlayerID()

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
	pid := g.Players[0].PlayerID()
	g.Step = PrecombatMain

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
	g.CurrentX = 5
	err := g.ActivateAbilityByIndex(pid, perm.ID(), abilityIdx, nil)
	if err != nil {
		t.Fatalf("ActivateAbilityByIndex failed: %v", err)
	}

	obj := g.Stack.Peek()
	if obj == nil {
		t.Fatal("expected stack object")
	}
	if obj.XValue != 5 {
		t.Errorf("stack object XValue = %d, want 5", obj.XValue)
	}
}

func TestCanAfford_RespectsManaConversions(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.Players[0].PlayerID()

	// One Mountain on the battlefield (produces {R})
	mountain := NewLand("Mountain", WithManaAbility(Red))
	mountain.SetOwner(pid)
	perm := g.PutOnBattlefield(mountain, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	// Sunglasses of Urza style: Red → White conversion
	g.Players[0].ManaPool().ManaConversions = map[Color]Color{Red: White}

	// Should be able to afford {W} because Mountain produces {R} which converts to {W}
	if !g.CanAfford(pid, ManaCost{White: 1}) {
		t.Error("should afford {W} with Mountain + Red→White conversion")
	}

	// Should NOT be able to afford {R}{W} — only one source, conversion makes it {W} not both
	if g.CanAfford(pid, ManaCost{Red: 1, White: 1}) {
		t.Error("should not afford {R}{W} with only one Mountain")
	}
}
