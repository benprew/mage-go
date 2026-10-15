package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageLifeFloorResults(t *testing.T) {
	for _, life := range []int{-2, 0, 1, 2, 20} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.PutOnBattlefield(NewCreature("Rule", "", 0, 1, WithAbility(Static(LimitDamageLifeLoss(You, 1)))), a.PlayerID())
		a.LoseLife(20 - life)
		card := NewCreature("Source", "", 1, 1, WithKeyword(core.Lifelink))
		p := g.PutOnBattlefield(card, b.PlayerID())
		source := g.CaptureDamageSource(p.ID())
		outcomes := g.dealSimultaneousDamageOutcomes(PlayerDamageAssignment(a, 4, source), PlayerDamageAssignment(a, 30, source))
		if a.Life() != min(life, 1) || b.Life() != 54 || len(outcomes) != 2 || outcomes[0].amount != 4 || outcomes[1].amount != 30 {
			t.Fatalf("start %d life %d/%d outcomes %+v", life, a.Life(), b.Life(), outcomes)
		}
	}
}

func TestDamageLifeFloorSimultaneousDeparture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	rule := g.PutOnBattlefield(NewCreature("Rule", "", 0, 1, WithAbility(Static(LimitDamageLifeLoss(You, 1)))), a.PlayerID())
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), b.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	g.DealSimultaneousDamage(PermanentDamageAssignment(rule, 1, saved), PlayerDamageAssignment(a, 25, saved))
	if a.Life() != 1 || rule.Damage != 1 {
		t.Fatal("simultaneous damage lost protection")
	}
	g.RemoveFromBattlefield(g.FindPermanent(rule.ID()))
	g.DealDamageToPlayer(a, 2, source.ID())
	if a.Life() != -1 {
		t.Fatal("departed source still protects")
	}
}

func TestDamageLifeFloorPrevention(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewCreature("Rule", "", 0, 1, WithAbility(Static(LimitDamageLifeLoss(You, 1)))), a.PlayerID())
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1, WithKeyword(core.Lifelink)), b.PlayerID())
	g.AddPreventionShield(a.PlayerID(), 3)
	g.DealDamageToPlayer(a, 5, source.ID())
	if a.Life() != 18 || b.Life() != 22 {
		t.Fatal("prevention did not precede life result")
	}
}

func TestDamageLifeFloorDefinition(t *testing.T) {
	for _, rule := range []StaticRule{LimitDamageLifeLoss(nil, 1), LimitDamageLifeLoss(You, 0), LimitDamageLifeLoss(You, -1)} {
		card := NewCreature("Bad", "", 0, 1, WithAbility(Static(rule)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid rule accepted")
		}
	}
}

func TestDamageLifeFloorSimultaneousLifeGains(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, doubled := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			rule := g.PutOnBattlefield(NewCreature("Rule", "", 0, 1, WithAbility(Static(LimitDamageLifeLoss(You, 1)))), a.PlayerID())
			a.LoseLife(18)
			if doubled {
				g.AddLifeGainModifier(rule.ID(), DoubleLifeGainForController(rule.ID(), a.PlayerID()))
			}
			source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), b.PlayerID())
			lifelink := g.PutOnBattlefield(NewCreature("Lifelink", "", 1, 1, WithKeyword(core.Lifelink)), a.PlayerID())
			packets := []DamageAssignment{PlayerDamageAssignment(a, 5, g.CaptureDamageSource(source.ID())), PlayerDamageAssignment(b, 5, g.CaptureDamageSource(lifelink.ID()))}
			if reverse {
				packets[0], packets[1] = packets[1], packets[0]
			}
			g.DealSimultaneousDamage(packets...)
			want := 2
			if doubled {
				want = 7
			}
			if a.Life() != want || b.Life() != 15 {
				t.Fatalf("reverse %v doubled %v life %d/%d", reverse, doubled, a.Life(), b.Life())
			}
		}
	}
}

func TestDamageLifeFloorPreventionLifeGain(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	rule := g.PutOnBattlefield(NewCreature("Rule", "", 0, 1, WithAbility(Static(LimitDamageLifeLoss(You, 1)))), a.PlayerID())
	a.LoseLife(18)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), b.PlayerID())
	g.effects.AddReplacement(&sourcePreventionReplacement{replacementBase: replacementBase{sourceID: rule.ID()}, playerID: a.PlayerID(), dmgSource: source.ID(), amount: proposedDamageAmount{}, gainLifePlayerID: a.PlayerID()})
	g.DealSimultaneousDamage(PlayerDamageAssignment(a, 5, g.CaptureDamageSource(source.ID())), PlayerDamageAssignment(a, 5, g.CaptureDamageSource(other.ID())))
	if a.Life() != 2 {
		t.Fatalf("life %d want 2", a.Life())
	}
}

func TestDamageLifeFloorDamageEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("damage")
	g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerIsDealtDamage(amount), GainLifeFor(You, FromNumber(amount))))), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Rule", "", 0, 1, WithAbility(Static(LimitDamageLifeLoss(You, 1)))), a.PlayerID())
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), b.PlayerID())
	g.DealDamageToPlayer(a, 25, source.ID())
	g.PutTriggersOnStack()
	if a.Life() != 1 || g.StackSize() != 1 || !g.HasDealtDamageToPlayer(source.ID(), a.PlayerID()) {
		t.Fatal("full damage event or history lost")
	}
	g.ResolveStack()
	if a.Life() != 26 {
		t.Fatalf("damage event saved %d instead of 25", a.Life()-1)
	}
}

func TestDamageLifeFloorReplacedGains(t *testing.T) {
	for _, gain := range []int{0, 3, 10} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.PutOnBattlefield(NewCreature("Rule", "", 0, 1, WithAbility(Static(LimitDamageLifeLoss(You, 1)))), a.PlayerID())
		source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1, WithKeyword(core.Lifelink)), a.PlayerID())
		a.LoseLife(18)
		g.effects.AddReplacement(&lifeOutcomeReplacement{amount: gain})
		g.DealDamageToPlayer(a, 5, source.ID())
		if a.Life() != max(1, 2+gain-5) {
			t.Fatalf("gain %d life %d", gain, a.Life())
		}
	}
}

func TestDamageLifeFloorMultipleRules(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		floors := []int{1, 5}
		if reverse {
			floors[0], floors[1] = floors[1], floors[0]
		}
		for _, floor := range floors {
			g.PutOnBattlefield(NewCreature("Rule", "", 0, 1, WithAbility(Static(LimitDamageLifeLoss(You, floor)))), a.PlayerID())
		}
		source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), b.PlayerID())
		g.DealDamageToPlayer(a, 25, source.ID())
		if a.Life() != 5 {
			t.Fatal("replacement recheck lost the higher floor")
		}
	}
}

func TestDamageLifeFloorAPNAP(t *testing.T) {
	var order []string
	a := &damageCapOrderPlayer{BasePlayer: NewBasePlayer("A"), order: &order}
	b := &damageCapOrderPlayer{BasePlayer: NewBasePlayer("B"), order: &order}
	g := NewGame(a, b)
	for _, player := range []Player{a, b} {
		for _, floor := range []int{1, 5} {
			g.PutOnBattlefield(NewCreature("Rule", "", 0, 1, WithAbility(Static(LimitDamageLifeLoss(You, floor)))), player.PlayerID())
		}
	}
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), b.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	g.DealSimultaneousDamage(PlayerDamageAssignment(b, 25, saved), PlayerDamageAssignment(a, 25, saved))
	if len(order) != 2 || order[0] != "A" || order[1] != "B" || a.Life() != 5 || b.Life() != 5 {
		t.Fatalf("order %v life %d/%d", order, a.Life(), b.Life())
	}
}
