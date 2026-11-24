package mage

import (
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCastSpellAdjustsTotalManaAfterAnnouncingX(t *testing.T) {
	for _, tc := range []struct {
		name           string
		x              int
		reduction      int
		startingLife   int
		colorlessMana  int
		additionalLife int
		channel        bool
		wantLife       int
	}{
		{name: "reduction covers target surcharge", x: 2, reduction: 3, wantLife: 20},
		{name: "reduction with a nonmana cost", x: 2, reduction: 3, additionalLife: 1, wantLife: 19},
		{name: "Channel pays the reduced generic total", x: 4, reduction: 1, channel: true, wantLife: 16},
		{name: "mana pays the surcharge when life is low", x: 2, startingLife: 4, colorlessMana: 1, additionalLife: 1, channel: true, wantLife: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := newModalGame()
			g.SetStep(PrecombatMain)
			a.ManaPool().Add(Red, 1)
			a.ManaPool().Add(Colorless, tc.colorlessMana)
			if tc.startingLife > 0 {
				a.SetLife(tc.startingLife)
			}
			if tc.channel {
				g.SetChannelActive(a.PlayerID())
			}
			if tc.reduction > 0 {
				g.PutOnBattlefield(NewArtifact("Cost Reducer", "{0}",
					WithStaticAbility(ReduceSpellCostStatic(SpellAny(), FixedAmount(tc.reduction), nil)),
				), a.PlayerID())
			}

			creature := g.PutOnBattlefield(NewCreature("Target Creature", "{0}", 1, 4), b.PlayerID())
			spell := NewSorcery("Divided Damage", "{X}{R}",
				NewMultiTargetSpell([]Target{TargetPlayer(), TargetCreature()}, DealDamageDividedEvenly(XValue())),
				WithAdditionalCost(ManaCostPerTargetBeyondFirst("{1}")),
			)
			if tc.additionalLife > 0 {
				WithAdditionalCost(LifePayCost(tc.additionalLife))(spell)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)

			if options := g.SpellTargetOptions(a.PlayerID(), spell, tc.x); len(options) != 1 || options[0].XValue != tc.x {
				t.Errorf("expected an affordable X=%d announcement, got %+v", tc.x, options)
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID(), creature.ID()}, tc.x); err != nil {
				t.Fatalf("cast with X=%d and two targets: %v", tc.x, err)
			}
			if cast := g.Stack().Peek(); cast.XValue != tc.x || cast.ManaValue() != tc.x+1 {
				t.Fatalf("stack X and mana value = %d and %d, want %d and %d", cast.XValue, cast.ManaValue(), tc.x, tc.x+1)
			}
			if life, mana := a.Life(), a.ManaPool().TotalMana(); life != tc.wantLife || mana != 0 {
				t.Fatalf("after payment got %d life and %d mana, want %d life and 0 mana", life, mana, tc.wantLife)
			}
			g.ResolveStack()
			if want := 20 - tc.x/2; b.Life() != want {
				t.Fatalf("opponent has %d life after resolution, want %d", b.Life(), want)
			}
		})
	}
}

func TestCastSpellCombinesPrintedAndAdditionalMana(t *testing.T) {
	g := newPriorityTestGame()
	playerID := g.players[0].PlayerID()
	g.SetStep(PrecombatMain)
	g.players[0].ManaPool().Add(White, 1)
	g.players[0].ManaPool().Add(Blue, 1)

	spell := NewSorcery("Combined Additional Mana", "{1}",
		NewSpellAbility(GainLife(1)),
		WithAdditionalCost(ManaCostOf("{W}")),
	)
	spell.SetOwner(playerID)
	g.players[0].AddToHand(spell)

	if err := g.CastSpellByID(playerID, spell.ID(), nil, 0); err != nil {
		t.Fatalf("casting {1} plus additional {W} from {W}{U}: %v", err)
	}
	if got := g.players[0].ManaPool().TotalMana(); got != 0 {
		t.Fatalf("payment left %d mana, want 0", got)
	}
}

func TestCastSpellCombinesPrintedAndActionMana(t *testing.T) {
	g := newPriorityTestGame()
	playerID := g.players[0].PlayerID()
	g.SetStep(PrecombatMain)
	g.players[0].ManaPool().Add(White, 1)
	g.players[0].ManaPool().Add(Blue, 1)

	spell := NewSorcery("Combined Action Mana", "{1}",
		NewSpell(GainLife(1), WithCost(ManaCostOf("{W}"))),
	)
	spell.SetOwner(playerID)
	g.players[0].AddToHand(spell)

	if err := g.CastSpellByID(playerID, spell.ID(), nil, 0); err != nil {
		t.Fatalf("casting {1} plus action {W} from {W}{U}: %v", err)
	}
	if got := g.players[0].ManaPool().TotalMana(); got != 0 {
		t.Fatalf("payment left %d mana, want 0", got)
	}
}

func TestCastSpellUnpayableTotalCostDoesNotTapManaSources(t *testing.T) {
	g := newPriorityTestGame()
	playerID := g.players[0].PlayerID()
	g.SetStep(PrecombatMain)
	mountain := addLand(t, g, playerID, "Mountain", Red)

	spell := NewSorcery("Unpayable Additional Life", "{R}",
		NewSpellAbility(GainLife(1)),
		WithAdditionalCost(LifePayCost(100)),
	)
	spell.SetOwner(playerID)
	g.players[0].AddToHand(spell)
	startingLife := g.players[0].Life()

	if err := g.CastSpellByID(playerID, spell.ID(), nil, 0); err == nil {
		t.Fatal("expected the unpayable total cost to reject the cast")
	}
	if mountain.Tapped {
		t.Fatal("failed cast tapped a mana source")
	}
	if got := g.players[0].ManaPool().TotalMana(); got != 0 {
		t.Fatalf("failed cast left %d mana in the pool, want 0", got)
	}
	if got := g.players[0].Life(); got != startingLife {
		t.Fatalf("failed cast changed life from %d to %d", startingLife, got)
	}
	if _, ok := g.players[0].RemoveFromHand(spell.ID()); !ok {
		t.Fatal("failed cast removed the spell from hand")
	}
}

func TestCastSpellLocksEitherCostAfterRemovingSpellFromHand(t *testing.T) {
	g := newPriorityTestGame()
	playerID := g.players[0].PlayerID()
	g.SetStep(PrecombatMain)
	g.players[0].ManaPool().Add(Black, 1)
	g.players[0].ManaPool().Add(Colorless, 7)

	spell := NewSorcery("Locked Either Cost", "{2}{B}",
		NewSpellAbility(GainLife(1)),
		WithAdditionalCost(EitherCost(DiscardCost(1), ManaCostOf("{5}"))),
	)
	spell.SetOwner(playerID)
	g.players[0].AddToHand(spell)

	if err := g.CastSpellByID(playerID, spell.ID(), nil, 0); err != nil {
		t.Fatalf("casting through the only legal {5} branch: %v", err)
	}
	if got := g.players[0].ManaPool().TotalMana(); got != 0 {
		t.Fatalf("locked total payment left %d mana, want 0", got)
	}
}

func TestCastSpellColorsSpentExcludeManaAbilityActivationCosts(t *testing.T) {
	g := newPriorityTestGame()
	playerID := g.players[0].PlayerID()
	g.SetStep(PrecombatMain)
	addLand(t, g, playerID, "Mountain 1", Red)
	addLand(t, g, playerID, "Mountain 2", Red)
	filter := NewArtifact("Mana Filter", "{3}",
		WithActivatedAbility(AddAnyMana(1, Colorless), GenericCost(2), WithCost(Tap())),
	)
	filter.SetOwner(playerID)
	filterPermanent := g.PutOnBattlefield(filter, playerID)
	filterPermanent.RevokeBaseAttr(AttrSummonSick)

	spell := NewSorcery("Colors Spent Transaction", "{U}", NewSpellAbility(GainLife(1)))
	spell.SetOwner(playerID)
	g.players[0].AddToHand(spell)

	if err := g.CastSpellByID(playerID, spell.ID(), nil, 0); err != nil {
		t.Fatalf("casting through the mana filter: %v", err)
	}
	cast := g.stack.Peek()
	if cast == nil || cast.CastContext == nil {
		t.Fatal("expected a cast context on the spell")
	}
	if got := cast.CastContext.ColorsSpent[Blue]; got != 1 {
		t.Fatalf("blue mana spent to cast = %d, want 1", got)
	}
	if got := cast.CastContext.ColorsSpent[Red]; got != 0 {
		t.Fatalf("red mana spent activating the filter leaked into cast metadata: %d", got)
	}
}
