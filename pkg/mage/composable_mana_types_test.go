package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type producedTypeChoicePlayer struct {
	*BasePlayer
	choice core.Color
	calls  int
}

func (p *producedTypeChoicePlayer) ChooseManaColor(string) core.Color {
	p.calls++
	return p.choice
}

func TestLandManaTypesCapturedForTriggeredMana(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("producer")
	types := BindManaTypes("produced types")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithPlayer(player).WithManaTypes(types),
		AddManaOfProducedTypeToPlayer(FromPlayer(player), types, Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Blue, Fixed(2)))))
	g.PutOnBattlefield(land, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if got := b.ManaPool().Count(core.Blue); got != 3 || g.StackSize() != 0 {
		t.Fatalf("blue mana = %d, stack = %d; want 3 and 0", got, g.StackSize())
	}
}

func TestLandManaTypesDeclaration(t *testing.T) {
	types := BindManaTypes("types")
	for _, pattern := range []landTappedForManaPattern{
		LandTappedForMana().WithManaTypes(nil),
		LandTappedForMana().WithManaTypes(BindManaTypes("")),
		LandTappedForMana().WithManaTypes(types).WithManaTypes(types),
	} {
		card := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern,
			AddManaOfProducedTypeToPlayer(You, types, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid type binding")
		}
	}
	card := NewEnchantment("Unbound", "", WithAbility(WhenEvent(LandTappedForMana(),
		AddManaOfProducedTypeToPlayer(You, types, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("accepted unbound type set")
	}
	for _, effect := range []ResolvingEffect{
		AddManaOfProducedTypeToPlayer(nil, types, Fixed(1)),
		AddManaOfProducedTypeToPlayer(You, nil, Fixed(1)),
		AddManaOfProducedTypeToPlayer(You, types, nil),
	} {
		card := NewEnchantment("Bad action", "", WithAbility(WhenEvent(
			LandTappedForMana().WithManaTypes(types), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid mana action")
		}
	}
}

func TestLandManaTypesMultiTypeChoiceAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	b := &producedTypeChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: core.Red}
	g := NewGame(a, b)
	player := BindPlayer("producer")
	types := BindManaTypes("produced types")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithPlayer(player).WithManaTypes(types),
		AddManaOfProducedTypeToPlayer(FromPlayer(player), types, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), Sequence(
		AddManaAmount(core.Blue, Fixed(1)), AddManaAmount(core.Red, Fixed(1))))))
	g.PutOnBattlefield(land, b.PlayerID())
	clone := g.Clone()
	if err := clone.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(land.ID()).Tapped || b.ManaPool().Count(core.Red) != 0 || clone.GetPlayer(b.PlayerID()).ManaPool().Count(core.Blue) != 2 {
		t.Fatal("clone mana choice changed original or missed produced type")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 2 || b.ManaPool().Count(core.Blue) != 1 || b.calls != 1 {
		t.Fatal("mana type choice did not use the produced set")
	}
}

func TestLandManaTypesInvalidChoiceAndSourcePhasing(t *testing.T) {
	a := &producedTypeChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: core.Green}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	types := BindManaTypes("types")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithManaTypes(types), AddManaOfProducedTypeToPlayer(You, types, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), Sequence(
		AddManaAmount(core.Blue, Fixed(1)), AddManaAmount(core.Red, Fixed(1))))))
	g.PutOnBattlefield(land, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(core.Blue) != 1 || a.ManaPool().Count(core.Green) != 0 {
		t.Fatal("invalid color choice produced a type absent from the event")
	}
	g.FindPermanent(watcher.ID()).PhasedOut = true
	g.FindPermanent(land.ID()).Tapped = false
	if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(core.Blue) != 1 {
		t.Fatal("phased trigger source produced mana")
	}
}
