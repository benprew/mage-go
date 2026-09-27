package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTappedForManaEventRecordsProducedTypes(t *testing.T) {
	cases := []struct {
		name     string
		card     Card
		activate func(*Game, uuid.UUID, uuid.UUID) error
		amount   int
		types    []Color
	}{
		{"ordinary tap", NewLand("Mixed", WithDynamicManaAbility(func(GameReader, uuid.UUID) []ManaProduction {
			return []ManaProduction{{Color: Blue, Amount: 2}, {Color: Green, Amount: 1}}
		})), func(g *Game, player, land uuid.UUID) error { return g.TapForMana(player, land) }, 3, []Color{Blue, Green}},
		{"direct activation", NewLand("Blue", WithManaAbility(Blue)), func(g *Game, player, land uuid.UUID) error {
			return g.ActivateAbilityByIndex(player, land, 0, nil)
		}, 1, []Color{Blue}},
		{"typed activation", NewLand("Typed", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Red, Fixed(2))))), func(g *Game, player, land uuid.UUID) error {
			return g.ActivateAbilityByIndex(player, land, 0, nil)
		}, 2, []Color{Red}},
		{"planned payment", NewLand("Green", WithManaAbility(Green)), func(g *Game, player, _ uuid.UUID) error {
			return g.AutoTapForCost(player, ManaCost{Green: 1})
		}, 1, []Color{Green}},
		{"planned color choice", NewLand("Choice", WithAbility(Activate(Costs(TapCost(This)), AddManaOfAnyOneColor(Fixed(2))))), func(g *Game, player, _ uuid.UUID) error {
			return g.AutoTapForCost(player, ManaCost{Green: 2})
		}, 2, []Color{Green}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			land := g.PutOnBattlefield(tc.card, a.PlayerID())
			var observed []GameEvent
			watcher := NewEnchantment("Observer", "", WithAbility(NewTriggered(EvtTappedForMana, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				observed = append(observed, *evt)
				return false
			})))
			g.PutOnBattlefield(watcher, b.PlayerID())
			if err := tc.activate(g, a.PlayerID(), land.ID()); err != nil {
				t.Fatal(err)
			}
			if len(observed) != 1 || observed[0].Amount != tc.amount || !slices.Equal(observed[0].ManaTypes, tc.types) {
				t.Fatalf("events = %+v, want amount %d and types %v", observed, tc.amount, tc.types)
			}
			if observed[0].SourceIncarnation != land.incarnationID || observed[0].PlayerID != a.PlayerID() {
				t.Fatalf("wrong source or player: %+v", observed[0])
			}
		})
	}
}

func TestLandTappedForManaTypeFilter(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blue := NewEnchantment("Blue watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().OfManaType(Blue), GainLifeFor(You, Fixed(1)))))
	green := NewEnchantment("Green watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().OfManaType(Green), GainLifeFor(You, Fixed(1)))))
	for _, card := range []Card{blue, green} {
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(card, b.PlayerID())
	}
	land := g.PutOnBattlefield(NewLand("Blue land", WithManaAbility(Blue)), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("trigger count = %d, want 1", g.StackSize())
	}
	g.ResolveStack()
	if b.Life() != 21 {
		t.Fatalf("life = %d, want 21", b.Life())
	}
	bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(
		LandTappedForMana().OfManaType(AnyColor), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("accepted a non-mana type")
	}
	duplicate := NewEnchantment("Duplicate", "", WithAbility(WhenEvent(
		LandTappedForMana().OfManaType(Blue).OfManaType(Green), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil {
		t.Fatal("accepted duplicate mana type filter")
	}
}
