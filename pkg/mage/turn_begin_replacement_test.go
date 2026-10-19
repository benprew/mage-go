package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type turnSkipPlayer struct {
	*BasePlayer
	answers []bool
	asks    int
	order   int
}

func (p *turnSkipPlayer) ChooseMayAbility(string) bool {
	p.asks++
	if len(p.answers) == 0 {
		return false
	}
	answer := p.answers[0]
	p.answers = p.answers[1:]
	return answer
}
func (p *turnSkipPlayer) ChooseMode(_ []string, _ string) int { return p.order }

func turnSkipFixture() (*Game, *turnSkipPlayer, *BasePlayer, *Permanent) {
	a := &turnSkipPlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Skip source", "", WithAbility(Static(MaySkipYourTurnToUntap(This, PermanentMatchesFilter(This, IsTapped))))), a.PlayerID())
	g.TapPermanent(source)
	return g, a, b, source
}

func TestTurnBeginReplacementTiming(t *testing.T) {
	g, a, b, source := turnSkipFixture()
	a.answers = []bool{true}
	if beginTurnAtMain(g) || a.asks != 1 || !g.FindPermanent(source.ID()).Tapped {
		t.Fatal("skip did not defer its continuation")
	}
	if beginTurnAtMain(g) || a.asks != 1 {
		t.Fatal("repeated start repeated the decision")
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.SkipNextTurnFor(b.PlayerID())
		game.AdvanceToNextTurn()
		if beginTurnAtMain(game) || !game.FindPermanent(source.ID()).Tapped {
			t.Fatal("skipped next turn ran continuation")
		}
		game.AdvanceToNextTurn()
		if !beginTurnAtMain(game) || game.FindPermanent(source.ID()).Tapped {
			t.Fatal("next actual turn did not run continuation")
		}
		game.TapPermanent(game.FindPermanent(source.ID()))
		if !beginTurnAtMain(game) || !game.FindPermanent(source.ID()).Tapped {
			t.Fatal("continuation repeated")
		}
	}
}

func TestTurnBeginReplacementLifecycle(t *testing.T) {
	for _, scenario := range []string{"decline", "untapped", "opponent", "phased", "removed", "control", "stun", "return-after", "phased-after", "removed-after", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b, source := turnSkipFixture()
			a.answers = []bool{true}
			switch scenario {
			case "decline":
				a.answers = []bool{false}
			case "untapped":
				g.UntapPermanent(source)
			case "opponent":
				g.SetActivePlayerIndex(1)
			case "phased":
				g.PhaseOut(source.ID())
			case "removed":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			starts := beginTurnAtMain(g)
			skipped := scenario != "decline" && scenario != "untapped" && scenario != "opponent" && scenario != "phased" && scenario != "removed" && scenario != "control"
			if starts == skipped {
				t.Fatalf("start = %v", starts)
			}
			if !skipped {
				return
			}
			switch scenario {
			case "stun":
				g.MutablePermanent(source.ID()).AddCounter(core.Stun, 1)
			case "return-after":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				source = g.PutOnBattlefield(source.Card, a.PlayerID())
				g.TapPermanent(source)
			case "phased-after":
				g.PhaseOut(source.ID())
			case "removed-after":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			clone := g.Clone()
			g.AdvanceToNextTurn()
			if !beginTurnAtMain(g) {
				t.Fatal("opponent turn was skipped")
			}
			p := g.FindPermanentIncludingPhased(source.ID())
			wantTapped := scenario == "stun" || scenario == "return-after" || scenario == "phased-after"
			if p.Tapped != wantTapped {
				t.Fatal("continuation lost visit or untap replacement")
			}
			if scenario == "stun" && p.Counters[core.Stun] != 0 {
				t.Fatal("stun not consumed")
			}
			if !clone.FindPermanentIncludingPhased(source.ID()).Tapped {
				t.Fatal("original changed clone")
			}
			clone.AdvanceToNextTurn()
			if !beginTurnAtMain(clone) {
				t.Fatal("clone turn skipped")
			}
			if clone.FindPermanentIncludingPhased(source.ID()).Tapped != wantTapped {
				t.Fatal("clone lost continuation")
			}
		})
	}
}

func TestTurnBeginReplacementMultipleAndSchedule(t *testing.T) {
	for _, scheduled := range []bool{false, true} {
		g, a, _, first := turnSkipFixture()
		second := g.PutOnBattlefield(first.Card.Copy(), a.PlayerID())
		g.TapPermanent(second)
		a.answers = []bool{false, true}
		if scheduled {
			g.SkipNextTurnFor(a.PlayerID())
		}
		if beginTurnAtMain(g) || a.asks != 2 {
			t.Fatal("replacement decisions were not independent")
		}
		if scheduled && g.turns.Schedule().SkipNextTurnFor[a.PlayerID()] != 1 {
			t.Fatal("optional skip consumed another skip")
		}
		g.AdvanceToNextTurn()
		if !beginTurnAtMain(g) {
			t.Fatal("next turn did not start")
		}
		if !g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped {
			t.Fatal("wrong source untapped")
		}
	}
}

func TestTurnBeginReplacementDefinition(t *testing.T) {
	for i, rule := range []StaticRule{
		MaySkipYourTurnToUntap(This, PermanentMatchesFilter(This, IsTapped)),
		MaySkipYourTurnToUntap(nil, PermanentMatchesFilter(This, IsTapped)),
		MaySkipYourTurnToUntap(This, nil),
		MaySkipYourTurnToUntap(DeclarePermanentTarget("undeclared", IsLand), PermanentMatchesFilter(This, IsTapped)),
	} {
		err := ValidateCardDefinition("Skip", NewArtifact("Skip", "", WithAbility(Static(rule))))
		if (err != nil) != (i > 0) {
			t.Fatalf("declaration %d: %v", i, err)
		}
	}
}

func TestTurnBeginReplacementNativeRun(t *testing.T) {
	g, a, b, source := turnSkipFixture()
	a.answers = []bool{true}
	for range 5 {
		a.AddToLibrary(NewLand("A land"))
		b.AddToLibrary(NewLand("B land"))
	}
	g.SetOnPriority(func(game *Game, _ int, _ bool) PriorityAction {
		if game.CurrentTurn() == 2 && game.FindPermanent(source.ID()).Tapped {
			t.Fatal("continuation did not precede priority")
		}
		return PriorityAction{Type: PriorityPass}
	})
	g.Run(0, core.Untap, 1)
	if g.CurrentTurn() != 2 || len(a.Hand()) != 0 || a.asks != 1 || !g.FindPermanent(source.ID()).Tapped {
		t.Fatal("native driver did not skip entire turn")
	}
	g.Run(0, core.Untap, 2)
	if g.CurrentTurn() != 3 || len(b.Hand()) != 1 || g.FindPermanent(source.ID()).Tapped {
		t.Fatal("native driver did not resume with continuation")
	}
}

func TestTurnBeginReplacementScheduledFirst(t *testing.T) {
	g, a, _, source := turnSkipFixture()
	a.order = 1
	a.answers = []bool{true}
	g.SkipNextTurnFor(a.PlayerID())
	if beginTurnAtMain(g) || a.asks != 0 || !g.FindPermanent(source.ID()).Tapped || g.turns.Schedule().SkipNextTurnFor[a.PlayerID()] != 0 {
		t.Fatal("affected player did not choose scheduled replacement first")
	}
	g.AdvanceToNextTurn()
	if !beginTurnAtMain(g) || !g.FindPermanent(source.ID()).Tapped {
		t.Fatal("unused optional replacement queued an untap")
	}
}

func TestTurnBeginReplacementControlAndPhasing(t *testing.T) {
	g, a, _, source := turnSkipFixture()
	g.PhaseOut(source.ID())
	g.PhaseIn([]uuid.UUID{source.ID()})
	a.answers = []bool{true}
	if beginTurnAtMain(g) {
		t.Fatal("phase cycle removed replacement")
	}
	g.PhaseOut(source.ID())
	g.PhaseIn([]uuid.UUID{source.ID()})
	g.AdvanceToNextTurn()
	if !beginTurnAtMain(g) || g.FindPermanent(source.ID()).Tapped {
		t.Fatal("phase cycle lost pending visit")
	}

	b := &turnSkipPlayer{BasePlayer: NewBasePlayer("B"), answers: []bool{true}}
	g = NewGame(a, b)
	source = g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Static(MaySkipYourTurnToUntap(This, PermanentMatchesFilter(This, IsTapped))))), a.PlayerID())
	g.TapPermanent(source)
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.AdvanceToNextTurn()
	if beginTurnAtMain(g) || b.asks != 1 {
		t.Fatal("current controller did not receive decision")
	}
}

func TestTurnBeginReplacementCannotSkipStartedTurn(t *testing.T) {
	g, a, _, source := turnSkipFixture()
	a.answers = []bool{true}
	g.RunStepWithPriority(core.PrecombatMain)
	if !beginTurnAtMain(g) || a.asks != 0 || !g.FindPermanent(source.ID()).Tapped {
		t.Fatal("a started turn was offered a skip")
	}
}

func TestTurnBeginReplacementAttachedSubject(t *testing.T) {
	a := &turnSkipPlayer{BasePlayer: NewBasePlayer("A"), answers: []bool{true}}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 1, 4), b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Skip aura", "", WithAbility(Static(MaySkipYourTurnToUntap(AttachedPermanent, PermanentMatchesFilter(This, IsTapped))))), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	if g.FindPermanent(aura.ID()).AttachedTo != host.ID() {
		t.Fatal("attachment failed")
	}
	g.TapPermanent(aura)
	g.TapPermanent(host)
	if beginTurnAtMain(g) {
		t.Fatal("attached subject was not available")
	}
	g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
	g.AdvanceToNextTurn()
	if !beginTurnAtMain(g) || g.FindPermanent(host.ID()).Tapped {
		t.Fatal("source departure cancelled saved host untap")
	}
}

func beginTurnAtMain(g *Game) bool {
	if !g.TryBeginTurn() {
		return false
	}
	g.RunStepWithPriority(core.PrecombatMain)
	return true
}

func TestTurnBeginReplacementFirstActualStep(t *testing.T) {
	g, a, b, source := turnSkipFixture()
	a.answers = []bool{true}
	if g.TryBeginTurn() {
		t.Fatal("turn was not skipped")
	}
	g.PutOnBattlefield(NewArtifact("Skip untap", "", WithAbility(Static(SkipUntapSteps()))), b.PlayerID())
	observer := NewTriggered(core.EvtBecameUntapped, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if reader.(*Game).GetStep() != core.Upkeep {
			t.Fatal("untap event did not use first actual step")
		}
		return true
	})
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer)), b.PlayerID())
	g.SetStep(core.Cleanup)
	g.AdvanceToNextTurn()
	if !g.TryBeginTurn() {
		t.Fatal("next turn was skipped")
	}
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(source.ID()).Tapped || b.Life() != 20 {
		t.Fatal("skipped step performed continuation")
	}
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(source.ID()).Tapped || b.Life() != 21 {
		t.Fatal("first actual step did not perform continuation before priority")
	}
}
