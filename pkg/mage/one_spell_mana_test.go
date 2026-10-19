package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type oneSpellManaPlayer struct {
	*BasePlayer
	accept bool
	offers int
}

func (p *oneSpellManaPlayer) ChooseMayAbility(string) bool { p.offers++; return p.accept }

func oneSpellManaGame(t *testing.T) (*Game, *oneSpellManaPlayer) {
	t.Helper()
	a := &oneSpellManaPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	if err := SpendAnyManaForOneSpellThisTurn(You).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	return g, a
}

func TestOneSpellManaPartition(t *testing.T) {
	for _, tax := range []bool{false, true} {
		t.Run(map[bool]string{false: "additional", true: "tax"}[tax], func(t *testing.T) {
			g, a := oneSpellManaGame(t)
			c := NewInstant("Probe", "{U}", Spell(GainLifeFor(You, Fixed(1))))
			if tax {
				g.PutOnBattlefield(NewEnchantment("Tax", "", WithStaticAbility(IncreaseSpellCostStatic(SpellAny(), core.ManaCost{Red: 1}, false))), a.PlayerID())
			} else {
				WithAdditionalCost(ManaCostOf("{R}"))(c)
			}
			a.AddToHand(c)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
				t.Fatal("permission paid additional colored mana")
			}
			if a.ManaPool().TotalMana() != 2 || len(a.Hand()) != 1 {
				t.Fatal("failed cast changed resources")
			}
			a.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().LastDrainedColors[core.Colorless] != 1 || a.ManaPool().LastDrainedColors[core.Red] != 1 {
				t.Fatal("wrong actual spending")
			}
			g.ResolveStack()
			d := NewInstant("Second", "{G}", Spell(GainLifeFor(You, Fixed(1))))
			a.AddToHand(d)
			if err := g.CastSpellByID(a.PlayerID(), d.ID(), nil, 0); err == nil {
				t.Fatal("permission used twice")
			}
		})
	}
}

func TestOneSpellManaDeclineCloneAndExpiry(t *testing.T) {
	g, a := oneSpellManaGame(t)
	c := NewInstant("Probe", "{U}", Spell(GainLifeFor(You, Fixed(1))))
	a.AddToHand(c)
	a.ManaPool().Add(core.Colorless, 1)
	a.accept = false
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
		t.Fatal("decline ignored")
	}
	a.accept = true
	clone := g.Clone()
	clone.players[0] = &oneSpellManaPlayer{BasePlayer: clone.GetPlayer(a.PlayerID()).(*SearchPlayer).BasePlayer, accept: true}
	for _, game := range []*Game{g, clone} {
		if err := game.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	g, a = oneSpellManaGame(t)
	g.RunStepWithPriority(core.Cleanup)
	c = NewInstant("Expired", "{U}", Spell(GainLifeFor(You, Fixed(1))))
	a.AddToHand(c)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
		t.Fatal("permission survived cleanup")
	}
}

func TestOneSpellManaDefinitions(t *testing.T) {
	c := NewInstant("Invalid", "", Spell(SpendAnyManaForOneSpellThisTurn(nil)))
	if err := ValidateCardDefinition(c.Name(), c); err == nil {
		t.Fatal("nil player accepted")
	}
}

func TestOneSpellManaReductionAndRestrictions(t *testing.T) {
	g, a := oneSpellManaGame(t)
	g.PutOnBattlefield(NewArtifact("Reduction", "", WithStaticAbility(ReduceSpellCostStatic(SpellAny(), FixedAmount(2), nil))), a.PlayerID())
	c := NewInstant("Probe", "{1}{U}{U}", Spell(GainLifeFor(You, Fixed(1))))
	a.AddToHand(c)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
		t.Fatal("generic reduction removed a colored symbol")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g, a = oneSpellManaGame(t)
	c = NewInstant("Restricted", "{U}", Spell(GainLifeFor(You, Fixed(1))))
	a.AddToHand(c)
	a.ManaPool().AddRestricted(core.Colorless, 1, core.ArtifactSpellsOnly{})
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
		t.Fatal("permission removed a spending restriction")
	}
	artifact := NewArtifact("Eligible", "{U}")
	a.AddToHand(artifact)
	if err := g.CastSpellByID(a.PlayerID(), artifact.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
}

func TestOneSpellManaMultipleAndAlternate(t *testing.T) {
	g, a := oneSpellManaGame(t)
	if err := SpendAnyManaForOneSpellThisTurn(You).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		c := NewInstant("Probe", "{U}", Spell(GainLifeFor(You, Fixed(1))))
		a.AddToHand(c)
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.oneSpellManaPermissions) != 0 {
		t.Fatal("distinct permissions were not consumed")
	}
	g, a = oneSpellManaGame(t)
	c := NewInstant("Alternate", "{U}", Spell(GainLifeFor(You, Fixed(1))))
	a.AddToHand(c)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastCardFromZoneWithAlternateCost(a.PlayerID(), c.ID(), core.ZoneHand, core.ManaCost{Red: 1}, nil, 0); err == nil {
		t.Fatal("permission paid an alternative cost")
	}
	if len(g.oneSpellManaPermissions) != 1 || a.offers != 0 {
		t.Fatal("alternative cast consumed or offered permission")
	}
}

func TestOneSpellManaStaleAndNonManaCost(t *testing.T) {
	g, a := oneSpellManaGame(t)
	c := NewInstant("Probe", "{U}", Spell(GainLifeFor(You, Fixed(1))))
	a.AddToHand(c)
	a.ManaPool().Add(core.Colorless, 1)
	tx, err := g.prepareSpellPaymentTransaction(spellPaymentSpec{Card: c, Controller: a.PlayerID(), Zone: core.ZoneHand, Mana: c.ManaCost(), PayingManaCost: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(g.oneSpellManaPermissions) != 1 || a.ManaPool().TotalMana() != 1 {
		t.Fatal("preparation spent resources")
	}
	g.oneSpellManaPermissions = nil
	if err := tx.Commit(); err == nil || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 1 {
		t.Fatal("stale permission changed resources")
	}
	g, a = oneSpellManaGame(t)
	c = NewInstant("Cost", "{U}", Spell(GainLifeFor(You, Fixed(1))), WithAdditionalCost(LifePayCost(21)))
	a.AddToHand(c)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil || a.Life() != 20 || a.ManaPool().TotalMana() != 1 || len(g.oneSpellManaPermissions) != 1 || len(a.Hand()) != 1 {
		t.Fatal("unpayable total changed resources")
	}
}

func TestOneSpellManaAutomaticHybridAndX(t *testing.T) {
	for _, cost := range []string{"{W/U}{G}", "{X}{1}{B}"} {
		t.Run(cost, func(t *testing.T) {
			g, a := oneSpellManaGame(t)
			c := NewInstant("Probe", cost, Spell(GainLifeFor(You, Fixed(1))))
			x := 0
			if c.ManaCost().HasX {
				c = restrictedXSpell()
				x = 2
			}
			a.AddToHand(c)
			for range 2 + x {
				g.PutOnBattlefield(NewLand("Source", WithManaAbility(core.Colorless)), a.PlayerID())
			}
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, x); err != nil {
				t.Fatal(err)
			}
			for _, p := range g.zones.battlefield {
				if !p.Tapped {
					t.Fatal("solver did not activate enough sources")
				}
			}
			if a.ManaPool().LastDrainedColors[core.Colorless] != 2+x || c.ManaCost().Colors()[0] == core.Colorless {
				t.Fatal("payment changed printed cost or actual spending")
			}
			if g.StackPeek().XValue != x {
				t.Fatal("payment changed announced X")
			}
			if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("spell copy failed")
			}
			if len(g.oneSpellManaPermissions) != 0 {
				t.Fatal("copy installed a permission")
			}
		})
	}
}

func TestOneSpellManaOtherPlayerAndActivation(t *testing.T) {
	g, a := oneSpellManaGame(t)
	b := g.players[1]
	c := NewInstant("Other", "{U}", Spell(GainLifeFor(You, Fixed(1))))
	b.AddToHand(c)
	b.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(b.PlayerID(), c.ID(), nil, 0); err == nil {
		t.Fatal("another player used permission")
	}
	p := g.PutOnBattlefield(NewArtifact("Activation", "", WithAbility(Activate(Costs(ManaCostOf("{U}")), GainLifeFor(You, Fixed(1))))), a.PlayerID())
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("activation used spell permission")
	}
	if len(g.oneSpellManaPermissions) != 1 {
		t.Fatal("failed actions consumed permission")
	}
}

func TestOneSpellManaQueries(t *testing.T) {
	g, a := oneSpellManaGame(t)
	c := NewInstant("Probe", "{X}{U}", Spell(GainLifeFor(You, Fixed(1))))
	a.AddToHand(c)
	a.ManaPool().Add(core.Colorless, 3)
	if !g.CanPaySpellManaCost(a.PlayerID(), c, 2) || g.CanPaySpellManaCost(a.PlayerID(), c, 3) {
		t.Fatal("wrong mana-cost affordability")
	}
	if len(g.GetCastableSpells(a.PlayerID())) != 1 {
		t.Fatal("permission-funded spell was hidden")
	}
	if a.offers != 0 || a.ManaPool().TotalMana() != 3 || len(g.oneSpellManaPermissions) != 1 {
		t.Fatal("query made a choice or spent resources")
	}
	if g.CanPaySpellManaCost(g.players[1].PlayerID(), c, 0) || g.CanPaySpellManaCost(a.PlayerID(), nil, 0) || g.CanPaySpellManaCost(a.PlayerID(), c, -1) {
		t.Fatal("invalid query accepted")
	}
}

func TestOneSpellManaAddedXReduction(t *testing.T) {
	g, a := oneSpellManaGame(t)
	c := restrictedXSpell()
	WithAdditionalCost(ManaCostOf("{X}"))(c)
	g.PutOnBattlefield(NewArtifact("Reduction", "", WithStaticAbility(ReduceSpellCostStatic(SpellAny(), FixedAmount(1), nil))), a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.Colorless, 4)
	a.ManaPool().Add(core.Black, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 2); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().LastDrainedColors[core.Black] != 1 || a.ManaPool().LastDrainedColors[core.Colorless] != 4 || g.StackPeek().XValue != 2 {
		t.Fatal("reduction did not remove added restricted X first")
	}
}
