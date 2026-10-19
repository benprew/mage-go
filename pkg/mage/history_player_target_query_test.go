package mage

import (
	"reflect"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestHistoryPlayerTargetQuery(t *testing.T) {
	for _, opponent := range []bool{false, true} {
		g, _, obj, _ := targetQueryFixture(t)
		options := []PlayerTargetOption{TargetPlayerAttackedThisTurn()}
		if opponent {
			options = append(options, TargetOpponentOnly())
		}
		target := DeclarePlayerTarget("history", options...)
		query, ok := InspectPlayerTarget(target)
		if !ok {
			t.Fatal("native history declaration rejected")
		}
		obj.TargetSpecs = []Target{target.targetSpec()}
		obj.Targets = []uuid.UUID{obj.Controller}
		check := func() {
			t.Helper()
			random := g.random.Clone()
			for _, player := range g.players {
				obj.Targets[0] = player.PlayerID()
				want := slices.Contains(target.Possible(obj.Controller, obj.Card, g), player.PlayerID())
				for range 2 {
					id, known := query.Select(g, obj)
					if !known || (id != uuid.Nil) != want || id != uuid.Nil && id != player.PlayerID() {
						t.Fatalf("selection=%v/%v want eligible=%v", id, known, want)
					}
				}
			}
			if !reflect.DeepEqual(random, g.random) {
				t.Fatal("query changed RNG")
			}
		}
		check()
		for _, player := range g.players {
			p := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 4), player.PlayerID())
			g.GetCombat().AddAttackerIncarnation(p.ID(), p.incarnationID, g.GetOpponent(player.PlayerID()).PlayerID())
			check()
			g.RemoveFromCombat(p.ID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(player.PlayerID(), []uuid.UUID{p.ID()})
			check()
			g.PhaseIn(g.PhaseOut(p.ID()))
			check()
			g.RemoveFromBattlefield(p)
			g.PutOnBattlefield(p.Card, g.GetOpponent(player.PlayerID()).PlayerID())
			check()
		}
		clone := g.Clone()
		clone.trackers.Turn.ResetForNewTurn(obj.Controller)
		if id, known := query.Select(clone, clone.StackPeek()); !known || id != uuid.Nil {
			t.Fatal("clone did not read its own reset history")
		}
		check()
		obj.Targets[0] = uuid.New()
		if id, known := query.Select(g, obj); !known || id != uuid.Nil {
			t.Fatal("absent player used history")
		}
		foreign, _ := InspectPlayerTarget(DeclarePlayerTarget("history", options...))
		if _, known := foreign.Select(g, obj); known {
			t.Fatal("name replaced declaration identity")
		}
		obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
		obj.Targets = append(obj.Targets, obj.Controller)
		if _, known := query.Select(g, obj); known {
			t.Fatal("duplicate declaration accepted")
		}
	}
}

func TestHistoryPlayerTargetQuerySpell(t *testing.T) {
	for _, state := range []string{"only", "all illegal", "eligible prefix", "zero prefix", "ineligible prefix", "suffix", "foreign suffix", "duplicate", "opaque all illegal", "missing all illegal"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, land := targetQueryFixture(t)
			target := DeclarePlayerTarget("history", TargetPlayerAttackedThisTurn())
			obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
			obj.Targets = append(obj.Targets, obj.Controller)
			g.FireEvent(core.GameEvent{Type: core.EvtDeclaredAttacker, PlayerID: obj.Controller, SourceID: uuid.New()})
			damage := (PacketDamageInstruction{Source: This, Packets: []DamagePacket{DamagePacketTo(target, Fixed(4))}}).Effect()
			obj.Effects = []Effect{damage}
			want, known := false, true
			switch state {
			case "eligible prefix":
				obj.Effects = []Effect{damage, Destroy(land)}
				known = false
			case "zero prefix":
				obj.Effects = []Effect{DealDamagePackets(This, DamagePacketTo(target, Fixed(0))), Destroy(land)}
				known = false
			case "all illegal":
				g.trackers.Turn.ResetForNewTurn(obj.Controller)
				g.PhaseOut(p.ID())
			case "ineligible prefix":
				g.trackers.Turn.ResetForNewTurn(obj.Controller)
				obj.Effects = []Effect{damage, Destroy(land)}
				want = true
			case "suffix":
				obj.Effects = []Effect{Destroy(land), damage}
				want = true
			case "foreign suffix":
				obj.TargetSpecs[1] = DeclarePlayerTarget("history", TargetPlayerAttackedThisTurn()).targetSpec()
				obj.Effects = []Effect{Destroy(land), damage}
				known = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				obj.Targets = append(obj.Targets, obj.Controller)
				known = false
			case "opaque all illegal", "missing all illegal":
				g.trackers.Turn.ResetForNewTurn(obj.Controller)
				g.PhaseOut(p.ID())
				known = false
				if state == "opaque all illegal" {
					obj.Effects = append(obj.Effects, destructionInspectionTrap{})
				} else {
					obj.TargetSpecs = obj.TargetSpecs[:1]
					obj.Targets = obj.Targets[:1]
				}
			}
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					if got, k := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}}); got != want || k != known {
						t.Fatalf("proof=%v/%v want=%v/%v", got, k, want, known)
					}
				}
			}
			if g.GetPlayer(obj.Controller).Life() != 20 || g.FindPermanentIncludingPhased(p.ID()) == nil {
				t.Fatal("query changed state")
			}
		})
	}
}

type historyPlayerTargetTrap struct{ Target }

func (historyPlayerTargetTrap) Possible(uuid.UUID, Card, *Game) []uuid.UUID {
	panic("target callback")
}

func TestHistoryPlayerTargetQueryInvalid(t *testing.T) {
	target := DeclarePlayerTarget("history", TargetPlayerAttackedThisTurn())
	var absentPlayer *PlayerTarget
	var absentOpponent *OpponentTarget
	for _, spec := range []Target{
		absentPlayer, absentOpponent,
		&PlayerTarget{BaseTarget: BaseTarget{min: 0, max: 1}, attackedThisTurn: true},
		&OpponentTarget{BaseTarget: BaseTarget{min: 1, max: 2}, attackedThisTurn: true},
		historyPlayerTargetTrap{TargetPlayer()},
	} {
		target.spec = spec
		if _, ok := InspectPlayerTarget(target); ok {
			t.Fatalf("invalid declaration accepted: %T", spec)
		}
	}
}

func TestHistoryPlayerTargetQueryConsumers(t *testing.T) {
	g, _, obj, _ := targetQueryFixture(t)
	target := DeclarePlayerTarget("history", TargetPlayerAttackedThisTurn())
	obj.TargetSpecs = []Target{target.targetSpec()}
	obj.Targets = []uuid.UUID{obj.Controller}
	for _, eligible := range []bool{false, true} {
		if eligible {
			g.FireEvent(core.GameEvent{Type: core.EvtDeclaredAttacker, PlayerID: obj.Controller, SourceID: uuid.New()})
		}
		for _, effect := range []Effect{
			(LifeInstruction{Subject: target, Amount: Fixed(1), Gain: true}).Effect(),
			(FuturePreventionInstruction{Recipient: target, Amount: Fixed(1)}).Effect(),
			(ManaLossInstruction{Subject: target}).Effect(),
			ReturnEachToHand(PermanentsOwnedBy(target, IsLand)),
		} {
			obj.Effects = []Effect{effect}
			if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
				t.Fatalf("native history consumer rejected: %T eligible=%v", effect, eligible)
			}
		}
	}
}
