package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticSpellCostIncreasePayment(t *testing.T) {
	for _, controllerOnly := range []bool{false, true} {
		for _, opponent := range []bool{false, true} {
			for _, alternate := range []bool{false, true} {
				g, a, b := newActivationDivisionGame()
				var caster Player = a
				if opponent {
					caster = b
				}
				g.PutOnBattlefield(NewEnchantment("Tax", "", WithStaticAbility(IncreaseSpellCostStatic(SpellAny(), core.ManaCost{Generic: 2, Black: 1}, controllerOnly))), a.PlayerID())
				spell := NewInstant("Probe", "{X}{U}", Spell(GainLifeFor(You, Fixed(1))), WithAdditionalCost(GenericCost(1)))
				spell.SetOwner(caster.PlayerID())
				if alternate {
					caster.AddToGraveyard(spell)
				} else {
					caster.AddToHand(spell)
				}
				g.PutOnBattlefield(NewArtifact("Reduction", "", WithStaticAbility(ReduceSpellCostStatic(SpellAny(), FixedAmount(2), nil))), caster.PlayerID())
				taxed := !controllerOnly || !opponent
				caster.ManaPool().Add(core.Blue, 1)
				generic := 1
				if taxed {
					generic += 2
				}
				caster.ManaPool().Add(core.Colorless, generic)
				cast := func() error {
					if alternate {
						return g.CastCardFromZoneWithAlternateCost(caster.PlayerID(), spell.ID(), core.ZoneGraveyard, spell.ManaCost(), nil, 2)
					}
					return g.CastSpellByID(caster.PlayerID(), spell.ID(), nil, 2)
				}
				if taxed {
					if err := cast(); err == nil {
						t.Fatal("generic mana paid the colored increase")
					}
					if caster.ManaPool().TotalMana() != generic+1 || g.StackSize() != 0 {
						t.Fatal("failed payment changed resources")
					}
					caster.ManaPool().Add(core.Black, 1)
				}
				if err := cast(); err != nil {
					t.Fatal(err)
				}
				if caster.ManaPool().TotalMana() != 0 || g.StackPeek().XValue != 2 {
					t.Fatal("incorrect increased total or announced X")
				}
				if g.CopyStackObjectDirect(g.StackPeek(), caster.PlayerID(), false) == nil {
					t.Fatal("copy required another payment")
				}
				g.ResolveStack()
				if caster.Life() != 22 {
					t.Fatal("spell and copy did not resolve")
				}
			}
		}
	}
}

func TestStaticSpellCostIncreaseLifecycle(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	card := NewArtifact("Tax", "", WithStaticAbility(IncreaseSpellCostStatic(SpellAny(), core.ManaCost{Black: 1}, true)))
	card.SetOwner(b.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	probe := NewInstant("Probe", "{U}")
	check := func(game *Game, caster uuid.UUID, want int) {
		t.Helper()
		total, _ := game.modifySpellTotalCost(spellPaymentSpec{Card: probe, Controller: caster}, probe.ManaCost())
		if total.Black != want || total.Blue != 1 {
			t.Fatalf("cost = %v, want B=%d U=1", total, want)
		}
	}
	check(g, a.PlayerID(), 1)
	check(g, b.PlayerID(), 0)
	clone := g.Clone()
	g.PhaseOut(source.ID())
	check(g, a.PlayerID(), 0)
	check(clone, a.PlayerID(), 1)
	g.PhaseIn([]uuid.UUID{source.ID()})
	check(g, a.PlayerID(), 1)
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	check(g, a.PlayerID(), 0)
	check(g, b.PlayerID(), 1)
	remover := g.PutOnBattlefield(NewArtifact("Remover", ""), a.PlayerID())
	removal := RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(remover.ID())
	g.AddContinuousEffect(removal)
	check(g, b.PlayerID(), 0)
	g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
	check(g, b.PlayerID(), 1)
	saved := g.effects.Rules.spellCostTaxes[0]
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	check(g, b.PlayerID(), 0)
	g.PutOnBattlefield(card, a.PlayerID())
	check(g, a.PlayerID(), 1)
	if saved.applies(g, a.PlayerID(), probe) || saved.applies(g, b.PlayerID(), probe) {
		t.Fatal("returned source restored the old tax")
	}
	check(clone, a.PlayerID(), 1)
}

func TestStaticSpellCostIncreaseDeclarations(t *testing.T) {
	for _, cost := range []core.ManaCost{{HasX: true, XCount: 1}, {XCount: 1}, {Generic: -1}, {White: -1}, {Blue: -1}, {Black: -1}, {Red: -1}, {Green: -1}, {Hybrid: []core.HybridSymbol{{}}}} {
		card := NewArtifact("Invalid", "", WithStaticAbility(IncreaseSpellCostStatic(SpellAny(), cost, true)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted invalid increase %v", cost)
		}
	}
	card := NewArtifact("Invalid", "", WithStaticAbility(IncreaseSpellCostStatic(nil, core.ManaCost{Black: 1}, true)))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("accepted nil filter")
	}
}

func TestStaticSpellCostIncreaseStackingAndLock(t *testing.T) {
	for _, sacrificeSource := range []bool{false, true} {
		g, a, b := newActivationDivisionGame()
		filter := SpellsOr(func(_ *Game, _ uuid.UUID, c Card, _ uuid.UUID) bool { return HasColorCardFilter(core.Black).Match(c) }, func(_ *Game, _ uuid.UUID, c Card, _ uuid.UUID) bool { return HasColorCardFilter(core.White).Match(c) })
		tax := g.PutOnBattlefield(NewCreature("Tax", "", 1, 1, WithStaticAbility(IncreaseSpellCostStatic(filter, core.ManaCost{Generic: 2, Black: 1}, false))), b.PlayerID())
		if sacrificeSource {
			g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: tax.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
		}
		g.PutOnBattlefield(NewArtifact("Second tax", "", WithStaticAbility(IncreaseSpellCostStatic(filter, core.ManaCost{White: 1}, false))), b.PlayerID())
		spell := NewInstant("Multicolor", "{W}{B}", Spell(GainLifeFor(You, Fixed(1))))
		if sacrificeSource {
			WithAdditionalCost(SacrificeNMatchingIncludingSourceCost(1, IsCreature, "Sacrifice a creature"))(spell)
		}
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		a.ManaPool().Add(core.White, 2)
		a.ManaPool().Add(core.Black, 2)
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil {
			t.Fatal("underpaid stacked taxes")
		}
		if a.ManaPool().TotalMana() != 5 || g.FindPermanent(tax.ID()) == nil || g.StackSize() != 0 {
			t.Fatal("tax failure committed sacrifice or mana")
		}
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if a.ManaPool().TotalMana() != 0 || (g.FindPermanent(tax.ID()) == nil) != sacrificeSource {
			t.Fatal("tax counted twice for multicolor or changed during payment")
		}
	}
}
