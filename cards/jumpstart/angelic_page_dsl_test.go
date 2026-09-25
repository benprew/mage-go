package jumpstart

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestAngelicPageTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Angelic Page")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	ability := mage.UnwrapAbility(card.Abilities()[0]).(mage.ActivatedAbility)
	if ability.SorcerySpeed() || len(ability.Targets()) != 1 {
		t.Fatal("expected one target and instant timing")
	}
	if _, ok := ability.Targets()[0].(mage.TargetBinding); !ok {
		t.Error("missing typed creature target")
	}
	meta := mage.InspectDefinition(ability.Effects()[0])
	if meta.Role != "until" || meta.Properties.PowerBoost != 1 || meta.Properties.ToughnessBoost != 1 || meta.Properties.Mass || meta.Properties.Outcome != mage.OutcomeBenefit {
		t.Fatalf("metadata = %+v", meta)
	}
}

func TestAngelicPageCombatTargets(t *testing.T) {
	for _, blocking := range []bool{false, true} {
		for _, opponent := range []bool{false, true} {
			for _, leavesBefore := range []bool{false, true} {
				t.Run(fmtPageCase(blocking, opponent, leavesBefore), func(t *testing.T) {
					g := gametest.NewTestGame(t)
					a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
					source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Angelic Page")
					owner := a.PlayerID()
					if opponent {
						owner = b.PlayerID()
					}
					target := g.PutOnBattlefield(mage.NewCreature("Combatant", "", 2, 2), owner)
					idle := g.PutOnBattlefield(mage.NewCreature("Idle", "", 2, 2), owner)
					artifact := g.PutOnBattlefield(mage.NewArtifact("Noncreature", ""), owner)
					g.GetCombat().AddAttacker(artifact.ID(), b.PlayerID())
					if blocking {
						attacker := uuid.New()
						g.GetCombat().AddAttacker(attacker, owner)
						g.GetCombat().AddBlocker(target.ID(), attacker)
					} else {
						g.GetCombat().AddAttacker(target.ID(), b.PlayerID())
					}
					g.SetStep(core.DeclareBlockers)
					ability := mage.UnwrapAbility(g.FindPermanent(source).Card.Abilities()[0]).(mage.ActivatedAbility)
					possible := ability.Targets()[0].Possible(a.PlayerID(), g.FindPermanent(source).Card, g.Game)
					if !slices.Contains(possible, target.ID()) || slices.Contains(possible, idle.ID()) || slices.Contains(possible, artifact.ID()) {
						t.Fatalf("incorrect combat targets: %v", possible)
					}
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source, 0, []uuid.UUID{target.ID()}); err != nil {
						t.Fatal(err)
					}
					if !g.FindPermanent(source).Tapped {
						t.Fatal("source did not pay tap cost")
					}
					if leavesBefore {
						g.RemoveFromCombat(target.ID())
					}
					g.RemoveFromBattlefield(g.FindPermanent(source))
					g.ResolveStack()
					g.RemoveFromCombat(target.ID())
					g.ApplyContinuousEffects()
					want := 3
					if leavesBefore {
						want = 2
					}
					if target.CurrentPower(g.Game) != want || target.CurrentToughness(g.Game) != want {
						t.Fatalf("P/T = %d/%d, want %d/%d", target.CurrentPower(g.Game), target.CurrentToughness(g.Game), want, want)
					}
					g.GetEffects().RemoveEndOfTurn()
					g.ApplyContinuousEffects()
					if target.CurrentPower(g.Game) != 2 || target.CurrentToughness(g.Game) != 2 {
						t.Fatal("boost did not expire")
					}
				})
			}
		}
	}
}

func fmtPageCase(blocking, opponent, leavesBefore bool) string {
	name := "attacker"
	if blocking {
		name = "blocker"
	}
	if opponent {
		name += "/opponent"
	} else {
		name += "/controller"
	}
	if leavesBefore {
		name += "/leaves before resolution"
	} else {
		name += "/leaves after resolution"
	}
	return name
}
