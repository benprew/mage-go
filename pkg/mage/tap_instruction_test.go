package mage

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTapInstructionDeclaration(t *testing.T) {
	binding := DeclarePermanentTarget("subject", AnyPermanent)
	set := MatchingPermanents(IsLand)
	for _, declaration := range []TapInstruction{
		{Subject: binding, Operation: TapOperation}, {Subject: binding, Operation: UntapOperation},
		{Subject: binding, Operation: MayTapOrUntapOperation}, {Subjects: set, Operation: TapOperation}, {Subjects: set, Operation: UntapOperation},
	} {
		effect := declaration.Effect()
		instructions, ok := InspectTapInstructions(Sequence(effect, Sequence(effect)))
		if !ok || len(instructions) != 2 || instructions[0].Subject != declaration.Subject || instructions[0].Operation != declaration.Operation || InspectDefinition(instructions[0].Subjects).Text != InspectDefinition(declaration.Subjects).Text {
			t.Fatal("inspection lost declaration or simultaneous group")
		}
		instructions[0] = TapInstruction{}
		again, _ := InspectTapInstructions(effect)
		if again[0].Operation != declaration.Operation {
			t.Fatal("inspection aliases declaration")
		}
	}
	var absent *PermanentTargetBinding
	for _, declaration := range []TapInstruction{
		{}, {Subject: absent, Operation: TapOperation}, {Subject: binding, Subjects: set, Operation: TapOperation},
		{Subjects: set, Operation: MayTapOrUntapOperation}, {Subject: binding, Operation: TapOperation + 99},
	} {
		if declaration.Effect() != nil {
			t.Fatal("malformed declaration built an effect")
		}
	}
	var absentEffect *tapSubjectEffect
	for _, effect := range []Effect{nil, absentEffect, TapSubject(nil), UntapPermanent(nil), TapEach(nil), UntapEach(nil), MayTapOrUntap(nil), Destroy(binding),
		TapSubject(binding, TapAmountInto(nil)), UntapPermanentWithResult(binding),
		TapSubject(binding, TapAmountInto(&NumberBinding{})),
		Sequence(TapSubject(binding), destructionInspectionTrap{}),
	} {
		if _, ok := InspectTapInstructions(effect); ok {
			t.Fatal("incomplete tap tree accepted")
		}
	}
}

func TestTapInstructionSpellProof(t *testing.T) {
	for _, operation := range []TapOperationKind{TapOperation, UntapOperation, MayTapOrUntapOperation} {
		for _, state := range []string{"only", "prefix", "suffix", "illegal prefix", "empty group", "group", "phase", "phase cycle", "return", "wrong declaration", "opaque collection", "opaque suffix", "opaque replacement", "clone", "copy"} {
			t.Run(fmt.Sprintf("%d/%s", operation, state), func(t *testing.T) {
				g, p, obj, binding := targetQueryFixture(t)
				other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
				watched := []PermanentVisit{{other.ID(), other.incarnationID}}
				instruction := TapInstruction{Subject: binding, Operation: operation}
				want, known := false, true
				switch state {
				case "illegal prefix":
					p.GrantBaseAttr(core.Shroud)
				case "phase":
					g.PhaseOut(p.ID())
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "return":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(p.Card, p.ControllerID())
				case "wrong declaration":
					instruction.Subject = DeclarePermanentTarget("other", IsLand)
					known = false
				case "group", "empty group", "opaque collection":
					if operation == MayTapOrUntapOperation {
						return
					}
					instruction.Subject = nil
					filter := IsLand
					if state == "empty group" {
						filter = IsArtifact
					}
					if state == "opaque collection" {
						filter = NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("inspection called filter") })
						known = false
					}
					instruction.Subjects = MatchingPermanents(filter)
				case "opaque replacement":
					g.effects.AddReplacement(&proposalTrap{})
				}
				tap := instruction.Effect()
				obj.Effects = []Effect{tap}
				switch state {
				case "prefix", "illegal prefix", "empty group", "group", "phase", "phase cycle", "return":
					obj.Effects = []Effect{Sequence(tap, DestroyEach(MatchingPermanents(IsLand)))}
					known = state == "illegal prefix" || state == "empty group" || state == "phase" || state == "return"
					want = state == "empty group"
					if operation == UntapOperation && (state == "prefix" || state == "group" || state == "phase cycle") {
						want, known = true, true
					}
				case "suffix":
					obj.Effects = []Effect{Sequence(DestroyEach(MatchingPermanents(IsLand)), tap)}
					want = true
				case "opaque suffix":
					obj.Effects = []Effect{Sequence(tap, destructionInspectionTrap{})}
					known = false
				case "copy":
					obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
				}
				before := g.Clone()
				for range 2 {
					if got, gotKnown := g.InspectSpellDestruction(obj, watched); got != want || gotKnown != known {
						t.Fatalf("certainty=%v/%v want=%v/%v", got, gotKnown, want, known)
					}
				}
				if p.Tapped || !reflect.DeepEqual(g.random, before.random) || !reflect.DeepEqual(g.effects.replacements, before.effects.replacements) || g.StackSize() != before.StackSize() {
					t.Fatal("inspection changed state")
				}
				clone := g.Clone()
				if got, gotKnown := clone.InspectSpellDestruction(clone.stack.FindByID(obj.ID), watched); got != want || gotKnown != known {
					t.Fatal("clone changed proof")
				}
			})
		}
	}
}

func TestTapInstructionChangedLayerBarrier(t *testing.T) {
	for _, untap := range []bool{false, true} {
		g, p, obj, binding := targetQueryFixture(t)
		g.RemoveFromBattlefield(p)
		condition := PermanentIsTapped(This)
		if untap {
			condition = PermanentIsUntapped(This)
		}
		card := NewLand("Conditional land", WithAbility(Static(While(condition, GrantKeywordModification(This, core.Indestructible)))))
		p = g.PutOnBattlefield(card, obj.Controller)
		p.Tapped = untap
		obj.Targets[0] = p.ID()
		g.stack.Pop()
		g.PushStack(obj)
		tap := (TapInstruction{Subject: binding, Operation: TapOperation}).Effect()
		if untap {
			tap = (TapInstruction{Subject: binding, Operation: UntapOperation}).Effect()
		}
		obj.Effects = []Effect{Sequence(tap, Destroy(binding))}
		g.ApplyContinuousEffects()
		if p.HasAttr(core.Indestructible) {
			t.Fatal("fixture starts indestructible")
		}
		if certain, known := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); certain || known {
			t.Fatal("tap prefix used old continuous state")
		}
		g.ResolveStack()
		if g.FindPermanent(p.ID()) == nil || !g.FindPermanent(p.ID()).HasAttr(core.Indestructible) {
			t.Fatalf("native resolution did not apply changed layer: untap=%v present=%v tapped=%v attr=%v", untap, g.FindPermanent(p.ID()) != nil, p.Tapped, p.HasAttr(core.Indestructible))
		}
	}
}

func TestTapInstructionGroupLayers(t *testing.T) {
	for _, untap := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		condition := PermanentIsTapped(This)
		if untap {
			condition = PermanentIsUntapped(This)
		}
		subjects := make([]*Permanent, 0, 2)
		for range 2 {
			p := g.PutOnBattlefield(NewLand("Subject", WithAbility(Static(While(condition, GrantKeywordModification(This, core.Indestructible))))), a.PlayerID())
			p.Tapped = untap
			subjects = append(subjects, p)
		}
		g.ApplyContinuousEffects()
		if untap {
			if err := ApplyEffect(g, (TapInstruction{Subjects: MatchingPermanents(IsLand), Operation: UntapOperation}).Effect(), subjects[0].ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := ApplyEffect(g, (TapInstruction{Subjects: MatchingPermanents(IsLand), Operation: TapOperation}).Effect(), subjects[0].ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		}
		for _, p := range subjects {
			if !g.FindPermanent(p.ID()).HasKeyword(core.Indestructible) {
				t.Fatal("group did not refresh layers")
			}
		}
		clone := g.Clone()
		if untap {
			if err := ApplyEffect(clone, (TapInstruction{Subject: This, Operation: TapOperation}).Effect(), subjects[0].ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := ApplyEffect(clone, (TapInstruction{Subject: This, Operation: UntapOperation}).Effect(), subjects[0].ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		}
		if clone.FindPermanent(subjects[0].ID()).HasKeyword(core.Indestructible) || !g.FindPermanent(subjects[0].ID()).HasKeyword(core.Indestructible) {
			t.Fatal("layer refresh crossed clone state")
		}
	}
}
