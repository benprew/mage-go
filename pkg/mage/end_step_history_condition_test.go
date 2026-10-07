package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceAttackHistoryThisTurn(t *testing.T) {
	for _, attacked := range []bool{false, true} {
		for _, change := range []string{"live", "leave", "return", "phase", "phase-cycle", "control", "unknown", "cleanup", "combat-only"} {
			t.Run(map[bool]string{false: "idle/", true: "attacked/"}[attacked]+change, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				g.SetTurn(1)
				c := NewCreature("Source", "", 2, 4)
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				ctx := StateContext{SourceID: p.ID(), ControllerID: a.PlayerID(), SourceIncarnation: p.incarnationID}
				if attacked {
					g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
				}
				g.GetCombat().RemoveFromCombat(p.ID())
				switch change {
				case "leave", "return":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					if change == "return" {
						a.RemoveFromHand(c.ID())
						q := g.PutOnBattlefield(c, a.PlayerID())
						q.RevokeBaseAttr(core.AttrSummonSick)
						if !attacked {
							g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{q.ID()})
						}
					}
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "unknown":
					ctx.SourceIncarnation = uuid.New()
				case "cleanup":
					g.RunStepWithPriority(core.Cleanup)
				case "combat-only":
					g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
				}
				wantAttack := attacked && change != "cleanup"
				for _, game := range []*Game{g, g.Clone()} {
					for _, expected := range []bool{false, true} {
						want := change != "unknown" && wantAttack == expected
						if SourceAttackHistoryThisTurn(expected).checkState(game, ctx) != want {
							t.Fatalf("history=%v, want %v", expected, want)
						}
					}
				}
				if change == "leave" || change == "phase" {
					ctx.SourceIncarnation = uuid.Nil
					if SourceAttackHistoryThisTurn(false).checkState(g, ctx) {
						t.Fatal("uncaptured absent source matched")
					}
				}
			})
		}
	}
}

func TestCreatureDiedThisTurnCondition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := CreatureDiedThisTurnCondition()
	if condition.checkState(g, StateContext{}) {
		t.Fatal("empty history")
	}
	landCard := NewLand("Land")
	landCard.SetOwner(a.PlayerID())
	land := g.PutOnBattlefield(landCard, a.PlayerID())
	g.DestroyPermanent(land)
	if condition.checkState(g, StateContext{}) {
		t.Fatal("noncreature death counted")
	}
	tokenCard := NewCreature("Token", "", 1, 1)
	tokenCard.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(tokenCard, b.PlayerID())
	p.IsToken = true
	g.DestroyPermanent(p)
	clone := g.Clone()
	if !condition.checkState(g, StateContext{}) || !condition.checkState(clone, StateContext{}) {
		t.Fatal("token death absent")
	}
	g.RunStepWithPriority(core.Cleanup)
	if condition.checkState(g, StateContext{}) || !condition.checkState(clone, StateContext{}) {
		t.Fatal("cleanup or clone history")
	}
	invalid := NewInstant("Invalid", "", WithEffect(IfState(SourceAttackHistoryThisTurn(false), GainLifeFor(You, Fixed(1)))))
	if ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("nonpermanent source allowed")
	}
}
