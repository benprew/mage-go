package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestGrantedModificationLifetime(t *testing.T) {
	for _, duration := range []Duration{Indefinite, EndOfTurn, EndOfCombat} {
		t.Run(duration.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			donor, recipient := NewAura("Donor", ""), NewCreature("Recipient", "", 2, 2)
			g.PutOnBattlefield(donor, a.PlayerID())
			g.PutOnBattlefield(recipient, b.PlayerID())
			g.Attach(donor.ID(), recipient.ID())
			effect := Until(duration, ActivatedAbilityModification(AttachedPermanent, Costs(RemoveCountersCost(Matrix, 1)), Sequence(AddCountersTo(This, Charge, Fixed(1)), GainLifeFor(You, Fixed(2))), ActivationLimit(1)))
			g.MutablePermanent(recipient.ID()).AddCounter(Matrix, 3)
			if err := ApplyEffect(g, effect, donor.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
			g.ApplyContinuousEffects()
			if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(recipient.ID()).Counters[Matrix] != 2 {
				t.Fatal("counter cost was not paid")
			}
			g.PhaseOut(recipient.ID())
			g.ApplyContinuousEffects()
			g.PhaseIn([]uuid.UUID{recipient.ID()})
			g.ApplyContinuousEffects()
			if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err == nil {
				t.Fatal("phasing reset limit")
			}
			clone := g.Clone()
			clone.RemoveFromBattlefield(clone.FindPermanent(recipient.ID()))
			clone.PutOnBattlefield(recipient, b.PlayerID())
			clone.ApplyContinuousEffects()
			clone.ResolveStack()
			if len(clone.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 || clone.FindPermanent(recipient.ID()).Counters[Charge] != 0 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
				t.Fatal("grant or pending body followed returned recipient")
			}
			g.ResolveStack()
			if g.FindPermanent(recipient.ID()).Counters[Charge] != 1 || b.Life() != 22 || a.Life() != 20 {
				t.Fatal("wrong granted context")
			}
			g.effects.RemoveEndOfTurn()
			g.effects.RemoveEndOfCombat()
			g.ApplyContinuousEffects()
			want := 0
			if duration == Indefinite {
				want = 1
			}
			if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != want {
				t.Fatal("wrong duration or clone shared lifetime")
			}
		})
	}
}

func TestGrantedModificationSeparateGrantsAndControl(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := NewCreature("Recipient", "", 2, 2)
	g.PutOnBattlefield(recipient, a.PlayerID())
	effect := Until(Indefinite, ActivatedAbilityModification(This, Costs(), GainLifeFor(You, Fixed(1)), ActivationLimit(1)))
	for range 2 {
		if err := ApplyEffect(g, effect, recipient.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: recipient.ID(), TargetID: recipient.ID(), ControllerID: b.PlayerID(), Duration: EndOfTurn})
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err == nil {
		t.Fatal("control change reset limit")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 1, nil); err != nil {
		t.Fatal("separate grants share limit", err)
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatal("pending grant changed controller")
	}
}

func TestGrantedModificationStaticIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	modification := ActivatedAbilityModification(This, Costs(), GainLifeFor(You, Fixed(1)), ActivationLimit(1))
	donor := NewCreature("Donor", "", 2, 2, WithAbility(Static(While(Controls(IsCreature), modification, modification))))
	g.PutOnBattlefield(donor, a.PlayerID())
	g.ApplyContinuousEffects()
	for index := range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), donor.ID(), index+1, nil); err != nil {
			t.Fatal(err)
		}
		g.ApplyContinuousEffects()
		if err := g.ActivateAbilityByIndex(a.PlayerID(), donor.ID(), index+1, nil); err == nil {
			t.Fatal("static reevaluation reset limit")
		}
	}
}

func TestGrantedModificationAbsentAndSourceDuration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor, recipient := NewAura("Donor", ""), NewCreature("Recipient", "", 2, 2)
	g.PutOnBattlefield(donor, a.PlayerID())
	g.PutOnBattlefield(recipient, a.PlayerID())
	effect := UntilSourceLeaves(This, ActivatedAbilityModification(AttachedPermanent, Costs(), Regenerate(This)))
	if err := ApplyEffect(g, effect, donor.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.Attach(donor.ID(), recipient.ID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 {
		t.Fatal("absent subject acquired later grant")
	}
	if err := ApplyEffect(g, effect, donor.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 1 {
		t.Fatal("source duration grant missing")
	}
	g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 {
		t.Fatal("source duration grant outlived source")
	}
}

func TestGrantedModificationLayerOrder(t *testing.T) {
	for _, removalFirst := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		recipient := NewCreature("Recipient", "", 2, 2)
		g.PutOnBattlefield(recipient, a.PlayerID())
		if removalFirst {
			g.AddContinuousEffect(RemoveAllAbilities(recipient.ID(), Indefinite))
		}
		if err := ApplyEffect(g, Until(Indefinite, ActivatedAbilityModification(This, Costs(), Regenerate(This))), recipient.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if !removalFirst {
			g.AddContinuousEffect(RemoveAllAbilities(recipient.ID(), Indefinite))
		}
		g.ApplyContinuousEffects()
		want := 0
		if removalFirst {
			want = 1
		}
		if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != want {
			t.Fatal("grant ignored layer timestamp")
		}
	}
}
