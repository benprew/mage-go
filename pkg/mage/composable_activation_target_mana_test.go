package mage

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestActivationTargetManaCountsAndColors(t *testing.T) {
	for _, color := range []Color{White, Blue, Black, Red, Green} {
		for count := 0; count <= 3; count++ {
			t.Run(fmt.Sprintf("%v/%d", color, count), func(t *testing.T) {
				g, a, b := newActivationDivisionGame()
				targets := DeclareAnyNumberDamageTargets("targets")
				source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(
					Costs(ManaCostOf("{2}"), TapCost(This), LifePayCost(2)), GainLifeFor(You, Fixed(1)),
					WithTargets(targets), ActivationManaPerTarget(color, 2)))), a.PlayerID())
				victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
				ids := []uuid.UUID{a.PlayerID(), b.PlayerID(), victim.ID()}[:count]
				a.ManaPool().Add(Colorless, 2)
				a.ManaPool().Add(color, count*2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != 0 || a.Life() != 18 || !g.FindPermanent(source.ID()).Tapped || len(g.StackPeek().Targets) != count {
					t.Fatal("wrong combined payment or announcement")
				}
			})
		}
	}
}

func TestActivationTargetManaGroupsAutotapAndReduction(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	first, second := DeclarePlayerTarget("first"), DeclarePlayerTarget("second")
	source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(
		Costs(ManaCostOf("{3}"), TapCost(This)), GainLifeFor(You, Fixed(1)),
		WithActivationTargetGroups(first, second), ActivationManaPerTarget(Red, 1)))), a.PlayerID())
	g.PutOnBattlefield(NewEnchantment("Reducer", "", WithStaticAbility(
		FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(game *Game, _ uuid.UUID) error {
			game.AddActivationCostReduction(source.ID(), 2)
			return nil
		}))), a.PlayerID())
	lands := make([]uuid.UUID, 0, 3)
	for i, color := range []Color{Blue, Red, Red} {
		land := g.PutOnBattlefield(NewLand(fmt.Sprintf("Land %d", i), WithManaAbility(color)), a.PlayerID())
		lands = append(lands, land.ID())
	}
	if err := g.ActivateAbilityByIndexWithTargetGroups(a.PlayerID(), source.ID(), 0, [][]uuid.UUID{{b.PlayerID()}, {b.PlayerID()}}); err != nil {
		t.Fatal(err)
	}
	for _, id := range lands {
		if !g.FindPermanent(id).Tapped {
			t.Fatal("combined cost did not use all mana sources")
		}
	}
	if a.ManaPool().TotalMana() != 0 || len(g.StackPeek().Targets) != 2 {
		t.Fatal("repeated target occurrence or reduction changed colored payment")
	}
}

func TestActivationTargetManaRollback(t *testing.T) {
	for _, state := range []string{"underpaid", "wrong color", "absent", "phased", "duplicate", "unpaid life"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			targets := DeclareAnyNumberDamageTargets("targets")
			life := 2
			if state == "unpaid life" {
				life = 21
			}
			source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(
				Costs(ManaCostOf("{2}"), TapCost(This), LifePayCost(life)), GainLifeFor(You, Fixed(1)),
				WithTargets(targets), ActivationManaPerTarget(Red, 1)))), a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
			ids := []uuid.UUID{b.PlayerID(), victim.ID()}
			a.ManaPool().Add(Colorless, 2)
			color, amount := Red, 2
			switch state {
			case "underpaid":
				amount = 1
			case "wrong color":
				color = Blue
			case "absent":
				ids[1] = uuid.New()
			case "phased":
				g.PhaseOut(victim.ID())
			case "duplicate":
				ids[1] = ids[0]
			}
			a.ManaPool().Add(color, amount)
			events := 0
			for _, event := range []EventType{EvtAbilityActivated, EvtLifeLost, EvtBecomesTarget, EvtTapped} {
				g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(NewTriggered(event, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
					if reader == g {
						events++
					}
					return false
				}))), a.PlayerID())
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids); err == nil {
				t.Fatal("invalid activation succeeded")
			}
			if a.Life() != 20 || a.ManaPool().TotalMana() != 2+amount || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 || events != 0 {
				t.Fatal("failed activation changed live resources or events")
			}
		})
	}
}

func TestActivationTargetManaCopiesAndIdentity(t *testing.T) {
	for _, state := range []string{"absent", "returned", "phased"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			targets := DeclareAnyNumberDamageTargets("targets")
			source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(),
				DealDamageEach(This, targets, Fixed(2)), WithTargets(targets), ActivationManaPerTarget(Red, 1)))), a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
			a.ManaPool().Add(Red, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID(), victim.ID()}); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
			clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
			p := clone.FindPermanent(victim.ID())
			if state == "phased" {
				clone.PhaseOut(p.ID())
			} else {
				clone.RemoveFromBattlefield(p)
				if state == "returned" {
					clone.PutOnBattlefield(p.Card, b.PlayerID())
				}
			}
			clone.ResolveStackObject(copyObj)
			if clone.GetPlayer(b.PlayerID()).Life() != 18 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
				t.Fatal("copy required payment or lost source LKI")
			}
			if p := clone.FindPermanent(victim.ID()); p != nil && p.Damage != 0 {
				t.Fatal("illegal old target took damage")
			}
			if b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 0 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if b.Life() != 18 || g.FindPermanent(victim.ID()).Damage != 2 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("original payment or targets changed")
			}
		})
	}
}

func TestActivationTargetManaDeclarationsAndReuse(t *testing.T) {
	for _, option := range []ActivationOption{ActivationManaPerTarget(Red, 0), ActivationManaPerTarget(Red, -1), ActivationManaPerTarget(Colorless, 1), ActivationManaPerTarget(AnyColor, 1), ActivationManaPerTarget(Color(255), 1)} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), option)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "colored mana") {
			t.Fatalf("invalid declaration: %v", err)
		}
	}
	g, a, b := newActivationDivisionGame()
	option := ActivationManaPerTarget(Red, 1)
	targets := DeclareAnyNumberDamageTargets("targets")
	source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), WithTargets(targets), option))), a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source.Card); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, node := range InspectDefinition(source.Card.Abilities()[0]).Children {
		found = found || node.Role == "activation-mana-per-target"
	}
	if !found {
		t.Fatal("missing cost inspection")
	}
	a.ManaPool().Add(Red, 3)
	for _, ids := range [][]uuid.UUID{{a.PlayerID(), b.PlayerID()}, {b.PlayerID()}} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids); err != nil {
			t.Fatal(err)
		}
	}
	if a.ManaPool().TotalMana() != 0 || g.StackSize() != 2 {
		t.Fatal("pending instances shared their target count")
	}
}
