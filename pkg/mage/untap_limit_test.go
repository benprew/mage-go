package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type untapChoicePlayer struct {
	*BasePlayer
	choices  []uuid.UUID
	observed bool
}

func (p *untapChoicePlayer) ChooseMayAbility(string) bool { return len(p.choices) > 0 }

func (p *untapChoicePlayer) ChoosePermanent(candidates []*Permanent, _ string, g GameReader) *Permanent {
	for _, candidate := range candidates {
		if !candidate.Tapped {
			p.observed = true
		}
	}
	if len(p.choices) == 0 {
		return nil
	}
	id := p.choices[0]
	p.choices = p.choices[1:]
	for _, candidate := range candidates {
		if candidate.ID() == id {
			return candidate
		}
	}
	return &Permanent{Card: NewArtifact("Invalid choice", "")}
}

func TestUntapLimitsChoiceAndOverlap(t *testing.T) {
	for _, scenario := range []string{"separate", "overlap", "decline", "invalid", "zero", "stun", "blocked"} {
		t.Run(scenario, func(t *testing.T) {
			a := &untapChoicePlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			limit := 1
			if scenario == "zero" {
				limit = 0
			}
			g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(Static(LimitUntapsDuringStep(IsCreature, limit), LimitUntapsDuringStep(IsArtifact, 1)))), b.PlayerID())
			creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), a.PlayerID())
			artifact := g.PutOnBattlefield(NewArtifact("Artifact", ""), a.PlayerID())
			both := g.PutOnBattlefield(NewCreature("Both", "", 2, 2, WithCardType(core.TypeArtifact)), a.PlayerID())
			for _, p := range []*Permanent{creature, artifact, both} {
				g.TapPermanent(p)
			}
			a.choices = []uuid.UUID{creature.ID(), artifact.ID()}
			switch scenario {
			case "overlap":
				a.choices = []uuid.UUID{both.ID()}
			case "decline":
				a.choices = nil
			case "invalid":
				a.choices = []uuid.UUID{uuid.New()}
			case "zero":
				a.choices = []uuid.UUID{artifact.ID()}
			case "stun":
				g.MutablePermanent(creature.ID()).AddCounter(core.Stun, 1)
			case "blocked":
				g.SkipNextUntap(creature.ID())
				a.choices = []uuid.UUID{both.ID()}
			}
			g.RunStepWithPriority(core.Untap)
			if a.observed {
				t.Fatal("choice observed an intermediate untap")
			}
			wantCreature := scenario == "separate" || scenario == "invalid"
			wantArtifact := scenario == "separate" || scenario == "zero" || scenario == "stun"
			wantBoth := scenario == "overlap" || scenario == "blocked"
			for _, check := range []struct {
				p    *Permanent
				want bool
			}{{creature, wantCreature}, {artifact, wantArtifact}, {both, wantBoth}} {
				if g.FindPermanent(check.p.ID()).Tapped == check.want {
					t.Fatalf("%s: tapped=%v", check.p.Name(), !check.want)
				}
			}
			if scenario == "stun" && g.FindPermanent(creature.ID()).Counters[core.Stun] != 0 {
				t.Fatal("stun replacement did not run")
			}
		})
	}
}

func TestUntapLimitInvalid(t *testing.T) {
	card := NewEnchantment("Invalid", "", WithAbility(Static(LimitUntapsDuringStep(IsCreature, -1))))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("accepted negative limit")
	}
}

func TestUntapLimitsSnapshotAndSimultaneous(t *testing.T) {
	a := &untapChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	rule := NewArtifact("Rule", "")
	filter := NewPermanentFilter("creatures while source is tapped", func(p *Permanent, g *Game) bool {
		return p.HasType(core.TypeCreature) && g.FindPermanent(rule.ID()).Tapped
	})
	rule.AddAbility(Static(LimitUntapsDuringStep(filter, 1)))
	source := g.PutOnBattlefield(rule, a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 2), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 2), a.PlayerID())
	for _, p := range []*Permanent{source, first, second} {
		g.TapPermanent(p)
	}
	a.choices = []uuid.UUID{first.ID()}
	events, intermediate := 0, 0
	observer := NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtBecameUntapped, false, GainLifeFor(You, Fixed(1))).SetCondition(func(event *core.GameEvent, game GameReader, _, _ uuid.UUID) bool {
		if event.SourceID == source.ID() || event.SourceID == first.ID() {
			events++
			if game.FindPermanent(source.ID()).Tapped || game.FindPermanent(first.ID()).Tapped {
				intermediate++
			}
		}
		return false
	})))
	g.PutOnBattlefield(observer, a.PlayerID())
	g.RunStepWithPriority(core.Untap)
	if events != 2 || intermediate != 0 || !g.FindPermanent(second.ID()).Tapped {
		t.Fatalf("events=%d intermediate=%d", events, intermediate)
	}
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(second.ID()).Tapped {
		t.Fatal("next decision retained old membership")
	}
}
