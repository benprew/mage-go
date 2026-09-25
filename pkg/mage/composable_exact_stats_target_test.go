package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestExactStatsTargetResolution(t *testing.T) {
	for _, change := range []string{"none", "power", "toughness", "return"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := randomTestGame()
			card := NewCreature("Recipient", "", 1, 1)
			card.SetOwner(b.PlayerID())
			recipient := g.PutOnBattlefield(card, b.PlayerID())
			target := DeclarePermanentTarget("creature", And(IsCreature, HasPowerEqual(1), HasToughnessEqual(1)))
			ability := Activate(Costs(TapCost(This)), Until(EndOfTurn, ModifyPT(target, Fixed(1), Fixed(2))), WithTargets(target))
			source := g.PutOnBattlefield(NewLand("Booster", WithAbility(ability)), a.PlayerID())
			g.pushStack(&StackObject{ID: uuid.New(), SourceID: source.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{recipient.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
			clone := g.Clone()
			power, toughness := 1, 1
			switch change {
			case "power", "toughness":
				dp, dt := 1, 0
				if change == "toughness" {
					dp, dt = 0, 1
				}
				effect := Until(EndOfTurn, ModifyPT(This, Fixed(dp), Fixed(dt)))
				if err := ApplyEffect(clone, effect, recipient.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				power, toughness = power+dp, toughness+dt
			case "return":
				p := clone.FindPermanent(recipient.ID())
				clone.RemoveFromBattlefield(p)
				clone.PutOnBattlefield(p.Card, b.PlayerID())
			default:
				power, toughness = 2, 3
			}
			clone.ResolveStack()
			p := clone.FindPermanent(recipient.ID())
			if p.CurrentPower(clone) != power || p.CurrentToughness(clone) != toughness {
				t.Fatalf("got %d/%d, want %d/%d", p.CurrentPower(clone), p.CurrentToughness(clone), power, toughness)
			}
			if recipient.CurrentPower(g) != 1 || recipient.CurrentToughness(g) != 1 {
				t.Fatal("clone changed original")
			}
		})
	}
}
