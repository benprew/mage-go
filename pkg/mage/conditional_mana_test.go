package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestConditionalMana(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	yes := []ManaProduction{{Color: core.Colorless, Amount: 3}}
	no := []ManaProduction{{Color: core.Green, Amount: 1}}
	ability := NewConditionalManaAbility(PlayerControls(You, HasSubType("Tower")), yes, no)
	source := g.PutOnBattlefield(NewLand("Source", WithAbility(ability)), a.PlayerID())
	yes[0].Amount = 99
	no[0].Amount = 99
	tower := g.PutOnBattlefield(NewLand("Other", WithSubTypes("Tower")), a.PlayerID())
	check := func(game *Game, amount int, color core.Color) {
		t.Helper()
		got := ManaProductionsForAbilityInGame(ability, game, source.ID())
		if len(got) != 1 || got[0].Amount != amount || got[0].Color != color {
			t.Fatalf("production = %+v", got)
		}
		got[0].Amount = 98
	}
	check(g, 3, core.Colorless)
	clone := g.Clone()
	g.PhaseOut(tower.ID())
	check(g, 1, core.Green)
	check(clone, 3, core.Colorless)
	g.PhaseIn([]uuid.UUID{tower.ID()})
	g.AddControlEffect(ControlEffectSpec{TargetID: tower.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
	check(g, 1, core.Green)
	g.AddControlEffect(ControlEffectSpec{TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
	check(g, 3, core.Colorless)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 0 || b.ManaPool().Count(core.Colorless) != 3 || !g.FindPermanent(source.ID()).Tapped {
		t.Fatal("activation differs from query")
	}
	g.PhaseOut(source.ID())
	if got := ManaProductionsForAbilityInGame(ability, g, source.ID()); len(got) != 0 {
		t.Fatal("phased source production")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	if got := ManaProductionsForAbilityInGame(ability, g, source.ID()); len(got) != 0 {
		t.Fatal("absent source production")
	}
	if got := ManaProductionsForAbility(ability); len(got) != 0 {
		t.Fatal("context-free dynamic production")
	}
}

func TestConditionalManaInvalid(t *testing.T) {
	for _, tc := range []struct {
		condition StateCondition
		yes, no   []ManaProduction
	}{
		{nil, []ManaProduction{{Color: core.Green, Amount: 1}}, []ManaProduction{{Color: core.Green, Amount: 1}}},
		{PlayerControls(DeclarePlayerTarget("missing"), IsLand), []ManaProduction{{Color: core.Green, Amount: 1}}, []ManaProduction{{Color: core.Green, Amount: 1}}},
		{Controls(IsLand), nil, []ManaProduction{{Color: core.Green, Amount: 1}}},
		{Controls(IsLand), []ManaProduction{{Color: core.Green, Amount: 0}}, []ManaProduction{{Color: core.Green, Amount: 1}}},
		{Controls(IsLand), []ManaProduction{{Color: core.Color(99), Amount: 1}}, []ManaProduction{{Color: core.Green, Amount: 1}}},
	} {
		card := NewLand("Invalid", WithAbility(NewConditionalManaAbility(tc.condition, tc.yes, tc.no)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid conditional mana")
		}
	}
}
