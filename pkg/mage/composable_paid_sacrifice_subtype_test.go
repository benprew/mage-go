package mage

import (
	"strings"
	"testing"
)

func TestPaidSacrificeSubtypeAndStackFrames(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	thrull := BindNumber("was Thrull")
	source := NewCreature("Praetor", "", 5, 5, WithAbility(Activate(
		Costs(SacrificeCreatureCostWithSubtype("Thrull", thrull)),
		IfPositive(FromNumber(thrull), GainLifeFor(You, Fixed(1))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(NewCreature("Thrull", "", 1, 1, WithSubTypes("Thrull")), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Other", "", 1, 1), a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("clone life = %d, want 21", got)
	}
	g.ResolveStack()
	if got := a.Life(); got != 21 {
		t.Fatalf("life = %d, want 21", got)
	}
}

func TestPaidSacrificeSubtypeRollbackAndDeclaration(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	creature := NewCreature("Thrull", "", 1, 1, WithSubTypes("Thrull"))
	g.PutOnBattlefield(creature, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{SacrificeCreatureCostWithSubtype("Thrull", paid)}})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 {
		t.Fatal("returned creature produced a payment value")
	}
	for _, tc := range []struct {
		subtype string
		binding *NumberBinding
		want    string
	}{
		{"Thrull", paid, ""}, {"", paid, "subtype"}, {"Island", paid, "creature subtype"}, {"Thrull", nil, "non-nil"},
	} {
		card := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(SacrificeCreatureCostWithSubtype(tc.subtype, tc.binding)), GainLifeFor(You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("subtype %q: validation = %v", tc.subtype, err)
		}
	}
}

func TestPaidSacrificeSubtypeSourceAndStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	matched := BindNumber("matched")
	source := NewCreature("Self", "", 2, 2, WithSubTypes("Thrull"), WithAbility(Activate(
		Costs(SacrificeCreatureCostWithSubtype("Thrull", matched)),
		IfPositive(FromNumber(matched), GainLifeFor(You, Fixed(2))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("source was not sacrificed")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("source payment subtype was not kept in both stack frames")
	}
}

func TestPaidSacrificeSubtypeSpellAndResolutionCost(t *testing.T) {
	paid := BindNumber("paid subtype")
	spell := NewInstant("Offering", "", WithAbility(Spell(
		IfPositive(FromNumber(paid), GainLifeFor(You, Fixed(1))))),
		WithAdditionalCost(SacrificeCreatureCostWithSubtype("Thrull", paid)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatalf("spell additional cost: %v", err)
	}
	invalid := NewInstant("Invalid", "", WithAbility(Spell(
		IfPaid(You, Costs(SacrificeCreatureCostWithSubtype("Thrull", paid)), GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil || !strings.Contains(err.Error(), "activated ability cost or spell additional cost") {
		t.Fatalf("resolution payment validation = %v", err)
	}
}
