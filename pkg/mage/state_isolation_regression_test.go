package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCopyEffectUsesCopiableCharacteristics(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	target := g.PutOnBattlefield(NewCreature("Model", "{2}{G}", 3, 4, WithSubTypes("Elf"), WithAbility(NewActivatedAbility(GainLife(1), ManaCostOf("{0}"), WithOncePerTurn()))), owner)
	target.grantedAttrs[Flying] = 1
	first := g.PutOnBattlefield(NewCreature("First", "{0}", 0, 0), owner)
	g.effects.AddCopyEffect(first.ID(), target)
	g.ApplyContinuousEffects()
	second := g.PutOnBattlefield(NewCreature("Second", "{0}", 0, 0), owner)
	g.effects.AddCopyEffect(second.ID(), first)
	g.ApplyContinuousEffects()
	for _, p := range []*Permanent{first, second} {
		if p.Name() != "Model" || p.CurrentPower(g) != 3 || !p.HasSubType("Elf") || !slices.Equal(p.Card.Colors(), target.Card.Colors()) || len(p.RuntimeAbilities) != 1 {
			t.Fatalf("incomplete copy: %s %d %v", p.Name(), p.CurrentPower(g), p.RuntimeAbilities)
		}
		if p.HasKeyword(Flying) {
			t.Error("copied granted flying")
		}
	}
	action := first.RuntimeAbilities[0].(*ActionDefinition)
	action.activatedThisTurn = true
	g.ApplyContinuousEffects()
	if !first.RuntimeAbilities[0].(*ActionDefinition).activatedThisTurn {
		t.Fatal("copy reapplication reset activation state")
	}
}

func TestCopyEffectUpdateAndLeave(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	source := g.PutOnBattlefield(NewCreature("Original", "{U}", 0, 0), owner)
	first := g.PutOnBattlefield(NewCreature("First", "{G}", 2, 3, WithKeyword(Flying)), owner)
	second := g.PutOnBattlefield(NewCreature("Second", "{R}", 4, 5, WithAbility(NewActivatedAbility(GainLife(1), ManaCostOf("{0}"), WithOncePerTurn()))), owner)
	g.effects.AddCopyEffect(source.ID(), first)
	g.ApplyContinuousEffects()
	branch := g.Clone()
	branch.effects.UpdateCopyEffect(source.ID(), branch.FindPermanent(second.ID()))
	branch.ApplyContinuousEffects()
	copied := branch.FindPermanent(source.ID())
	if copied.Name() != "Second" || copied.HasKeyword(Flying) || copied.CurrentPower(branch) != 4 || len(copied.RuntimeAbilities) != 1 {
		t.Fatal("update did not replace copied characteristics")
	}
	if source.Name() != "First" || !source.HasKeyword(Flying) {
		t.Fatal("branch update changed original game")
	}
	branch.RemoveFromBattlefield(copied)
	if copied.Card.Name() != "Original" || copied.Card.Power() != 0 {
		t.Fatal("copy survived zone change")
	}
	if branch.zones.LKI(copied.ID()).Snapshot.Name() != "Second" {
		t.Fatal("LKI lost copied characteristics")
	}
}

func TestCopyEffectStaticAbilities(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	model := g.PutOnBattlefield(NewCreature("Model", "{0}", 1, 1, WithAbility(StaticAbility(FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, id uuid.UUID) error {
		g.MutablePermanent(id).powerBonus++
		return nil
	})))), owner)
	source := g.PutOnBattlefield(NewCreature("Source", "{0}", 0, 0), owner)
	g.effects.AddCopyEffect(source.ID(), model)
	g.ApplyContinuousEffects()
	if source.CurrentPower(g) != 2 {
		t.Fatal("copied static effect did not apply")
	}
	plain := g.PutOnBattlefield(NewCreature("Plain", "{0}", 3, 3), owner)
	g.effects.UpdateCopyEffect(source.ID(), plain)
	g.ApplyContinuousEffects()
	if source.CurrentPower(g) != 3 {
		t.Fatal("old static effect survived update")
	}
}
