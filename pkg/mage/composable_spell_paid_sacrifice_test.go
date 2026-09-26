package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellPaidSacrificeStats(t *testing.T) {
	for _, alternate := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		power, toughness := BindNumber("power"), BindNumber("toughness")
		card := NewInstant("Offering", "", WithAbility(Spell(Sequence(
			GainLifeFor(You, FromNumber(power)), GainLifeFor(You, FromNumber(toughness)),
		))), WithAdditionalCost(SacrificeCreatureCostWithStats(power, toughness)))
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		creature := NewCreature("Creature", "", 2, 3)
		g.PutOnBattlefield(creature, a.PlayerID())
		g.MutablePermanent(creature.ID()).AddCounter(core.P1P1, 1)
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		var err error
		if alternate {
			err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), card.ID(), core.ZoneHand, core.ManaCost{}, nil, 0)
		} else {
			err = g.CastSpellByName(a.PlayerID(), card.Name(), nil)
		}
		if err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(creature.ID()) != nil {
			t.Fatal("payment did not sacrifice creature")
		}
		g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
		clone := g.Clone()
		clone.ResolveStack()
		if clone.GetPlayer(a.PlayerID()).Life() != 27 || clone.GetPlayer(b.PlayerID()).Life() != 27 {
			t.Fatalf("clone or copy lost paid values: A=%d B=%d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life())
		}
		g.ResolveStack()
		if a.Life() != 27 || b.Life() != 27 {
			t.Fatalf("spell stack frames shared paid values: A=%d B=%d", a.Life(), b.Life())
		}
	}
}

func TestSpellPaidSacrificeDefinition(t *testing.T) {
	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"valid", SacrificeCreatureCostWithStats(BindNumber("power"), BindNumber("toughness")), ""},
		{"nil binding", SacrificeCreatureCostWithStats(nil, BindNumber("toughness")), "non-nil"},
		{"duplicate binding", SacrificeCreatureCostWithStats(BindNumber("same"), BindNumber("same")), "duplicate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewInstant("Offering", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)))), WithAdditionalCost(tc.cost))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

func TestSpellPaidSacrificeFailureKeepsResources(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("power"), BindNumber("toughness")
	card := NewInstant("Offering", "", WithAbility(Spell(GainLifeFor(You, FromNumber(power)))), WithAdditionalCost(SacrificeCreatureCostWithStats(power, toughness)))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err == nil {
		t.Fatal("cast without creature succeeded")
	}
	if len(a.Hand()) != 1 || !g.stack.IsEmpty() || a.Life() != 20 {
		t.Fatal("failed cast changed resources")
	}
}

func TestSpellPaidSacrificePendingSpells(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("power"), BindNumber("toughness")
	for _, tc := range []struct {
		name  string
		power int
	}{
		{"First", 2}, {"Second", 5},
	} {
		card := NewInstant(tc.name, "", WithAbility(Spell(GainLifeFor(You, FromNumber(power)))), WithAdditionalCost(SacrificeCreatureCostWithStats(power, toughness)))
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		g.PutOnBattlefield(NewCreature(tc.name+" creature", "", tc.power, 1), a.PlayerID())
		if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if a.Life() != 27 {
		t.Fatalf("life = %d, want separate paid powers", a.Life())
	}
}

func TestSpellPaidSacrificeModalBody(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("power"), BindNumber("toughness")
	card := NewInstant("Modal offering", "", WithAbility(SpellModes(
		ModeOf("power", GainLifeFor(You, FromNumber(power))),
		ModeOf("toughness", GainLifeFor(You, FromNumber(toughness))),
	)), WithAdditionalCost(SacrificeCreatureCostWithStats(power, toughness)))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	g.PutOnBattlefield(NewCreature("Creature", "", 2, 4), a.PlayerID())
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("life = %d, want chosen mode's paid power", a.Life())
	}
}
