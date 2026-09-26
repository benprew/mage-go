package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastCasterEventIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	caster := BindPlayer("caster")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByOpponent().WithCaster(caster), GainLifeFor(FromPlayer(caster), Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Spell", "")
	spell.SetOwner(b.PlayerID())
	g.pushStack(&StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: a.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("own cast triggered: stack=%d", g.StackSize())
	}
	g.pushStack(&StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), PlayerID: b.PlayerID()})
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("opponent cast trigger: stack=%d, want 3", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("clone changed original caster")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 21 {
		t.Fatalf("life=(%d,%d), want (20,21)", a.Life(), b.Life())
	}
}

func TestSpellCastCasterEventDeclaration(t *testing.T) {
	caster := BindPlayer("caster")
	valid := NewArtifact("Valid", "", WithAbility(WhenEvent(
		SpellCast().ByOpponent().WithCaster(caster), GainLifeFor(FromPlayer(caster), Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		card Card
	}{
		{"missing binding", NewArtifact("Missing", "", WithAbility(WhenEvent(SpellCast().WithCaster(nil), GainLifeFor(You, Fixed(1)))))},
		{"unbound read", NewArtifact("Unbound", "", WithAbility(WhenEvent(SpellCast(), GainLifeFor(FromPlayer(caster), Fixed(1)))))},
		{"wrong source zone", NewInstant("Wrong", "", WithAbility(WhenEvent(SpellCast().WithCaster(caster), GainLifeFor(FromPlayer(caster), Fixed(1)))))},
		{"duplicate capture", NewArtifact("Duplicate", "", WithAbility(WhenEvent(SpellCast().WithCaster(caster).WithCaster(BindPlayer("second")), GainLifeFor(You, Fixed(1)))))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateCardDefinition(tc.card.Name(), tc.card); err == nil {
				t.Fatal("invalid declaration accepted")
			}
		})
	}
	node := InspectDefinition(SpellCast().ByOpponent().WithCaster(caster))
	if node.Role != "event-pattern" || !strings.Contains(node.Text, "opponent") {
		t.Fatalf("inspection=%+v", node)
	}
}

func TestSpellCastCasterEventCopyAndSeparateCasts(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	caster := BindPlayer("caster")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithCaster(caster), GainLifeFor(FromPlayer(caster), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	for _, player := range []*BasePlayer{a, b} {
		spell := NewInstant("Spell", "")
		spell.SetOwner(player.PlayerID())
		player.AddToHand(spell)
		if err := g.CastSpellByName(player.PlayerID(), spell.Name(), nil); err != nil {
			t.Fatal(err)
		}
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatalf("stack=%d, want 4", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 21 {
		t.Fatalf("life=(%d,%d), want (22,21)", a.Life(), b.Life())
	}
}

func TestSpellCastCasterEventPhasingAndFailedCast(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	caster := BindPlayer("caster")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByOpponent().WithCaster(caster), GainLifeFor(FromPlayer(caster), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PhaseOut(watcher.ID())
	spell := NewInstant("Unpaid", "{U}")
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByName(b.PlayerID(), spell.Name(), nil); err == nil {
		t.Fatal("cast without payment succeeded")
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || b.Life() != 20 {
		t.Fatal("phased source or failed cast created a trigger")
	}
}

func TestSpellCastCasterEventColorComposition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	caster := BindPlayer("caster")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByOpponent().OfColor(Blue).WithCaster(caster),
		GainLifeFor(FromPlayer(caster), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	for _, tc := range []struct {
		controller uuid.UUID
		cost       string
	}{
		{b.PlayerID(), "{R}"},
		{a.PlayerID(), "{U}"},
		{b.PlayerID(), "{U}"},
	} {
		spell := NewInstant("Spell", tc.cost)
		spell.SetOwner(tc.controller)
		object := &StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: tc.controller}
		g.pushStack(object)
		g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: object.ID, PlayerID: tc.controller})
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatalf("stack=%d, want 4", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 21 {
		t.Fatalf("life=(%d,%d), want (20,21)", a.Life(), b.Life())
	}
}
