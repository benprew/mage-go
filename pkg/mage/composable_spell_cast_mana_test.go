package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastManaSpentCapturedBeforeSpellLeaves(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spent := BindNumber("spent")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByOpponent().WithManaSpent(spent), GainLifeFor(You, FromNumber(spent)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Spell", "", Spell(DrawFor(You, Fixed(0))))
	spell.SetOwner(b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: b.PlayerID(),
		CastContext: &CastContext{ColorsSpent: map[Color]int{Red: 2, Colorless: 3}}}
	g.pushStack(obj)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	obj.CastContext.ColorsSpent[Red] = 10
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	g.stack.RemoveByID(obj.ID)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 25 {
			t.Fatalf("captured mana = %d", got)
		}
	}
}

func TestSpellCastManaSpentZeroAndPhasedSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spent := BindNumber("spent")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithManaSpent(spent), GainLifeFor(You, FromNumber(spent)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	permanent := g.FindPermanent(watcher.ID())
	spell := NewInstant("Free", "", Spell(DrawFor(You, Fixed(0))))
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: b.PlayerID(), CastContext: &CastContext{}}
	g.pushStack(obj)
	permanent.PhasedOut = true
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("phased source triggered: stack = %d", g.StackSize())
	}
	permanent.PhasedOut = false
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("free cast trigger: stack = %d", g.StackSize())
	}
	g.stack.RemoveByID(obj.ID)
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatalf("zero mana capture = %d", a.Life())
	}
}

func TestSpellCastManaSpentDeclaration(t *testing.T) {
	spent := BindNumber("spent")
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"valid", NewArtifact("Valid", "", WithAbility(WhenEvent(SpellCast().WithManaSpent(spent), GainLifeFor(You, FromNumber(spent))))), ""},
		{"nil", NewArtifact("Nil", "", WithAbility(WhenEvent(SpellCast().WithManaSpent(nil), GainLifeFor(You, Fixed(1))))), "binding"},
		{"unnamed", NewArtifact("Unnamed", "", WithAbility(WhenEvent(SpellCast().WithManaSpent(BindNumber("")), GainLifeFor(You, Fixed(1))))), "name"},
		{"duplicate", NewArtifact("Duplicate", "", WithAbility(WhenEvent(SpellCast().WithManaSpent(spent).WithManaSpent(BindNumber("other")), GainLifeFor(You, Fixed(1))))), "one"},
		{"unbound", NewArtifact("Unbound", "", WithAbility(WhenEvent(SpellCast(), GainLifeFor(You, FromNumber(spent))))), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCardDefinition(tc.card.Name(), tc.card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	if got := InspectDefinition(SpellCast().WithManaSpent(spent)); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatalf("inspection = %#v", got)
	}
}
