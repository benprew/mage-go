package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableManaProductionAndResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	addTappedForManaWatcher(g, a)
	ability := Activate(Costs(TapCost(This)), Sequence(AddMana(Blue, 1), Sequence(AddMana(Green, 2))))
	card := NewLand("Composed mana", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	permanent := g.PutOnBattlefield(card, a.PlayerID())
	got := ManaProductionsForAbility(ability)
	if len(got) != 2 || got[0].Color != Blue || got[0].Amount != 1 || got[1].Color != Green || got[1].Amount != 2 {
		t.Errorf("productions = %+v", got)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), permanent.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.GetStack().Size() != 0 || a.ManaPool().Count(Blue) != 1 || a.ManaPool().Count(Green) != 2 {
		t.Fatal("composed mana ability did not resolve immediately")
	}
	resolvePendingTriggers(g)
	if a.Life() != 21 {
		t.Fatal("composed mana activation did not emit exactly one tapped-for-mana event")
	}
}

func TestComposableManaPlanningAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := Activate(Costs(TapCost(This)), Sequence(AddMana(Blue, 1), AddMana(Green, 1)))
	card := NewLand("Composed mana", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	permanent := g.PutOnBattlefield(card, a.PlayerID())
	cost := ManaCost{Blue: 1, Green: 1}
	if !g.CanAfford(a.PlayerID(), cost, nil) {
		t.Fatal("planner did not find composed production")
	}
	clone := g.Clone()
	if err := clone.AutoTapForCost(a.PlayerID(), cost); err != nil {
		t.Fatal(err)
	}
	if !clone.FindPermanent(permanent.ID()).Tapped || g.FindPermanent(permanent.ID()).Tapped || a.ManaPool().Count(Blue) != 0 {
		t.Fatal("planned activation leaked between clones")
	}
	cp := clone.GetPlayer(a.PlayerID())
	if cp.ManaPool().Count(Blue) != 1 || cp.ManaPool().Count(Green) != 1 {
		t.Fatal("planned output differs from metadata")
	}
}

func TestComposableManaTargetAndMixedEffects(t *testing.T) {
	target := DeclarePlayerTarget("player")
	targeted := Activate(Costs(TapCost(This)), Sequence(AddMana(Blue, 1)), WithTargets(target))
	if got := ManaProductionsForAbility(targeted); got != nil {
		t.Fatalf("targeted ability advertised mana: %+v", got)
	}
	mixed := Activate(Costs(TapCost(This)), Sequence(AddMana(Blue, 1), GainLifeFor(You, Fixed(2))))
	if got := ManaProductionsForAbility(mixed); got != nil {
		t.Fatalf("planner omitted a non-mana instruction: %+v", got)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Mixed mana", WithAbility(mixed))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || a.ManaPool().Count(Blue) != 1 || g.GetStack().Size() != 0 {
		t.Fatal("mixed mana ability did not resolve all instructions immediately")
	}
	targetedCard := NewLand("Targeted mana", WithAbility(targeted))
	targetedCard.SetOwner(a.PlayerID())
	targetedPermanent := g.PutOnBattlefield(targetedCard, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), targetedPermanent.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.GetStack().Size() != 1 || a.ManaPool().Count(Blue) != 1 {
		t.Fatal("targeted mana instruction bypassed the stack")
	}
	g.ResolveTopOfStack()
	if a.ManaPool().Count(Blue) != 2 {
		t.Fatal("targeted instruction did not resolve normally")
	}
}

func TestComposableManaPlanningUntapRestriction(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := Activate(Costs(TapCost(This)), Sequence(AddMana(Blue, 1)), OnlyDuring(Untap))
	card := NewLand("Restricted mana", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.turns.SetStep(Upkeep)
	if _, ok := manaSourceAbilityForPlanning(ability, p.ID(), 0, g); ok {
		t.Fatal("planner ignored the untap-step restriction")
	}
}

func TestComposableManaWithoutTapCost(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	addTappedForManaWatcher(g, a)
	ability := Activate(Costs(ManaPayment("{1}")), Sequence(AddMana(Blue, 2)))
	card := NewArtifact("Mana filter", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	a.ManaPool().Add(Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.GetStack().Size() != 0 || a.ManaPool().Count(Blue) != 2 || a.ManaPool().Count(Colorless) != 0 || g.FindPermanent(p.ID()).Tapped {
		t.Fatal("mana ability without a tap cost did not pay and resolve immediately")
	}
	resolvePendingTriggers(g)
	if a.Life() != 20 {
		t.Fatal("untapped mana ability emitted a tapped-for-mana event")
	}
}

func TestComposableManaNonManaRestriction(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := Activate(Costs(TapCost(This)), Sequence(AddMana(Blue, 1)))
	card := NewLand("Mana source", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	p.GrantBaseAttr(AttrCantActivateNonManaAbilities)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(Blue) != 1 {
		t.Fatal("mana production was blocked by a non-mana restriction")
	}
}

func TestComposableManaChoiceGroups(t *testing.T) {
	for _, separate := range []bool{false, true} {
		t.Run(map[bool]string{false: "one color", true: "independent colors"}[separate], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			effect := Sequence(AddAnyMana(2, AnyColor))
			if separate {
				effect = Sequence(AddAnyMana(1, AnyColor), Sequence(AddAnyMana(1, AnyColor)))
			}
			card := NewLand("Choice source", WithAbility(Activate(Costs(TapCost(This)), effect)))
			card.SetOwner(a.PlayerID())
			g.PutOnBattlefield(card, a.PlayerID())
			cost := ManaCost{Blue: 1, Green: 1}
			if got := g.CanAfford(a.PlayerID(), cost, nil); got != separate {
				t.Fatalf("can pay two colors = %v, want %v", got, separate)
			}
			if separate {
				if err := g.AutoTapForCost(a.PlayerID(), cost); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().Count(Blue) != 1 || a.ManaPool().Count(Green) != 1 {
					t.Fatal("planned independent choices were not preserved")
				}
			}
		})
	}
}

func TestComposableManaCostFailureLeavesState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Costed mana", WithAbility(Activate(Costs(TapCost(This), ManaPayment("{2}")), Sequence(AddMana(Blue, 3)))))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	a.ManaPool().Add(Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("unpayable activation succeeded")
	}
	if g.FindPermanent(p.ID()).Tapped || a.ManaPool().Count(Colorless) != 1 || a.ManaPool().Count(Blue) != 0 || g.GetStack().Size() != 0 {
		t.Fatal("failed mana activation changed resources")
	}
}
