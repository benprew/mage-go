package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceTargetedBySpellEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		BecomesTargetOfSpell(This), Sacrifice(This))))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	other := NewCreature("Other", "", 2, 2)
	g.PutOnBattlefield(other, b.PlayerID())
	for _, evt := range []GameEvent{
		{Type: EvtBecomesTarget, TargetID: other.ID()},
		{Type: EvtBecomesTarget, TargetID: card.ID(), Flag: true},
	} {
		g.FireEvent(evt)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("nonmatching target event triggered")
	}
	g.FireEvent(GameEvent{Type: EvtBecomesTarget, TargetID: card.ID(), SourceID: uuid.New(), PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("spell target triggers = %d", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if g.FindPermanent(card.ID()) == nil || clone.FindPermanent(card.ID()) != nil {
		t.Fatal("clone did not isolate trigger resolution")
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("source did not sacrifice")
	}
}

func TestSourceTargetedBySpellCast(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		BecomesTargetOfSpell(This), Sacrifice(This))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	target := DeclarePermanentTarget("target", IsCreature)
	spell := NewInstant("Tap spell", "", WithAbility(Spell(TapSubject(target), WithTargets(target))))
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByName(b.PlayerID(), spell.Name(), []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("cast stack size = %d", g.StackSize())
	}
	g.ResolveStack()
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("target trigger did not sacrifice source")
	}
}

func TestSourceTargetedBySpellDepartureAndPhasing(t *testing.T) {
	for _, tc := range []string{"phased", "absent", "left", "returned"} {
		t.Run(tc, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
				BecomesTargetOfSpell(This), GainLifeFor(You, Fixed(1)))))
			card.SetOwner(a.PlayerID())
			g.PutOnBattlefield(card, a.PlayerID())
			switch tc {
			case "phased":
				g.PhaseOut(card.ID())
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
			}
			g.FireEvent(GameEvent{Type: EvtBecomesTarget, TargetID: card.ID()})
			if tc == "phased" || tc == "absent" {
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("unavailable source triggered")
				}
				return
			}
			g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
			if tc == "returned" {
				g.PutOnBattlefield(card, b.PlayerID())
			}
			g.PutTriggersOnStack()
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 20 {
				t.Fatal("pending trigger lost its controller")
			}
		})
	}
}

func TestSourceTargetedBySpellDeclaration(t *testing.T) {
	card := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(BecomesTargetOfSpell(This), Sacrifice(This))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if info := InspectDefinition(BecomesTargetOfSpell(This)); info.Role != "event-pattern" {
		t.Fatalf("inspection role = %q", info.Role)
	}
	bad := NewInstant("Invalid", "", WithAbility(WhenEvent(BecomesTargetOfSpell(This), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("registration error = %v", err)
	}
}
