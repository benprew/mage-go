package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestManaSystem_InitializationAndClone(t *testing.T) {
	ms := NewManaSystem()
	if len(ms.scratch) != 0 {
		t.Fatalf("expected empty scratch, got len %d", len(ms.scratch))
	}

	clone := ms.Clone()
	if len(clone.scratch) != 0 {
		t.Fatalf("expected empty clone scratch, got len %d", len(clone.scratch))
	}
}

func TestManaSystem_EmptyManaPools(t *testing.T) {
	ms := NewManaSystem()
	p1 := NewBasePlayer("Player 1")
	p2 := NewBasePlayer("Player 2")

	p1.ManaPool().Add(Red, 3)
	p2.ManaPool().Add(Blue, 2)
	if p1.ManaPool().TotalMana() != 3 {
		t.Errorf("expected 3 mana for p1, got %d", p1.ManaPool().TotalMana())
	}
	if p2.ManaPool().TotalMana() != 2 {
		t.Errorf("expected 2 mana for p2, got %d", p2.ManaPool().TotalMana())
	}

	ms.EmptyManaPools([]Player{p1, p2})
	if p1.ManaPool().TotalMana() != 0 {
		t.Errorf("expected 0 mana for p1, got %d", p1.ManaPool().TotalMana())
	}
	if p2.ManaPool().TotalMana() != 0 {
		t.Errorf("expected 0 mana for p2, got %d", p2.ManaPool().TotalMana())
	}
}

func TestManaSystem_SourceDiscoveryAndAffordability(t *testing.T) {
	g := newPriorityTestGame()
	p1ID := g.players[0].PlayerID()

	mountain := NewLand("Mountain", WithManaAbility(Red))
	mountain.SetOwner(p1ID)
	permM := g.PutOnBattlefield(mountain, p1ID)
	permM.RevokeBaseAttr(AttrSummonSick)

	forest := NewLand("Forest", WithManaAbility(Green))
	forest.SetOwner(p1ID)
	permF := g.PutOnBattlefield(forest, p1ID)
	permF.RevokeBaseAttr(AttrSummonSick)

	sources := g.mana.GetUntappedManaSources(g, p1ID)
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}

	if h := g.mana.HypotheticalMana(g, p1ID); h != 2 {
		t.Errorf("expected 2 hypothetical mana, got %d", h)
	}
	if !g.mana.CanAfford(g, p1ID, ParseManaCost("{R}{G}"), nil) {
		t.Errorf("expected {R}{G} to be affordable")
	}
	if g.mana.CanAfford(g, p1ID, ParseManaCost("{R}{R}"), nil) {
		t.Errorf("expected {R}{R} to not be affordable")
	}
	if x := g.mana.MaxXValue(g, p1ID, ParseManaCost("{X}{R}"), nil); x != 1 {
		t.Errorf("expected MaxXValue 1 for {X}{R}, got %d", x)
	}
}

func TestManaSystem_TapForMana_And_Events(t *testing.T) {
	g := newPriorityTestGame()
	p1 := g.players[0]
	p1ID := p1.PlayerID()

	watcher := NewEnchantment("Tapped for Mana Watcher", "{0}",
		WithAbility(NewTriggered(EvtTappedForMana, false, GainLife(1))),
	)
	watcher.SetOwner(p1ID)
	g.PutOnBattlefield(watcher, p1ID)

	island := NewLand("Island", WithManaAbility(Blue))
	island.SetOwner(p1ID)
	perm := g.PutOnBattlefield(island, p1ID)
	perm.RevokeBaseAttr(AttrSummonSick)

	err := g.mana.TapForMana(g, p1ID, perm.ID(), Blue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !perm.Tapped {
		t.Errorf("expected permanent to be tapped")
	}
	if p1.ManaPool().Count(Blue) != 1 {
		t.Errorf("expected 1 blue mana, got %d", p1.ManaPool().Count(Blue))
	}

	g.PutTriggersOnStack()
	g.ResolveStack()

	if got := p1.Life(); got != 21 {
		t.Errorf("expected life 21, got %d", got)
	}
}

func TestManaSystem_ManaBonuses(t *testing.T) {
	g := newPriorityTestGame()
	p1ID := g.players[0].PlayerID()

	mountain := NewLand("Mountain", WithManaAbility(Red), WithSubTypes("Mountain"))
	mountain.SetOwner(p1ID)
	mPerm := g.PutOnBattlefield(mountain, p1ID)
	mPerm.RevokeBaseAttr(AttrSummonSick)

	gauntlet := NewArtifact("Gauntlet of Might", "{4}", WithAbility(NewManaBonusAbility(HasSubType("Mountain"), Red)))
	gauntlet.SetOwner(p1ID)
	g.PutOnBattlefield(gauntlet, p1ID)

	bonuses := g.mana.ManaBonuses(g, mPerm.ID())
	if len(bonuses) != 1 {
		t.Fatalf("expected 1 bonus, got %d", len(bonuses))
	}
	if bonuses[0] != ManaBonusColor(Red) {
		t.Errorf("expected Red bonus, got %v", bonuses[0])
	}

	err := g.mana.TapForMana(g, p1ID, mPerm.ID(), Red)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.players[0].ManaPool().Count(Red) != 2 {
		t.Errorf("expected 2 red mana, got %d", g.players[0].ManaPool().Count(Red))
	}
}

func TestManaSystem_PreservationScoreAndDrawback(t *testing.T) {
	g := newPriorityTestGame()
	p1ID := g.players[0].PlayerID()

	mountain := NewLand("Mountain", WithManaAbility(Red))
	mountain.SetOwner(p1ID)
	mPerm := g.PutOnBattlefield(mountain, p1ID)
	mPerm.RevokeBaseAttr(AttrSummonSick)

	cityOfBrass := NewLand("City of Brass",
		WithManaAbility(AnyColor),
		WithAbility(NewTriggered(EvtTapped, false,
			DealDamageToPlayers(Fixed(1), SelectController()),
		).SetConditionData(EventSourceIsSelf{})),
	)
	cityOfBrass.SetOwner(p1ID)
	cPerm := g.PutOnBattlefield(cityOfBrass, p1ID)
	cPerm.RevokeBaseAttr(AttrSummonSick)

	sources := g.mana.GetUntappedManaSources(g, p1ID)
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}

	var mScore, cScore int
	for _, src := range sources {
		if src.PermanentID == mPerm.ID() {
			mScore = g.mana.PreservationScore(g, src, AutoTapHint{}, [AnyColor + 1]int{})
		} else if src.PermanentID == cPerm.ID() {
			cScore = g.mana.PreservationScore(g, src, AutoTapHint{}, [AnyColor + 1]int{})
		}
	}

	if cScore <= mScore {
		t.Errorf("expected city of brass preservation score (%d) to be greater than mountain (%d)", cScore, mScore)
	}
}
