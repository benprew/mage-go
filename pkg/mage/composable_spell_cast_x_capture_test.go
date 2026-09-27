package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastAnnouncedXCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	x := BindNumber("cast X")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithXInManaCost().WithAnnouncedX(x), GainLifeFor(You, FromNumber(x)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	for _, value := range []int{0, 3, 5} {
		spell := NewInstant("Spell", "{X}{R}")
		obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, XValue: value, Controller: b.PlayerID()}
		g.stack.Push(obj)
		g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
		obj.XValue = 99
		g.stack.RemoveByID(obj.ID)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("stack = %d, want 3", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 28 {
			t.Fatalf("captured X: life = %d, want 28", got)
		}
	}
}

func TestSpellCastAnnouncedXDeclaration(t *testing.T) {
	x := BindNumber("cast X")
	pattern := SpellCast().WithAnnouncedX(x)
	if got := InspectDefinition(pattern); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatalf("inspection = %#v", got)
	}
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"valid", NewArtifact("Valid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, FromNumber(x))))), ""},
		{"nil", NewArtifact("Nil", "", WithAbility(WhenEvent(SpellCast().WithAnnouncedX(nil), GainLifeFor(You, Fixed(1))))), "binding"},
		{"unnamed", NewArtifact("Unnamed", "", WithAbility(WhenEvent(SpellCast().WithAnnouncedX(BindNumber("")), GainLifeFor(You, Fixed(1))))), "name"},
		{"duplicate", NewArtifact("Duplicate", "", WithAbility(WhenEvent(pattern.WithAnnouncedX(BindNumber("other")), GainLifeFor(You, Fixed(1))))), "one"},
		{"unbound", NewArtifact("Unbound", "", WithAbility(WhenEvent(SpellCast(), GainLifeFor(You, FromNumber(x))))), "producer"},
		{"wrong zone", NewInstant("Wrong", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, FromNumber(x))))), "battlefield"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCardDefinition(tc.card.Name(), tc.card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

func TestSpellCastAnnouncedXSourceLifecycleAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	x := BindNumber("cast X")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByOpponent().WithAnnouncedX(x), GainLifeFor(You, FromNumber(x)))))
	permanent := g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Spell", "{X}{R}")
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, XValue: 4, Controller: b.PlayerID()}
	g.stack.Push(obj)
	permanent.PhasedOut = true
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	permanent.PhasedOut = false
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d, want spell and one trigger", g.StackSize())
	}
	g.RemoveFromBattlefield(permanent)
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.stack.RemoveByID(obj.ID)
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 24 {
			t.Fatalf("original trigger life = %d, want 24", got)
		}
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 24 {
			t.Fatalf("copied trigger life = %d, want 24", got)
		}
	}
}
