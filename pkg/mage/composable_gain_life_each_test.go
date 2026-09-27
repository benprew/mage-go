package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGainLifeEachFinalTotalsAndReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	seen := 0
	watcher := NewEnchantment("Watcher", "", WithAbility(NewTriggered(core.EvtLifeGained, false, GainLifeFor(You, Fixed(1))).SetCondition(func(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if a.Life() != 26 || b.Life() != 26 || event.Amount != 6 {
			t.Errorf("event saw life %d/%d and amount %d", a.Life(), b.Life(), event.Amount)
		}
		seen++
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.AddLifeGainModifier(watcher.ID(), func(_ *Game, _ uuid.UUID, amount int) int {
		if a.Life() != 20 || b.Life() != 20 {
			t.Errorf("modifier saw a partial batch: %d/%d", a.Life(), b.Life())
		}
		return amount * 2
	})
	if err := ApplyEffect(g, GainLifeEach(EachPlayer, Fixed(3)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 26 || b.Life() != 26 || seen != 2 {
		t.Fatalf("life %d/%d, events %d", a.Life(), b.Life(), seen)
	}
}

func TestGainLifeEachZeroAbsentAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	clone := g.Clone()
	for _, effect := range []ResolvingEffect{
		GainLifeEach(EachPlayer, Fixed(0)),
		GainLifeEach(OpponentsOf(FromPlayer(BindPlayer("absent"))), Fixed(4)),
		GainLifeEach(OpponentsOf(You), Fixed(2)),
	} {
		if err := ApplyEffect(clone, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 22 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("wrong selected players or clone changed original")
	}
}

func TestGainLifeEachReplacementOutcomes(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		amount  int
		consume bool
	}{
		{"changed", 5, false},
		{"consumed", 0, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.effects.AddReplacement(&lifeOutcomeReplacement{amount: scenario.amount, consume: scenario.consume})
			if err := ApplyEffect(g, GainLifeEach(EachPlayer, Fixed(2)), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			for _, player := range []Player{a, b} {
				if player.Life() != 20+scenario.amount || g.PlayerLifeGainedThisTurn(player.PlayerID()) != scenario.amount {
					t.Errorf("life or event outcome for %s: %d/%d", player.Name(), player.Life(), g.PlayerLifeGainedThisTurn(player.PlayerID()))
				}
			}
		})
	}
}

func TestGainLifeEachDeclaration(t *testing.T) {
	valid := NewInstant("Valid", "", Spell(GainLifeEach(EachPlayer, Fixed(2))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{GainLifeEach(nil, Fixed(1)), GainLifeEach(EachPlayer, nil), GainLifeEach(OpponentsOf(FromPlayer(BindPlayer("missing"))), Fixed(1))} {
		card := NewInstant("Invalid", "", Spell(effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("expected declaration error, got %v", err)
		}
	}
}
