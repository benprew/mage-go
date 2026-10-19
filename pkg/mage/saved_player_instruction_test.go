package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSavedPlayerInstructionDeclaration(t *testing.T) {
	binding := BindPlayer("saved")
	body := AtNext(EachEndStep, DestroyEach(PermanentsControlledBy(FromPlayer(binding), IsCreature)))
	effect := (SavedPlayerInstruction{Subject: ActivePlayer, Binding: binding, Body: body}).Effect()
	for range 2 {
		got, ok := InspectSavedPlayerInstruction(effect)
		if !ok || got.Subject != ActivePlayer || got.Binding != binding || got.Body != body {
			t.Fatal("lost declaration")
		}
	}
	for _, bad := range []SavedPlayerInstruction{
		{}, {Binding: binding, Body: body}, {Subject: You, Body: body},
		{Subject: You, Binding: &PlayerBinding{}, Body: body},
		{Subject: You, Binding: binding}, {Subject: (*PlayerTargetBinding)(nil), Binding: binding, Body: body},
		{Subject: You, Binding: binding, Body: (*sequenceEffect)(nil)},
	} {
		if bad.Effect() != nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	for _, bad := range []Effect{nil, (*sequenceEffect)(nil), Sequence(), Sequence(SnapshotPlayer(You, binding)), Sequence(body, SnapshotPlayer(You, binding)), Sequence(SnapshotPlayer(You, binding), body, body), Sequence((*playerSnapshot)(nil), body), destructionInspectionTrap{}} {
		if _, ok := InspectSavedPlayerInstruction(bad); ok {
			t.Fatal("invalid scope accepted")
		}
	}
}

func TestSavedPlayerInstructionQuery(t *testing.T) {
	for _, state := range []string{"alone", "prefix", "suffix", "foreign binding", "opaque player", "foreign player", "opaque filter", "opaque exclusion", "opaque body", "unknown suffix", "illegal", "unknown illegal", "copy", "clone", "escaped", "read before write", "nested scope", "wrong duration", "foreign collection target"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			binding := BindPlayer("saved")
			ref := FromPlayer(binding)
			var player PlayerRef = ActivePlayer
			filter := IsCreature
			known, want := true, false
			trap := NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter ran") })
			switch state {
			case "foreign binding":
				ref = FromPlayer(BindPlayer("saved"))
				known = false
			case "opaque player":
				player = lifeInspectionPlayerTrap{}
				known = false
			case "foreign player":
				player = DeclarePlayerTarget("foreign")
				known = false
			case "opaque filter", "unknown illegal":
				filter = And(IsCreature, trap)
				known = false
			}
			excluded := MatchingPermanents(Not(ControlledSinceCurrentTurnStart))
			if state == "opaque exclusion" {
				excluded = MatchingPermanents(trap)
				known = false
			}
			subjects := IntersectPermanents(PermanentsControlledBy(ref, filter), WithoutPermanents(MatchingPermanents(AnyPermanent), excluded))
			if state == "foreign collection target" {
				subjects = IntersectPermanents(subjects, PermanentsControlledBy(DeclarePlayerTarget("foreign"), AnyPermanent))
				known = false
			}
			body := DestroyEach(subjects)
			if state == "opaque body" {
				body = Sequence(body, destructionInspectionTrap{})
				known = false
			}
			delay := AtNext(EachEndStep, body)
			if state == "wrong duration" {
				delay = AtEach(EachEndStep, body)
				known = false
			}
			effect := (SavedPlayerInstruction{Subject: player, Binding: binding, Body: Sequence(delay)}).Effect()
			obj.Effects = []Effect{effect}
			switch state {
			case "prefix":
				obj.Effects = []Effect{effect, Destroy(target)}
				want = true
			case "suffix", "copy", "clone":
				obj.Effects = []Effect{Destroy(target), effect}
				want = true
			case "unknown suffix":
				obj.Effects = []Effect{Destroy(target), effect, destructionInspectionTrap{}}
				known = false
			case "illegal", "unknown illegal":
				p.GrantBaseAttr(core.Shroud)
			case "escaped":
				obj.Effects = []Effect{effect, AtNext(EachEndStep, body)}
				known = false
			case "read before write":
				obj.Effects = []Effect{AtNext(EachEndStep, body), effect}
				known = false
			case "nested scope":
				obj.Effects = []Effect{(SavedPlayerInstruction{Subject: You, Binding: BindPlayer("outer"), Body: effect}).Effect()}
				known = false
			}
			if state == "copy" {
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			}
			if state == "clone" {
				g = g.Clone()
				obj = g.StackPeek()
			}
			for range 2 {
				certain, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
				if certain != want || k != known {
					t.Fatalf("got %v/%v want %v/%v", certain, k, want, known)
				}
			}
			if g.FindPermanent(p.ID()) == nil || len(g.DelayedTriggers()) != 0 || len(g.resolution.playerBindings) != 0 {
				t.Fatal("inspection changed state")
			}
		})
	}
}

func TestSavedPlayerInstructionHistoryCollection(t *testing.T) {
	for _, state := range []string{"normal", "regain", "phase", "cycle", "return", "clone", "new turn", "attacked"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 2, 3)
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.SetTurn(2)
			subjects := WithoutPermanents(MatchingPermanents(DidntAttackThisTurn), MatchingPermanents(Not(ControlledSinceCurrentTurnStart)))
			predicate, ok := InspectContextualPermanentCollection(subjects)
			if !ok {
				t.Fatal("native history or exclusion unknown")
			}
			selected := true
			switch state {
			case "regain":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				selected = false
			case "phase":
				g.PhaseOut(p.ID())
				selected = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(card, a.PlayerID())
				selected = false
			case "clone":
				clone := g.Clone()
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g = clone
			case "new turn":
				g.SetTurn(3)
			case "attacked":
				p.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
				g.GetCombat().RemoveFromCombat(p.ID())
				selected = false
			}
			wasTapped := g.FindPermanentIncludingPhased(p.ID()).Tapped
			g.SetRandomResults([]int{7})
			for range 2 {
				got := predicate.Select(&EffectContext{Game: g, Controller: a.PlayerID()})
				if (len(got) == 1) != selected {
					t.Fatal("native selection differs from current history")
				}
				if len(got) == 1 && (got[0].ID != p.ID() || got[0].Incarnation != g.FindPermanent(p.ID()).incarnationID) {
					t.Fatal("wrong visit")
				}
			}
			if g.RandIntn(10) != 7 || g.FindPermanentIncludingPhased(p.ID()).Tapped != wasTapped {
				t.Fatal("query changed state")
			}
		})
	}
}

func TestSavedPlayerInstructionDefinition(t *testing.T) {
	saved := BindPlayer("saved")
	body := AtNext(EachEndStep, DestroyEach(PermanentsControlledBy(FromPlayer(saved), IsCreature)))
	valid := (SavedPlayerInstruction{Subject: ActivePlayer, Binding: saved, Body: body}).Effect()
	if err := ValidateCardDefinition("Valid", NewInstant("Valid", "", Spell(valid))); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Effect{
		(SavedPlayerInstruction{Subject: FromPlayer(saved), Binding: saved, Body: body}).Effect(),
		Sequence(body, valid),
		(SavedPlayerInstruction{Subject: You, Binding: BindPlayer("saved"), Body: body}).Effect(),
	} {
		if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", Spell(bad))); err == nil {
			t.Fatal("unbound saved player accepted")
		}
	}
}
