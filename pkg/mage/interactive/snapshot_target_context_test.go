package interactive

import (
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAvailableActionsManaValueTargets(t *testing.T) {
	for _, includeLabels := range []bool{false, true} {
		a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		a.ManaPool().Add(core.Red, 4)
		zero := g.PutOnBattlefield(mage.NewArtifact("Zero", "{0}"), b.PlayerID())
		three := g.PutOnBattlefield(mage.NewArtifact("Three", "{3}"), b.PlayerID())
		g.PutOnBattlefield(mage.NewArtifact("Too expensive", "{9}"), b.PlayerID())
		spell := mage.NewSorcery("Removal", "{X}{R}", mage.NewTargetedSpell(mage.TargetArtifactWithManaValueX(), mage.DestroyTarget()))
		a.AddToHand(spell)
		g.SetXValue(9)

		found := false
		for _, action := range getAvailableActions(g, a.PlayerID(), includeLabels) {
			if action.CardID != spell.ID() {
				continue
			}
			found = true
			if !slices.Equal(action.ValidTargets, []uuid.UUID{zero.ID(), three.ID()}) {
				t.Fatalf("labels %t: targets = %v, want affordable artifacts", includeLabels, action.ValidTargets)
			}
			wantOptions := []mage.SpellTargetOption{
				{XValue: 0, Candidates: [][]uuid.UUID{{zero.ID()}}},
				{XValue: 3, Candidates: [][]uuid.UUID{{three.ID()}}},
			}
			if !reflect.DeepEqual(action.XTargetOptions, wantOptions) {
				t.Fatalf("labels %t: X target options = %+v, want %+v", includeLabels, action.XTargetOptions, wantOptions)
			}
			if !action.NeedsX || action.MaxXValue != 3 {
				t.Fatalf("wrong X options: %+v", action)
			}
		}
		if !found {
			t.Fatalf("labels %t: missing affordable spell", includeLabels)
		}
		if g.XValue() != 9 || a.ManaPool().TotalMana() != 4 {
			t.Fatal("action query changed announcement state")
		}
	}
}

func TestAvailableActionsRetainXTargetGroups(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	a.ManaPool().Add(core.Red, 8)
	three := g.PutOnBattlefield(mage.NewArtifact("Three", "{3}"), b.PlayerID())
	creature := g.PutOnBattlefield(mage.NewCreature("Creature", "{1}", 1, 1), b.PlayerID())
	spell := mage.NewSorcery("Removal", "{X}{R}", mage.NewMultiTargetSpell([]mage.Target{
		mage.TargetPermanent(mage.IsCardType(core.TypeArtifact), mage.ManaValueIs(mage.EqualTo(mage.HalfXRoundedDown()))),
		mage.TargetCreature(),
	}, mage.DestroyTarget()))
	a.AddToHand(spell)

	for _, action := range GetAvailableActions(g, a.PlayerID()) {
		if action.CardID != spell.ID() {
			continue
		}
		want := []mage.SpellTargetOption{
			{XValue: 6, Candidates: [][]uuid.UUID{{three.ID()}, {creature.ID()}}},
			{XValue: 7, Candidates: [][]uuid.UUID{{three.ID()}, {creature.ID()}}},
		}
		if !reflect.DeepEqual(action.XTargetOptions, want) {
			t.Fatalf("X target options = %+v, want %+v", action.XTargetOptions, want)
		}
		return
	}
	t.Fatal("missing affordable spell")
}

func TestAvailableActionsReportChannelX(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	a.ManaPool().Add(core.Red, 1)
	g.SetChannelActive(a.PlayerID())
	target := g.PutOnBattlefield(mage.NewArtifact("Three", "{3}"), b.PlayerID())
	spell := mage.NewSorcery("Removal", "{X}{R}", mage.NewTargetedSpell(mage.TargetArtifactWithManaValueX(), mage.DestroyTarget()))
	a.AddToHand(spell)

	for _, action := range GetAvailableActions(g, a.PlayerID()) {
		if action.CardID != spell.ID() {
			continue
		}
		want := []mage.SpellTargetOption{{XValue: 3, Candidates: [][]uuid.UUID{{target.ID()}}}}
		if !reflect.DeepEqual(action.XTargetOptions, want) || action.MaxXValue != 3 {
			t.Fatalf("expected an X=3 cast with Channel, got %+v", action)
		}
		return
	}
	t.Fatal("missing cast with Channel")
}
