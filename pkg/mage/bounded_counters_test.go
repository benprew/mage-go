package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type boundedCounterPlayer struct {
	*BasePlayer
	choice int
	calls  int
	upper  int
}

func (p *boundedCounterPlayer) ChooseNumber(_, maximum int, _ string) int {
	p.calls++
	p.upper = maximum
	if p.choice == -1 {
		return maximum
	}
	return p.choice
}

func TestAddCountersUpTo(t *testing.T) {
	for _, tc := range []struct {
		name                                               string
		current, amount, limit, choice, want, upper, calls int
		double                                             bool
	}{
		{"partial", 1, 5, 7, 2, 3, 5, 1, false},
		{"decline", 1, 5, 7, 0, 1, 5, 1, false},
		{"room", 6, 5, 7, -1, 7, 1, 1, false},
		{"full", 7, 5, 7, -1, 7, 0, 0, false},
		{"overfull", 9, 5, 7, -1, 9, 0, 0, false},
		{"zero", 1, 0, 7, -1, 1, 0, 0, false},
		{"zero-limit", 0, 5, 0, -1, 0, 0, 0, false},
		{"storage-bound", 254, 999, 255, -1, 255, 1, 1, false},
		{"negative", 1, -2, 7, -1, 1, 0, 0, false},
		{"invalid-high", 1, 3, 7, 20, 1, 3, 1, false},
		{"invalid-low", 1, 3, 7, -4, 1, 3, 1, false},
		{"replacement-cap", 5, 2, 7, -1, 7, 2, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &boundedCounterPlayer{BasePlayer: NewBasePlayer("A"), choice: tc.choice}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewArtifact("Subject", ""), a.PlayerID())
			p.AddCounter(core.P1P0, tc.current)
			if tc.double {
				g.AddCounterDoubler(p.ID(), core.P1P0, AnyPermanent)
			}
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
			if err := AddCountersUpTo(This, core.P1P0, Fixed(tc.amount), tc.limit).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if got := g.FindPermanent(p.ID()).Counters[core.P1P0]; got != tc.want || a.upper != tc.upper || a.calls != tc.calls {
				t.Fatalf("counters %d want %d; choice bounds/calls %d/%d want %d/%d", got, tc.want, a.upper, a.calls, tc.upper, tc.calls)
			}
		})
	}
}

func TestAddCountersUpToIdentity(t *testing.T) {
	for _, change := range []string{"none", "control", "phase", "cycle", "leave", "return", "clone"} {
		t.Run(change, func(t *testing.T) {
			a := &boundedCounterPlayer{BasePlayer: NewBasePlayer("A"), choice: -1}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Subject", "")
			p := g.PutOnBattlefield(card, a.PlayerID())
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
			switch change {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "leave", "return":
				g.RemoveFromBattlefield(p)
				if change == "return" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			}
			if change == "clone" {
				ctx.Game = g.Clone()
			}
			if err := AddCountersUpTo(This, core.P1P0, Fixed(3), 7).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 3
			if change == "phase" || change == "leave" || change == "return" {
				want = 0
			}
			if live := ctx.Game.FindPermanentIncludingPhased(p.ID()); live != nil && live.Counters[core.P1P0] != want {
				t.Fatal("wrong incarnation or phase state")
			}
			if change == "clone" && g.FindPermanent(p.ID()).Counters[core.P1P0] != 0 {
				t.Fatal("clone shares counters")
			}
			if (want == 0) != (a.calls == 0) && change != "clone" {
				t.Fatal("choice for absent subject")
			}
		})
	}
}

func TestAddCountersUpToDeclarations(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		AddCountersUpTo(nil, core.P1P0, Fixed(1), 7),
		AddCountersUpTo(This, core.P1P0, nil, 7),
		AddCountersUpTo(This, core.NumCounters, Fixed(1), 7),
		AddCountersUpTo(This, core.P1P0, Fixed(1), -1),
		AddCountersUpTo(This, core.P1P0, Fixed(1), 256),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition("Invalid", card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestAddCountersUpToReplacementRecheck(t *testing.T) {
	for _, scenario := range []string{"add", "prevent", "return", "phase"} {
		t.Run(scenario, func(t *testing.T) {
			a := &boundedCounterPlayer{BasePlayer: NewBasePlayer("A"), choice: -1}
			g := NewGame(a, NewBasePlayer("B"))
			card := NewArtifact("Subject", "")
			p := g.PutOnBattlefield(card, a.PlayerID())
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
			g.AddReplacementEffect(&testPrioritizedReplacement{
				replacementBase: replacementBase{duration: core.Indefinite},
				matchFunc:       func(action Action) bool { _, ok := action.(*AddCountersAction); return ok },
				replaceFunc: func(action Action, game *Game) Action {
					switch scenario {
					case "add":
						game.MutablePermanent(p.ID()).AddCounter(core.P1P0, 6)
						return action.(*AddCountersAction).WithAmount(10)
					case "prevent":
						return nil
					case "return":
						game.RemoveFromBattlefield(game.FindPermanent(p.ID()))
						game.PutOnBattlefield(card, a.PlayerID())
					case "phase":
						game.PhaseOut(p.ID())
					}
					return action
				},
			})
			if err := AddCountersUpTo(This, core.P1P0, Fixed(3), 7).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 0
			if scenario == "add" {
				want = 7
			}
			if g.FindPermanentIncludingPhased(p.ID()).Counters[core.P1P0] != want {
				t.Fatal("replacement recheck")
			}
		})
	}
}
