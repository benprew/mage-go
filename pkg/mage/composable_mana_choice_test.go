package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAddManaOfAnyOneColorChoiceAndEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	addTappedForManaWatcher(g, a)
	ability := Activate(Costs(TapCost(This)), AddManaOfAnyOneColor(Fixed(2)))
	source := g.PutOnBattlefield(NewLand("Choice source", WithAbility(ability)), a.PlayerID())
	if err := ValidateCardDefinition("Choice source", source.Card); err != nil {
		t.Fatal(err)
	}
	productions := ManaProductionsForAbility(ability)
	if len(productions) != 1 || productions[0].Color != AnyColor || productions[0].Amount != 2 || productions[0].AnyCombination {
		t.Fatalf("planner metadata = %+v", productions)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(White) != 2 || g.StackSize() != 0 {
		t.Fatal("choice did not add both mana immediately")
	}
	resolvePendingTriggers(g)
	if a.Life() != 21 {
		t.Fatal("tap-for-mana event was not emitted once")
	}
}

func TestAddManaOfAnyOneColorPlanningAndRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := Activate(Costs(TapCost(This)), AddManaOfAnyOneColor(Fixed(2)))
	source := g.PutOnBattlefield(NewLand("Choice source", WithAbility(ability)), a.PlayerID())
	if !g.CanAfford(a.PlayerID(), ManaCost{Green: 2}, nil) {
		t.Fatal("planner missed chosen-color production")
	}
	if g.CanAfford(a.PlayerID(), ManaCost{Green: 1, Blue: 1}, nil) {
		t.Fatal("planner split one color choice")
	}
	clone := g.Clone()
	if err := clone.AutoTapForCost(a.PlayerID(), ManaCost{Green: 2}); err != nil {
		t.Fatal(err)
	}
	if !clone.FindPermanent(source.ID()).Tapped || source.Tapped || a.ManaPool().Count(Green) != 0 {
		t.Fatal("planned mana leaked across clones")
	}
	if err := g.AutoTapForCost(a.PlayerID(), ManaCost{Green: 1, Blue: 1}); err == nil {
		t.Fatal("split-color payment succeeded")
	}
	if source.Tapped || a.ManaPool().Count(Green) != 0 || a.ManaPool().Count(Blue) != 0 {
		t.Fatal("failed payment changed source or pool")
	}
}

func TestAddManaOfAnyOneColorValidation(t *testing.T) {
	for _, amount := range []ResolutionValue{nil, FromNumber(BindNumber("missing"))} {
		card := NewSorcery("Invalid", "", Spell(AddManaOfAnyOneColor(amount)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid amount accepted")
		}
	}
	ability := Activate(Costs(TapCost(This)), AddManaOfAnyOneColor(PowerOf(This)))
	if !IsActivatedManaAbility(ability) || ManaProductionsForAbility(ability) != nil {
		t.Fatal("dynamic amount was misclassified or advertised to planner")
	}
}

func TestAddManaOfAnyOneColorDynamicAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := Activate(Costs(TapCost(This)), AddManaOfAnyOneColor(PowerOf(This)))
	source := g.PutOnBattlefield(NewCreature("Dynamic source", "", 3, 3, WithAbility(ability), WithKeyword(Haste)), a.PlayerID())
	clone := g.Clone()
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(a.PlayerID()).ManaPool().Count(White) != 3 || a.ManaPool().Count(White) != 0 || source.Tapped {
		t.Fatal("dynamic choice leaked across clones")
	}
	spell := NewInstant("Choice spell", "", Spell(AddManaOfAnyOneColor(Fixed(2))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.ResolveStack()
	if a.ManaPool().Count(White) != 2 || b.ManaPool().Count(White) != 2 {
		t.Fatal("copied mana instruction lost its resolving controller")
	}
}
