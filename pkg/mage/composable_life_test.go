package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestComposableLifeDynamicMetadata(t *testing.T) {
	effect := GainLifeFor(You, CountBattlefield(SelectController(), IsCreature))
	info := InspectDefinition(effect)
	if info.Properties.LifeGain != 0 || !strings.Contains(effect.Text(), "creature") {
		t.Fatalf("dynamic life metadata = %+v, text = %q", info, effect.Text())
	}
	if got := GainLifeFor(You, Fixed(3)).Properties().LifeGain; got != 3 {
		t.Fatalf("fixed life gain = %d", got)
	}
}

func TestComposableLifeValueContext(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	creature := NewCreature("Value source", "{4}", 4, 4)
	g.PutOnBattlefield(creature, b.PlayerID())
	effect := GainLifeFor(You, TargetPermanentManaValue())
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), []uuid.UUID{creature.ID()}); err != nil {
		t.Fatal(err)
	}
	if got := a.Life(); got != 24 {
		t.Fatalf("life = %d, want 24", got)
	}
}

func TestComposableLifeNonpositive(t *testing.T) {
	for _, amount := range []int{0, -2} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WheneverYouGainLifeTrigger(LoseLifeFor(You, Fixed(1)), false)))
		g.PutOnBattlefield(watcher, a.PlayerID())
		if err := ApplyEffect(g, Sequence(GainLifeFor(You, Fixed(amount)), LoseLifeFor(You, Fixed(amount))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.PutTriggersOnStack()
		if a.Life() != 20 || g.stack.Size() != 0 {
			t.Fatalf("amount %d: life = %d, stack = %d", amount, a.Life(), g.stack.Size())
		}
	}
}

func TestComposableLifeControllerAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1)
	g.PutOnBattlefield(source, b.PlayerID())
	effect := Sequence(LoseLifeFor(You, HalfRoundUp(PlayerLifeValue())), GainLifeFor(You, Fixed(2)))
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		if err := ApplyEffect(game, effect, source.ID(), a.PlayerID(), []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if game.GetPlayer(a.PlayerID()).Life() != 12 || game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatal("life instruction used the source controller or positional target")
		}
	}
	if a.Life() != 12 {
		t.Fatal("clone changed original life")
	}
}

func TestComposableLifeAbsentAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for _, effect := range []ResolvingEffect{GainLifeFor(ControllerOf(This), Fixed(3)), LoseLifeFor(ControllerOf(This), Fixed(3))} {
		if err := ApplyEffect(g, effect, uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("absent recipient fell back to controller")
	}
	for _, effect := range []ResolvingEffect{GainLifeFor(nil, Fixed(1)), GainLifeFor(You, nil), LoseLifeFor(nil, Fixed(1)), LoseLifeFor(You, nil)} {
		card := NewSorcery("Invalid life", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition("Invalid life", card); err == nil || !strings.Contains(err.Error(), "abilities[0].effects[0].") {
			t.Fatalf("validation = %v", err)
		}
	}
}

func TestComposableLifeGainReplacement(t *testing.T) {
	for _, lich := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := NewCreature("Watcher", "", 1, 1, WithAbility(WheneverYouGainLifeTrigger(LoseLifeFor(You, Fixed(1)), false)))
		g.PutOnBattlefield(source, a.PlayerID())
		if lich {
			g.SetLichActive(a.PlayerID(), source.ID())
			a.AddToLibrary(NewLand("One"))
			a.AddToLibrary(NewLand("Two"))
		}
		if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.PutTriggersOnStack()
		if lich {
			if a.Life() != 20 || len(a.Hand()) != 2 || g.stack.Size() != 0 {
				t.Fatal("replaced life gain must draw without a life-gain trigger")
			}
		} else {
			if a.Life() != 22 || g.stack.Size() != 1 {
				t.Fatal("life gain must emit one trigger")
			}
			g.ResolveStackObject(g.stack.Pop())
			if a.Life() != 21 {
				t.Fatal("triggered life loss did not resolve")
			}
		}
	}
}
