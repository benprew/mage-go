package mage

import (
	"math"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type nontramplePlayer struct {
	*BasePlayer
	assign        func(*Permanent, []*Permanent, int) map[uuid.UUID]int
	orders, calls int
}

func (p *nontramplePlayer) GetBlockerOrder(_ *Game, _ *Permanent, _ []*Permanent, _ int) []uuid.UUID {
	p.orders++
	return nil
}
func (p *nontramplePlayer) GetCombatDamageAssignment(_ *Game, source *Permanent, blockers []*Permanent, amount int) map[uuid.UUID]int {
	p.calls++
	if p.assign != nil {
		return p.assign(source, blockers, amount)
	}
	return nil
}

func nontrampleGame() (*Game, *nontramplePlayer, *BasePlayer, *Permanent, *Permanent, *Permanent) {
	a := &nontramplePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 3, 20, WithKeyword(core.Lifelink)), a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 20), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 20), b.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{source.ID()})
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: first.ID(), AttackerID: source.ID()}, {BlockerID: second.ID(), AttackerID: source.ID()}})
	return g, a, b, source, first, second
}

func TestNontrampleAssignmentFreeDivision(t *testing.T) {
	for _, split := range []int{0, 1, 2, 3} {
		t.Run(string(rune('0'+split)), func(t *testing.T) {
			g, a, b, source, first, second := nontrampleGame()
			assignment := map[uuid.UUID]int{first.ID(): split, second.ID(): 3 - split}
			a.assign = func(_ *Permanent, _ []*Permanent, _ int) map[uuid.UUID]int { return assignment }
			g.ExecuteCombatDamage()
			if first.Damage != split || second.Damage != 3-split || source.Damage != 2 || a.Life() != 23 || b.Life() != 20 {
				t.Fatalf("split=%d damage=%d/%d/%d life=%d/%d", split, first.Damage, second.Damage, source.Damage, a.Life(), b.Life())
			}
			if a.orders != 0 || a.calls != 1 {
				t.Fatalf("order calls=%d assignment calls=%d", a.orders, a.calls)
			}
			if assignment[first.ID()] != split || assignment[second.ID()] != 3-split {
				t.Fatal("caller map changed")
			}
		})
	}
}

func TestNontrampleAssignmentInvalid(t *testing.T) {
	for _, invalid := range []string{"nil", "empty", "under", "over", "negative", "foreign", "foreign-zero", "overflow"} {
		t.Run(invalid, func(t *testing.T) {
			g, a, b, source, first, second := nontrampleGame()
			first.Card.SetBasePT(1, 1)
			a.assign = func(_ *Permanent, _ []*Permanent, amount int) map[uuid.UUID]int {
				switch invalid {
				case "nil":
					return nil
				case "empty":
					return map[uuid.UUID]int{}
				case "under":
					return map[uuid.UUID]int{first.ID(): 1}
				case "over":
					return map[uuid.UUID]int{first.ID(): amount + 1}
				case "negative":
					return map[uuid.UUID]int{first.ID(): -1, second.ID(): amount + 1}
				case "foreign":
					return map[uuid.UUID]int{first.ID(): amount, uuid.New(): 1}
				case "foreign-zero":
					return map[uuid.UUID]int{first.ID(): amount, uuid.New(): 0}
				default:
					return map[uuid.UUID]int{first.ID(): math.MaxInt, second.ID(): math.MaxInt}
				}
			}
			g.ExecuteCombatDamage()
			if first.Damage != 1 || second.Damage != 2 || source.Damage != 2 || a.Life() != 23 || b.Life() != 20 {
				t.Fatalf("invalid plan used: %d/%d/%d life=%d/%d", first.Damage, second.Damage, source.Damage, a.Life(), b.Life())
			}
		})
	}
}

func TestNontrampleAssignmentTimingAndIdentity(t *testing.T) {
	for _, change := range []string{"first-strike", "double-strike", "zero", "negative", "removed-blocker", "returned-blocker", "phased-blocker", "phase-cycle", "changed-control", "returned-source", "phased-source", "stale-blocker", "stale-source"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, source, first, second := nontrampleGame()
			first.Card.SetBasePT(1, 1)
			a.assign = func(_ *Permanent, recipients []*Permanent, amount int) map[uuid.UUID]int {
				return map[uuid.UUID]int{recipients[len(recipients)-1].ID(): amount}
			}
			switch change {
			case "first-strike":
				source.GrantBaseAttr(core.FirstStrike)
			case "double-strike":
				source.GrantBaseAttr(core.DoubleStrike)
			case "zero":
				source.Card.SetBasePT(0, 20)
			case "negative":
				source.Card.SetBasePT(-2, 20)
			case "removed-blocker":
				g.RemoveFromCombat(first.ID())
			case "returned-blocker":
				card := first.Card
				g.BouncePermanentToHand(first)
				b.RemoveFromHand(first.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "phased-blocker":
				g.PhaseOut(first.ID())
			case "phase-cycle":
				g.PhaseOut(first.ID())
				g.PhaseIn([]uuid.UUID{first.ID()})
			case "changed-control":
				g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: first.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "returned-source":
				card := source.Card
				g.BouncePermanentToHand(source)
				a.RemoveFromHand(source.ID())
				source = g.PutOnBattlefield(card, a.PlayerID())
			case "phased-source":
				g.PhaseOut(source.ID())
			case "stale-blocker":
				g.GetCombat().Groups[0].BlockerIncarnations[0] = uuid.New()
			case "stale-source":
				g.GetCombat().AttackerIncarnations[source.ID()] = uuid.New()
			}
			g.GetCombat().ResolveDamage(g, true)
			wantFirst := 0
			if change == "first-strike" || change == "double-strike" {
				wantFirst = 3
			}
			if g.FindPermanent(second.ID()).Damage != wantFirst {
				t.Fatal("incorrect first-strike amount")
			}
			g.GetCombat().ResolveDamage(g, false)
			want := 3
			if change == "double-strike" {
				want = 6
			}
			if change == "zero" || change == "negative" || change == "returned-source" || change == "phased-source" || change == "stale-source" {
				want = 0
			}
			if g.FindPermanent(second.ID()).Damage != want || b.Life() != 20 {
				t.Fatalf("outgoing=%d life=%d", g.FindPermanent(second.ID()).Damage, b.Life())
			}
			wantIncoming := 2
			if change == "removed-blocker" || change == "returned-blocker" || change == "phased-blocker" || change == "phase-cycle" || change == "changed-control" || change == "stale-blocker" {
				wantIncoming = 1
			}
			if change == "returned-source" || change == "phased-source" || change == "stale-source" {
				wantIncoming = 0
			}
			if change == "phased-source" {
				g.PhaseIn([]uuid.UUID{source.ID()})
			}
			if g.FindPermanent(source.ID()).Damage != wantIncoming {
				t.Fatalf("incoming=%d want=%d", g.FindPermanent(source.ID()).Damage, wantIncoming)
			}
		})
	}
}

func TestNontrampleAssignmentBatchAndRetainedSource(t *testing.T) {
	g, a, baseB, source, first, second := nontrampleGame()
	b := &bandSourcePlayer{BasePlayer: baseB, numbers: []int{1}}
	g.players[1] = b
	first.Card.SetBasePT(0, 20)
	second.Card.SetBasePT(0, 20)
	other := g.PutOnBattlefield(NewCreature("Other source", "", 3, 20), a.PlayerID())
	other.RevokeBaseAttr(core.AttrSummonSick)
	g.GetCombat().AddAttackerIncarnation(other.ID(), other.incarnationID, b.PlayerID())
	g.GetCombat().AddBlockerIncarnation(first.ID(), first.incarnationID, other.ID())
	g.GetCombat().AddBlockerIncarnation(second.ID(), second.incarnationID, other.ID())
	shared := map[uuid.UUID]int{}
	a.assign = func(_ *Permanent, _ []*Permanent, amount int) map[uuid.UUID]int {
		clear(shared)
		shared[first.ID()] = 1
		shared[second.ID()] = amount - 1
		return shared
	}
	saved := g.CaptureDamageSource(source.ID())
	g.AddPreventionShield(first.ID(), 1)
	b.onNumber = func() {
		g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		returned := g.PutOnBattlefield(source.Card, b.PlayerID())
		returned.RevokeBaseAttr(core.Lifelink)
	}
	notifications := 0
	g.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
		notifications++
		if g.FindPermanent(first.ID()).Damage != 1 || g.FindPermanent(second.ID()).Damage != 4 || a.Life() != 22 {
			t.Error("notification saw a partial batch or lost original lifelink")
		}
	})
	g.ExecuteCombatDamage()
	if b.numberCalls != 1 || notifications != 3 || a.Life() != 22 || b.Life() != 20 || a.calls != 2 {
		t.Fatalf("choices=%d notifications=%d life=%d/%d calls=%d", b.numberCalls, notifications, a.Life(), b.Life(), a.calls)
	}
	if !g.HasDealtDamageToPermanentFromSource(saved, second) || g.HasDealtDamageToPermanentFromSource(g.CaptureDamageSource(source.ID()), second) {
		t.Fatal("returned source replaced original damage information")
	}
	if len(shared) != 2 || shared[first.ID()] != 1 || shared[second.ID()] != 2 {
		t.Fatal("caller map changed")
	}
}

func TestNontrampleAssignmentReplacementOrder(t *testing.T) {
	for _, redirectFirst := range []bool{false, true} {
		g, a, baseB, _, first, second := nontrampleGame()
		b := &bandSourcePlayer{BasePlayer: baseB}
		g.players[1] = b
		if redirectFirst {
			b.mode = 1
		}
		a.assign = func(_ *Permanent, _ []*Permanent, _ int) map[uuid.UUID]int {
			return map[uuid.UUID]int{first.ID(): 1, second.ID(): 2}
		}
		g.AddPreventionShield(second.ID(), 1)
		g.SetCreatureDamageRedirect(second.ID(), b.PlayerID())
		g.ExecuteCombatDamage()
		want, wantLife := 19, 22
		if redirectFirst {
			want, wantLife = 18, 23
		}
		if first.Damage != 1 || second.Damage != 0 || a.Life() != wantLife || b.Life() != want {
			t.Fatalf("redirect first=%v damage=%d/%d life=%d/%d", redirectFirst, first.Damage, second.Damage, a.Life(), b.Life())
		}
	}
}

func TestNontrampleAssignmentDeathtouch(t *testing.T) {
	g, a, _, source, first, second := nontrampleGame()
	source.GrantBaseAttr(core.Deathtouch)
	a.assign = func(_ *Permanent, _ []*Permanent, _ int) map[uuid.UUID]int {
		return map[uuid.UUID]int{first.ID(): 1, second.ID(): 2}
	}
	g.ExecuteCombatDamage()
	if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil || source.Damage != 2 || a.Life() != 23 {
		t.Fatal("deathtouch changed the division or simultaneous results")
	}
}
