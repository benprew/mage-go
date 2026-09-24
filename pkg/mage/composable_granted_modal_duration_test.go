package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestGrantedModalModificationLifetimeAndCopies(t *testing.T) {
	for _, duration := range []Duration{Indefinite, EndOfTurn, EndOfCombat} {
		t.Run(duration.String(), func(t *testing.T) {
			g, a, b := newModalGame()
			donor, recipient := NewAura("Donor", ""), NewArtifact("Recipient", "")
			g.PutOnBattlefield(donor, a.PlayerID())
			g.PutOnBattlefield(recipient, b.PlayerID())
			g.Attach(donor.ID(), recipient.ID())
			target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
			effect := Until(duration, ActivatedModesModification(AttachedPermanent, Costs(RemoveCountersCost(Matrix, 1)),
				ModeOf("counter", AddCountersTo(This, Charge, Fixed(1))),
				ModeOf("lose", LoseLifeFor(target, Fixed(3)), WithTargets(target)), ActivationLimit(1)))
			g.MutablePermanent(recipient.ID()).AddCounter(Matrix, 2)
			if err := ApplyEffect(g, effect, donor.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
			g.ApplyContinuousEffects()
			b.modes = []int{1}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			original := g.Stack().Peek()
			a.targets = [][]uuid.UUID{{b.PlayerID()}}
			cp := g.CopyStackObjectDirect(original, a.PlayerID(), true)
			if cp == nil || cp.ModeChoice != 1 || cp.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() {
				t.Fatal("copy lost mode or independent target")
			}
			g.PhaseOut(recipient.ID())
			g.ApplyContinuousEffects()
			g.PhaseIn([]uuid.UUID{recipient.ID()})
			g.ApplyContinuousEffects()
			b.modes = []int{0}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err == nil {
				t.Fatal("modes or phasing reset shared limit")
			}
			if g.FindPermanent(recipient.ID()).Counters[Matrix] != 1 {
				t.Fatal("failed activation or copy paid a counter")
			}
			clone := g.Clone()
			clone.RemoveFromBattlefield(clone.FindPermanent(recipient.ID()))
			clone.PutOnBattlefield(recipient, b.PlayerID())
			clone.ApplyContinuousEffects()
			clone.ResolveStack()
			if len(clone.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 || clone.GetPlayer(a.PlayerID()).Life() != 17 || clone.GetPlayer(b.PlayerID()).Life() != 17 {
				t.Fatal("returned recipient kept grant or pending modes lost context")
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.effects.RemoveEndOfTurn()
			g.effects.RemoveEndOfCombat()
			g.ApplyContinuousEffects()
			want := 0
			if duration == Indefinite {
				want = 1
			}
			if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != want {
				t.Fatal("wrong grant duration")
			}
			g.ResolveStack()
			if a.Life() != 17 || b.Life() != 17 || g.FindPermanent(recipient.ID()).Counters[Charge] != 0 {
				t.Fatal("expiration changed pending modes or unselected mode ran")
			}
		})
	}
}

func TestGrantedModalModificationSourceDurationAndWhile(t *testing.T) {
	for _, dynamic := range []bool{false, true} {
		g, a, _ := newModalGame()
		modification := ActivatedModesModification(AttachedPermanent, Costs(),
			ModeOf("counter", AddCountersTo(This, Charge, Fixed(1))),
			ModeOf("gain", GainLifeFor(You, Fixed(2))), ActivationLimit(1))
		donor, recipient := NewAura("Donor", ""), NewArtifact("Recipient", "")
		if dynamic {
			donor = NewAura("Donor", "", WithAbility(Static(While(Controls(IsArtifact), modification))))
		}
		g.PutOnBattlefield(donor, a.PlayerID())
		g.PutOnBattlefield(recipient, a.PlayerID())
		g.Attach(donor.ID(), recipient.ID())
		if !dynamic {
			if err := ApplyEffect(g, UntilSourceLeaves(This, modification), donor.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		}
		g.ApplyContinuousEffects()
		a.modes = []int{0}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ApplyContinuousEffects()
		if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err == nil {
			t.Fatal("reevaluation reset limit")
		}
		g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
		g.ApplyContinuousEffects()
		if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 {
			t.Fatal("grant outlived donor")
		}
		g.RemoveFromBattlefield(g.FindPermanent(recipient.ID()))
		g.PutOnBattlefield(recipient, a.PlayerID())
		g.ResolveStack()
		if g.FindPermanent(recipient.ID()).Counters[Charge] != 0 {
			t.Fatal("pending mode followed returned source")
		}
	}
}
