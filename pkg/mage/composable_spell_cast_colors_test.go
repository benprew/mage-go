package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastDistinctColorsCapturedAtEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	colors := BindNumber("colors")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().OfType(TypeInstant).ByOpponent().WithColorsSpent(colors),
		GainLifeFor(You, FromNumber(colors)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Spell", "", Spell(DrawFor(You, Fixed(0))))
	spell.SetOwner(b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: b.PlayerID(),
		CastContext: &CastContext{ColorsSpent: map[Color]int{Red: 2, Blue: 1, Colorless: 3, Green: 0}}}
	g.pushStack(obj)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	obj.CastContext.ColorsSpent[Green] = 5
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutTriggersOnStack()
	g.stack.RemoveByID(obj.ID)
	clone := g.Clone()
	clone.ResolveStack()
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 20 {
		t.Fatalf("clone changed original life = %d", got)
	}
	g.ResolveStack()
	for _, game := range []*Game{g, clone} {
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 22 {
			t.Fatalf("colors spent = %d", got)
		}
	}
}

func TestSpellCastDistinctColorsZeroAndSourcePhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	colors := BindNumber("colors")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithColorsSpent(colors), GainLifeFor(You, MaxValue(Fixed(1), FromNumber(colors))))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	permanent := g.FindPermanent(watcher.ID())
	spell := NewInstant("Free", "", Spell(DrawFor(You, Fixed(0))))
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: b.PlayerID(), CastContext: &CastContext{}}
	g.pushStack(obj)
	permanent.PhasedOut = true
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 1 {
		t.Fatalf("phased source stack = %d", got)
	}
	permanent.PhasedOut = false
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.stack.RemoveByID(obj.ID)
	g.ResolveStack()
	if got := a.Life(); got != 21 {
		t.Fatalf("zero-color cast life = %d", got)
	}
}

func TestSpellCastDistinctColorsDeclaration(t *testing.T) {
	colors := BindNumber("colors")
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"valid", NewArtifact("Valid", "", WithAbility(WhenEvent(SpellCast().WithColorsSpent(colors), GainLifeFor(You, FromNumber(colors))))), ""},
		{"nil", NewArtifact("Nil", "", WithAbility(WhenEvent(SpellCast().WithColorsSpent(nil), GainLifeFor(You, Fixed(1))))), "binding"},
		{"unnamed", NewArtifact("Unnamed", "", WithAbility(WhenEvent(SpellCast().WithColorsSpent(BindNumber("")), GainLifeFor(You, Fixed(1))))), "name"},
		{"duplicate", NewArtifact("Duplicate", "", WithAbility(WhenEvent(SpellCast().WithColorsSpent(colors).WithColorsSpent(BindNumber("other")), GainLifeFor(You, Fixed(1))))), "one"},
		{"unbound", NewArtifact("Unbound", "", WithAbility(WhenEvent(SpellCast(), GainLifeFor(You, FromNumber(colors))))), "producer"},
		{"wrong zone", NewInstant("Wrong", "", WithAbility(WhenEvent(SpellCast().WithColorsSpent(colors), GainLifeFor(You, FromNumber(colors))))), "battlefield"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCardDefinition(tc.card.Name(), tc.card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	if got := InspectDefinition(SpellCast().WithColorsSpent(colors)); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatalf("inspection = %#v", got)
	}
}

func TestDistinctColorsSpentExcludesColorless(t *testing.T) {
	ctx := &CastContext{ColorsSpent: map[Color]int{
		White: 1, Blue: 2, Black: 1, Red: 1, Green: 1, Colorless: 4,
	}}
	if got := ctx.DistinctColorsSpent(); got != 5 {
		t.Fatalf("distinct colors = %d", got)
	}
	if got := (*CastContext)(nil).DistinctColorsSpent(); got != 0 {
		t.Fatalf("nil cast context colors = %d", got)
	}
}
