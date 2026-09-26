package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPaidLifeAmountActivationCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("life paid")
	source := NewArtifact("Life source", "", WithAbility(Activate(
		Costs(LifePayCostInto(3, paid)), GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 17 {
		t.Fatalf("life after payment = %d", a.Life())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 23 || a.Life() != 17 {
		t.Fatal("payment result escaped or was lost by clone")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 23 {
		t.Fatal("payment result escaped its stack instance")
	}
}

func TestPaidLifeAmountBoundaryAndFailure(t *testing.T) {
	for _, tc := range []struct {
		amount, starting, want int
		fails                  bool
	}{
		{0, 0, 0, false},
		{3, 3, 3, false},
		{4, 3, 3, true},
	} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		g.PlayerLoseLife(a, 20-tc.starting)
		paid := BindNumber("paid")
		source := NewArtifact("Source", "", WithAbility(Activate(
			Costs(LifePayCostInto(tc.amount, paid)), GainLifeFor(You, FromNumber(paid)))))
		g.PutOnBattlefield(source, a.PlayerID())
		err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
		if (err != nil) != tc.fails {
			t.Fatalf("amount %d: activation = %v", tc.amount, err)
		}
		g.ResolveStack()
		if a.Life() != tc.want {
			t.Fatalf("amount %d: life = %d", tc.amount, a.Life())
		}
	}
}

func TestPaidLifeAmountDefinition(t *testing.T) {
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{LifePayCostInto(2, BindNumber("paid")), ""},
		{LifePayCostInto(2, nil), "non-nil"},
		{LifePayCostInto(-1, BindNumber("paid")), "negative"},
		{LifePayCostInto(2, BindNumber("")), "name"},
	} {
		paid := BindNumber("paid")
		if c, ok := tc.cost.(*lifePayCost); ok && c.paid != nil {
			paid = c.paid
		}
		card := NewArtifact("Source", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, FromNumber(paid)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
}

func TestPaidLifeAmountPendingActivationsAndSpell(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("life paid")
	target := DeclarePlayerTarget("recipient")
	source := NewArtifact("Source", "", WithAbility(Activate(
		Costs(LifePayCostInto(2, paid)), GainLifeFor(target, FromNumber(paid)), WithTargets(target))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 22 {
		t.Fatal("pending activations shared a payment frame")
	}
	spellPaid := BindNumber("spell life")
	spell := NewInstant("Spell", "", WithAbility(Spell(GainLifeFor(You, FromNumber(spellPaid)))), WithAdditionalCost(LifePayCostInto(3, spellPaid)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 18 {
		t.Fatalf("spell did not read its life payment: %d", a.Life())
	}
}

func TestPaidLifeAmountInvalidDeclarations(t *testing.T) {
	b := BindNumber("paid")
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"duplicate", NewArtifact("Duplicate", "", WithAbility(Activate(Costs(LifePayCostInto(1, b), LifePayCostInto(1, b)), GainLifeFor(You, FromNumber(b))))), "duplicate"},
		{"branch", NewArtifact("Branch", "", WithAbility(Activate(Costs(), IfPaid(You, Costs(LifePayCostInto(1, b)), GainLifeFor(You, FromNumber(b)))))), "requires an activated ability"},
		{"unbound", NewArtifact("Unbound", "", WithAbility(Activate(Costs(), GainLifeFor(You, FromNumber(b))))), "producer"},
	} {
		err := ValidateCardDefinition(tc.card.Name(), tc.card)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: validation = %v", tc.name, err)
		}
	}
}

func TestPaidLifeAmountTotalCostFailure(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	source := NewArtifact("Source", "", WithAbility(Activate(
		Costs(LifePayCostInto(3, paid), LifePayCost(100)), GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("unpayable total cost was accepted")
	}
	if a.Life() != 20 || len(g.StackObjects()) != 0 {
		t.Fatal("failed total cost changed life or stack")
	}
}
