package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func characteristicGame() (*Game, *BasePlayer, *BasePlayer) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	for range 3 {
		g.PutOnBattlefield(NewLand("A land"), a.PlayerID())
	}
	g.PutOnBattlefield(NewLand("B land"), b.PlayerID())
	return g, a, b
}

func assertCharacteristicPT(t *testing.T, g *Game, c Card, power, toughness int) {
	t.Helper()
	p, q := g.CardPowerToughness(c)
	if p != power || q != toughness {
		t.Fatalf("P/T = %d/%d, want %d/%d", p, q, power, toughness)
	}
}

func TestCharacteristicPTZonesAndController(t *testing.T) {
	g, a, b := characteristicGame()
	c := NewCreature("Count", "", 0, 0, WithStaticAbility(PTEqualsControlledCount(IsLand)))
	c.SetOwner(a.PlayerID())
	assertCharacteristicPT(t, g, c, 3, 3)
	a.AddToLibrary(c)
	assertCharacteristicPT(t, g, c, 3, 3)
	a.SetLibrary(nil)
	a.AddToHand(c)
	assertCharacteristicPT(t, g, c, 3, 3)
	a.RemoveFromHand(c.ID())
	a.AddToGraveyard(c)
	assertCharacteristicPT(t, g, c, 3, 3)
	if _, ok := g.MoveFromGraveyard(a.PlayerID(), c.ID(), core.ZoneExile); !ok {
		t.Fatal("graveyard removal failed")
	}
	g.ExileCard(c, c.ID())
	if g.FindExiledCard(c.ID()) == nil {
		t.Fatal("card missing from exile")
	}
	assertCharacteristicPT(t, g, c, 3, 3)
	g.RemoveFromExile(c.ID())
	p := g.PutOnBattlefield(c, b.PlayerID())
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, c, 1, 1)
	g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.EndOfTurn})
	assertCharacteristicPT(t, g, c, 3, 3)
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	assertCharacteristicPT(t, g, c, 3, 3)
	a.RemoveFromHand(c.ID())
	g.PutOnBattlefield(c, b.PlayerID())
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, c, 1, 1)
}

func TestCharacteristicPTSpellAndCopies(t *testing.T) {
	g, a, b := characteristicGame()
	c := NewCreature("Spell", "", 0, 0, WithStaticAbility(PTEqualsControlledCount(IsLand)))
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.StackPeek().Controller = b.PlayerID()
	assertCharacteristicPT(t, g, c, 1, 1)
	copied := g.CopySpellOnStack(c.ID(), a.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	assertCharacteristicPT(t, g, copied.Card, 3, 3)
	clone := g.Clone()
	g.ResolveStack()
	assertCharacteristicPT(t, clone, c, 1, 1)
	assertCharacteristicPT(t, g, c, 1, 1)
}

func TestCharacteristicPTLayersRemovalAndPhase(t *testing.T) {
	g, a, _ := characteristicGame()
	c := NewCreature("Count", "", 0, 0, WithStaticAbility(PTEqualsControlledCount(IsLand)))
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, c, 3, 3)
	g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, c, 0, 0)
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, c, 3, 3)
	g.AddContinuousEffect(TargetEffectWithSublayer(core.LayerPT, core.SublayerPT_Set, core.EndOfTurn, p.ID(), func(_ *Game, p *Permanent) error { p.BasePTOverride = &[2]int{2, 5}; return nil }))
	g.AddContinuousEffect(TargetEffectWithSublayer(core.LayerPT, core.SublayerPT_Modify, core.EndOfTurn, p.ID(), func(_ *Game, p *Permanent) error { p.BoostPT(1, 2); return nil }))
	g.AddContinuousEffect(TargetEffectWithSublayer(core.LayerPT, core.SublayerPT_Switch, core.EndOfTurn, p.ID(), func(_ *Game, p *Permanent) error { p.SwitchPowerToughness(); return nil }))
	g.AddCountersWithReplacement(g.FindPermanent(p.ID()), core.P1P1, 1, p.ID(), false)
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, c, 8, 4)
	ids := g.PhaseOut(p.ID())
	if len(ids) == 0 {
		t.Fatal("phase failed")
	}
	assertCharacteristicPT(t, g, c, 8, 4)
	g.PhaseIn(ids)
	assertCharacteristicPT(t, g, c, 8, 4)
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	assertCharacteristicPT(t, g, c, 3, 3)
}

func TestCharacteristicPTSharedDeclarationsAndClone(t *testing.T) {
	g, a, b := characteristicGame()
	declaration := PTEqualsControlledCount(IsLand)
	c := NewCreature("Count", "", 0, 0, WithStaticAbility(declaration))
	c.SetOwner(a.PlayerID())
	copied := c.Copy()
	copied.SetOwner(b.PlayerID())
	first := g.PutOnBattlefield(c, a.PlayerID())
	second := g.PutOnBattlefield(copied, b.PlayerID())
	for range 2 {
		g.ApplyContinuousEffects()
		assertCharacteristicPT(t, g, c, 3, 3)
		assertCharacteristicPT(t, g, copied, 1, 1)
	}
	clone := g.Clone()
	clone.PutOnBattlefield(NewLand("New"), b.PlayerID())
	clone.ApplyContinuousEffects()
	assertCharacteristicPT(t, clone, copied, 2, 2)
	assertCharacteristicPT(t, g, copied, 1, 1)
	g.BouncePermanentToHand(g.FindPermanent(first.ID()))
	a.RemoveFromHand(c.ID())
	g.PutOnBattlefield(c, b.PlayerID())
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, c, 1, 1)
	assertCharacteristicPT(t, g, g.FindPermanent(second.ID()).Card, 1, 1)
}

func TestCharacteristicPTKindsAndPurity(t *testing.T) {
	g, a, b := characteristicGame()
	for _, tc := range []struct {
		effect ContinuousEffect
		p, q   int
	}{
		{PTEqualsCount(IsLand), 4, 4}, {PTEqualsControlledCount(IsLand), 3, 3}, {ToughnessEqualsControlledCount(IsLand), 7, 3},
	} {
		c := NewCreature("Count", "", 7, 8, WithStaticAbility(tc.effect))
		c.SetOwner(a.PlayerID())
		if err := ValidateCardDefinition(c.Name(), c); err != nil {
			t.Fatal(err)
		}
		for range 3 {
			assertCharacteristicPT(t, g, c, tc.p, tc.q)
		}
		if g.StackSize() != 0 || a.Life() != 20 || b.Life() != 20 || len(g.AllBattlefield()) != 4 {
			t.Fatal("query changed state")
		}
	}
	assertCharacteristicPT(t, g, nil, 0, 0)
	assertCharacteristicPT(t, g, NewCreature("Printed", "", -1, 2), -1, 2)
}

func TestCharacteristicPTCountsMembership(t *testing.T) {
	g, a, b := characteristicGame()
	c := NewCreature("Count", "", 0, 0, WithStaticAbility(PTEqualsControlledCount(IsLand)))
	c.SetOwner(a.PlayerID())
	land := g.AllBattlefield()[0]
	ids := g.PhaseOut(land.ID())
	assertCharacteristicPT(t, g, c, 2, 2)
	g.PhaseIn(ids)
	assertCharacteristicPT(t, g, c, 3, 3)
	g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: land.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
	assertCharacteristicPT(t, g, c, 2, 2)
	named := NewCreature("Count", "", 0, 0, WithStaticAbility(PTEqualsCount(And(IsCreature, Named("Member")))))
	named.SetOwner(a.PlayerID())
	member := g.PutOnBattlefield(NewCreature("Member", "", 1, 1), b.PlayerID())
	assertCharacteristicPT(t, g, named, 1, 1)
	g.PutOnBattlefield(NewLand("Member"), a.PlayerID())
	assertCharacteristicPT(t, g, named, 1, 1)
	g.RemoveFromBattlefield(member)
	assertCharacteristicPT(t, g, named, 0, 0)
}

func TestCharacteristicPTCopiedDefinitionAndStaleEffect(t *testing.T) {
	g, a, b := characteristicGame()
	source := NewCreature("Original", "", 0, 0, WithStaticAbility(PTEqualsControlledCount(IsLand)))
	source.SetOwner(a.PlayerID())
	card := NewCreature("Destination", "", 9, 9)
	card.SetOwner(b.PlayerID())
	card.CloneFrom(source)
	assertCharacteristicPT(t, g, card, 1, 1)
	p := g.PutOnBattlefield(card, b.PlayerID())
	g.ApplyContinuousEffects()
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
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	b.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, a.PlayerID())
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, card, 3, 3)
	if old.IsActive(g) {
		t.Fatal("old source incarnation became active")
	}
	if err := old.ApplyContinuous(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	assertCharacteristicPT(t, g, card, 3, 3)
}

func TestCharacteristicPTDirectEffectCaller(t *testing.T) {
	g, a, _ := characteristicGame()
	card := NewCreature("Direct", "", 0, 0)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	effect := PTEqualsControlledCount(IsLand)
	effect.SetSourceID(p.ID())
	g.AddContinuousEffect(effect)
	g.ApplyContinuousEffects()
	assertCharacteristicPT(t, g, card, 3, 3)
}
