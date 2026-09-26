package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastColorEvent(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cost    string
		change  Color
		castByB bool
		phase   bool
		want    int
	}{
		{name: "blue", cost: "{U}", want: 1},
		{name: "opponent blue", cost: "{U}", castByB: true, want: 1},
		{name: "multicolor blue", cost: "{U}{R}", want: 1},
		{name: "red", cost: "{R}"},
		{name: "changed red to blue", cost: "{R}", change: Blue, want: 1},
		{name: "changed blue to red", cost: "{U}", change: Red},
		{name: "phased source", cost: "{U}", phase: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
				SpellCast().OfColor(Blue), GainLifeFor(You, Fixed(1)))))
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phase {
				g.PhaseOut(watcher.ID())
			}
			caster := a.PlayerID()
			if tc.castByB {
				caster = b.PlayerID()
			}
			spell := NewInstant("Spell", tc.cost)
			spell.SetOwner(caster)
			g.pushStack(&StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: caster})
			if tc.change != Colorless && !g.ChangeObjectColor(spell.ID(), tc.change) {
				t.Fatal("could not change spell color")
			}
			g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), PlayerID: caster})
			g.PutTriggersOnStack()
			if got := g.StackSize() - 1; got != tc.want {
				t.Fatalf("trigger count=%d, want %d", got, tc.want)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			if got := a.Life(); got != 20+tc.want {
				t.Fatalf("life=%d, want %d", got, 20+tc.want)
			}
		})
	}
}

func TestSpellCastColorEventPendingController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(SpellCast().OfColor(Blue), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Blue Spell", "{U}")
	spell.SetOwner(b.PlayerID())
	g.pushStack(&StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), PlayerID: b.PlayerID()})
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack size=%d, want 2", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("life A=%d B=%d", a.Life(), b.Life())
	}
}

func TestSpellCastColorEventRealCast(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(SpellCast().OfColor(Blue), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Blue Spell", "")
	spell.SetColorOverride([]Color{Blue})
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByName(b.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack size=%d, want 2", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("life=%d, want 21", a.Life())
	}
}

func TestSpellCastColorEventFailedCastDoesNotTrigger(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(SpellCast().OfColor(Blue), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Unpaid", "{U}")
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

func TestSpellCastColorEventUsesCastStackIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(SpellCast().OfColor(Blue), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Recast", "{R}")
	spell.SetOwner(b.PlayerID())
	blue := []Color{Blue}
	oldCopy := &StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID(), ColorOverride: &blue, IsCopy: true}
	cast := &StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID()}
	g.pushStack(oldCopy)
	g.pushStack(cast)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: cast.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack size=%d, want 2; cast red spell matched blue copy", g.StackSize())
	}
}

func TestSpellCastColorEventDeclaration(t *testing.T) {
	pattern := SpellCast().OfColor(Blue)
	if got := InspectDefinition(pattern); got.Role != "event-pattern" || got.Text != "a blue spell is cast" {
		t.Fatalf("inspection=%+v", got)
	}
	for _, invalid := range []EventPattern{
		SpellCast().OfColor(Colorless),
		SpellCast().OfColor(AnyColor),
		SpellCast().OfColor(Blue).OfColor(Red),
	} {
		card := NewArtifact("Invalid", "", WithAbility(WhenEvent(invalid, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "color") {
			t.Fatalf("validation=%v", err)
		}
	}
	spell := NewInstant("Invalid Source", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("validation=%v", err)
	}
}
