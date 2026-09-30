package mage

import (
	"maps"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type activationDivisionPlayer struct {
	*recordingPlayer
	choose    func([]uuid.UUID, int, *Game) map[uuid.UUID]int
	divisions int
}

func (p *activationDivisionPlayer) ChooseDamageDistribution(ids []uuid.UUID, total int, _ string, g *Game) map[uuid.UUID]int {
	p.divisions++
	return p.choose(ids, total, g)
}

func newActivationDivisionGame() (*Game, *activationDivisionPlayer, *recordingPlayer) {
	a := &activationDivisionPlayer{recordingPlayer: newRecordingPlayer("A")}
	b := newRecordingPlayer("B")
	return NewGame(a, b), a, b
}

type activationDivisionLife struct{}

func (activationDivisionLife) Text() string { return "controller life" }
func (activationDivisionLife) Resolve(g GameReader, _, controller uuid.UUID, _ []uuid.UUID) int {
	return g.GetPlayer(controller).Life()
}

func TestActivationDamageDivisionBeforePayment(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	a.SetLife(4)
	a.ManaPool().Add(Red, 1)
	source := g.PutOnBattlefield(NewArtifact("Division source", "",
		WithActivatedAbility(DealDividedDamage(activationDivisionLife{}), Tap(),
			WithCost(ManaCostOf("{R}")), WithCost(LifePayCost(2)),
			WithTarget(TargetUpToNCreaturesOrPlayers(2)))), a.PlayerID())
	response := map[uuid.UUID]int{a.PlayerID(): 1, b.PlayerID(): 3}
	a.choose = func(ids []uuid.UUID, total int, game *Game) map[uuid.UUID]int {
		if total != 4 || a.Life() != 4 || a.ManaPool().Count(Red) != 1 || game.FindPermanent(source.ID()).Tapped || game.StackSize() != 0 {
			t.Errorf("division occurred after payment: total=%d life=%d mana=%d tapped=%v", total, a.Life(), a.ManaPool().Count(Red), source.Tapped)
		}
		return response
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID(), b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	if a.divisions != 1 || !maps.Equal(obj.DamageDistribution, response) || a.Life() != 2 || !source.Tapped || a.ManaPool().Count(Red) != 0 {
		t.Fatal("division or payment was not locked")
	}
	response[b.PlayerID()] = 99
	clone := g.Clone()
	copied := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	copied.DamageDistribution[b.PlayerID()] = 7
	if obj.DamageDistribution[b.PlayerID()] != 3 || clone.StackObjects()[0].DamageDistribution[b.PlayerID()] != 3 {
		t.Fatal("division shares response, copy, or clone state")
	}
	g.ResolveStack()
	if b.Life() != 17 {
		t.Fatalf("payment changed announced damage: life=%d", b.Life())
	}
}

func TestActivationDamageDivisionSourceSacrifice(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	source := g.PutOnBattlefield(NewArtifact("Division source", "",
		WithActivatedAbility(DealDividedDamage(Fixed(3)), SacrificeSourceCost(),
			WithTarget(TargetUpToNCreaturesOrPlayers(1)))), a.PlayerID())
	a.choose = func(ids []uuid.UUID, total int, game *Game) map[uuid.UUID]int {
		if game.FindPermanent(source.ID()) == nil {
			t.Error("source left before division")
		}
		return map[uuid.UUID]int{b.PlayerID(): total}
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()) != nil || a.divisions != 1 {
		t.Fatal("source cost or choice was not completed")
	}
	g.ResolveStack()
	if b.Life() != 17 {
		t.Fatal("departed source lost its allocation")
	}
}

func TestActivationDamageDivisionImpossibleBeforePayment(t *testing.T) {
	for _, tc := range []struct {
		name         string
		total, count int
	}{
		{"too many", 1, 2}, {"zero with target", 0, 1}, {"negative", -1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			a.ManaPool().Add(Red, 1)
			source := g.PutOnBattlefield(NewArtifact("Division source", "",
				WithActivatedAbility(DealDividedDamage(Fixed(tc.total)), Tap(), WithCost(ManaCostOf("{R}")), WithCost(LifePayCost(2)), WithTarget(TargetUpToNCreaturesOrPlayers(2)))), a.PlayerID())
			events := 0
			for _, event := range []EventType{EvtTapped, EvtLifeLost, EvtAbilityActivated, EvtBecomesTarget} {
				g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(NewTriggered(event, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *GameEvent, _ GameReader, _, _ uuid.UUID) bool { events++; return false }))), a.PlayerID())
			}
			a.choose = func(ids []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
				return sanitizeDamageDistribution(nil, ids, total)
			}
			targets := []uuid.UUID{a.PlayerID(), b.PlayerID()}[:tc.count]
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err == nil {
				t.Error("accepted impossible division")
			}
			if source.Tapped || a.Life() != 20 || a.ManaPool().Count(Red) != 1 || g.StackSize() != 0 || events != 0 || a.divisions != 0 {
				t.Fatalf("invalid division changed state or prompted: tapped=%v life=%d mana=%d stack=%d events=%d choices=%d", source.Tapped, a.Life(), a.ManaPool().Count(Red), g.StackSize(), events, a.divisions)
			}
		})
	}
}

func TestActivationDamageDivisionFailedPayment(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	source := g.PutOnBattlefield(NewArtifact("Division source", "", WithActivatedAbility(DealDividedDamage(Fixed(3)), Tap(), WithCost(LifePayCost(11)), WithCost(LifePayCost(11)), WithTarget(TargetUpToNCreaturesOrPlayers(1)))), a.PlayerID())
	a.choose = func(ids []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
		return map[uuid.UUID]int{b.PlayerID(): total}
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil {
		t.Fatal("paid impossible cost")
	}
	if a.divisions != 1 || source.Tapped || a.Life() != 20 || g.StackSize() != 0 {
		t.Fatal("failed payment skipped announcement or changed state")
	}
}

func TestActivationDamageDivisionWithoutTargets(t *testing.T) {
	for _, total := range []int{-1, 0, 3} {
		g, a, _ := newActivationDivisionGame()
		source := g.PutOnBattlefield(NewArtifact("Division source", "", WithActivatedAbility(DealDividedDamage(Fixed(total)), Tap(), WithTarget(TargetUpToNCreaturesOrPlayers(1)))), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if !source.Tapped || a.divisions != 0 || len(g.StackPeek().DamageDistribution) != 0 {
			t.Fatal("empty division prompted or failed payment")
		}
	}
}

func TestActivationDamageDivisionSelectedMode(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	a.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
	source := g.PutOnBattlefield(NewArtifact("Modal division", "", WithAbility(NewModalActivated(Tap(), []Mode{
		{Label: "Divide", Targets: []Target{TargetUpToNCreaturesOrPlayers(1)}, Effects: []Effect{DealDividedDamage(Fixed(3))}},
		{Label: "Other", Effects: []Effect{DealDividedDamage(Fixed(-1))}},
	}))), a.PlayerID())
	a.choose = func(ids []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
		if source.Tapped {
			t.Error("modal division occurred after payment")
		}
		return map[uuid.UUID]int{b.PlayerID(): total}
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.divisions != 1 || g.StackPeek().ModeChoice != 0 || g.StackPeek().DamageDistribution[b.PlayerID()] != 3 {
		t.Fatal("division used an inactive mode")
	}
}

func TestActivationDamageDivisionIllegalTargetsBeforeChoice(t *testing.T) {
	for _, state := range []string{"absent", "phased", "duplicate"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			source := g.PutOnBattlefield(NewArtifact("Division source", "", WithActivatedAbility(DealDividedDamage(Fixed(3)), Tap(), WithTarget(TargetUpToNCreaturesOrPlayers(2)))), a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
			targets := []uuid.UUID{victim.ID()}
			switch state {
			case "absent":
				targets[0] = uuid.New()
			case "phased":
				g.PhaseOut(victim.ID())
			case "duplicate":
				targets = append(targets, victim.ID())
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err == nil {
				t.Fatal("accepted illegal targets")
			}
			if a.divisions != 0 || source.Tapped || g.StackSize() != 0 {
				t.Fatal("illegal targets prompted or paid")
			}
		})
	}
}
