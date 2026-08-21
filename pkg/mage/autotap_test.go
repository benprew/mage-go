package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
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
	if g.CanAfford(pid, ManaCost{Green: 1, Red: 1}, nil) {
		t.Error("should not afford {G}{R} with only a Forest")
	}

	// Add {R} to pool
	g.players[0].ManaPool().Add(Red, 1)

	// Now can afford {G}{R}
	if !g.CanAfford(pid, ManaCost{Green: 1, Red: 1}, nil) {
		t.Error("should afford {G}{R} with Forest + {R} in pool")
	}
}

func TestCanAfford_PoolCoversEntireCost(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// No lands, just pool mana
	g.players[0].ManaPool().Add(Blue, 2)

	if !g.CanAfford(pid, ManaCost{Blue: 1, Generic: 1}, nil) {
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
	if !g.CanAfford(pid, ManaCost{White: 1}, nil) {
		t.Error("should afford {W} with Mountain + Red→White conversion")
	}

	// Should NOT be able to afford {R}{W} — only one source, conversion makes it {W} not both
	if g.CanAfford(pid, ManaCost{Red: 1, White: 1}, nil) {
		t.Error("should not afford {R}{W} with only one Mountain")
	}
}

func TestAutoTapForCost_ManaBonusReducesTapping(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// Two Mountains on the battlefield
	for range 2 {
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

func TestAutoTapForCost_PrefersLandsWithoutTapDrawback(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	painful := NewLand("Painful Island",
		WithManaAbility(Blue),
		WithAbility(NewTriggered(EvtTapped, false,
			DealDamageToPlayers(Fixed(1), SelectController()),
		).SetConditionData(EventSourceIsSelf{})),
	)
	painful.SetOwner(pid)
	painfulPerm := g.PutOnBattlefield(painful, pid)

	islands := make([]*Permanent, 0, 2)
	for range 2 {
		island := NewLand("Island", WithSubTypes("Island"), WithManaAbility(Blue))
		island.SetOwner(pid)
		islands = append(islands, g.PutOnBattlefield(island, pid))
	}

	if err := g.AutoTapForCost(pid, ManaCost{Blue: 2}); err != nil {
		t.Fatalf("AutoTapForCost({U}{U}) failed: %v", err)
	}
	if painfulPerm.Tapped {
		t.Fatal("solver tapped the land with a damage drawback")
	}
	for _, island := range islands {
		if !island.Tapped {
			t.Fatal("solver did not tap both drawback-free Islands")
		}
	}
}

func TestMaxXValue_BasicLands(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// 3 Mountains on battlefield
	for range 3 {
		m := NewLand("Mountain", WithManaAbility(Red))
		m.SetOwner(pid)
		p := g.PutOnBattlefield(m, pid)
		p.RevokeBaseAttr(AttrSummonSick)
	}

	// Fireball {X}{R}: fixed cost is {R}, so maxX = 3 - 1 = 2
	mc := ParseManaCost("{X}{R}")
	if got := g.MaxXValue(pid, mc, nil); got != 2 {
		t.Errorf("MaxXValue for {X}{R} with 3 Mountains = %d, want 2", got)
	}
}

func TestMaxXValue_WithPoolMana(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// 1 Mountain + 2 colorless in pool
	m := NewLand("Mountain", WithManaAbility(Red))
	m.SetOwner(pid)
	p := g.PutOnBattlefield(m, pid)
	p.RevokeBaseAttr(AttrSummonSick)
	g.players[0].ManaPool().Add(Colorless, 2)

	// {X}{R}: pool has 2 colorless, Mountain gives R for the colored cost, so maxX = 2
	mc := ParseManaCost("{X}{R}")
	if got := g.MaxXValue(pid, mc, nil); got != 2 {
		t.Errorf("MaxXValue = %d, want 2", got)
	}
}

func TestMaxXValue_DoubleX(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// 5 Mountains
	for range 5 {
		m := NewLand("Mountain", WithManaAbility(Red))
		m.SetOwner(pid)
		p := g.PutOnBattlefield(m, pid)
		p.RevokeBaseAttr(AttrSummonSick)
	}

	// {X}{X}{R}: fixed cost {R}, 4 mana left, divided by 2 X's = maxX 2
	mc := ParseManaCost("{X}{X}{R}")
	if got := g.MaxXValue(pid, mc, nil); got != 2 {
		t.Errorf("MaxXValue for {X}{X}{R} with 5 Mountains = %d, want 2", got)
	}
}

func TestMaxXValue_NoXInCost(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	mc := ParseManaCost("{2}{R}")
	if got := g.MaxXValue(pid, mc, nil); got != 0 {
		t.Errorf("MaxXValue for non-X spell = %d, want 0", got)
	}
}

func TestMaxXValue_CantAffordBase(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// No mana sources at all
	mc := ParseManaCost("{X}{R}")
	if got := g.MaxXValue(pid, mc, nil); got != 0 {
		t.Errorf("MaxXValue with no mana = %d, want 0", got)
	}
}

func TestMaxXValue_WithManaBonus(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// 2 Mountains + Mana Flare (doubles land mana)
	for range 2 {
		m := NewLand("Mountain", WithManaAbility(Red))
		m.SetOwner(pid)
		p := g.PutOnBattlefield(m, pid)
		p.RevokeBaseAttr(AttrSummonSick)
	}
	flare := NewEnchantment("Mana Flare", "{2}{R}",
		WithAbility(NewManaFlareAbility(IsLand)),
	)
	flare.SetOwner(pid)
	g.PutOnBattlefield(flare, pid)

	// 2 Mountains each producing 2 = 4 total. {X}{R}: maxX = 4 - 1 = 3
	mc := ParseManaCost("{X}{R}")
	if got := g.MaxXValue(pid, mc, nil); got != 3 {
		t.Errorf("MaxXValue with Mana Flare = %d, want 3", got)
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
	if !g.CanAfford(pid, ManaCost{Generic: 2}, nil) {
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

// Mana Vault style: tap-for-mana built via WithActivatedAbility(AddMana(...), Tap())
// rather than WithMultiManaAbility. Auto-tap must discover these too.
func TestAutoTapForCost_ActivatedManaAbility(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	vault := NewArtifact("Mana Vault", "{1}",
		WithActivatedAbility(AddMana(Colorless, 3), Tap()),
	)
	vault.SetOwner(pid)
	perm := g.PutOnBattlefield(vault, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	if err := g.AutoTapForCost(pid, ManaCost{Generic: 2}); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}
	if !perm.Tapped {
		t.Error("expected Mana Vault to be tapped")
	}
	pool := g.players[0].ManaPool()
	if pool.Count(Colorless) != 3 {
		t.Errorf("expected 3 colorless mana, got %d", pool.Count(Colorless))
	}
}

func TestTapForMana_ActivatedManaAbility(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	vault := NewArtifact("Mana Vault", "{1}",
		WithActivatedAbility(AddMana(Colorless, 3), Tap()),
	)
	vault.SetOwner(pid)
	perm := g.PutOnBattlefield(vault, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	if err := g.TapForMana(pid, perm.ID()); err != nil {
		t.Fatalf("TapForMana failed: %v", err)
	}
	if !perm.Tapped {
		t.Error("expected Mana Vault to be tapped")
	}
	pool := g.players[0].ManaPool()
	if pool.Count(Colorless) != 3 {
		t.Errorf("expected 3 colorless mana, got %d", pool.Count(Colorless))
	}
}

func TestCanAfford_ActivatedManaAbility(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	vault := NewArtifact("Mana Vault", "{1}",
		WithActivatedAbility(AddMana(Colorless, 3), Tap()),
	)
	vault.SetOwner(pid)
	perm := g.PutOnBattlefield(vault, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	if !g.CanAfford(pid, ManaCost{Generic: 3}, nil) {
		t.Error("should afford {3} with untapped Mana Vault")
	}
	if g.CanAfford(pid, ManaCost{Generic: 4}, nil) {
		t.Error("should not afford {4} with only Mana Vault")
	}
}

func TestCanAfford_CostedActivatedManaAbility(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	for i, color := range []Color{Red, Black, Black} {
		addLand(t, g, pid, fmt.Sprintf("Land %d", i), color)
	}
	prism := NewArtifact("Celestial Prism", "{3}",
		WithActivatedAbility(
			AddAnyMana(1, Colorless),
			GenericCost(2),
			WithCost(Tap()),
		),
	)
	prism.SetOwner(pid)
	prismPerm := g.PutOnBattlefield(prism, pid)
	prismPerm.RevokeBaseAttr(AttrSummonSick)

	falconCost := ManaCost{Generic: 1, Blue: 1}
	if !g.CanAfford(pid, falconCost, nil) {
		t.Fatal("should afford {1}{U} by paying {2} to activate Celestial Prism")
	}
	if err := g.AutoTapForCost(pid, falconCost); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}
	if !prismPerm.Tapped {
		t.Error("expected Celestial Prism to be tapped")
	}
	if err := g.players[0].ManaPool().Pay(falconCost, nil); err != nil {
		t.Fatalf("paying {1}{U} after auto-tap failed: %v", err)
	}
}

func TestCastSpell_CostedActivatedManaAbility(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()
	g.step = PrecombatMain

	for i, color := range []Color{Red, Black, Black} {
		addLand(t, g, pid, fmt.Sprintf("Land %d", i), color)
	}
	prism := NewArtifact("Celestial Prism", "{3}",
		WithActivatedAbility(
			AddAnyMana(1, Colorless),
			GenericCost(2),
			WithCost(Tap()),
		),
	)
	prism.SetOwner(pid)
	prismPerm := g.PutOnBattlefield(prism, pid)
	prismPerm.RevokeBaseAttr(AttrSummonSick)

	falcon := NewCreature("Zephyr Falcon", "{1}{U}", 1, 1)
	falcon.SetOwner(pid)
	g.players[0].AddToHand(falcon)

	castable := g.GetCastableSpells(pid)
	if len(castable) != 1 || castable[0].ID() != falcon.ID() {
		t.Fatal("expected Zephyr Falcon to be included in castable spells")
	}
	if err := g.CastSpellByID(pid, falcon.ID(), nil, 0); err != nil {
		t.Fatalf("casting Zephyr Falcon failed: %v", err)
	}
	if !prismPerm.Tapped {
		t.Error("expected Celestial Prism to be tapped while casting Zephyr Falcon")
	}
}

func TestCanAfford_CostedActivatedManaAbilityRequiresItsActivationCost(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	addLand(t, g, pid, "Mountain", Red)
	addLand(t, g, pid, "Swamp", Black)
	prism := NewArtifact("Celestial Prism", "{3}",
		WithActivatedAbility(
			AddAnyMana(1, Colorless),
			GenericCost(2),
			WithCost(Tap()),
		),
	)
	prism.SetOwner(pid)
	prismPerm := g.PutOnBattlefield(prism, pid)
	prismPerm.RevokeBaseAttr(AttrSummonSick)

	if g.CanAfford(pid, ManaCost{Generic: 1, Blue: 1}, nil) {
		t.Fatal("should not afford {1}{U} when activating Celestial Prism consumes both lands")
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

	if !g.CanAfford(pid, ManaCost{Generic: 2}, nil) {
		t.Error("should afford {2} with Sol Ring")
	}

	// Should not afford {3} with just Sol Ring
	if g.CanAfford(pid, ManaCost{Generic: 3}, nil) {
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

// addLand puts a basic-style land on the battlefield, returns the permanent.
func addLand(t *testing.T, g *Game, pid uuid.UUID, name string, color Color) *Permanent {
	t.Helper()
	land := NewLand(name, WithManaAbility(color))
	land.SetOwner(pid)
	perm := g.PutOnBattlefield(land, pid)
	perm.RevokeBaseAttr(AttrSummonSick)
	return perm
}

// Atog example: Mountain + Mountain + Forest in play, Atog and Lightning Bolt
// in hand. Casting Atog ({1}{R}) should prefer to tap a Mountain for the {R}
// and the Forest for the {1}, leaving a Mountain untapped for the Bolt's {R}.
func TestAutoTapForCost_AtogPreservesRedForBolt(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	m1 := addLand(t, g, pid, "Mountain", Red)
	m2 := addLand(t, g, pid, "Mountain", Red)
	forest := addLand(t, g, pid, "Forest", Green)

	bolt := NewCreature("Lightning Bolt", "{R}", 0, 0) // shape doesn't matter, only cost
	bolt.SetOwner(pid)
	g.players[0].AddToHand(bolt)
	// "Atog" stand-in is the one being cast; only Bolt should contribute to
	// hand-demand. Atog isn't an actual card here, so pass a fresh ID as the
	// excluded "casting card" (it won't match anything in hand → Bolt counts).
	atogCost := ManaCost{Generic: 1, Red: 1}
	if err := g.AutoTapForCostWithHint(pid, atogCost, AutoTapHint{CastingCard: uuid.New()}); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}

	if !forest.Tapped {
		t.Error("expected Forest tapped (non-cost color, low score)")
	}
	tappedMountains := 0
	if m1.Tapped {
		tappedMountains++
	}
	if m2.Tapped {
		tappedMountains++
	}
	if tappedMountains != 1 {
		t.Errorf("expected exactly 1 Mountain tapped, got %d", tappedMountains)
	}
}

// Mishra's Factory animate ability costs {1} (no {T}). When activated, the
// algorithm should not tap the Factory itself for its own {1} cost — it
// should pick the Mountain instead.
func TestAutoTapForCost_FactoryNotTappedForOwnAbility(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// Stand-in Factory: colorless mana + an unrelated {1}-cost activated
	// ability. Avoids importing the cards package into engine tests.
	factory := NewLand("Mishra's Factory",
		WithManaAbility(Colorless),
		WithActivatedAbility(GainLife(1), GenericCost(1)),
	)
	factory.SetOwner(pid)
	fperm := g.PutOnBattlefield(factory, pid)
	fperm.RevokeBaseAttr(AttrSummonSick)

	mountain := addLand(t, g, pid, "Mountain", Red)

	hint := AutoTapHint{ActivationSource: fperm.ID(), ActivationTapsSource: false}
	if err := g.AutoTapForCostWithHint(pid, ManaCost{Generic: 1}, hint); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}

	if fperm.Tapped {
		t.Error("expected Factory to stay untapped")
	}
	if !mountain.Tapped {
		t.Error("expected Mountain to be tapped for the {1}")
	}
}

// Casting an unrelated spell with a utility land in play: prefer to tap the
// basic, leaving the utility land available.
func TestAutoTapForCost_PrefersBasicOverFactory(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	factory := NewLand("Mishra's Factory",
		WithManaAbility(Colorless),
		WithActivatedAbility(GainLife(1), GenericCost(1)),
	)
	factory.SetOwner(pid)
	fperm := g.PutOnBattlefield(factory, pid)
	fperm.RevokeBaseAttr(AttrSummonSick)

	mountain := addLand(t, g, pid, "Mountain", Red)

	// Pay {1} for a generic cost — should pick Mountain (basic, no utility).
	if err := g.AutoTapForCostWithHint(pid, ManaCost{Generic: 1}, AutoTapHint{}); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}
	if fperm.Tapped {
		t.Error("expected Factory to stay untapped (utility penalty)")
	}
	if !mountain.Tapped {
		t.Error("expected Mountain to be tapped")
	}
}

// Strip Mine has a utility ability; pay {1} should tap the Mountain.
func TestAutoTapForCost_StripMinePreservedForGeneric(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	strip := NewLand("Strip Mine",
		WithManaAbility(Colorless),
		WithActivatedAbility(GainLife(1), Tap()), // stand-in utility ability
	)
	strip.SetOwner(pid)
	sperm := g.PutOnBattlefield(strip, pid)
	sperm.RevokeBaseAttr(AttrSummonSick)

	mountain := addLand(t, g, pid, "Mountain", Red)

	if err := g.AutoTapForCostWithHint(pid, ManaCost{Generic: 1}, AutoTapHint{}); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}
	if sperm.Tapped {
		t.Error("expected Strip Mine to stay untapped (utility penalty)")
	}
	if !mountain.Tapped {
		t.Error("expected Mountain to be tapped")
	}
}

// If the only mana source is the activated permanent itself, the penalty
// shouldn't prevent tapping it — fallback must still succeed.
func TestAutoTapForCost_FallsBackWhenSourceIsOnlyOption(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	factory := NewLand("Mishra's Factory",
		WithManaAbility(Colorless),
		WithActivatedAbility(GainLife(1), GenericCost(1)),
	)
	factory.SetOwner(pid)
	fperm := g.PutOnBattlefield(factory, pid)
	fperm.RevokeBaseAttr(AttrSummonSick)

	hint := AutoTapHint{ActivationSource: fperm.ID(), ActivationTapsSource: false}
	if err := g.AutoTapForCostWithHint(pid, ManaCost{Generic: 1}, hint); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}
	if !fperm.Tapped {
		t.Error("expected Factory to be tapped as last-resort fallback")
	}
}

// Sol Ring + Mountain, pay {2}: Sol Ring should be picked (colorless, no
// color-flex penalty) and its Amount=2 covers the full cost in one tap.
func TestAutoTapForCost_PrefersColorlessForGeneric(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	ring := NewArtifact("Sol Ring", "{1}", WithMultiManaAbility(ManaProduction{Color: Colorless, Amount: 2}))
	ring.SetOwner(pid)
	rperm := g.PutOnBattlefield(ring, pid)
	rperm.RevokeBaseAttr(AttrSummonSick)

	mountain := addLand(t, g, pid, "Mountain", Red)

	if err := g.AutoTapForCostWithHint(pid, ManaCost{Generic: 2}, AutoTapHint{}); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}
	if !rperm.Tapped {
		t.Error("expected Sol Ring tapped (colorless preferred for generic)")
	}
	if mountain.Tapped {
		t.Error("expected Mountain untapped")
	}
}

// Urza's Mine (colorless) + Forest, pay {1}{G}: Forest pays {G}, Mine pays {1}.
func TestAutoTapForCost_UrzaLandPreferredOverColoredForGeneric(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	mine := NewLand("Urza's Mine", WithManaAbility(Colorless))
	mine.SetOwner(pid)
	mperm := g.PutOnBattlefield(mine, pid)
	mperm.RevokeBaseAttr(AttrSummonSick)

	forest := addLand(t, g, pid, "Forest", Green)

	if err := g.AutoTapForCostWithHint(pid, ManaCost{Generic: 1, Green: 1}, AutoTapHint{}); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}
	if !mperm.Tapped {
		t.Error("expected Urza's Mine tapped for the {1}")
	}
	if !forest.Tapped {
		t.Error("expected Forest tapped for the {G}")
	}
}

// A single-tap dual-emit land ("{T}: Add {G}{W}") should pay both {G} and
// {W} with one tap. Before per-tap output tracking, the solver picked the
// source for {G}, marked it unavailable, and then failed to find a {W}
// source even though the same tap produced one.
func TestAutoTapForCost_DualEmitSingleTapPaysTwoColors(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	dual := NewLand("Savannah",
		WithMultiManaAbility(
			ManaProduction{Color: Green, Amount: 1},
			ManaProduction{Color: White, Amount: 1},
		),
	)
	dual.SetOwner(pid)
	perm := g.PutOnBattlefield(dual, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	if err := g.AutoTapForCost(pid, ManaCost{Green: 1, White: 1}); err != nil {
		t.Fatalf("AutoTapForCost({G}{W}) failed on dual-emit land: %v", err)
	}
	if !perm.Tapped {
		t.Error("expected dual-emit land to be tapped")
	}
}

// Dual-emit land tapped for one color, with the other color showing up in
// the floating pool — verify the surplus actually reached the pool via
// TapForMana, not just the solver bookkeeping.
func TestAutoTapForCost_DualEmitSurplusGoesToPool(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	dual := NewLand("Savannah",
		WithMultiManaAbility(
			ManaProduction{Color: Green, Amount: 1},
			ManaProduction{Color: White, Amount: 1},
		),
	)
	dual.SetOwner(pid)
	perm := g.PutOnBattlefield(dual, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	// Pay only {G}; the {W} from the same tap should float in the pool.
	if err := g.AutoTapForCost(pid, ManaCost{Green: 1}); err != nil {
		t.Fatalf("AutoTapForCost({G}) failed: %v", err)
	}
	if got := g.players[0].ManaPool().Count(White); got != 1 {
		t.Errorf("expected {W} in pool from dual-emit surplus, got %d", got)
	}
}

// A dual land registered with two separate mana abilities ("{T}: Add {U}"
// and "{T}: Add {B}") must add the color the solver picked it for, not the
// color of whichever ability happens to be listed first. Underground Sea
// is the canonical example.
func TestAutoTapForCost_MultiAbilityDualLandRespectsSolverColor(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	dual := NewLand("Underground Sea",
		WithManaAbility(Blue),
		WithManaAbility(Black),
	)
	dual.SetOwner(pid)
	perm := g.PutOnBattlefield(dual, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	if err := g.AutoTapForCost(pid, ManaCost{Black: 1}); err != nil {
		t.Fatalf("AutoTapForCost({B}) failed: %v", err)
	}
	if !perm.Tapped {
		t.Fatal("expected Underground Sea to be tapped")
	}
	pool := g.players[0].ManaPool()
	if got := pool.Count(Black); got != 1 {
		t.Errorf("expected 1 {B} in pool, got %d", got)
	}
	if got := pool.Count(Blue); got != 0 {
		t.Errorf("expected 0 {U} in pool (Sea was tapped for {B}), got %d", got)
	}
}

// When the ability cost includes {T} on the source, the source must be hard-
// excluded from auto-tap (cost payment will tap it). Pay the mana portion
// from the other land; the source remains untapped at this point so that the
// later {T} cost can pay it.
func TestAutoTapForCost_TapCostHardExcludesSource(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// Land with {T}, {1}: gain 1 — when activated, both {T} and {1} are paid.
	utility := NewLand("Utility Land",
		WithManaAbility(Colorless),
		WithActivatedAbility(GainLife(1), Tap(), WithCost(GenericCost(1))),
	)
	utility.SetOwner(pid)
	uperm := g.PutOnBattlefield(utility, pid)
	uperm.RevokeBaseAttr(AttrSummonSick)

	mountain := addLand(t, g, pid, "Mountain", Red)

	hint := AutoTapHint{ActivationSource: uperm.ID(), ActivationTapsSource: true}
	if err := g.AutoTapForCostWithHint(pid, ManaCost{Generic: 1}, hint); err != nil {
		t.Fatalf("AutoTapForCost failed: %v", err)
	}
	if uperm.Tapped {
		t.Error("expected source land to remain untapped after auto-tap (its {T} cost pays later)")
	}
	if !mountain.Tapped {
		t.Error("expected Mountain to pay the {1}")
	}
}
