package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type pileBlockingPlayer struct {
	*BasePlayer
	modes        []int
	calls        int
	declarations int
	beforeChoice func()
}

func (p *pileBlockingPlayer) ChooseMode(_ []string, _ string) int {
	p.calls++
	if p.beforeChoice != nil {
		f := p.beforeChoice
		p.beforeChoice = nil
		f()
	}
	if len(p.modes) == 0 {
		return 0
	}
	result := p.modes[0]
	p.modes = p.modes[1:]
	return result
}
func (p *pileBlockingPlayer) DeclareBlockers(*Game) []BlockAssignment { p.declarations++; return nil }
func randomBlockingGame(t *testing.T) (*Game, *BasePlayer, *pileBlockingPlayer, []*Permanent) {
	t.Helper()
	a := NewBasePlayer("A")
	b := &pileBlockingPlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	attackers := make([]*Permanent, 0, 2)
	for range 2 {
		p := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 8), a.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		attackers = append(attackers, p)
	}
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attackers[0].ID(), attackers[1].ID()})
	if err := ApplyEffect(g, RandomizeBlockingPilesThisTurn(), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	return g, a, b, attackers
}
func TestRandomBlockingPiles(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(map[bool]string{false: "search", true: "native"}[native], func(t *testing.T) {
			g, _, b, attackers := randomBlockingGame(t)
			blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
			b.modes = []int{1}
			g.SetRandomResults([]int{0, 37})
			g.SetStep(core.DeclareBlockers)
			if native {
				g.doDeclareBlockers()
			} else {
				g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attackers[0].ID()}})
			}
			if b.declarations != 0 || b.calls != 1 {
				t.Fatal("ordinary declaration used", b.declarations, b.calls)
			}
			if !slices.Contains(g.combat.GroupFor(attackers[1].ID()).BlockerIDs, blocker.ID()) || g.combat.GroupFor(attackers[0].ID()).Blocked {
				t.Fatal("pile was not permuted before blocks")
			}
			if g.RandIntn(100) != 37 {
				t.Fatal("wrong RNG consumption")
			}
			if len(g.GetBlockedThisTurn(blocker.ID())) != 1 {
				t.Fatal("missing blocking history")
			}
		})
	}
}
func TestRandomBlockingPilesLegality(t *testing.T) {
	for _, condition := range []string{"ground", "flying", "tapped", "phased", "cant-block", "landwalk", "menace", "requirement"} {
		t.Run(condition, func(t *testing.T) {
			g, _, b, attackers := randomBlockingGame(t)
			blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
			want := condition == "ground" || condition == "flying"
			switch condition {
			case "flying":
				g.MutablePermanent(attackers[0].ID()).GrantBaseAttr(core.Flying)
				g.MutablePermanent(blocker.ID()).GrantBaseAttr(core.Reach)
			case "tapped":
				g.TapPermanent(blocker)
			case "phased":
				g.PhaseOut(blocker.ID())
			case "cant-block":
				g.MutablePermanent(blocker.ID()).RevokeBaseAttr(core.AttrCanBlock)
			case "landwalk":
				g.MutablePermanent(attackers[0].ID()).GrantBaseAttr(core.Forestwalk)
				g.PutOnBattlefield(NewLand("Forest", WithSubTypes("Forest")), b.PlayerID())
			case "menace":
				g.MutablePermanent(attackers[0].ID()).GrantBaseAttr(core.Menace)
			case "requirement":
				g.MutablePermanent(attackers[0].ID()).GrantBaseAttr(core.MustBeBlocked)
			}
			g.ApplyContinuousEffects()
			b.modes = []int{1}
			if condition == "requirement" {
				b.modes = []int{0}
			}
			g.SetRandomResults([]int{1})
			g.ExecuteBlockers(nil)
			if g.combat.GroupFor(attackers[0].ID()).Blocked != want {
				t.Fatal("incorrect legal block", condition)
			}
		})
	}
}
func TestRandomBlockingPilesCapacity(t *testing.T) {
	for _, capacity := range []core.Attr{0, core.CanBlockAdditional, core.CanBlockAny} {
		g, _, b, attackers := randomBlockingGame(t)
		blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
		if capacity != 0 {
			blocker.GrantBaseAttr(capacity)
			g.ApplyContinuousEffects()
		}
		b.modes = []int{1, 1}
		g.SetRandomResults([]int{1})
		g.ExecuteBlockers(nil)
		if !g.combat.GroupFor(attackers[0].ID()).Blocked || g.combat.GroupFor(attackers[1].ID()).Blocked != (capacity != 0) {
			t.Fatal("incorrect extra pile capacity", capacity)
		}
	}
}
func TestRandomBlockingPilesLifecycle(t *testing.T) {
	g, a, b, attackers := randomBlockingGame(t)
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
	b.modes = []int{1}
	g.SetRandomResults([]int{1, 19})
	clone := g.Clone()
	b.beforeChoice = func() { g.RemoveFromBattlefield(blocker); g.PutOnBattlefield(blocker.Card, b.PlayerID()) }
	g.ExecuteBlockers(nil)
	if g.combat.GroupFor(attackers[0].ID()).Blocked {
		t.Fatal("returned visit inherited a pile")
	}
	if clone.RandIntn(2) != 1 || g.RandIntn(100) != 19 {
		t.Fatal("clone shared RNG")
	}
	clone.SetTurn(clone.CurrentTurn() + 1)
	if clone.turns.randomBlockingPiles == g.turns.randomBlockingPiles {
		t.Fatal("turn replacement state shared or not cleared")
	}
	g.doCleanupActions()
	if g.turns.randomBlockingPiles {
		t.Fatal("cleanup retained turn effect")
	}
	if err := ApplyEffect(g, RandomizeBlockingPilesThisTurn(), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.combat.Reset()
	late := g.PutOnBattlefield(NewCreature("Late", "", 1, 8), a.PlayerID())
	late.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{late.ID()})
	b.modes = []int{1}
	g.ExecuteBlockers(nil)
	if !g.combat.GroupFor(late.ID()).Blocked {
		t.Fatal("later combat did not use live creatures")
	}
}
func TestRandomBlockingPilesDefinition(t *testing.T) {
	card := NewInstant("Random piles", "", WithAbility(Spell(RandomizeBlockingPilesThisTurn())))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestRandomBlockingPilesBatchAndBands(t *testing.T) {
	for _, band := range []bool{false, true} {
		g, a, b, attackers := randomBlockingGame(t)
		if band {
			g.combat.AddBand([]uuid.UUID{attackers[0].ID(), attackers[1].ID()})
		}
		g.MutablePermanent(attackers[0].ID()).GrantBaseAttr(core.Menace)
		for range 2 {
			g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
		}
		var observations int
		listener := NewTriggered(core.EvtBlockerGroup, false).SetCondition(func(evt *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
			if evt.SourceID == attackers[0].ID() {
				observations++
				if len(evt.RelatedIDs) != 2 || len(g.combat.GroupFor(attackers[0].ID()).BlockerIDs) != 2 {
					t.Error("event observed incomplete batch")
				}
			}
			return false
		})
		g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(listener)), a.PlayerID())
		b.modes = []int{1, 1}
		g.SetRandomResults([]int{1})
		g.ExecuteBlockers(nil)
		if observations != 1 || !g.combat.GroupFor(attackers[0].ID()).Blocked || g.combat.GroupFor(attackers[1].ID()).Blocked != band {
			t.Fatal("wrong group or band", observations)
		}
	}
}
func TestRandomBlockingPilesEmptyAndClone(t *testing.T) {
	g, a, b, _ := randomBlockingGame(t)
	g.combat.Reset()
	g.SetRandomResults([]int{71})
	g.ExecuteBlockers(nil)
	if b.calls != 0 || g.RandIntn(100) != 71 {
		t.Fatal("empty combat made choices")
	}
	for range 2 {
		g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
	}
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 8), a.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
	clone := g.Clone()
	if !clone.turns.randomBlockingPiles {
		t.Fatal("clone lost pending effect")
	}
	b.modes = []int{-1, 1000}
	g.ExecuteBlockers(nil)
	if g.combat.GroupFor(attacker.ID()).Blocked {
		t.Fatal("invalid optional answers forced a block")
	}
	clone.doCleanupActions()
	if !g.turns.randomBlockingPiles {
		t.Fatal("clone cleanup changed original")
	}
}
