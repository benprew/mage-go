package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEvenCollectionDamageContract(t *testing.T) {
	g, a, b := newSpellCopyGame()
	recipients := PermanentDamageRecipients(MatchingPermanents(IsCreature))
	effect := DealDamageEvenlyToRecipients(This, recipients, Fixed(7))
	spell := NewInstant("Equal collection", "", Spell(effect))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 20), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 20), b.PlayerID())
	g.AddPreventionShield(first.ID(), 1)
	events := 0
	g.damage.SetOnDamageDealt(func(string, string, int, bool) {
		events++
		if first.Damage != 2 || second.Damage != 3 {
			t.Error("observer saw an incomplete batch")
		}
	})
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if events != 2 || b.Life() != 20 {
		t.Fatal("wrong collection damage events")
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: NewInstant("Valid", "", Spell(effect))},
		{Name: "nil source", Card: NewInstant("Invalid", "", Spell(DealDamageEvenlyToRecipients(nil, recipients, Fixed(7)))), WantErr: true},
		{Name: "nil collection", Card: NewInstant("Invalid", "", Spell(DealDamageEvenlyToRecipients(This, nil, Fixed(7)))), WantErr: true},
		{Name: "nil total", Card: NewInstant("Invalid", "", Spell(DealDamageEvenlyToRecipients(This, recipients, nil))), WantErr: true},
	})
}

func TestEvenCollectionDamageSourceAndCapture(t *testing.T) {
	g, a, b := newSpellCopyGame()
	calls := 0
	effect := DealDamageEvenlyToRecipients(This, PermanentDamageRecipients(MatchingPermanents(IsCreature)), countedLifeAmount{&calls})
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 2, WithKeyword(core.Lifelink), WithAbility(Activate(Costs(SacrificeSourceCost()), effect))), a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 30), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 30), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	preparations := 0
	g.effects.AddReplacement(&dividedDamagePreparationProbe{check: func() {
		preparations++
		if preparations == 1 {
			g.PutOnBattlefield(NewCreature("Later", "", 1, 30), b.PlayerID())
		}
	}})
	g.ResolveStack()
	if calls != 1 || g.FindPermanent(first.ID()).Damage != 10 || g.FindPermanent(second.ID()).Damage != 10 || a.Life() != 40 {
		t.Fatalf("calls=%d damage=%d/%d life=%d preparations=%d", calls, first.Damage, second.Damage, a.Life(), preparations)
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("clone shared life changes")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 40 || calls != 2 {
		t.Fatal("clone lost saved source or read total more than once")
	}
}
