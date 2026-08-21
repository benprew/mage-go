package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCastSpellCombinesPrintedAndAdditionalMana(t *testing.T) {
	g := newPriorityTestGame()
	playerID := g.players[0].PlayerID()
	g.step = PrecombatMain
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
	g.step = PrecombatMain
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
	g.step = PrecombatMain
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
	g.step = PrecombatMain
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
	g.step = PrecombatMain
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
