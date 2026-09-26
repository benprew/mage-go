package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastTypeEventIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	caster := BindPlayer("artifact caster")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().OfType(TypeArtifact).ByOpponent().WithCaster(caster),
		GainLifeFor(FromPlayer(caster), Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Cast artifact", "")
	artifact.SetOwner(b.PlayerID())
	instant := NewInstant("Cast instant", "")
	instant.SetOwner(b.PlayerID())
	for _, card := range []Card{instant, artifact} {
		object := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID()}
		g.pushStack(object)
		g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: object.ID, PlayerID: b.PlayerID()})
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("stack=%d, want two spells and one trigger", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("clone changed original caster")
	}
	g.ResolveStack()
	if b.Life() != 21 {
		t.Fatalf("caster life=%d, want 21", b.Life())
	}
}

func TestSpellCastTypeEventUsesStackObject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().OfType(TypeArtifact), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Shared card", "")
	artifact.SetOwner(b.PlayerID())
	old := &StackObject{ID: uuid.New(), Card: artifact, SourceID: artifact.ID(), Controller: b.PlayerID()}
	current := &StackObject{ID: uuid.New(), Card: NewInstant("Current form", ""), SourceID: artifact.ID(), Controller: b.PlayerID()}
	g.pushStack(old)
	g.pushStack(current)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: artifact.ID(), TargetID: current.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, type matched another object with the same source card", g.StackSize())
	}
}

func TestSpellCastTypeEventDeclaration(t *testing.T) {
	pattern := SpellCast().OfType(TypeArtifact)
	if got := InspectDefinition(pattern); got.Role != "event-pattern" || !strings.Contains(got.Text, "artifact") {
		t.Fatalf("inspection=%+v", got)
	}
	for _, invalid := range []EventPattern{
		SpellCast().OfType(TypeLand),
		SpellCast().OfType(CardType(-1)),
		SpellCast().OfType(TypeArtifact).OfType(TypeCreature),
	} {
		card := NewArtifact("Invalid", "", WithAbility(WhenEvent(invalid, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "type") {
			t.Fatalf("validation=%v", err)
		}
	}
}

func TestSpellCastTypeEventLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name   string
		phase  bool
		remove bool
		want   int
	}{
		{name: "present", want: 1},
		{name: "phased source", phase: true},
		{name: "absent source", remove: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
				SpellCast().OfType(TypeArtifact), GainLifeFor(You, Fixed(1)))))
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phase {
				g.PhaseOut(watcher.ID())
			}
			if tc.remove {
				g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
			}
			spell := NewArtifact("Spell", "")
			spell.SetOwner(b.PlayerID())
			b.AddToHand(spell)
			g.turns.SetActivePlayerIndex(1)
			g.turns.SetStep(PrecombatMain)
			if err := g.CastSpellByName(b.PlayerID(), spell.Name(), nil); err != nil {
				t.Fatal(err)
			}
			g.PutTriggersOnStack()
			if got := g.StackSize() - 1; got != tc.want {
				t.Fatalf("trigger count=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestSpellCastTypeEventFailedCast(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().OfType(TypeArtifact), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewArtifact("Unpaid", "{2}")
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByName(b.PlayerID(), spell.Name(), nil); err == nil {
		t.Fatal("cast without mana succeeded")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || a.Life() != 20 {
		t.Fatal("failed cast created a trigger")
	}
}
