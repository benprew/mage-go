package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestSpellSingleTargetOccurrences(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	one, two := uuid.New(), uuid.New()
	filter := SpellWithSingleTarget()
	for _, tc := range []struct {
		name    string
		targets []uuid.UUID
		ability bool
		want    bool
	}{
		{"none", nil, false, false},
		{"one", []uuid.UUID{one}, false, true},
		{"two", []uuid.UUID{one, two}, false, false},
		{"same twice", []uuid.UUID{one, one}, false, false},
		{"ability", []uuid.UUID{one}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj := &StackObject{ID: uuid.New(), Card: NewInstant("Spell", ""), Targets: tc.targets, IsAbility: tc.ability}
			if got := filter.matches(g, a.PlayerID(), nil, obj); got != tc.want {
				t.Fatalf("match = %v, want %v", got, tc.want)
			}
		})
	}
	if filter.matches(g, a.PlayerID(), nil, nil) || filter.matches(g, a.PlayerID(), nil, &StackObject{Targets: []uuid.UUID{one}}) {
		t.Fatal("absent spell matched")
	}
	target := DeclareSpellTarget("spell").WithFilter(filter)
	card := NewInstant("Counter", "", Spell(Counter(target), WithTargets(target)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestSpellSingleTargetSavedAndCurrentLists(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	subject := g.PutOnBattlefield(NewCreature("Subject", "", 1, 4), a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: uuid.New(), Card: NewInstant("Spell", ""), Controller: a.PlayerID(), Targets: []uuid.UUID{subject.ID()}}
	g.pushStack(obj)
	filter := SpellWithSingleTarget()
	g.PhaseOut(subject.ID())
	if !filter.matches(g, a.PlayerID(), nil, obj) {
		t.Fatal("phasing removed a target occurrence")
	}
	g.PhaseIn([]uuid.UUID{subject.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(subject.ID()))
	if !filter.matches(g, a.PlayerID(), nil, obj) {
		t.Fatal("departure removed a target occurrence")
	}
	clone := g.Clone()
	copiedSpell := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	if copiedSpell == nil || !filter.matches(g, a.PlayerID(), nil, copiedSpell) {
		t.Fatal("copy lost its target")
	}
	obj.Targets = append(obj.Targets, uuid.New())
	if filter.matches(g, a.PlayerID(), nil, obj) || !filter.matches(g, a.PlayerID(), nil, copiedSpell) || !filter.matches(clone, a.PlayerID(), nil, clone.Stack().FindByID(obj.ID)) {
		t.Fatal("target lists are not independent")
	}
}
