package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestFightTypedTargetsAndIdentity(t *testing.T) {
	for _, changed := range []string{"none", "first returned", "second returned", "second noncreature"} {
		t.Run(changed, func(t *testing.T) {
			g, a, b := newTriggerTargetGame()
			first := NewCreature("First", "", 2, 4)
			first.SetOwner(a.PlayerID())
			firstPerm := g.PutOnBattlefield(first, a.PlayerID())
			second := NewCreature("Second", "", 3, 4)
			second.SetOwner(b.PlayerID())
			secondPerm := g.PutOnBattlefield(second, b.PlayerID())
			firstRef := DeclarePermanentTarget("first", IsCreature)
			secondRef := DeclarePermanentTarget("second", IsCreature)
			ability := Spell(Fight(firstRef, secondRef), WithTargets(firstRef, secondRef))
			card := NewInstant("Fight", "", WithAbility(ability))
			card.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition("Fight", card); err != nil {
				t.Fatal(err)
			}
			obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
			g.pushStack(obj)
			switch changed {
			case "first returned":
				g.RemoveFromBattlefield(firstPerm)
				firstPerm = g.PutOnBattlefield(first, a.PlayerID())
			case "second returned":
				g.RemoveFromBattlefield(secondPerm)
				secondPerm = g.PutOnBattlefield(second, b.PlayerID())
			case "second noncreature":
				secondPerm.RevokeBaseAttr(AttrIsCreature)
			}
			g.ResolveStackObject(obj)
			wantFirst, wantSecond := 0, 0
			if changed == "none" {
				wantFirst, wantSecond = 3, 2
			}
			if firstPerm.Damage != wantFirst || secondPerm.Damage != wantSecond {
				t.Fatalf("damage = (%d, %d), want (%d, %d)", firstPerm.Damage, secondPerm.Damage, wantFirst, wantSecond)
			}
		})
	}
}

func TestFightSelfDealsTwicePower(t *testing.T) {
	g, a, _ := newTriggerTargetGame()
	c := NewCreature("Self", "", 2, 5)
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	if err := ApplyEffect(g, Fight(This, This), c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if p.Damage != 4 {
		t.Fatalf("damage = %d, want 4", p.Damage)
	}
}

func TestFightDamageObserversSeeBothResults(t *testing.T) {
	g, a, b := newTriggerTargetGame()
	first := NewCreature("First", "", 2, 5)
	first.SetOwner(a.PlayerID())
	firstPerm := g.PutOnBattlefield(first, a.PlayerID())
	second := NewCreature("Second", "", 3, 5)
	second.SetOwner(b.PlayerID())
	secondPerm := g.PutOnBattlefield(second, b.PlayerID())
	observations := 0
	g.damage.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
		observations++
		if firstPerm.Damage != 3 || secondPerm.Damage != 2 {
			t.Errorf("observer saw damage (%d, %d), want (3, 2)", firstPerm.Damage, secondPerm.Damage)
		}
	})
	fightPermanents(&EffectContext{Game: g, Controller: a.PlayerID()}, firstPerm, secondPerm)
	if observations != 2 {
		t.Fatalf("observations = %d, want 2", observations)
	}
}

func TestFightBindingValidation(t *testing.T) {
	first := DeclarePermanentTarget("first", IsCreature)
	second := DeclarePermanentTarget("second", IsCreature)
	for _, tc := range []struct {
		name     string
		body     ResolvingEffect
		declared []TargetBinding
		want     string
	}{
		{"nil first", Fight(nil, second), []TargetBinding{second}, ".first"},
		{"undeclared second", Fight(first, second), []TargetBinding{first}, ".second"},
		{"nil second", Fight(first, nil), []TargetBinding{first}, ".second"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewInstant("Invalid Fight", "", WithAbility(Spell(tc.body, WithTargets(tc.declared...))))
			if err := ValidateCardDefinition("Invalid Fight", card); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validation error = %v, want path %q", err, tc.want)
			}
		})
	}
}

func TestFightTypedCopyAndClone(t *testing.T) {
	g, a, b := newTriggerTargetGame()
	first := NewCreature("First", "", 2, 5)
	first.SetOwner(a.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	second := NewCreature("Second", "", 3, 5)
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	firstRef := DeclarePermanentTarget("first", IsCreature)
	secondRef := DeclarePermanentTarget("second", IsCreature)
	ability := Spell(Fight(firstRef, secondRef), WithTargets(firstRef, secondRef))
	card := NewInstant("Fight", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	clone := g.Clone()
	copyObj := clone.CopySpellOnStack(card.ID(), a.PlayerID(), false)
	clone.ResolveStackObject(copyObj)
	if clone.FindPermanent(first.ID()).Damage != 3 || clone.FindPermanent(second.ID()).Damage != 2 {
		t.Fatal("copied fight lost its target bindings")
	}
	if g.FindPermanent(first.ID()).Damage != 0 || g.FindPermanent(second.ID()).Damage != 0 {
		t.Fatal("cloned fight changed original game")
	}
	g.ResolveStackObject(obj)
	if g.FindPermanent(first.ID()).Damage != 3 || g.FindPermanent(second.ID()).Damage != 2 {
		t.Fatal("original fight lost its target bindings")
	}
}
