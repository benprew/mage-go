package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetEventChooser(t *testing.T) {
	g, a, b := newSpellCopyGame()
	player := BindPlayer("upkeep")
	target := DeclarePermanentTarget("chosen", IsCreature, TargetControlledBy(FromPlayer(player)), TargetChosenBy(FromPlayer(player)))
	card := NewArtifact("Watcher", "", WithAbility(WhenEvent(TurnStepBegins(EachUpkeep, player), TapSubject(target), WithTargets(target))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 3), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 3), b.PlayerID())
	a.chooseQueue = [][]uuid.UUID{{first.ID()}}
	b.chooseQueue = [][]uuid.UUID{{second.ID()}}
	g.resolution.playerBindings = map[*PlayerBinding]uuid.UUID{player: a.PlayerID()}
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || !slices.Equal(g.StackPeek().Targets, []uuid.UUID{second.ID()}) || len(a.chooseQueue) != 1 || len(b.chooseQueue) != 0 {
		t.Fatal("saved event player did not choose")
	}
	if g.resolution.playerBindings[player] != a.PlayerID() {
		t.Fatal("query changed live binding")
	}
	copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
	if copied == nil || !slices.Equal(copied.Targets, []uuid.UUID{first.ID()}) || len(a.chooseQueue) != 0 {
		t.Fatal("copy controller did not retarget using saved event")
	}
	g.ResolveStack()
	if !g.FindPermanent(first.ID()).Tapped {
		t.Fatal("copy did not resolve")
	}
	g.ResolveStack()
	if !g.FindPermanent(second.ID()).Tapped {
		t.Fatal("original did not resolve")
	}
}

func TestTargetEventChooserInvalid(t *testing.T) {
	for _, ref := range []PlayerRef{nil, FromPlayer(nil), FromPlayer(BindPlayer("absent")), ControllerOf(This)} {
		target := DeclarePermanentTarget("chosen", IsCreature, TargetChosenBy(ref))
		c := NewInstant("Bad", "", Spell(TapSubject(target), WithTargets(target)))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("invalid chooser accepted")
		}
	}
}

func TestTargetEventChooserConflict(t *testing.T) {
	player := BindPlayer("upkeep")
	for _, extra := range []PermanentTargetOption{TargetChosenBy(FromPlayer(player)), TargetChosenByOpponent()} {
		target := DeclarePermanentTarget("chosen", IsCreature, TargetChosenBy(FromPlayer(player)), extra)
		c := NewArtifact("Bad", "", WithAbility(WhenEvent(TurnStepBegins(EachUpkeep, player), TapSubject(target), WithTargets(target))))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("conflicting chooser accepted")
		}
	}
}
