package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerControlsSavedIntervening(t *testing.T) {
	for _, lost := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		player := BindPlayer("upkeep player")
		trigger := WhenEvent(TurnStepBegins(EachUpkeep, player), GainLifeFor(FromPlayer(player), Fixed(1)), WithInterveningIf(PlayerControls(FromPlayer(player), HasSubType("Plains"))))
		card := NewEnchantment("Watcher", "", WithAbility(trigger))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		source := g.PutOnBattlefield(card, a.PlayerID())
		plains := g.PutOnBattlefield(NewLand("Plains", WithSubTypes("Plains")), b.PlayerID())
		g.resolution.playerBindings = map[*PlayerBinding]uuid.UUID{player: a.PlayerID()}
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatal("event binding was not read at detection", g.StackSize())
		}
		g.BouncePermanentToHand(g.FindPermanent(source.ID()))
		g.SetActivePlayerIndex(0)
		clone := g.Clone()
		if lost {
			g.PhaseOut(plains.ID())
		}
		g.ResolveStack()
		clone.ResolveStack()
		want := 21
		if lost {
			want = 20
		}
		if b.Life() != want || a.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 21 {
			t.Fatal("saved player recheck or clone isolation")
		}
	}
}

func TestOpponentControlsMoreThanYou(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := OpponentControlsMoreThanYou(IsLand)
	ctx := StateContext{ControllerID: a.PlayerID()}
	if c.checkState(g, ctx) {
		t.Fatal("empty counts matched")
	}
	g.PutOnBattlefield(NewLand("Own"), a.PlayerID())
	g.PutOnBattlefield(NewLand("Other"), b.PlayerID())
	if c.checkState(g, ctx) {
		t.Fatal("equal counts matched")
	}
	land := NewLand("Extra")
	land.SetOwner(a.PlayerID())
	extra := g.PutOnBattlefield(land, b.PlayerID())
	g.PutOnBattlefield(NewCreature("Noise", "", 1, 1), a.PlayerID())
	if !c.checkState(g, ctx) || c.checkState(g, StateContext{ControllerID: b.PlayerID()}) || c.checkState(g, StateContext{ControllerID: uuid.New()}) {
		t.Fatal("control or filter comparison")
	}
	clone := g.Clone()
	clone.PhaseOut(extra.ID())
	if c.checkState(clone, ctx) || !c.checkState(g, ctx) {
		t.Fatal("phasing or clone query")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: extra.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	if c.checkState(g, ctx) {
		t.Fatal("stale control count")
	}
	card := NewEnchantment("Count", "", WithAbility(At(YourUpkeep, GainLifeFor(You, Fixed(1)), WithInterveningIf(c))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestPlayerControlsSavedDeclarations(t *testing.T) {
	p := BindPlayer("missing")
	invalid := NewInstant("Unbound", "", WithAbility(Spell(IfState(PlayerControls(FromPlayer(p), IsLand), GainLifeFor(You, Fixed(1))))))
	if ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("unbound player accepted")
	}
}

func TestOpponentControlsMoreIntervening(t *testing.T) {
	for _, lost := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		card := NewEnchantment("Counts", "", WithAbility(At(YourUpkeep, GainLifeFor(You, Fixed(1)), WithInterveningIf(OpponentControlsMoreThanYou(IsLand)))))
		source := g.PutOnBattlefield(card, a.PlayerID())
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatal("false count triggered")
		}
		land := g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatal("true count did not trigger")
		}
		g.BouncePermanentToHand(g.FindPermanent(source.ID()))
		clone := g.Clone()
		if lost {
			g.BouncePermanentToHand(g.FindPermanent(land.ID()))
		}
		g.ResolveStack()
		clone.ResolveStack()
		want := 21
		if lost {
			want = 20
		}
		if a.Life() != want || b.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 21 {
			t.Fatal("count recheck or captured controller")
		}
	}
}
