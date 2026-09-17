package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableSourceUntapLifetime(t *testing.T) {
	for _, change := range []string{"unchanged", "absent", "returned", "phased", "phase cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Source", "", 1, 1, WithAbility(At(YourUpkeep, UntapPermanent(This))))
			g.PutOnBattlefield(source, a.PlayerID())
			g.TapPermanent(g.FindPermanent(source.ID()))
			g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.stack.Size() != 1 {
				t.Fatal("missing trigger")
			}
			switch change {
			case "absent", "returned":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "returned" {
					g.PutOnBattlefield(source, a.PlayerID())
					g.TapPermanent(g.FindPermanent(source.ID()))
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			clone := g.Clone()
			clone.ResolveStack()
			if change == "absent" {
				return
			}
			if change == "phased" {
				clone.PhaseIn([]uuid.UUID{source.ID()})
				g.PhaseIn([]uuid.UUID{source.ID()})
			}
			acted := change != "returned" && change != "phased"
			if clone.FindPermanent(source.ID()).Tapped == acted {
				t.Fatal("untap used the wrong source lifetime")
			}
			if !g.FindPermanent(source.ID()).Tapped {
				t.Fatal("clone untapped original source")
			}
		})
	}
}

func TestComposableSourceUntapStunAndEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1)
	target := NewCreature("Unrelated target", "", 1, 1)
	watcher := NewEnchantment("Watcher", "", WithAbility(NewTriggered(EvtBecameUntapped, false, GainLifeFor(You, Fixed(1)))))
	for _, card := range []Card{source, target, watcher} {
		g.PutOnBattlefield(card, a.PlayerID())
	}
	g.TapPermanent(g.FindPermanent(source.ID()))
	g.TapPermanent(g.FindPermanent(target.ID()))
	g.MutablePermanent(source.ID()).AddCounter(Stun, 1)
	effect := UntapPermanent(This)
	for i := range 3 {
		if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
		resolvePendingTriggers(g)
		wantLife := 21
		if i == 0 {
			wantLife = 20
		}
		if a.Life() != wantLife || g.FindPermanent(source.ID()).Tapped != (i == 0) || g.FindPermanent(source.ID()).Counters[Stun] != 0 {
			t.Fatalf("attempt %d: life %d, source %+v", i, a.Life(), g.FindPermanent(source.ID()))
		}
		if !g.FindPermanent(target.ID()).Tapped {
			t.Fatal("untapped positional target")
		}
	}
}

func TestComposableSourceUntapValidation(t *testing.T) {
	for _, effect := range []ResolvingEffect{UntapPermanent(This), UntapPermanent(nil)} {
		card := NewSorcery("Invalid untap", "", WithAbility(Spell(effect)))
		err := ValidateCardDefinition("Invalid untap", card)
		if err == nil || !strings.Contains(err.Error(), "abilities[0].effects[0].subject") {
			t.Fatalf("validation = %v", err)
		}
	}
	effect := UntapPermanent(This)
	if !strings.Contains(effect.Text(), "this") || effect.Properties().Outcome != OutcomeBenefit {
		t.Fatal("missing untap diagnostics or benefit metadata")
	}
}
