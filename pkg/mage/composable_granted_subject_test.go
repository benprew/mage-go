package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticGrantedSubjectAttachment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor := NewAura("Donor", "", WithAbility(Static(GrantActivatedAbility(AttachedPermanent,
		Costs(ManaPayment("{1}")), Sequence(AddCountersTo(This, Charge, Fixed(1)), GainLifeFor(You, Fixed(2))), ActivationLimit(1)))))
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	donor.SetOwner(a.PlayerID())
	first.SetOwner(b.PlayerID())
	second.SetOwner(a.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.PutOnBattlefield(donor, a.PlayerID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(first.ID()).RuntimeAbilities) != 0 {
		t.Fatal("unattached donor granted an ability")
	}
	g.Attach(donor.ID(), first.ID())
	g.ApplyContinuousEffects()
	b.ManaPool().Add(Colorless, 2)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), first.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.Attach(donor.ID(), second.ID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(first.ID()).RuntimeAbilities) != 0 || len(g.FindPermanent(second.ID()).RuntimeAbilities) != 1 {
		t.Fatal("grant did not follow attachment")
	}
	g.ResolveStack()
	if g.FindPermanent(first.ID()).Counters[Charge] != 1 || g.FindPermanent(second.ID()).Counters[Charge] != 0 || b.Life() != 22 || a.Life() != 20 {
		t.Fatal("pending ability changed recipient or controller")
	}
	g.Attach(donor.ID(), first.ID())
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(b.PlayerID(), first.ID(), 0, nil); err == nil {
		t.Fatal("reattachment reset activation limit")
	}
	clone := g.Clone()
	clone.PhaseOut(donor.ID())
	clone.ApplyContinuousEffects()
	if len(clone.FindPermanent(first.ID()).RuntimeAbilities) != 0 {
		t.Fatal("phased donor retained grant")
	}
	clone.PhaseIn([]uuid.UUID{donor.ID()})
	clone.ApplyContinuousEffects()
	if len(clone.FindPermanent(first.ID()).RuntimeAbilities) != 1 {
		t.Fatal("grant did not resume")
	}
	if err := clone.ActivateAbilityByIndex(b.PlayerID(), first.ID(), 0, nil); err == nil {
		t.Fatal("phasing reset limit")
	}
	clone.RemoveFromBattlefield(clone.FindPermanent(donor.ID()))
	clone.ApplyContinuousEffects()
	if len(clone.FindPermanent(first.ID()).RuntimeAbilities) != 0 || len(g.FindPermanent(first.ID()).RuntimeAbilities) != 1 {
		t.Fatal("donor lifetime or clone isolation failed")
	}
}

func TestStaticGrantedSubjectModes(t *testing.T) {
	g, a, b := newModalGame()
	target := DeclarePlayerTarget("player")
	donor := NewAura("Donor", "", WithAbility(Static(GrantActivatedModes(AttachedPermanent, Costs(),
		ModeOf("counter", AddCountersTo(This, Charge, Fixed(1))),
		ModeOf("life", GainLifeFor(target, Fixed(3)), WithTargets(target)), ActivationLimit(1)))))
	recipient := NewCreature("Recipient", "", 2, 2)
	g.PutOnBattlefield(recipient, b.PlayerID())
	g.PutOnBattlefield(donor, a.PlayerID())
	g.Attach(donor.ID(), recipient.ID())
	g.ApplyContinuousEffects()
	b.modes = []int{1}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.Stack().Peek().ModeChoice != 1 {
		t.Fatal("lost selected mode")
	}
	g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 {
		t.Fatal("lost mode target after donor departure")
	}
}

func TestStaticGrantedSubjectValidation(t *testing.T) {
	for _, subject := range []PermanentRef{DeclarePermanentTarget("outer", IsCreature), From(BindPermanent("outer"))} {
		card := NewEnchantment("Invalid subject", "", WithAbility(Static(GrantActivatedAbility(subject, Costs(), Regenerate(This)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "subjects") {
			t.Fatalf("missing subject diagnostic: %v", err)
		}
	}
	for _, subject := range []PermanentRef{This, AttachedPermanent} {
		card := NewEnchantment("Valid subject", "", WithAbility(Static(GrantActivatedAbility(subject, Costs(), Regenerate(This)))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStaticGrantedSubjectReturnedRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor := NewAura("Donor", "", WithAbility(Static(GrantActivatedAbility(AttachedPermanent, Costs(),
		Sequence(AddCountersTo(This, Charge, Fixed(1)), GainLifeFor(You, Fixed(2)))))))
	recipient := NewCreature("Recipient", "", 2, 2)
	donor.SetOwner(a.PlayerID())
	recipient.SetOwner(b.PlayerID())
	g.PutOnBattlefield(recipient, b.PlayerID())
	g.PutOnBattlefield(donor, a.PlayerID())
	g.Attach(donor.ID(), recipient.ID())
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(recipient.ID()))
	g.PutOnBattlefield(recipient, b.PlayerID())
	g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
	g.ResolveStack()
	if g.FindPermanent(recipient.ID()).Counters[Charge] != 0 || b.Life() != 22 {
		t.Fatal("pending grant followed returned recipient or stopped remaining instructions")
	}
}

func TestStaticGrantedSubjectNil(t *testing.T) {
	var typedNil *PermanentTargetBinding
	for _, subject := range []PermanentRef{nil, typedNil} {
		for _, modal := range []bool{false, true} {
			func() {
				defer func() {
					if recover() == nil {
						t.Error("nil subject accepted")
					}
				}()
				if modal {
					GrantActivatedModes(subject, Costs(), ModeOf("one", Regenerate(This)), ModeOf("two", GainLifeFor(You, Fixed(1))))
				} else {
					GrantActivatedAbility(subject, Costs(), Regenerate(This))
				}
			}()
		}
	}
}
