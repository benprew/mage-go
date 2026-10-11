package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellAbilityExclusionIdentity(t *testing.T) {
	for _, change := range []string{"none", "self", "other-target", "spell", "missing-visit", "departed", "returned", "phased", "phase-cycle", "noncreature", "renamed", "removed", "resolving"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			self := g.PutOnBattlefield(NewCreature("Worker", "", 1, 1), a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Worker", "", 1, 1), b.PlayerID())
			victim := &StackObject{ID: uuid.New(), SourceID: uuid.New(), Card: NewArtifact("Victim", ""), Controller: a.PlayerID()}
			g.PushStack(victim)
			source := other
			if change == "self" {
				source = self
			}
			ability := &StackObject{ID: uuid.New(), SourceID: source.ID(), IsAbility: true, Controller: b.PlayerID(), Targets: []uuid.UUID{victim.ID}}
			if change == "other-target" {
				ability.Targets = []uuid.UUID{uuid.New()}
			}
			if change == "spell" {
				ability.IsAbility = false
				ability.Card = NewInstant("Spell", "")
			}
			if change == "missing-visit" {
				g.GetStack().Push(ability)
			} else {
				g.PushStack(ability)
			}
			clone := g.Clone()
			other = g.mutablePermanentIncludingPhased(other.ID())
			switch change {
			case "departed", "returned":
				g.BouncePermanentToHand(other)
				if change == "returned" {
					b.RemoveFromHand(other.Card.ID())
					returned := g.PutOnBattlefield(other.Card, b.PlayerID())
					returned.RevokeBaseAttr(core.AttrIsCreature)
					returned.GrantBaseAttr(core.AttrIsArtifact)
				}
			case "phased":
				g.PhaseOut(other.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(other.ID()))
			case "noncreature":
				other.RevokeBaseAttr(core.AttrIsCreature)
				other.GrantBaseAttr(core.AttrIsArtifact)
			case "renamed":
				card := *other.Card.(*BaseCard)
				card.name = "Different"
				other.Card = &card
			case "removed":
				g.GetStack().RemoveByID(ability.ID)
			case "resolving":
				g.GetStack().Pop()
				g.stack.resolvingReferenceObjects = append(g.stack.resolvingReferenceObjects, ability)
			}
			target := DeclareSpellTarget("victim").WithFilter(SpellNotTargetedByPermanentAbility(And(IsCreature, Named("Worker")), true))
			want := slices.Contains([]string{"self", "other-target", "spell", "missing-visit", "noncreature", "renamed", "removed", "resolving"}, change)
			if got := slices.Contains(target.Possible(a.PlayerID(), self.Card, g), victim.ID); got != want {
				t.Fatalf("legal = %v, want %v", got, want)
			}
			cloneWant := slices.Contains([]string{"self", "other-target", "spell", "missing-visit"}, change)
			if got := slices.Contains(target.Possible(a.PlayerID(), self.Card, clone), victim.ID); got != cloneWant {
				t.Fatal("clone lost separate stack or characteristics")
			}
		})
	}
}
