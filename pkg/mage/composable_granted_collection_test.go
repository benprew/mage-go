package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestGrantedCollectionMembership(t *testing.T) {
	for _, modal := range []bool{false, true} {
		for _, dynamic := range []bool{false, true} {
			g, a, b := newModalGame()
			set := ControlledPermanents(IsArtifact)
			mod := EachActivatedAbilityModification(set, Costs(), GainLifeFor(You, Fixed(2)), ActivationLimit(1))
			if modal {
				mod = EachActivatedModesModification(set, Costs(), ModeOf("gain", GainLifeFor(You, Fixed(2))), ModeOf("counter", AddCountersTo(This, Charge, Fixed(1))), ActivationLimit(1))
			}
			donor := NewEnchantment("Donor", "")
			if dynamic {
				donor = NewEnchantment("Donor", "", WithAbility(Static(While(Controls(IsArtifact), mod))))
			}
			first, second, late, enemy := NewArtifact("First", ""), NewArtifact("Second", ""), NewArtifact("Late", ""), NewArtifact("Enemy", "")
			g.PutOnBattlefield(donor, a.PlayerID())
			g.PutOnBattlefield(first, a.PlayerID())
			g.PutOnBattlefield(second, a.PlayerID())
			g.PutOnBattlefield(enemy, b.PlayerID())
			if !dynamic {
				if err := ApplyEffect(g, Until(EndOfTurn, mod), donor.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.PutOnBattlefield(late, a.PlayerID())
			g.ApplyContinuousEffects()
			wantLate := 0
			if dynamic {
				wantLate = 1
			}
			if len(g.FindPermanent(late.ID()).RuntimeAbilities) != wantLate || len(g.FindPermanent(enemy.ID()).RuntimeAbilities) != 0 {
				t.Fatalf("membership: modal=%v dynamic=%v", modal, dynamic)
			}
			for _, c := range []Card{first, second} {
				a.modes = []int{0}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), c.ID(), 0, nil); err != nil {
					t.Fatal("recipients shared limit", err)
				}
			}
			original := g.Stack().Peek()
			if g.CopyStackObjectDirect(original, b.PlayerID(), false) == nil {
				t.Fatal("missing copy")
			}
			g.PhaseOut(first.ID())
			g.ApplyContinuousEffects()
			g.PhaseIn([]uuid.UUID{first.ID()})
			g.ApplyContinuousEffects()
			a.modes = []int{0}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err == nil {
				t.Fatal("phasing reset limit")
			}
			clone := g.Clone()
			clone.RemoveFromBattlefield(clone.FindPermanent(first.ID()))
			clone.PutOnBattlefield(first, a.PlayerID())
			clone.ApplyContinuousEffects()
			wantReturned := 0
			if dynamic {
				wantReturned = 1
			}
			if len(clone.FindPermanent(first.ID()).RuntimeAbilities) != wantReturned {
				t.Fatal("wrong returned membership")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 24 || clone.GetPlayer(b.PlayerID()).Life() != 22 || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("copy context or clone isolation")
			}
			g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
			g.ApplyContinuousEffects()
			wantRemaining := 1
			if dynamic {
				wantRemaining = 0
			}
			if len(g.FindPermanent(second.ID()).RuntimeAbilities) != wantRemaining {
				t.Fatal("wrong donor lifetime")
			}
			g.effects.RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if len(g.FindPermanent(second.ID()).RuntimeAbilities) != 0 {
				t.Fatal("grant did not expire")
			}
			g.ResolveStack()
			if a.Life() != 24 || b.Life() != 22 {
				t.Fatal("expiration cancelled pending abilities")
			}
		}
	}
}

func TestGrantedCollectionEmptyAndSourceDuration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor, recipient := NewEnchantment("Donor", ""), NewArtifact("Recipient", "")
	g.PutOnBattlefield(donor, a.PlayerID())
	effect := UntilSourceLeaves(This, EachActivatedAbilityModification(MatchingPermanents(IsArtifact), Costs(), Regenerate(This)))
	if err := ApplyEffect(g, effect, donor.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(recipient, a.PlayerID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 {
		t.Fatal("empty set acquired a later member")
	}
	if err := ApplyEffect(g, effect, donor.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 1 {
		t.Fatal("missing source-bound grant")
	}
	g.PhaseOut(donor.ID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 1 {
		t.Fatal("phasing ended duration")
	}
	g.PhaseIn([]uuid.UUID{donor.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 {
		t.Fatal("source departure did not end grant")
	}
}

func TestGrantedCollectionTargetsAndControl(t *testing.T) {
	for _, modal := range []bool{false, true} {
		g, a, b := newModalGame()
		first, second := NewArtifact("First", ""), NewArtifact("Second", "")
		g.PutOnBattlefield(first, a.PlayerID())
		g.PutOnBattlefield(second, a.PlayerID())
		target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
		mod := EachActivatedAbilityModification(ControlledPermanents(IsArtifact), Costs(RemoveCountersCost(Matrix, 1)), LoseLifeFor(target, Fixed(3)), WithTargets(target), ActivationLimit(1))
		if modal {
			mod = EachActivatedModesModification(ControlledPermanents(IsArtifact), Costs(RemoveCountersCost(Matrix, 1)), ModeOf("lose", LoseLifeFor(target, Fixed(3)), WithTargets(target)), ModeOf("gain", GainLifeFor(You, Fixed(1))), ActivationLimit(1))
		}
		for _, c := range []Card{first, second} {
			g.MutablePermanent(c.ID()).AddCounter(Matrix, 2)
		}
		if err := ApplyEffect(g, Until(Indefinite, mod), first.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		a.modes = []int{0}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		g.AddControlEffect(ControlEffectSpec{SourceID: first.ID(), TargetID: second.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
		g.ApplyContinuousEffects()
		b.modes = []int{0}
		if err := g.ActivateAbilityByIndex(b.PlayerID(), second.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		for _, c := range []Card{first, second} {
			if g.FindPermanent(c.ID()).Counters[Matrix] != 1 {
				t.Fatal("wrong recipient paid cost")
			}
		}
		g.ResolveStack()
		if a.Life() != 17 || b.Life() != 17 {
			t.Fatal("recipient target or controller context shared")
		}
	}
}
