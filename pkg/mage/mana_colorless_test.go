package mage

import (
	"slices"
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestColorlessSpellPaymentAndStackManaValue(t *testing.T) {
	for _, name := range []string{"parsed", "card definition"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := newModalGame()
			spell := NewInstant("Colorless spell", "{C}")
			if name == "card definition" {
				definition, err := NewCardDefinition(CardDefinitionSpec{
					Name:     "Colorless spell",
					TypeLine: TypeLine{CardTypes: []CardType{TypeInstant}},
					ManaCost: ManaCost{Colorless: 1},
				})
				if err != nil {
					t.Fatal(err)
				}
				spell = definition.NewInstance(a.PlayerID())
			}
			a.AddToHand(spell)
			a.ManaPool().Add(Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil {
				t.Fatal("cast succeeded with colored mana only")
			}
			if a.ManaPool().Count(Blue) != 1 || !slices.Contains(a.Hand(), spell) || !g.Stack().IsEmpty() {
				t.Fatal("rejected cast changed the pool, hand, or stack")
			}
			wastes := g.PutOnBattlefield(NewLand("Wastes", WithManaAbility(Colorless)), a.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if !wastes.Tapped || a.ManaPool().Count(Blue) != 1 || a.ManaPool().Count(Colorless) != 0 {
				t.Fatal("cast did not use the colorless mana source")
			}
			b.ManaPool().Add(Blue, 2)
			counter := NewInstant("Counter", "{X}{U}",
				NewTargetedSpell(TargetSpellOnStack(ManaValueIs(EqualTo(XValue()))), CounterSpell()))
			options := g.SpellTargetOptions(b.PlayerID(), counter)
			if len(options) != 1 || options[0].XValue != 1 || !slices.Equal(options[0].Candidates[0], []uuid.UUID{spell.ID()}) {
				t.Fatalf("colorless spell requires X=1, got %+v", options)
			}
		})
	}
}

func TestColorlessManaPaymentWithConversions(t *testing.T) {
	pool := NewManaPool()
	pool.ManaConversions = map[Color]Color{Red: White}
	cost := ManaCost{Colorless: 1, White: 1}
	pool.Add(Blue, 1)
	pool.Add(Red, 1)
	if pool.CanPay(cost, nil) {
		t.Fatal("accepted blue mana for a colorless requirement")
	}
	pool.Add(Colorless, 1)
	if err := pool.Pay(cost, nil); err != nil {
		t.Fatal(err)
	}
	if pool.TotalMana() != 1 || pool.Count(Blue) != 1 {
		t.Fatal("payment must consume red and colorless mana")
	}
}

func TestColorlessManaCostPerTarget(t *testing.T) {
	g, a, b := newModalGame()
	spell := NewInstant("Per-target spell", "{U}", NewSpell(
		GainLife(1),
		WithTarget(TargetPlayer()),
		WithTarget(TargetPlayer()),
		WithCost(ManaCostPerTarget("{C}")),
	))
	a.AddToHand(spell)
	a.ManaPool().Add(Blue, 2)
	a.ManaPool().Add(Colorless, 1)
	targets := []uuid.UUID{a.PlayerID(), b.PlayerID()}
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), targets, 0); err == nil {
		t.Fatal("cast succeeded with less than one colorless mana per target")
	}
	a.ManaPool().Add(Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), targets, 0); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 1 || a.ManaPool().Count(Blue) != 1 {
		t.Fatal("cast must consume two colorless mana and one blue mana")
	}
}

func TestAnyColorCastingPermissionRequiresColorlessMana(t *testing.T) {
	g, a, b := newModalGame()
	spell := NewInstant("Exiled spell", "{U}{C}")
	spell.SetOwner(b.PlayerID())
	g.ExileCard(spell, uuid.Nil())
	g.GrantCastFromExile(a.PlayerID(), spell.ID(), true)
	a.ManaPool().Add(Red, 2)
	if err := g.CastExiledCardWithPermission(a.PlayerID(), spell.ID(), nil, 0); err == nil {
		t.Fatal("permission to use any color paid a colorless requirement")
	}
	a.ManaPool().Add(Colorless, 1)
	if err := g.CastExiledCardWithPermission(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 1 || a.ManaPool().Count(Red) != 1 {
		t.Fatal("cast must consume one colorless mana and one red mana")
	}
}

func TestMaxXValueKeepsColorlessRequirement(t *testing.T) {
	g, a, _ := newModalGame()
	a.ManaPool().Add(Blue, 3)
	cost := ParseManaCost("{X}{C}")
	if x := g.MaxXValue(a.PlayerID(), cost, nil); x != 0 {
		t.Fatalf("maximum X with no colorless mana = %d, want 0", x)
	}
	a.ManaPool().Add(Colorless, 1)
	if x := g.MaxXValue(a.PlayerID(), cost, nil); x != 3 {
		t.Fatalf("maximum X = %d, want 3", x)
	}
}
