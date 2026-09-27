package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestBecomesTargetedCapturesEventController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	actor := BindPlayer("actor")
	source := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		BecomesTargeted(This).ByOpponent().WithController(actor),
		GainLifeFor(FromPlayer(actor), Fixed(1)))))
	g.PutOnBattlefield(source, a.PlayerID())
	other := NewCreature("Other", "", 2, 2)
	g.PutOnBattlefield(other, a.PlayerID())
	for _, event := range []GameEvent{
		{Type: EvtBecomesTarget, TargetID: other.ID(), PlayerID: b.PlayerID()},
		{Type: EvtBecomesTarget, TargetID: source.ID(), PlayerID: a.PlayerID()},
	} {
		g.FireEvent(event)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("unrelated or own targeting matched")
	}
	for _, ability := range []bool{false, true} {
		g.FireEvent(GameEvent{Type: EvtBecomesTarget, TargetID: source.ID(), SourceID: uuid.New(), PlayerID: b.PlayerID(), Flag: ability})
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("targeting triggers = %d, want 2", g.StackSize())
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.ResolveStack()
	if g.GetPlayer(b.PlayerID()).Life() != 22 || g.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("event controller changed after source return")
	}
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("clone lost targeting capture")
	}
}

func TestBecomesTargetedUnavailableSource(t *testing.T) {
	for _, change := range []string{"absent", "phased"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(BecomesTargeted(This), GainLifeFor(You, Fixed(1)))))
			g.PutOnBattlefield(source, a.PlayerID())
			if change == "absent" {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			} else {
				g.PhaseOut(source.ID())
			}
			g.FireEvent(GameEvent{Type: EvtBecomesTarget, TargetID: source.ID(), PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("unavailable source triggered")
			}
		})
	}
}

func TestBecomesTargetedByActivatedAbility(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	actor := BindPlayer("actor")
	source := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		BecomesTargeted(This).ByOpponent().WithController(actor), GainLifeFor(FromPlayer(actor), Fixed(1)))))
	g.PutOnBattlefield(source, a.PlayerID())
	target := DeclarePermanentTarget("target", IsCreature)
	activator := NewArtifact("Activator", "", WithAbility(Activate(Costs(), TapSubject(target), WithTargets(target))))
	g.PutOnBattlefield(activator, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), activator.ID(), 0, []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack size = %d, want 2", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[1], a.PlayerID(), false)
	g.ResolveStack()
	if b.Life() != 22 || a.Life() != 20 {
		t.Fatal("activation or copied trigger lost its event controller")
	}
}

func TestBecomesTargetedDeclaration(t *testing.T) {
	actor := BindPlayer("actor")
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
		BecomesTargeted(This).ByOpponent().WithController(actor), GainLifeFor(FromPlayer(actor), Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if info := InspectDefinition(BecomesTargeted(This)); info.Role != "event-pattern" {
		t.Fatalf("inspection role = %q", info.Role)
	}
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"nil", NewCreature("Nil", "", 1, 1, WithAbility(WhenEvent(BecomesTargeted(This).WithController(nil), GainLifeFor(You, Fixed(1))))), "controller binding"},
		{"duplicate", NewCreature("Duplicate", "", 1, 1, WithAbility(WhenEvent(BecomesTargeted(This).WithController(actor).WithController(actor), GainLifeFor(FromPlayer(actor), Fixed(1))))), "one controller"},
		{"unbound", NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(BecomesTargeted(This), GainLifeFor(FromPlayer(actor), Fixed(1))))), "producer"},
		{"spell", NewInstant("Spell", "", WithAbility(WhenEvent(BecomesTargeted(This), GainLifeFor(You, Fixed(1))))), "battlefield"},
	} {
		err := ValidateCardDefinition(tc.card.Name(), tc.card)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: validation = %v, want %q", tc.name, err, tc.want)
		}
	}
}
