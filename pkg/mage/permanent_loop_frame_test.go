package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestPermanentLoopFrameIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Member", "", 1, 10), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Other", "", 1, 10), b.PlayerID())
	member := BindPermanent("member")
	source := BindDamageSource("saved source")
	result := BindDamageResults("results")
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{source: g.CaptureDamageSource(p.ID())}
	g.resolution.damageResultsBindings = map[*DamageResultsBinding][]damageOutcome{result: {{amount: 3}}}
	calls := 0
	body := FuncEffect("check frame", EffectProperties{}, func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		calls++
		if !g.resolution.damageSourceBindings[source].Valid() || g.resolution.damageResultsBindings[result][0].amount != 3 {
			t.Error("previous iteration changed the frame")
		}
		delete(g.resolution.damageSourceBindings, source)
		g.resolution.damageResultsBindings[result][0].amount = 9
		return nil
	})
	if err := ForEach(MatchingPermanents(IsCreature), member, body).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || !g.resolution.damageSourceBindings[source].Valid() || g.resolution.damageResultsBindings[result][0].amount != 3 {
		t.Fatal("loop changed outer bindings")
	}
}

func TestPermanentLoopSourceQuery(t *testing.T) {
	member := BindPermanent("member")
	for _, effect := range []ResolvingEffect{
		ForEach(SingletonPermanent(This), member, TapSubject(From(member))),
		ForEach(MatchingPermanents(IsCreature), member, TapSubject(This)),
	} {
		if !effectReadsSourcePermanent(effect) {
			t.Fatal("loop lost source reference")
		}
	}
}

type permanentLoopOrderPlayer struct {
	*BasePlayer
	reverse bool
}

func (p *permanentLoopOrderPlayer) ChoosePermanent(candidates []*Permanent, _ string, _ GameReader) *Permanent {
	if p.reverse {
		return candidates[len(candidates)-1]
	}
	return candidates[0]
}

func TestPermanentLoopTurnOrder(t *testing.T) {
	a := &permanentLoopOrderPlayer{BasePlayer: NewBasePlayer("A"), reverse: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	other := g.PutOnBattlefield(NewCreature("Nonactive", "", 1, 10), b.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 10), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 10), a.PlayerID())
	member := BindPermanent("member")
	var visits []uuid.UUID
	body := FuncEffect("visit", EffectProperties{}, func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		visits = append(visits, g.resolution.permanentBindings[member].id)
		return nil
	})
	if err := ForEachInTurnOrder(MatchingPermanents(IsCreature), member, body).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(visits) != 3 || visits[0] != second.ID() || visits[1] != first.ID() || visits[2] != other.ID() {
		t.Fatal("loop ignored turn order or controller choice", visits)
	}
}

func TestPermanentLoopCapturedVisits(t *testing.T) {
	for _, change := range []string{"return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first := g.PutOnBattlefield(NewCreature("First", "", 1, 10), a.PlayerID())
			second := g.PutOnBattlefield(NewCreature("Second", "", 1, 10), a.PlayerID())
			member := BindPermanent("member")
			calls := 0
			body := Sequence(FuncEffect("change next visit", EffectProperties{}, func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
				calls++
				if calls == 1 {
					switch change {
					case "return":
						g.RemoveFromBattlefield(second)
						g.PutOnBattlefield(second.Card, a.PlayerID())
					case "phase":
						g.PhaseOut(second.ID())
					case "control":
						g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: second.ID(), ControllerID: b.PlayerID()})
					}
					g.PutOnBattlefield(NewCreature("Later", "", 1, 10), a.PlayerID())
				}
				return nil
			}), TapSubject(From(member)))
			if err := ForEachInTurnOrder(MatchingPermanents(IsCreature), member, body).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if calls != 2 || !g.FindPermanent(first.ID()).Tapped {
				t.Fatal("loop reread membership")
			}
			for _, p := range g.AllBattlefield() {
				if p.Name() == "Later" && p.Tapped {
					t.Fatal("new member entered loop")
				}
			}
			p := g.FindPermanentIncludingPhased(second.ID())
			if p.Tapped != (change == "control") {
				t.Fatal("loop followed returned or phased visit, or lost changed control")
			}
		})
	}
}
