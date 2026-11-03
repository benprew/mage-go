package mage

import (
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSharedAbilityActivationCountsUseSourceIncarnation(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	ability := NewActivated(ManaCostOf("{0}"), GainLife(1), WithOncePerTurn())
	a := g.PutOnBattlefield(NewArtifact("A", "{0}", WithAbility(ability)), owner)
	b := g.PutOnBattlefield(NewArtifact("B", "{0}", WithAbility(ability)), owner)
	if err := g.ActivateAbilityByIndex(owner, a.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if ability.CanActivate(AbilityContext{SourceID: a.ID(), Controller: owner}, g) {
		t.Fatal("source can activate twice")
	}
	if !ability.CanActivate(AbilityContext{SourceID: b.ID(), Controller: owner}, g) {
		t.Fatal("shared definition used another source's count")
	}
	branch := g.Clone()
	if err := branch.ActivateAbilityByIndex(owner, b.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !ability.CanActivate(AbilityContext{SourceID: b.ID(), Controller: owner}, g) {
		t.Fatal("branch changed original activation count")
	}
	g.RemoveFromBattlefield(a)
	a = g.PutOnBattlefield(a.Card, owner)
	if !ability.CanActivate(AbilityContext{SourceID: a.ID(), Controller: owner}, g) {
		t.Fatal("new incarnation retained activation count")
	}
}

func TestRepeatedAbilitySpecificationsHaveIndependentLimits(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	ability := NewActivated(ManaCostOf("{0}"), GainLife(1), WithOncePerTurn())
	permanent := g.PutOnBattlefield(NewArtifact("Repeated", "{0}", WithAbility(ability), WithAbility(ability)), owner)
	for _, index := range []int{0, 1} {
		if err := g.ActivateAbilityByIndex(owner, permanent.ID(), index, nil); err != nil {
			t.Fatal("repeated ability shared another occurrence's limit:", err)
		}
		if err := g.ActivateAbilityByIndex(owner, permanent.ID(), index, nil); err == nil {
			t.Fatal("ability exceeded its limit")
		}
	}
}

func TestRepeatedStaticSpecificationsApplyEachOccurrence(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	effect := FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, id uuid.UUID) error { g.MutablePermanent(id).BoostPT(1, 1); return nil })
	permanent := g.PutOnBattlefield(NewCreature("Repeated", "{0}", 1, 1, WithStaticAbility(effect, effect)), owner)
	if permanent.CurrentPower(g) != 3 {
		t.Fatal("repeated static effect was lost")
	}
	branch := g.Clone()
	branch.ApplyContinuousEffects()
	if branch.FindPermanent(permanent.ID()).CurrentPower(branch) != 3 || permanent.CurrentPower(g) != 3 {
		t.Fatal("static effect bindings changed across branches")
	}
}

func TestGrantedActivationLimitSurvivesAnotherGrantLeaving(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	host := g.PutOnBattlefield(NewCreature("Host", "{0}", 2, 2), owner)
	effect := GrantActivatedAbilityToAttached(GainLife(1), ManaCostOf("{0}"), AttachAura, WithOncePerTurn())
	definition, err := NewCardDefinition(CardDefinitionSpec{Name: "Grant", TypeLine: TypeLine{CardTypes: []CardType{TypeEnchantment}}, Abilities: []Ability{StaticAbility(effect)}})
	if err != nil {
		t.Fatal(err)
	}
	first := g.PutOnBattlefield(NewCard(definition), owner)
	second := g.PutOnBattlefield(NewCard(definition), owner)
	first.AttachedTo = host.ID()
	second.AttachedTo = host.ID()
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(owner, host.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(first)
	if err := g.ActivateAbilityByIndex(owner, host.ID(), 0, nil); err == nil {
		t.Fatal("remaining grant lost its activation count")
	}
}

func TestRepeatedStateTriggerSpecificationsRemainSeparate(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	ability := NewStateTriggered(false, GainLife(1))
	g.PutOnBattlefield(NewArtifact("Repeated state", "{0}", WithAbility(ability), WithAbility(ability)), owner)
	g.CheckStateTriggers()
	if len(g.triggers.Pending()) != 2 {
		t.Fatal("repeated state trigger specifications shared suppression state")
	}
}
