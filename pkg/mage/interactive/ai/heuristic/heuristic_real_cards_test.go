package heuristic

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai/combatsolver"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"

	_ "git.sr.ht/~cdcarter/mage-go/cards" // register all card sets
)

// TestHeuristicAttackers_RealCardLethalScenario reproduces a real-game
// position the AI refused to attack from. Uses actual registered cards so
// every static effect (Bad Moon's +1/+1, Unholy Strength's +2/+2, Lance's
// first strike) and triggered ability (Abu Ja'far's "destroy whatever
// damaged me") is in play.
func TestHeuristicAttackers_RealCardLethalScenario(t *testing.T) {
	pa := mage.NewBasePlayer("AI")
	pb := mage.NewBasePlayer("Opp")
	g := mage.NewGame(pa, pb)

	pa.SetLife(11)
	pb.SetLife(3)

	put := func(name string, owner *mage.BasePlayer) *mage.Permanent {
		t.Helper()
		card, err := mage.CreateCard(name)
		if err != nil {
			t.Fatalf("CreateCard %q: %v", name, err)
		}
		card.SetOwner(owner.PlayerID())
		perm := g.PutOnBattlefield(card, owner.PlayerID())
		perm.RevokeBaseAttr(core.AttrSummonSick)
		return perm
	}

	// AI side
	put("Drudge Skeletons", pa)
	scathe := put("Scathe Zombies", pa)
	put("Lost Soul", pa)
	put("Bad Moon", pa)
	put("Bad Moon", pa)
	put("Flying Carpet", pa)
	put("Flying Carpet", pa)
	g.Attach(put("Unholy Strength", pa).ID(), scathe.ID())
	g.Attach(put("Unholy Strength", pa).ID(), scathe.ID())
	for range 5 {
		put("Swamp", pa)
	}

	// Opponent side
	put("Pikemen", pb)
	blacksmith := put("Argivian Blacksmith", pb)
	put("Abu Ja'far", pb)
	put("Ivory Cup", pb)
	g.Attach(put("Lance", pb).ID(), blacksmith.ID())
	g.Attach(put("Artifact Ward", pb).ID(), blacksmith.ID())
	for range 6 {
		put("Plains", pb)
	}

	if pow, tgh := scathe.CurrentPower(g), scathe.CurrentToughness(g); pow != 8 || tgh != 6 {
		t.Fatalf("Scathe Zombies = %d/%d, want 8/6 (setup is wrong)", pow, tgh)
	}

	personalities := []struct {
		name string
		w    ai.WeightedPersonality
	}{
		{"Midrange", ai.MidrangeWeighted},
		{"Aggro", ai.AggroWeighted},
		{"Control", ai.ControlWeighted},
		{"Tempo", ai.TempoWeighted},
		{"Burn", ai.BurnWeighted},
	}
	for _, p := range personalities {
		t.Run(p.name, func(t *testing.T) {
			s := New(p.w)

			r := combatsolver.SolveAttack(g, pa.PlayerID(), combatsolver.Options{Profile: combatsolver.Profile{
				Weights:        p.w.Weights,
				Aggression:     p.w.Aggression,
				BlockThreshold: p.w.BlockThreshold,
			}})
			lethal := eval.CalculateLethal(g, pa.PlayerID())
			t.Logf("SolveAttack: attackers=%s score=%d nodes=%d deadlineHit=%v heldTricks=%d  |  lethal=%+v",
				attackerNames(g, r.Attackers), r.Score, r.Nodes, r.DeadlineHit, len(r.HeldTricks), lethal)

			if !lethal.IHaveLethal {
				t.Errorf("CalculateLethal: IHaveLethal=false, expected true (Flying Carpet grants Lost Soul flying for lethal)")
			}

			attackers := s.Attackers(pa, g)
			t.Logf("Strategy.Attackers: %v", attackerNames(g, attackers))
			if len(attackers) == 0 {
				t.Errorf("declared zero attackers; expected at least one")
			}

			act := s.PriorityAction(pa, g, 0, true)
			tgtName := "(none)"
			if len(act.Targets) > 0 {
				tgtName = permName(g, act.Targets[0])
			}
			t.Logf("PriorityAction(main): type=%v perm=%s ability=%d target=%s",
				act.Type, permName(g, act.PermanentID), act.AbilityIndex, tgtName)

			if act.Type != interactive.ActionActivateAbility {
				t.Errorf("expected main-phase action ActivateAbility (Flying Carpet), got %v", act.Type)
			}
			if perm := g.FindPermanent(act.PermanentID); perm == nil || perm.Name() != "Flying Carpet" {
				t.Errorf("expected activation of Flying Carpet, got %s", permName(g, act.PermanentID))
			}
			if len(act.Targets) == 0 {
				t.Errorf("expected Flying Carpet to target a creature, got no target")
			} else if tgt := g.FindPermanent(act.Targets[0]); tgt == nil || tgt.Controller != pa.PlayerID() {
				t.Errorf("Flying Carpet targeted %s controlled by %v; expected one of AI's own creatures",
					permName(g, act.Targets[0]), tgt.Controller)
			}
		})
	}
}

func attackerNames(g *mage.Game, ids []uuid.UUID) string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		names = append(names, permName(g, id))
	}
	return "[" + strings.Join(names, ", ") + "]"
}

func permName(g *mage.Game, id uuid.UUID) string {
	if perm := g.FindPermanent(id); perm != nil {
		return perm.Name()
	}
	return id.String()
}
