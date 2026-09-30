package mage

import (
	"maps"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func addDivisionSpell(a Player, total ValueSource, costs ...CardOption) Card {
	card := NewInstant("Division spell", "{R}", NewMultiTargetSpell([]Target{TargetUpToNCreaturesOrPlayers(2)}, DealDividedDamage(total)))
	for _, option := range costs {
		option(card)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	return card
}

func TestSpellDamageDivisionBeforePayment(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	a.SetLife(4)
	a.ManaPool().Add(Red, 1)
	card := addDivisionSpell(a, activationDivisionLife{}, WithAdditionalCost(LifePayCost(2)))
	response := map[uuid.UUID]int{a.PlayerID(): 1, b.PlayerID(): 3}
	a.choose = func(_ []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
		if total != 4 || a.Life() != 4 || a.ManaPool().Count(Red) != 1 || len(a.Hand()) != 1 {
			t.Errorf("division after payment: total=%d life=%d mana=%d hand=%d", total, a.Life(), a.ManaPool().Count(Red), len(a.Hand()))
		}
		return response
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{a.PlayerID(), b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	if a.divisions != 1 || !maps.Equal(obj.DamageDistribution, response) || a.Life() != 2 {
		t.Fatal("division was not locked")
	}
	response[b.PlayerID()] = 99
	clone := g.Clone()
	copied := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	copied.DamageDistribution[b.PlayerID()] = 7
	if obj.DamageDistribution[b.PlayerID()] != 3 || clone.StackObjects()[0].DamageDistribution[b.PlayerID()] != 3 {
		t.Fatal("allocation maps share state")
	}
	g.ResolveStack()
	if b.Life() != 17 {
		t.Fatal("payment changed announced damage")
	}
}

func TestSpellDamageDivisionImpossibleBeforePayment(t *testing.T) {
	for _, total := range []int{-1, 0, 1} {
		g, a, b := newActivationDivisionGame()
		a.ManaPool().Add(Red, 1)
		card := addDivisionSpell(a, Fixed(total), WithAdditionalCost(LifePayCost(2)))
		events := 0
		for _, event := range []EventType{EvtLifeLost, EvtSpellCast, EvtBecomesTarget} {
			g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(NewTriggered(event, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *GameEvent, _ GameReader, _, _ uuid.UUID) bool { events++; return false }))), a.PlayerID())
		}
		a.choose = func(ids []uuid.UUID, amount int, _ *Game) map[uuid.UUID]int {
			return sanitizeDamageDistribution(nil, ids, amount)
		}
		if err := g.CastSpellByName(a.PlayerID(), card.Name(), []uuid.UUID{a.PlayerID(), b.PlayerID()}); err == nil {
			t.Errorf("accepted total %d for two targets", total)
		}
		if a.divisions != 0 || a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(a.Hand()) != 1 || g.StackSize() != 0 || events != 0 {
			t.Errorf("impossible total %d prompted or changed state", total)
		}
	}
}

func TestSpellDamageDivisionFailedPayment(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	a.ManaPool().Add(Red, 1)
	card := addDivisionSpell(a, Fixed(3), WithAdditionalCost(LifePayCost(11)), WithAdditionalCost(LifePayCost(11)))
	a.choose = func(_ []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
		return map[uuid.UUID]int{b.PlayerID(): total}
	}
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), []uuid.UUID{b.PlayerID()}); err == nil {
		t.Fatal("paid impossible cost")
	}
	if a.divisions != 1 || a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed payment skipped announcement or changed state")
	}
}

func TestSpellDamageDivisionWithoutTargets(t *testing.T) {
	for _, total := range []int{-1, 0, 3} {
		g, a, _ := newActivationDivisionGame()
		a.ManaPool().Add(Red, 1)
		card := addDivisionSpell(a, Fixed(total))
		if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err != nil {
			t.Fatal(err)
		}
		if a.divisions != 0 || len(g.StackPeek().DamageDistribution) != 0 || a.ManaPool().Count(Red) != 0 {
			t.Fatal("empty division prompted or failed payment")
		}
	}
}

func TestSpellDamageDivisionSelectedMode(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	a.ManaPool().Add(Red, 1)
	card := NewInstant("Modal division", "{R}", WithAbility(NewModalSpell([]Mode{
		{Label: "Divide", Targets: []Target{TargetUpToNCreaturesOrPlayers(1)}, Effects: []Effect{DealDividedDamage(Fixed(3))}},
		{Label: "Other", Effects: []Effect{DealDividedDamage(Fixed(-1))}},
	})))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.choose = func(_ []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
		if a.ManaPool().Count(Red) != 1 {
			t.Error("modal division after payment")
		}
		return map[uuid.UUID]int{b.PlayerID(): total}
	}
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if a.divisions != 1 || g.StackPeek().ModeChoice != 0 || g.StackPeek().DamageDistribution[b.PlayerID()] != 3 {
		t.Fatal("division used inactive mode")
	}
}

func TestSpellDamageDivisionIllegalTargetsBeforeChoice(t *testing.T) {
	for _, state := range []string{"absent", "phased", "duplicate", "wrong type"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			a.ManaPool().Add(Red, 1)
			card := addDivisionSpell(a, Fixed(3))
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
			targets := []uuid.UUID{victim.ID()}
			switch state {
			case "absent":
				targets[0] = uuid.New()
			case "phased":
				g.PhaseOut(victim.ID())
			case "duplicate":
				targets = append(targets, victim.ID())
			case "wrong type":
				targets[0] = g.PutOnBattlefield(NewArtifact("Other", ""), b.PlayerID()).ID()
			}
			if err := g.CastSpellByName(a.PlayerID(), card.Name(), targets); err == nil {
				t.Fatal("accepted illegal targets")
			}
			if a.divisions != 0 || a.ManaPool().Count(Red) != 1 || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("illegal targets prompted or paid")
			}
		})
	}
}

func TestSpellDamageDivisionBeforeManaAbilities(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	land := g.PutOnBattlefield(NewLand("Mana source", WithManaAbility(Red)), a.PlayerID())
	card := addDivisionSpell(a, Fixed(3))
	a.choose = func(_ []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
		if land.Tapped || a.ManaPool().TotalMana() != 0 {
			t.Error("mana ability ran before division")
		}
		return map[uuid.UUID]int{b.PlayerID(): total}
	}
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if !land.Tapped || a.divisions != 1 || g.StackPeek().DamageDistribution[b.PlayerID()] != 3 {
		t.Fatal("mana payment did not preserve division")
	}
}
