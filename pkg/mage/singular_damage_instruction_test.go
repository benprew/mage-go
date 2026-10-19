package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSingularDamageInstructionDeclaration(t *testing.T) {
	target := DeclareDamageTarget("recipient")
	declaration := SingularDamageInstruction{Source: This, Recipient: target, Amount: Fixed(3)}
	effect := declaration.Effect()
	instructions, ok := InspectSingularDamageInstructions(Sequence(effect, Sequence(effect)))
	if !ok || len(instructions) != 2 || instructions[0].Recipient != target {
		t.Fatal("lost damage declaration")
	}
	instructions[0].Recipient = nil
	again, _ := InspectSingularDamageInstructions(effect)
	if again[0].Recipient != target {
		t.Fatal("inspection aliases declaration")
	}
	var absent *DamageTargetBinding
	for _, bad := range []SingularDamageInstruction{{}, {Source: This, Recipient: absent, Amount: Fixed(3)}, {Recipient: target, Amount: Fixed(3)}, {Source: This, Recipient: target}} {
		if bad.Effect() != nil {
			t.Fatal("malformed declaration built an effect")
		}
	}
	var absentEffect *damageFromEffect
	for _, bad := range []Effect{nil, absentEffect, DealDamageFrom(nil, target, Fixed(1)), DealDamageFrom(This, target, nil), DealDamageFrom(This, nil, Fixed(1)), DealDamageFrom(This, target, Fixed(1), DamageAmountInto(&NumberBinding{})), DealDamageFrom(This, target, Fixed(1), nil), Sequence(effect, destructionInspectionTrap{})} {
		if _, ok := InspectSingularDamageInstructions(bad); ok {
			t.Fatal("incomplete damage declaration accepted")
		}
	}
	if _, ok := InspectDamageTarget(DeclareOpponentChosenDamageTarget("chosen")); ok {
		t.Fatal("opaque target wrapper accepted")
	}
	if _, ok := InspectDamageTarget(DeclarePlayerOrPlaneswalkerTarget("restricted")); !ok {
		t.Fatal("native player-or-planeswalker domain rejected")
	}
	if _, ok := InspectDamageTarget(absent); ok {
		t.Fatal("nil declaration accepted")
	}
}

func damageQueryFixture(t *testing.T) (*Game, *Permanent, *StackObject, *DamageTargetBinding) {
	t.Helper()
	g, p, obj, _ := targetQueryFixture(t)
	p.GrantBaseAttr(core.AttrIsCreature)
	target := DeclareDamageTarget("recipient")
	g.stack.Pop()
	obj.TargetSpecs = []Target{target.targetSpec()}
	obj.Effects = []Effect{(SingularDamageInstruction{Source: This, Recipient: target, Amount: Fixed(3)}).Effect()}
	g.PushStack(obj)
	return g, p, obj, target
}

func TestSingularDamageInstructionLegality(t *testing.T) {
	for _, state := range []string{"normal", "player", "planeswalker", "phase", "cycle", "return", "shroud", "hexproof", "protection", "other protection", "noncreature", "missing visit", "missing zone", "duplicate", "wrong declaration", "opaque protection", "opaque restriction", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := damageQueryFixture(t)
			predicate, ok := InspectDamageTarget(target)
			if !ok {
				t.Fatal("native any target rejected")
			}
			legal, known := true, true
			switch state {
			case "player":
				obj.Targets[0] = obj.Controller
			case "planeswalker":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(NewPlaneswalker("Walker", "", 5), p.ControllerID())
				g.stack.Pop()
				obj.Targets[0] = p.ID()
				g.PushStack(obj)
			case "phase":
				g.PhaseOut(p.ID())
				legal = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, p.ControllerID())
				legal = false
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
				legal = false
			case "hexproof":
				p.GrantBaseAttr(core.Hexproof)
				legal = false
			case "protection":
				p.RuntimeAbilities = append(p.RuntimeAbilities, ProtectionFromColor(core.Red))
				legal = false
			case "other protection":
				p.RuntimeAbilities = append(p.RuntimeAbilities, ProtectionFromColor(core.Blue))
			case "noncreature":
				p.RevokeBaseAttr(core.AttrIsCreature)
				legal = false
			case "missing visit":
				delete(obj.targetIncarnations, p.ID())
				known = false
			case "missing zone":
				delete(obj.TargetZones, p.ID())
				known = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, obj.TargetSpecs[0])
				obj.Targets = append(obj.Targets, obj.Targets[0])
				known = false
			case "wrong declaration":
				predicate, _ = InspectDamageTarget(DeclareDamageTarget("wrong"))
				known = false
			case "opaque protection":
				p.RuntimeAbilities = append(p.RuntimeAbilities, &ProtectionAbility{Filter: NewCardFilter("trap", func(Card) bool { panic("protection callback") })})
				known = false
			case "opaque restriction":
				source := g.PutOnBattlefield(NewAura("Restriction", "", WithAbility(Static(CantBeTargetedBySpells(AttachedPermanent, NewCardFilter("trap", func(Card) bool { panic("restriction callback") }))))), obj.Controller)
				g.Attach(source.ID(), p.ID())
				known = false
			case "copy":
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			case "clone":
				g = g.Clone()
				obj = g.StackPeek()
			}
			before := g.Clone()
			for range 2 {
				selection, gotKnown := predicate.Select(g, obj)
				if gotKnown != known || known && (!selection.Empty()) != legal {
					t.Fatalf("selection=%+v known=%v want legal=%v known=%v", selection, gotKnown, legal, known)
				}
				if known && legal && state == "player" && selection.Player != obj.Controller {
					t.Fatal("player identity lost")
				}
				if known && legal && state != "player" && selection.Permanent != (PermanentVisit{p.ID(), p.incarnationID}) {
					t.Fatal("permanent visit lost")
				}
			}
			if !reflect.DeepEqual(g.random, before.random) || g.StackSize() != before.StackSize() || p.Damage != 0 {
				t.Fatal("query changed state")
			}
		})
	}
}

func TestSingularDamageInstructionOrder(t *testing.T) {
	for _, state := range []string{"only", "prefix", "suffix", "illegal prefix", "all illegal", "player prefix", "zero prefix", "opaque amount", "opaque child", "opaque replacement"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := damageQueryFixture(t)
			other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
			second := DeclarePermanentTarget("second", IsLand)
			g.stack.Pop()
			obj.Targets = append(obj.Targets, other.ID())
			obj.TargetSpecs = append(obj.TargetSpecs, second.targetSpec())
			g.PushStack(obj)
			damage := obj.Effects[0]
			want, known := false, true
			switch state {
			case "prefix":
				obj.Effects = []Effect{Sequence(damage, Destroy(second))}
				known = false
			case "suffix":
				obj.Effects = []Effect{Sequence(Destroy(second), damage)}
				want = true
			case "illegal prefix":
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = []Effect{Sequence(damage, Destroy(second))}
				want = true
			case "all illegal":
				p.GrantBaseAttr(core.Shroud)
				other.GrantBaseAttr(core.Shroud)
				obj.Effects = []Effect{Sequence(damage, Destroy(second))}
			case "player prefix":
				obj.Targets[0] = obj.Controller
				obj.Effects = []Effect{Sequence(damage, Destroy(second))}
				known = false
			case "zero prefix":
				obj.Effects = []Effect{Sequence(DealDamageFrom(This, target, Fixed(0)), Destroy(second))}
				known = false
			case "opaque amount":
				obj.Effects = []Effect{DealDamageFrom(This, target, damageInspectionValueTrap{})}
				known = false
			case "opaque child":
				obj.Effects = []Effect{Sequence(damage, destructionInspectionTrap{})}
				known = false
			case "opaque replacement":
				g.effects.AddReplacement(&proposalTrap{})
			}
			for range 2 {
				if got, gotKnown := g.InspectSpellDestruction(obj, []PermanentVisit{{other.ID(), other.incarnationID}}); got != want || gotKnown != known {
					t.Fatalf("proof=%v/%v want=%v/%v", got, gotKnown, want, known)
				}
			}
			clone := g.Clone()
			if got, gotKnown := clone.InspectSpellDestruction(clone.StackPeek(), []PermanentVisit{{other.ID(), other.incarnationID}}); got != want || gotKnown != known {
				t.Fatal("clone proof differs")
			}
			if p.Damage != 0 {
				t.Fatal("inspection dealt damage")
			}
		})
	}
}

type damageInspectionValueTrap struct{}

func (damageInspectionValueTrap) Text() string                    { return "trap" }
func (damageInspectionValueTrap) resolveValue(*EffectContext) int { panic("inspection called value") }

func TestSingularDamageInstructionSourceLKI(t *testing.T) {
	for _, returned := range []bool{false, true} {
		g, a, b := newSpellCopyGame()
		target := DeclareDamageTarget("recipient")
		card := NewCreature("Green source", "{G}", 2, 2, WithAbility(Activate(Costs(SacrificeSourceCost()), DealDamageFrom(This, target, Fixed(2)), WithTargets(target))))
		source := g.PutOnBattlefield(card, a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Protected", "", 2, 9), b.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
			t.Fatal(err)
		}
		victim.RuntimeAbilities = append(victim.RuntimeAbilities, ProtectionFromColor(core.Green))
		if returned {
			g.PutOnBattlefield(card, b.PlayerID()).FaceDown = true
		}
		predicate, ok := InspectDamageTarget(target)
		if !ok {
			t.Fatal("target declaration rejected")
		}
		for _, game := range []*Game{g, g.Clone()} {
			obj := game.CopyStackObjectDirect(game.StackPeek(), a.PlayerID(), false)
			if selected, known := predicate.Select(game, obj); !known || !selected.Empty() {
				t.Fatal("target query lost source LKI")
			}
			game.ResolveStack()
			if game.FindPermanent(victim.ID()).Damage != 0 {
				t.Fatal("native resolution lost source LKI")
			}
		}
	}
}
