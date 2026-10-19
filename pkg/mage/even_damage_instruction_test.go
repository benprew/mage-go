package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEvenDamageInstructionDeclaration(t *testing.T) {
	recipients := DeclareAnyNumberDamageTargets("recipients")
	instruction := EvenDamageInstruction{Source: This, Recipients: recipients, Total: Fixed(5)}
	effect := instruction.Effect()
	got, ok := InspectEvenDamageInstructions(Sequence(effect, Sequence(effect)))
	if !ok || len(got) != 2 || got[0].Recipients != recipients {
		t.Fatal("lost division declaration")
	}
	got[0].Source = nil
	again, _ := InspectEvenDamageInstructions(effect)
	if again[0].Source == nil {
		t.Fatal("inspection aliases declaration")
	}
	var absent *DamageTargetsBinding
	for _, bad := range []EvenDamageInstruction{{}, {Source: This, Recipients: absent, Total: Fixed(1)}, {Recipients: recipients, Total: Fixed(1)}, {Source: This, Total: Fixed(1)}, {Source: This, Recipients: recipients}} {
		if bad.Effect() != nil {
			t.Fatal("incomplete declaration accepted")
		}
	}
	var absentEffect *damageEachEffect
	for _, bad := range []Effect{nil, absentEffect, DealDamageEach(This, recipients, Fixed(1)), DealAllocatedDamage(This, recipients, 3), DealDamageEvenly(nil, recipients, Fixed(1)), DealDamageEvenly(This, nil, Fixed(1)), DealDamageEvenly(This, recipients, nil), DealDamageEvenly(This, recipients, Fixed(1), nil), DealDamageEvenly(This, recipients, Fixed(1), DamageTotalInto(&NumberBinding{})), DealDamageEvenly(This, recipients, Fixed(1), DamageResultsInto(&DamageResultsBinding{})), Sequence(effect, destructionInspectionTrap{})} {
		if _, ok := InspectEvenDamageInstructions(bad); ok {
			t.Fatal("unsupported division accepted")
		}
	}
}

func TestEvenDamageInstructionOrder(t *testing.T) {
	for _, state := range []string{"only", "prefix", "empty prefix", "zero prefix", "player prefix", "suffix", "all illegal", "opaque amount", "opaque source", "opaque recipients", "opaque child", "opaque replacement"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			obj.Targets = nil
			obj.TargetSpecs = nil
			recipients := PermanentDamageRecipients(MatchingPermanents(IsCreature))
			p.GrantBaseAttr(core.AttrIsCreature)
			damage := DealDamageEvenlyToRecipients(This, recipients, Fixed(2))
			obj.Effects = []Effect{damage}
			want, known := false, true
			switch state {
			case "prefix":
				obj.Effects = []Effect{damage, DestroyEach(MatchingPermanents(IsLand))}
				known = false
			case "empty prefix":
				p.RevokeBaseAttr(core.AttrIsCreature)
				obj.Effects = []Effect{damage, DestroyEach(MatchingPermanents(IsLand))}
				want = true
			case "zero prefix":
				obj.Effects = []Effect{DealDamageEvenlyToRecipients(This, recipients, Fixed(0)), DestroyEach(MatchingPermanents(IsLand))}
				known = false
			case "player prefix":
				obj.Effects = []Effect{DealDamageEvenlyToRecipients(This, DamageRecipients(EachPlayer, MatchingPermanents(IsArtifact)), Fixed(1)), DestroyEach(MatchingPermanents(IsLand))}
				known = false
			case "suffix":
				obj.Effects = []Effect{DestroyEach(MatchingPermanents(IsLand)), damage}
				want = true
			case "all illegal":
				obj.Targets = []uuid.UUID{p.ID()}
				obj.TargetSpecs = []Target{target.targetSpec()}
				g.stack.Pop()
				g.PushStack(obj)
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = []Effect{damage, Destroy(target)}
			case "opaque amount":
				obj.Effects = []Effect{DealDamageEvenlyToRecipients(This, recipients, damageInspectionValueTrap{})}
				known = false
			case "opaque source":
				obj.Effects = []Effect{DealDamageEvenlyToRecipients(DamageSourceOf(This), recipients, Fixed(2))}
				known = false
			case "opaque recipients":
				obj.Effects = []Effect{DealDamageEvenlyToRecipients(This, PermanentDamageRecipients(collectionDamageSetTrap{}), Fixed(2))}
				known = false
			case "opaque child":
				obj.Effects = []Effect{Sequence(damage, destructionInspectionTrap{})}
				known = false
			case "opaque replacement":
				g.effects.AddReplacement(&proposalTrap{})
			}
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					if got, gotKnown := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}}); got != want || gotKnown != known {
						t.Fatalf("proof=%v/%v want=%v/%v", got, gotKnown, want, known)
					}
				}
			}
			if p.Damage != 0 || g.GetPlayer(obj.Controller).Life() != 20 {
				t.Fatal("inspection dealt damage")
			}
		})
	}
}
