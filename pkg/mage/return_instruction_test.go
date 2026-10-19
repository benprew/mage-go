package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestReturnInstructionDeclaration(t *testing.T) {
	binding := DeclarePermanentTarget("subject", AnyPermanent)
	set := MatchingPermanents(IsLand)
	for _, declaration := range []DirectReturnInstruction{{Subject: binding}, {Subjects: set}} {
		effect := declaration.Effect()
		instructions, ok := InspectDirectReturn(Sequence(effect, Sequence(effect)))
		if !ok || len(instructions) != 2 || instructions[0].Subject != declaration.Subject || InspectDefinition(instructions[0].Subjects).Text != InspectDefinition(declaration.Subjects).Text {
			t.Fatal("return declaration lost subject or group identity")
		}
		instructions[0] = DirectReturnInstruction{}
		again, _ := InspectDirectReturn(effect)
		if again[0].Subject != declaration.Subject || InspectDefinition(again[0].Subjects).Text != InspectDefinition(declaration.Subjects).Text {
			t.Fatal("inspection result aliases declaration")
		}
	}
	var absent *PermanentTargetBinding
	for _, declaration := range []DirectReturnInstruction{{}, {Subject: absent}, {Subject: binding, Subjects: set}} {
		if declaration.Effect() != nil {
			t.Fatal("malformed declaration built an effect")
		}
	}
	var absentEffect *returnToHandSubjectEffect
	for _, effect := range []Effect{nil, absentEffect, ReturnToHand(nil), ReturnEachToHand(nil), Sequence(ReturnToHand(binding), destructionInspectionTrap{}), Destroy(binding)} {
		if _, ok := InspectDirectReturn(effect); ok {
			t.Fatal("incomplete return tree accepted")
		}
	}
}

func TestReturnInstructionSpellOrder(t *testing.T) {
	for _, state := range []string{"return only", "return then destroy", "return group", "empty prefix", "destroy first", "other watch", "mixed watches", "shroud", "phase", "phase cycle", "returned visit", "copy", "shield", "indestructible", "opaque suffix", "opaque collection", "opaque target", "wrong declaration", "opaque replacement", "return tail"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, binding := targetQueryFixture(t)
			p.Card.SetOwner(p.ControllerID())
			other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
			watched := make([]PermanentVisit, 1, 2)
			watched[0] = PermanentVisit{p.ID(), p.incarnationID}
			obj.Effects = []Effect{Sequence(ReturnToHand(binding), DestroyEach(MatchingPermanents(IsLand)))}
			want, known := false, true
			trap := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("inspection ran filter") })
			switch state {
			case "return only":
				obj.Effects = []Effect{ReturnToHand(binding)}
			case "return group":
				obj.Effects = []Effect{Sequence(ReturnEachToHand(MatchingPermanents(IsLand)), DestroyEach(MatchingPermanents(IsLand)))}
			case "empty prefix":
				obj.Effects = []Effect{Sequence(ReturnEachToHand(MatchingPermanents(IsArtifact)), Destroy(binding))}
				want = true
			case "destroy first":
				obj.Effects = []Effect{Sequence(Destroy(binding), ReturnToHand(binding))}
				want = true
			case "other watch":
				watched[0] = PermanentVisit{other.ID(), other.incarnationID}
				known = false
			case "mixed watches":
				watched = append(watched, PermanentVisit{other.ID(), other.incarnationID})
				known = false
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
			case "phase":
				g.PhaseOut(p.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "returned visit":
				g.BouncePermanentToHand(p)
				g.GetPlayer(p.Card.Owner()).RemoveFromHand(p.ID())
				g.PutOnBattlefield(p.Card, p.ControllerID())
			case "copy":
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			case "shield":
				g.AddRegenerationShield(p.ID())
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
			case "opaque replacement":
				g.effects.AddReplacement(&proposalTrap{})
			case "return tail":
				obj.Effects = []Effect{Sequence(ReturnToHand(binding), ReturnEachToHand(MatchingPermanents(IsLand)))}
				watched[0] = PermanentVisit{other.ID(), other.incarnationID}
			case "opaque suffix":
				obj.Effects = []Effect{Sequence(ReturnToHand(binding), destructionInspectionTrap{})}
				known = false
			case "opaque collection":
				obj.Effects = []Effect{ReturnEachToHand(MatchingPermanents(trap))}
				known = false
			case "opaque target":
				binding.spec.(*PermanentTarget).Filters = []PermanentFilter{trap}
				known = false
			case "wrong declaration":
				obj.Effects = []Effect{ReturnToHand(DeclarePermanentTarget("subject", IsLand))}
				known = false
			}
			before := g.Clone()
			for range 2 {
				if got, gotKnown := g.InspectSpellDestruction(obj, watched); got != want || gotKnown != known {
					t.Fatalf("certainty=%v/%v want=%v/%v", got, gotKnown, want, known)
				}
			}
			if !reflect.DeepEqual(g.random, before.random) || !reflect.DeepEqual(g.effects.replacements, before.effects.replacements) || g.StackSize() != before.StackSize() || g.FindPermanent(other.ID()) == nil || p.Tapped {
				t.Fatal("inspection changed game state")
			}
			clone := g.Clone()
			if got, gotKnown := clone.InspectSpellDestruction(clone.stack.FindByID(obj.ID), watched); got != want || gotKnown != known {
				t.Fatal("clone changed proof")
			}
			if state == "return then destroy" || state == "return group" || state == "shield" || state == "indestructible" {
				g.ResolveStack()
				if g.FindPermanent(p.ID()) != nil || len(g.GetPlayer(p.Card.Owner()).Hand()) != 1 {
					t.Fatal("returned subject did not reach its owner")
				}
				if g.FindPermanent(other.ID()) != nil || (len(g.GetPlayer(obj.Controller).Hand()) == 1) != (state == "return group") {
					t.Fatal("later destruction did not use changed state")
				}
			}
		})
	}
}

func TestReturnInstructionPartialLegality(t *testing.T) {
	g, p, obj, first := targetQueryFixture(t)
	second := DeclarePermanentTarget("second", IsLand)
	other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
	obj.TargetSpecs = append(obj.TargetSpecs, second.targetSpec())
	obj.Targets = append(obj.Targets, other.ID())
	obj.targetIncarnations[other.ID()] = other.incarnationID
	obj.TargetZones[other.ID()] = core.ZoneBattlefield
	obj.Effects = []Effect{Sequence(ReturnToHand(first), Destroy(second))}
	p.GrantBaseAttr(core.Shroud)
	watched := []PermanentVisit{{other.ID(), other.incarnationID}}
	if certain, known := g.InspectSpellDestruction(obj, watched); !certain || !known {
		t.Fatal("illegal return blocked legal destruction")
	}
	other.GrantBaseAttr(core.Shroud)
	if certain, known := g.InspectSpellDestruction(obj, watched); certain || !known {
		t.Fatal("all-illegal spell executed")
	}
	obj.Targets[1] = uuid.Nil
	if certain, known := g.InspectSpellDestruction(obj, watched); certain || known {
		t.Fatal("missing target supplied a proof")
	}
}

func TestReturnInstructionPhasedWatch(t *testing.T) {
	g, source, obj, binding := targetQueryFixture(t)
	card := NewLand("Phased watch")
	card.SetOwner(obj.Controller)
	subject := g.PutOnBattlefield(card, obj.Controller)
	watched := []PermanentVisit{{subject.ID(), subject.incarnationID}}
	ref := DeclarePermanentTarget("phase subject", IsLand)
	g.resolution.targetSpecs = []Target{ref.targetSpec()}
	if err := ApplyEffect(g, PhaseOutUntilSourceLeaves(ref, false), source.ID(), source.ControllerID(), []uuid.UUID{subject.ID()}); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanentIncludingPhased(subject.ID()).PhasedOut {
		t.Fatal("subject did not phase out")
	}
	obj.Effects = []Effect{Sequence(ReturnToHand(binding), DestroyEach(MatchingPermanents(IsLand)))}
	if certain, known := g.InspectSpellDestruction(obj, watched); certain || known {
		t.Fatal("return treated a phased visit as permanently absent")
	}
	if !g.FindPermanentIncludingPhased(subject.ID()).PhasedOut || g.FindPermanent(source.ID()) == nil {
		t.Fatal("inspection changed phase state")
	}
	g.ResolveStack()
	inGraveyard := false
	for _, moved := range g.GetPlayer(obj.Controller).Graveyard() {
		inGraveyard = inGraveyard || moved.ID() == card.ID()
	}
	if g.FindPermanentIncludingPhased(subject.ID()) != nil || !inGraveyard {
		t.Fatal("source departure did not expose watched land to later destruction")
	}
}
