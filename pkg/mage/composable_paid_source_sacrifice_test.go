package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidSourceSacrificeStats(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	power, toughness := BindNumber("power"), BindNumber("toughness")
	source := NewCreature("Offering", "", 2, 3, WithAbility(Activate(
		Costs(SacrificeCreatureIncludingSourceCostWithStats(power, toughness)),
		GainLifeFor(You, FromNumber(toughness)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(core.P1P1, 2)
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("the source was not sacrificed")
	}
	g.ResolveStack()
	if a.Life() != 25 {
		t.Fatalf("life = %d, want 25", a.Life())
	}
}

func TestPaidSourceSacrificeStatsRollbackAndDefinition(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	power, toughness := BindNumber("power"), BindNumber("toughness")
	source := NewCreature("Offering", "", 2, 3)
	g.PutOnBattlefield(source, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: a.PlayerID(), SourceID: source.ID(),
		Costs: []Cost{SacrificeCreatureIncludingSourceCostWithStats(power, toughness)},
	})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || g.FindPermanent(source.ID()) == nil {
		t.Fatal("returned source paid a stale cost")
	}
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{SacrificeCreatureIncludingSourceCostWithStats(power, toughness), ""},
		{SacrificeCreatureIncludingSourceCostWithStats(nil, toughness), "non-nil"},
		{SacrificeCreatureIncludingSourceCostWithStats(power, power), "duplicate"},
	} {
		card := NewCreature("Definition", "", 1, 1, WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	missing := BindNumber("missing")
	invalid := NewCreature("Missing", "", 1, 1, WithAbility(Activate(
		Costs(SacrificeCreatureIncludingSourceCostWithStats(power, toughness)),
		GainLifeFor(You, FromNumber(missing)),
	)))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("unbound paid value validation = %v", err)
	}
	spell := NewInstant("Additional", "", WithAbility(Spell(GainLifeFor(You, FromNumber(power)))),
		WithAdditionalCost(SacrificeCreatureIncludingSourceCostWithStats(power, toughness)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatalf("spell additional cost validation = %v", err)
	}
}

func TestPaidSourceSacrificeStatsCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("power"), BindNumber("toughness")
	source := NewCreature("Offering", "", 3, 2, WithAbility(Activate(
		Costs(SacrificeCreatureIncludingSourceCostWithStats(power, toughness)),
		GainLifeFor(You, FromNumber(power)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || clone.GetPlayer(b.PlayerID()).Life() != 23 || a.Life() != 20 {
		t.Fatal("clone lost the paid source value")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 23 {
		t.Fatal("stack copy lost the paid source value")
	}
}
