package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttachedActivationCloneIsolation(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	host := g.PutOnBattlefield(NewCreature("Host", "{0}", 1, 1), owner)
	aura := g.PutOnBattlefield(NewEnchantment("Aura", "{0}"), owner)
	aura.AttachedTo = host.ID()
	effect := GrantActivatedAbilityToAttached(GainLife(1), ManaCostOf("{0}"), AttachAura, WithOncePerTurn())
	g.effects.Add(effect, aura.ID())
	g.ApplyContinuousEffects()
	original := UnwrapAbility(host.RuntimeAbilities[0]).(*ActionDefinition)
	branch := g.Clone()
	branch.ApplyContinuousEffects()
	cloned := UnwrapAbility(branch.FindPermanent(host.ID()).RuntimeAbilities[0]).(*ActionDefinition)
	branch.markAbilityActivated(host.ID(), cloned.AbilityID())
	if g.activationCount(host.ID(), original.AbilityID()) > 0 {
		t.Fatal("branch changed live granted ability")
	}
	branch.ApplyContinuousEffects()
	if branch.activationCount(host.ID(), UnwrapAbility(branch.FindPermanent(host.ID()).RuntimeAbilities[0]).AbilityID()) != 1 {
		t.Fatal("reapplication reset activation limit")
	}
}
