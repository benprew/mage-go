package interactive

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGetAvailableActions_IncludesUntappedLandManaAbility(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	forest := mage.NewLand("Forest", mage.WithManaAbility(core.Green))
	forest.SetOwner(pa.PlayerID())
	perm := g.PutOnBattlefield(forest, pa.PlayerID())

	actions := GetAvailableActions(g, pa.PlayerID())
	var manaOpt *ActionOption
	for i := range actions {
		if actions[i].Type == ActionActivateAbility && actions[i].PermanentID == perm.ID() {
			manaOpt = &actions[i]
			break
		}
	}
	if manaOpt == nil {
		t.Fatalf("expected ActionActivateAbility for untapped Forest, got %+v", actions)
	}
	if manaOpt.CardName != "Forest" {
		t.Errorf("expected CardName Forest, got %q", manaOpt.CardName)
	}
	if !manaOpt.IsManaAbility {
		t.Errorf("expected IsManaAbility to be true")
	}
}

func TestGetAvailableActions_ExcludesTappedLand(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	forest := mage.NewLand("Forest", mage.WithManaAbility(core.Green))
	forest.SetOwner(pa.PlayerID())
	perm := g.PutOnBattlefield(forest, pa.PlayerID())
	perm.Tapped = true

	actions := GetAvailableActions(g, pa.PlayerID())
	for _, a := range actions {
		if a.Type == ActionActivateAbility && a.PermanentID == perm.ID() {
			t.Fatalf("expected no ActionActivateAbility for tapped Forest, got %+v", a)
		}
	}
}

func TestGetAvailableActions_ManaDorkSummoningSickness(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	elf := mage.NewCreature("Llanowar Elves", "{G}", 1, 1, mage.WithManaAbility(core.Green))
	elf.SetOwner(pa.PlayerID())
	perm := g.PutOnBattlefield(elf, pa.PlayerID())
	// Creatures have AttrSummonSick on ETB.

	actions := GetAvailableActions(g, pa.PlayerID())
	for _, a := range actions {
		if a.Type == ActionActivateAbility && a.PermanentID == perm.ID() {
			t.Fatalf("expected no ActionActivateAbility for summoning-sick Llanowar Elves")
		}
	}

	// Remove summoning sickness (e.g. controlled since turn start)
	perm.RevokeBaseAttr(core.AttrSummonSick)
	actions = GetAvailableActions(g, pa.PlayerID())
	found := false
	for _, a := range actions {
		if a.Type == ActionActivateAbility && a.PermanentID == perm.ID() {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ActionActivateAbility for ready Llanowar Elves")
	}
}

func TestGetAvailableActions_DualLandOffersBothColors(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	taiga := mage.NewLand("Taiga",
		mage.WithManaAbility(core.Red),
		mage.WithManaAbility(core.Green),
	)
	taiga.SetOwner(pa.PlayerID())
	perm := g.PutOnBattlefield(taiga, pa.PlayerID())

	actions := GetAvailableActions(g, pa.PlayerID())
	var taigaOpts []ActionOption
	for _, a := range actions {
		if a.Type == ActionActivateAbility && a.PermanentID == perm.ID() {
			taigaOpts = append(taigaOpts, a)
		}
	}
	if len(taigaOpts) != 2 {
		t.Fatalf("expected 2 ActionActivateAbility for Taiga, got %d", len(taigaOpts))
	}
	if taigaOpts[0].AbilityIndex == taigaOpts[1].AbilityIndex {
		t.Errorf("expected distinct AbilityIndex for Taiga abilities")
	}
}

func TestActivateAbilityByIndex_ManaAbilityAddsToPool(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	forest := mage.NewLand("Forest", mage.WithManaAbility(core.Green))
	forest.SetOwner(pa.PlayerID())
	perm := g.PutOnBattlefield(forest, pa.PlayerID())

	err := g.ActivateAbilityByIndex(pa.PlayerID(), perm.ID(), 0, nil)
	if err != nil {
		t.Fatalf("ActivateAbilityByIndex failed: %v", err)
	}
	if !perm.Tapped {
		t.Errorf("expected Forest to be tapped")
	}
	if got := pa.ManaPool().Count(core.Green); got != 1 {
		t.Errorf("green mana in pool = %d, want 1", got)
	}
}

func TestActivateAbilityByIndex_DualLandSpecificAbility(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	taiga := mage.NewLand("Taiga",
		mage.WithManaAbility(core.Red),
		mage.WithManaAbility(core.Green),
	)
	taiga.SetOwner(pa.PlayerID())
	perm := g.PutOnBattlefield(taiga, pa.PlayerID())

	// Activate second ability (Green)
	err := g.ActivateAbilityByIndex(pa.PlayerID(), perm.ID(), 1, nil)
	if err != nil {
		t.Fatalf("ActivateAbilityByIndex failed: %v", err)
	}
	if !perm.Tapped {
		t.Errorf("expected Taiga to be tapped")
	}
	if got := pa.ManaPool().Count(core.Green); got != 1 {
		t.Errorf("green mana in pool = %d, want 1", got)
	}
	if got := pa.ManaPool().Count(core.Red); got != 0 {
		t.Errorf("red mana in pool = %d, want 0", got)
	}
}

func TestActivateAbilityByIndex_UndoManaAbility(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	forest := mage.NewLand("Forest", mage.WithManaAbility(core.Green))
	forest.SetOwner(pa.PlayerID())
	perm := g.PutOnBattlefield(forest, pa.PlayerID())

	snap := captureForUndo(g, pa.PlayerID(), 0)

	err := g.ActivateAbilityByIndex(pa.PlayerID(), perm.ID(), 0, nil)
	if err != nil {
		t.Fatalf("ActivateAbilityByIndex failed: %v", err)
	}
	if !perm.Tapped || pa.ManaPool().Count(core.Green) != 1 {
		t.Fatalf("pre-undo assertion failed")
	}

	restoreFromUndo(g, pa.PlayerID(), snap)

	if perm.Tapped {
		t.Errorf("expected Forest to be untapped after undo")
	}
	if got := pa.ManaPool().Count(core.Green); got != 0 {
		t.Errorf("expected empty mana pool after undo, got %d", got)
	}
}
