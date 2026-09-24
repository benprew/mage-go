package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

func TestComposableOpponentLifeSimultaneous(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	observed := 0
	watcher := NewTriggered(core.EvtLifeLost, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *core.GameEvent, game GameReader, _, _ uuid.UUID) bool {
		observed++
		if game.GetPlayer(a.PlayerID()).Life() != 0 || game.GetPlayer(b.PlayerID()).Life() != 0 || evt.Amount != 20 {
			t.Error("life-loss event observed an incomplete action")
		}
		return true
	})
	g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(watcher)), a.PlayerID())
	if err := ApplyEffect(g, LoseLifeEach(EachPlayer, ValueAtResolution(PlayerLifeValue())), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if observed != 2 || g.stack.Size() != 2 {
		t.Fatalf("events = %d, triggers = %d", observed, g.stack.Size())
	}
}

func TestComposableOpponentLife(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		controller, opponent := a, b
		if reverse {
			controller, opponent = b, a
		}
		source := NewCreature("Source", "", 1, 1)
		g.PutOnBattlefield(source, opponent.PlayerID())
		effect := Sequence(LoseLifeEach(OpponentsOf(You), Fixed(3)), GainLifeFor(You, Fixed(3)))
		clone := g.Clone()
		if err := ApplyEffect(clone, effect, source.ID(), controller.PlayerID(), []uuid.UUID{controller.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if clone.GetPlayer(controller.PlayerID()).Life() != 23 || clone.GetPlayer(opponent.PlayerID()).Life() != 17 {
			t.Fatal("life action used the active player, source controller, or positional target")
		}
		if a.Life() != 20 || b.Life() != 20 {
			t.Fatal("clone changed original life")
		}
	}
}

func TestComposableOpponentLifeAmountAndOrder(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := NewCreature("Amount", "{3}", 1, 1)
	g.PutOnBattlefield(target, b.PlayerID())
	effect := Sequence(LoseLifeEach(OpponentsOf(You), ValueAtResolution(TargetPermanentManaValue())), GainLifeFor(You, ValueAtResolution(PlayerLifeValue())))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 17 || a.Life() != 40 {
		t.Fatalf("life = %d/%d", a.Life(), b.Life())
	}
	if err := ApplyEffect(g, Sequence(LoseLifeEach(EachPlayer, Fixed(3)), GainLifeFor(You, ValueAtResolution(PlayerLifeValue()))), uuid.Nil, a.PlayerID(), []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 14 || a.Life() != 74 {
		t.Fatalf("loss must precede gain: life = %d/%d", a.Life(), b.Life())
	}
}

func TestComposableOpponentLifeAbsenceAndNonpositive(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for _, effect := range []ResolvingEffect{
		LoseLifeEach(OpponentsOf(ControllerOf(This)), Fixed(3)),
		LoseLifeEach(OpponentsOf(You), Fixed(0)),
		LoseLifeEach(OpponentsOf(You), Fixed(-3)),
	} {
		if err := ApplyEffect(g, effect, uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("absent or nonpositive instruction changed life")
	}
}

func TestComposableOpponentLifeReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Lich", "", 1, 1)
	g.PutOnBattlefield(source, a.PlayerID())
	g.SetLichActive(a.PlayerID(), source.ID())
	for range 3 {
		a.AddToLibrary(NewLand("Card"))
	}
	if err := ApplyEffect(g, Sequence(LoseLifeEach(OpponentsOf(You), Fixed(3)), GainLifeFor(You, Fixed(3))), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 17 || len(a.Hand()) != 3 {
		t.Fatal("life gain replacement did not preserve opponent life loss")
	}
}

func TestComposableOpponentLifeValidation(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		LoseLifeEach(nil, Fixed(3)),
		LoseLifeEach(OpponentsOf(nil), Fixed(3)),
		LoseLifeEach(OpponentsOf(You), nil),
	} {
		card := NewSorcery("Invalid opponents", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "abilities[0].effects[0].") {
			t.Fatalf("validation = %v", err)
		}
	}
	info := InspectDefinition(LoseLifeEach(OpponentsOf(You), Fixed(3)))
	if info.Properties.Outcome != OutcomeDetriment || !info.Properties.Mass || len(info.Children) != 2 {
		t.Fatalf("inspection = %+v", info)
	}
}
