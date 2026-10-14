package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCharacteristicAdditiveScopesAndZones(t *testing.T) {
	g, a, b := characteristicGame()
	for _, tc := range []struct {
		name   string
		effect ContinuousEffect
		want   int
	}{
		{"all", PTEqualsCountPlus(IsLand, 2), 6},
		{"controlled", PTEqualsControlledCountPlus(IsLand, -1), 2},
		{"opponents", PTEqualsOpponentCountPlus(IsLand, 1), 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCreature("Sum", "", 1, 1, WithStaticAbility(tc.effect))
			c.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			for range 3 {
				assertCharacteristicPT(t, g, c, tc.want, tc.want)
			}
			if c.Power() != 1 || c.Toughness() != 1 || g.StackSize() != 0 || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("query changed state or printed values")
			}
		})
	}
	c := NewCreature("Opponent sum", "", 1, 1, WithStaticAbility(PTEqualsOpponentCountPlus(IsLand, 1)))
	c.SetOwner(a.PlayerID())
	a.AddToLibrary(c)
	assertCharacteristicPT(t, g, c, 2, 2)
	a.SetLibrary(nil)
	a.AddToHand(c)
	assertCharacteristicPT(t, g, c, 2, 2)
	a.RemoveFromHand(c.ID())
	a.AddToGraveyard(c)
	assertCharacteristicPT(t, g, c, 2, 2)
	g.MoveFromGraveyard(a.PlayerID(), c.ID(), core.ZoneExile)
	g.ExileCard(c, c.ID())
	assertCharacteristicPT(t, g, c, 2, 2)
	g.RemoveFromExile(c.ID())
	p := g.PutOnBattlefield(c, b.PlayerID())
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, c, 4, 4)
	g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	assertCharacteristicPT(t, g, c, 2, 2)
	ids := g.PhaseOut(g.FilterBattlefield(And(IsLand, ControlledBy(b.PlayerID())))[0].ID())
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, c, 1, 1)
	g.PhaseIn(ids)
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, c, 2, 2)
	unknown := c.Copy()
	unknown.SetOwner(uuid.New())
	assertCharacteristicPT(t, g, unknown, 1, 1)
}

func TestCharacteristicAdditiveLayersAndIdentity(t *testing.T) {
	g, a, b := characteristicGame()
	c := NewCreature("Sum", "", 1, 1, WithStaticAbility(PTEqualsOpponentCountPlus(IsLand, 1)))
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, c, 2, 2)
	var old ContinuousEffect
	for _, effect := range g.effects.effects {
		if effect.SourceID() == p.ID() {
			old = effect
			break
		}
	}
	if old == nil {
		t.Fatal("missing bound effect")
	}
	g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, c, 1, 1)
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, c, 2, 2)
	g.AddContinuousEffect(TargetEffectWithSublayer(core.LayerPT, core.SublayerPT_Set, core.EndOfTurn, p.ID(), func(_ *Game, p *Permanent) error { p.BasePTOverride = &[2]int{2, 5}; return nil }))
	g.AddContinuousEffect(TargetEffectWithSublayer(core.LayerPT, core.SublayerPT_Modify, core.EndOfTurn, p.ID(), func(_ *Game, p *Permanent) error { p.BoostPT(1, 2); return nil }))
	g.AddContinuousEffect(TargetEffectWithSublayer(core.LayerPT, core.SublayerPT_Switch, core.EndOfTurn, p.ID(), func(_ *Game, p *Permanent) error { p.SwitchPowerToughness(); return nil }))
	g.AddCountersWithReplacement(g.FindPermanent(p.ID()), core.P1P1, 1, p.ID(), false)
	for range 2 {
		g.ApplyContinuousEffects()
		assertCharacteristicPT(t, g, c, 8, 4)
	}
	ids := g.PhaseOut(p.ID())
	assertCharacteristicPT(t, g, c, 8, 4)
	g.PhaseIn(ids)
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	a.RemoveFromHand(c.ID())
	g.PutOnBattlefield(c, b.PlayerID())
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, c, 4, 4)
	if old.IsActive(g) {
		t.Fatal("return restored old effect")
	}
	if err := old.ApplyContinuous(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	assertCharacteristicPT(t, g, c, 4, 4)
	clone := g.Clone()
	clone.PutOnBattlefield(NewLand("Extra"), a.PlayerID())
	clone.ApplyContinuousEffects()
	assertCharacteristicPT(t, clone, c, 5, 5)
	assertCharacteristicPT(t, g, c, 4, 4)
}
