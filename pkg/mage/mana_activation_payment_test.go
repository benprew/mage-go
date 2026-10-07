package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestManaActivationPaymentPreservesFinalColors(t *testing.T) {
	pool := NewManaPool()
	pool.Add(White, 1)
	pool.Add(Red, 1)
	pool.Add(Green, 1)
	input := ManaSolverInputs{Pool: pool, Cost: ManaCost{White: 1, Blue: 1}, Sources: []manaSourceInfo{{PermanentID: uuid.New(), Abilities: []manaSourceAbility{{Productions: []ManaProduction{{Color: AnyColor, Amount: 1}}, ManaCost: ManaCost{Generic: 2}}}}}}
	solution, err := SolveMana(input)
	if err != nil {
		t.Fatal("legal activation can spend red and green, preserving white:", err)
	}
	if !CanSolveMana(input) || len(solution.SourcesToTap) != 1 || pool.TotalMana() != 3 {
		t.Fatal("planning changed the pool or omitted the source")
	}
}

func TestManaActivationPaymentEnumeration(t *testing.T) {
	for _, costText := range []string{"", "{1}", "{2}", "{W}", "{1}{W}", "{W/U}", "{1}{W/U}", "{W}{U}"} {
		for _, convert := range []bool{false, true} {
			pool := NewManaPool()
			pool.Add(White, 2)
			pool.Add(Blue, 1)
			pool.Add(Red, 1)
			pool.pool = append(pool.pool, Mana{Color: Green, Restriction: ArtifactSpellsOnly{}})
			if convert {
				pool.ManaConversions = map[Color]Color{Red: White}
			}
			cost := ParseManaCost(costText)
			got := manaActivationPayments(pool, cost)
			seen := map[ManaActivationPayment]bool{}
			for _, payment := range got {
				if seen[payment] || !validManaActivationPayment(pool, cost, payment) {
					t.Fatalf("%s: duplicate or invalid payment %+v", costText, payment)
				}
				seen[payment] = true
			}
			want := 0
			for w := 0; w <= 2; w++ {
				for u := 0; u <= 1; u++ {
					for r := 0; r <= 1; r++ {
						selected := NewManaPool()
						selected.ManaConversions = pool.ManaConversions
						selected.Add(White, w)
						selected.Add(Blue, u)
						selected.Add(Red, r)
						if w+u+r == cost.CMC() && selected.CanPay(cost, nil) {
							want++
						}
					}
				}
			}
			if len(got) != want || pool.TotalMana() != 5 {
				t.Fatalf("%s conversion=%v: got %d want %d", costText, convert, len(got), want)
			}
		}
	}
}

func TestManaActivationPaymentConvertedPip(t *testing.T) {
	pool := NewManaPool()
	pool.Add(Blue, 1)
	pool.Add(Red, 1)
	in := ManaSolverInputs{Pool: pool, Conversions: map[Color]Color{Red: Blue}, Cost: ManaCost{Blue: 1, Green: 1}, Sources: []manaSourceInfo{{PermanentID: uuid.New(), Abilities: []manaSourceAbility{{ManaCost: ManaCost{Blue: 1}, Productions: []ManaProduction{{Color: Green, Amount: 1}}}}}}}
	solution, err := SolveMana(in)
	if err != nil {
		t.Fatal(err)
	}
	action := solution.SourcesToTap[0]
	pool.ManaConversions = in.Conversions
	if action.ActivationPayment == nil || !validManaActivationPayment(pool, ManaCost{Blue: 1}, *action.ActivationPayment) {
		t.Fatal("illegal converted payment")
	}
	if err := payManaActivationPayment(pool, ManaCost{Blue: 1}, *action.ActivationPayment); err != nil {
		t.Fatal(err)
	}
	pool.Add(Green, 1)
	if !pool.CanPay(in.Cost, nil) {
		t.Fatal("converted payment cannot fund final cost")
	}
}

func paymentTestSource(g *Game, player Player) *Permanent {
	card := NewArtifact("Payment source", "", WithActivatedAbility(AddAnyMana(1, Colorless), GenericCost(2), WithCost(Tap())))
	card.SetOwner(player.PlayerID())
	return g.PutOnBattlefield(card, player.PlayerID())
}

func TestManaActivationPaymentPlanLifecycle(t *testing.T) {
	for _, change := range []string{"none", "clone", "returned", "phase", "phase-cycle", "cost-change", "ability-removed", "tapped", "control", "unavailable", "negative", "wrong-cost", "restricted", "legacy"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := paymentTestSource(g, a)
			a.ManaPool().Add(White, 1)
			a.ManaPool().Add(Red, 1)
			a.ManaPool().Add(Green, 1)
			plan, err := g.mana.PlanManaForCost(g, a.PlayerID(), ManaCost{White: 1, Blue: 1}, AutoTapHint{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			action := plan.SourcesToTap[0]
			if action.ActivationPayment == nil || action.ActivationPayment[White] != 0 || action.SourceIncarnation != source.incarnationID {
				t.Fatal("missing exact payment or incarnation")
			}
			clone := g.Clone()
			switch change {
			case "returned":
				card := source.Card
				g.RemoveFromBattlefield(source)
				source = g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "cost-change":
				g.effects.Rules.ActivationCostReductions[source.ID()] = 1
			case "ability-removed":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), Indefinite))
				g.ApplyContinuousEffects()
			case "tapped":
				g.TapPermanent(source)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
				g.ApplyContinuousEffects()
			case "unavailable":
				a.ManaPool().Clear()
				a.ManaPool().Add(White, 3)
			case "negative":
				payment := *action.ActivationPayment
				payment[Red] = -1
				action.ActivationPayment = &payment
			case "wrong-cost":
				payment := *action.ActivationPayment
				payment[White] = 1
				action.ActivationPayment = &payment
			case "restricted":
				a.ManaPool().Clear()
				a.ManaPool().Add(White, 1)
				a.ManaPool().pool = append(a.ManaPool().pool, Mana{Color: Red, Restriction: ArtifactSpellsOnly{}}, Mana{Color: Green, Restriction: ArtifactSpellsOnly{}})
			case "legacy":
				action.ActivationPayment = nil
				action.SourceIncarnation = uuid.Nil
			}
			before := a.ManaPool().SnapshotPool()
			tapped := g.FindPermanentIncludingPhased(source.ID()).Tapped
			err = g.mana.ActivatePlannedManaSource(g, a.PlayerID(), action)
			valid := change == "none" || change == "clone" || change == "legacy" || change == "phase-cycle"
			if !valid {
				if err == nil || a.ManaPool().TotalMana() != len(before) || g.FindPermanentIncludingPhased(source.ID()).Tapped != tapped {
					t.Fatal("invalid plan changed payment or source")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanentIncludingPhased(source.ID()).Tapped || a.ManaPool().TotalMana() != 2 {
					t.Fatal("activation did not execute")
				}
				if change != "legacy" && (a.ManaPool().Count(White) != 1 || a.ManaPool().Count(Blue) != 1) {
					t.Fatal("execution spent different colors")
				}
			}
			if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 3 || clone.FindPermanent(source.ID()).Tapped {
				t.Fatal("execution changed clone")
			}
			if err := clone.mana.ApplyManaSolution(clone, a.PlayerID(), plan); err != nil {
				t.Fatal("clone cannot execute original plan:", err)
			}
			if clone.GetPlayer(a.PlayerID()).ManaPool().Count(White) != 1 || clone.GetPlayer(a.PlayerID()).ManaPool().Count(Blue) != 1 {
				t.Fatal("clone payment changed")
			}
		})
	}
}
