package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSelfEntryColorsSpentFromCastAndTriggerIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	colors := BindNumber("entry colors")
	creature := NewCreature("Converge entrant", "", 2, 2, WithAbility(WhenEvent(
		Enters(This).WithColorsSpent(colors), GainLifeFor(You, FromNumber(colors)))))
	creature.SetOwner(a.PlayerID())
	a.AddToHand(creature)
	if err := g.CastSpellByName(a.PlayerID(), creature.Name(), nil); err != nil {
		t.Fatal(err)
	}
	obj := g.stack.Pop()
	obj.CastContext.ColorsSpent = map[core.Color]int{core.Red: 2, core.Blue: 1, core.Colorless: 3, core.Green: 0}
	g.ResolveStackObject(obj)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	clone := g.Clone()
	clone.PhaseOut(creature.ID())
	g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
	g.PutOnBattlefield(creature, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatalf("original life = %d, %d", a.Life(), b.Life())
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("clone lost trigger capture")
	}
}

func TestSelfEntryColorsSpentWithoutCastAndDeclaration(t *testing.T) {
	colors := BindNumber("entry colors")
	pattern := Enters(This).WithColorsSpent(colors)
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, FromNumber(colors)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if node := InspectDefinition(pattern); node.Role != "event-pattern" || len(node.Children) != 1 {
		t.Fatalf("inspection = %+v", node)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(valid, a.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatalf("uncast entry gained %d life", a.Life()-20)
	}
	for _, tc := range []struct {
		name    string
		pattern EventPattern
		want    string
	}{
		{"nil", Enters(This).WithColorsSpent(nil), "binding"},
		{"unnamed", Enters(This).WithColorsSpent(BindNumber("")), "name"},
		{"repeated", Enters(This).WithColorsSpent(colors).WithColorsSpent(colors), "one"},
		{"wrong event", Dies(This).WithColorsSpent(colors), "entry"},
	} {
		card := NewCreature(tc.name, "", 1, 1, WithAbility(WhenEvent(tc.pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s validation = %v", tc.name, err)
		}
	}
	unbound := NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(Enters(This), GainLifeFor(You, FromNumber(colors)))))
	if err := ValidateCardDefinition(unbound.Name(), unbound); err == nil {
		t.Fatal("accepted unbound capture")
	}
}

func TestSelfEntryColorsSpentPaidCastAndRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	colors := BindNumber("entry colors")
	creature := NewCreature("Paid entrant", "{2}{R}", 2, 2, WithAbility(WhenEvent(
		Enters(This).WithColorsSpent(colors), GainLifeFor(You, FromNumber(colors)))))
	creature.SetOwner(a.PlayerID())
	a.AddToHand(creature)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByName(a.PlayerID(), creature.Name(), nil); err == nil {
		t.Fatal("cast succeeded without generic mana")
	}
	if g.StackSize() != 0 || a.Life() != 20 {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(core.Blue, 1)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByName(a.PlayerID(), creature.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("life after paid entry = %d", a.Life())
	}
}
