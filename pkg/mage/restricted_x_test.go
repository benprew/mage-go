package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func restrictedXSpell() *BaseCard {
	return NewInstant("Restricted X", "{X}{1}{B}", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, ValueAtResolution(XValue()))), WithXSpendingColor(core.Black))))
}

func TestRestrictedXPayment(t *testing.T) {
	for _, route := range []string{"hand", "alternate"} {
		for _, black := range []int{1, 2, 3} {
			t.Run(fmt.Sprintf("%s/black%d", route, black), func(t *testing.T) {
				g, a, _ := newActivationDivisionGame()
				c := restrictedXSpell()
				c.SetOwner(a.PlayerID())
				a.ManaPool().Add(core.Black, black)
				a.ManaPool().Add(core.Red, 4-black)
				if route == "hand" {
					a.AddToHand(c)
				} else {
					a.AddToGraveyard(c)
				}
				var err error
				if route == "hand" {
					err = g.CastSpellByID(a.PlayerID(), c.ID(), nil, 2)
				} else {
					err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), c.ID(), core.ZoneGraveyard, c.ManaCost(), nil, 2)
				}
				if (err == nil) != (black == 3) {
					t.Fatalf("payment: %v", err)
				}
				if err != nil {
					if a.ManaPool().TotalMana() != 4 || g.StackSize() != 0 || (len(a.Hand())+len(a.Graveyard())) != 1 {
						t.Fatal("failed payment changed resources")
					}
					return
				}
				cp := g.Clone()
				copied := cp.CopyStackObjectDirect(cp.StackPeek(), a.PlayerID(), false)
				if copied.XValue != 2 || cp.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
					t.Fatal("copy paid again or changed X")
				}
				g.ResolveStack()
				if a.Life() != 22 || cp.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("X or clone isolation")
				}
			})
		}
	}
}

func TestRestrictedXPlanning(t *testing.T) {
	g, a, _ := newActivationDivisionGame()
	c := restrictedXSpell()
	ctx := SpellContextForCard(c)
	a.ManaPool().Add(core.Black, 3)
	a.ManaPool().Add(core.Red, 4)
	if g.MaxXValue(a.PlayerID(), c.ManaCost(), ctx) != 2 || a.ManaPool().TotalMana() != 7 {
		t.Fatal("X bound or query mutation")
	}
	locked := *ctx
	locked.RestrictedGeneric = 2
	cost := core.ManaCost{Generic: 3, Black: 1}
	if !a.ManaPool().CanPay(cost, &locked) {
		t.Fatal("restricted plan")
	}
	if err := a.ManaPool().Pay(cost, &locked); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(core.Black) != 0 || a.ManaPool().Count(core.Red) != 3 {
		t.Fatal("wrong color allocation")
	}
	a.ManaPool().Clear()
	a.ManaPool().AddRestricted(core.Black, 3, core.ArtifactSpellsOnly{})
	a.ManaPool().Add(core.Red, 1)
	if a.ManaPool().CanPay(cost, &locked) {
		t.Fatal("spending restriction lost")
	}
	locked.IsArtifact = true
	if !a.ManaPool().CanPay(cost, &locked) {
		t.Fatal("eligible mana rejected")
	}
	locked.RestrictedGeneric = 4
	if a.ManaPool().CanPay(cost, &locked) {
		t.Fatal("invalid decomposition accepted")
	}
}

func TestRestrictedXAutomaticAndPermissions(t *testing.T) {
	g, a, _ := newActivationDivisionGame()
	permission := g.PutOnBattlefield(NewArtifact("Permission", "", WithStaticAbility(ControllerManaSpending(core.White, core.Black))), a.PlayerID())
	lands := make([]uuid.UUID, 0, 3)
	for range 3 {
		lands = append(lands, g.PutOnBattlefield(NewLand("White source", WithManaAbility(core.White)), a.PlayerID()).ID())
	}
	red := g.PutOnBattlefield(NewLand("Red source", WithManaAbility(core.Red)), a.PlayerID())
	c := restrictedXSpell()
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if g.MaxXValue(a.PlayerID(), c.ManaCost(), SpellContextForCard(c)) != 2 {
		t.Fatal("permission bound")
	}
	phased := g.PhaseOut(permission.ID())
	if g.MaxXValue(a.PlayerID(), c.ManaCost(), SpellContextForCard(c)) != 0 {
		t.Fatal("phased permission supplied restricted X")
	}
	g.PhaseIn(phased)
	if a.ManaPool().TotalMana() != 0 {
		t.Fatal("query produced mana")
	}
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 2); err != nil {
		t.Fatal(err)
	}
	for _, id := range lands {
		if !g.FindPermanent(id).Tapped {
			t.Fatal("missing white activation")
		}
	}
	if !g.FindPermanent(red.ID()).Tapped || a.ManaPool().LastDrainedColors[core.White] != 3 {
		t.Fatal("actual payment colors")
	}
}

func TestRestrictedXModifiers(t *testing.T) {
	for _, state := range []string{"zero", "reduction", "tax and reduction", "channel", "channel fail", "additional rollback"} {
		t.Run(state, func(t *testing.T) {
			g, a, _ := newActivationDivisionGame()
			c := restrictedXSpell()
			x := 2
			black, red, wantLife := 3, 1, 22
			switch state {
			case "zero":
				x = 0
				black = 1
				wantLife = 20
			case "reduction":
				g.effects.Rules.SpellCostReductions[core.Black] = 2
				black = 1
			case "tax and reduction":
				g.effects.Rules.SpellCostIncreases[core.Black] = 2
				g.effects.Rules.SpellCostReductions[core.Black] = 2
				black = 1
				red = 3
			case "channel":
				g.effects.Rules.SetChannelActive(a.PlayerID())
				red = 0
				wantLife = 21
			case "channel fail":
				g.effects.Rules.SetChannelActive(a.PlayerID())
				black = 1
				red = 0
			case "additional rollback":
				WithAdditionalCost(LifePayCost(30))(c)
			}
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			a.ManaPool().Add(core.Black, black)
			a.ManaPool().Add(core.Red, red)
			err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, x)
			fail := state == "channel fail" || state == "additional rollback"
			if (err != nil) != fail {
				t.Fatalf("cast: %v", err)
			}
			if fail {
				if a.Life() != 20 || a.ManaPool().TotalMana() != black+red || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("rollback")
				}
				return
			}
			g.ResolveStack()
			if a.Life() != wantLife || a.ManaPool().TotalMana() != 0 {
				t.Fatal("modifier payment or announced X")
			}
		})
	}
}

func TestRestrictedXDeclarations(t *testing.T) {
	for _, color := range []core.Color{core.Colorless, core.AnyColor, core.Color(99)} {
		c := NewInstant("Invalid", "{X}", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, Fixed(1))), WithXSpendingColor(color))))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid color")
		}
	}
	c := NewInstant("Duplicate", "{X}", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, Fixed(1))), WithXSpendingColor(core.Black), WithXSpendingColor(core.Black))))
	if ValidateCardDefinition(c.Name(), c) == nil {
		t.Fatal("duplicate restriction")
	}
	c = NewInstant("No X", "{1}", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, Fixed(1))), WithXSpendingColor(core.Black))))
	if ValidateCardDefinition(c.Name(), c) == nil {
		t.Fatal("missing X")
	}
	c = restrictedXSpell()
	WithSpellTextContinuation()(c.Abilities()[0].(*ActionDefinition))
	if ValidateCardDefinition(c.Name(), c) == nil {
		t.Fatal("unsupported text continuation")
	}
	c = NewArtifact("Activation", "", WithAbility(NewAction(ActionActivated, WithEffect(GainLifeFor(You, Fixed(1))), WithXSpendingColor(core.Black))))
	if ValidateCardDefinition(c.Name(), c) == nil {
		t.Fatal("activation restriction")
	}
}

func TestRestrictedXAlternateAnyColor(t *testing.T) {
	g, a, _ := newActivationDivisionGame()
	c := restrictedXSpell()
	c.SetOwner(a.PlayerID())
	a.AddToGraveyard(c)
	a.ManaPool().Add(core.Red, 4)
	cost := c.ManaCost()
	if err := g.castCardFromZoneOptsWithCosts(a.PlayerID(), c.ID(), core.ZoneGraveyard, nil, 2, &cost, false, false, nil, true); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().LastDrainedColors[core.Red] != 4 {
		t.Fatal("as-though payment changed actual color")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("announced X")
	}
}

func TestRestrictedXMultipleSymbolsAndCosts(t *testing.T) {
	for _, extra := range []bool{false, true} {
		g, a, _ := newActivationDivisionGame()
		c := NewInstant("Repeated X", "{X}{X}{1}{B}", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, ValueAtResolution(XValue()))), WithXSpendingColor(core.Black))))
		black := 5
		if extra {
			WithAdditionalCost(ManaCostOf("{X}"))(c)
			black += 2
		}
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		a.ManaPool().Add(core.Black, black)
		a.ManaPool().Add(core.Red, 1)
		if !extra && g.MaxXValue(a.PlayerID(), c.ManaCost(), SpellContextForCard(c)) != 2 {
			t.Fatal("repeated X bound")
		}
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 2); err != nil {
			t.Fatal(err)
		}
		if a.ManaPool().Count(core.Black) != 0 || a.ManaPool().Count(core.Red) != 0 {
			t.Fatal("repeated X payment")
		}
		g.ResolveStack()
		if a.Life() != 22 {
			t.Fatal("announced X changed")
		}
	}
}

func TestRestrictedXAutomaticRollback(t *testing.T) {
	g, a, _ := newActivationDivisionGame()
	black := g.PutOnBattlefield(NewLand("Black source", WithManaAbility(core.Black)), a.PlayerID())
	red := g.PutOnBattlefield(NewLand("Red source", WithManaAbility(core.Red)), a.PlayerID())
	c := restrictedXSpell()
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 2); err == nil {
		t.Fatal("insufficient black sources accepted")
	}
	if g.FindPermanent(black.ID()).Tapped || g.FindPermanent(red.ID()).Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("failed planning changed resources")
	}
}

func TestRestrictedXAnyColorProduction(t *testing.T) {
	g, a, _ := newActivationDivisionGame()
	g.PutOnBattlefield(NewLand("Any source", WithManaAbility(core.AnyColor)), a.PlayerID())
	a.ManaPool().Add(core.Black, 2)
	a.ManaPool().Add(core.Red, 1)
	c := restrictedXSpell()
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 2); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().LastDrainedColors[core.Black] != 3 {
		t.Fatal("choice demand ignored restriction")
	}
}

func TestRestrictedXReplacementsAndConversion(t *testing.T) {
	for _, color := range []core.Color{core.Black, core.Red} {
		g, a, _ := newActivationDivisionGame()
		land := g.PutOnBattlefield(NewLand("Replaced", WithMultiManaAbility(ManaProduction{Color: core.Green, Amount: 3})), a.PlayerID())
		if err := ApplyEffect(g, ReplaceTappedManaTypeForPlayerThisTurn(You, IsLand, color), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		a.ManaPool().Add(core.Red, 1)
		c := restrictedXSpell()
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 2)
		if (err == nil) != (color == core.Black) {
			t.Fatalf("replacement cast: %v", err)
		}
		if color == core.Red && (g.FindPermanent(land.ID()).Tapped || a.ManaPool().TotalMana() != 1) {
			t.Fatal("rejected production changed state")
		}
	}
	g, a, _ := newActivationDivisionGame()
	g.PutOnBattlefield(NewArtifact("Converter", "", WithAbility(Activate(Costs(ManaCostOf("{1}")), AddMana(core.Black, 1)))), a.PlayerID())
	a.ManaPool().Add(core.Black, 1)
	a.ManaPool().Add(core.Red, 3)
	c := restrictedXSpell()
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 2); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().LastDrainedColors[core.Black] != 3 || a.ManaPool().LastDrainedColors[core.Red] != 1 {
		t.Fatal("converter consumed final payment color")
	}
}

func TestRestrictedXHybridAllocation(t *testing.T) {
	g, a, _ := newActivationDivisionGame()
	ctx := SpellContextForCard(restrictedXSpell())
	ctx.RestrictedGeneric = 2
	cost := core.ManaCost{Generic: 3, Black: 1, Hybrid: []core.HybridSymbol{{A: core.White, B: core.Red}}}
	a.ManaPool().Add(core.Black, 3)
	a.ManaPool().Add(core.Red, 2)
	if !g.CanAfford(a.PlayerID(), cost, ctx) || ctx.RestrictedGeneric != 2 || a.ManaPool().TotalMana() != 5 {
		t.Fatal("hybrid query or context changed")
	}
	if err := a.ManaPool().Pay(cost, ctx); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 {
		t.Fatal("hybrid allocation")
	}
}

func TestRestrictedXAlternateWithoutX(t *testing.T) {
	for _, x := range []int{0, 2, -1} {
		g, a, _ := newActivationDivisionGame()
		c := restrictedXSpell()
		c.SetOwner(a.PlayerID())
		a.AddToGraveyard(c)
		a.ManaPool().Add(core.Red, 1)
		err := g.CastCardFromZoneWithAlternateCost(a.PlayerID(), c.ID(), core.ZoneGraveyard, core.ManaCost{Red: 1}, nil, x)
		if (err == nil) != (x == 0) {
			t.Fatalf("alternate X=%d: %v", x, err)
		}
		if err != nil && (a.ManaPool().TotalMana() != 1 || len(a.Graveyard()) != 1 || g.StackSize() != 0) {
			t.Fatal("illegal X changed resources")
		}
	}
}
