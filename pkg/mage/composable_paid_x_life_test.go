package mage

import (
	"strings"
	"testing"
)

func TestPaidXLifeSpellBoundaryAndRollback(t *testing.T) {
	for _, tc := range []struct {
		x, life, want int
		fails         bool
	}{
		{0, 0, 0, false},
		{3, 3, 3, false},
		{4, 3, 3, true},
	} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		g.PlayerLoseLife(a, 20-tc.life)
		paid := BindNumber("paid")
		spell := NewInstant("Variable life", "", WithAbility(Spell(GainLifeFor(You, FromNumber(paid)))), WithAdditionalCost(LifePayXCostInto(paid)))
		if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
			t.Fatal(err)
		}
		a.AddToHand(spell)
		err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil, tc.x)
		if (err != nil) != tc.fails {
			t.Fatalf("X=%d: cast = %v", tc.x, err)
		}
		if tc.fails && (len(g.StackObjects()) != 0 || len(a.Hand()) != 1) {
			t.Fatal("failed payment changed stack or hand")
		}
		g.ResolveStack()
		if a.Life() != tc.want {
			t.Fatalf("X=%d: life = %d, want %d", tc.x, a.Life(), tc.want)
		}
	}
}

func TestPaidXLifeSeparateInstances(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("paid")
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(LifePayXCostInto(paid)), GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.SetXValue(2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.SetXValue(3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 15 {
		t.Fatalf("life after payment = %d", a.Life())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 22 || a.Life() != 15 {
		t.Fatal("clone or copy lost instance payment")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 22 {
		t.Fatal("pending payments shared state")
	}
}

func TestPaidXLifeRegistration(t *testing.T) {
	paid := BindNumber("paid")
	for _, tc := range []struct {
		card Card
		want string
	}{
		{NewInstant("Nil", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)))), WithAdditionalCost(LifePayXCostInto(nil))), "non-nil"},
		{NewInstant("Unnamed", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)))), WithAdditionalCost(LifePayXCostInto(BindNumber("")))), "name"},
		{NewInstant("Duplicate", "", WithAbility(Spell(GainLifeFor(You, FromNumber(paid)))), WithAdditionalCost(LifePayXCostInto(paid)), WithAdditionalCost(LifePayXCostInto(paid))), "duplicate"},
		{NewInstant("Unbound", "", WithAbility(Spell(GainLifeFor(You, FromNumber(paid))))), "producer"},
	} {
		err := ValidateCardDefinition(tc.card.Name(), tc.card)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: %v", tc.card.Name(), err)
		}
	}
}
