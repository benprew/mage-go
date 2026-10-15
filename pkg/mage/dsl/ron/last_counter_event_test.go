package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const lastCounterEvent = `WhenAPlayerRemovesTheLastCounterOfTypeFromAPermanent(SinglePlayer(You),InterventionCounter,SinglePermanent(ThisPermanent))`

func TestRONLastCounterEventDivineClause(t *testing.T) {
	n := chosenDiscardRecord(t, "LEG", "Divine Intervention")
	if _, err := Compile(n); err == nil || !strings.Contains(err.Error(), "unsupported action (DrawTheGame)") {
		t.Fatalf("remaining full-card blocker: %v", err)
	}
	projected := false
	var rewrite func(Node) Node
	rewrite = func(node Node) Node {
		if node.Kind == Symbol && node.Name == "DrawTheGame" {
			projected = true
			return nameScopeNode(t, `PlayerAction(Trigger_ThatPlayer,GainLife(Integer(3)))`)
		}
		for i, child := range node.Args {
			node.Args[i] = rewrite(child)
		}
		for key, child := range node.Fields {
			node.Fields[key] = rewrite(child)
		}
		return node
	}
	n = rewrite(n)
	if !projected {
		t.Fatal("missing draw clause")
	}
	factory, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	if p.Counters[core.Intervention] != 2 {
		t.Fatal("entry counters")
	}
	queueRONUpkeep(g, b)
	if g.StackSize() != 0 {
		t.Fatal("opponent upkeep")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if p.Counters[core.Intervention] != 1 || a.Life() != 20 {
		t.Fatal("first removal")
	}
	queueRONUpkeep(g, a)
	g.ResolveTopOfStack()
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || a.Life() != 20 {
		t.Fatal("last removal must queue an independent trigger")
	}
	g.DestroyPermanent(g.FindPermanent(p.ID()))
	clone := g.Clone()
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("departure or pending clone")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 {
		t.Fatal("clone lost actor")
	}
}

func TestRONLastCounterEventInstances(t *testing.T) {
	for _, scenario := range []string{"normal", "wrong player", "control before", "control after", "replenished", "phase", "phase cycle", "copy", "returned"} {
		t.Run(scenario, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `TriggerA(`+lastCounterEvent+`,ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(2)))]))`)
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(factory(), b.PlayerID())
			p.AddCounter(core.Intervention, 1)
			other.AddCounter(core.Intervention, 1)
			actor := a.PlayerID()
			want := 2
			switch scenario {
			case "wrong player":
				actor, want = b.PlayerID(), 0
			case "control before":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				want = 0
			case "phase", "phase cycle":
				g.PhaseOut(p.ID())
				want = 0
				if scenario == "phase cycle" {
					g.PhaseIn([]uuid.UUID{p.ID()})
					want = 2
				}
			}
			if err := mage.ApplyEffect(g, mage.RemoveAllCounters(mage.This, core.Intervention), p.ID(), actor, nil); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "control after":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "replenished":
				g.MutablePermanent(p.ID()).AddCounter(core.Intervention, 3)
			case "returned":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
			}
			g.PutTriggersOnStack()
			if scenario == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				want = 4
			}
			g.ResolveStack()
			if a.Life() != 20+want || b.Life() != 20 || g.FindPermanent(other.ID()).Counters[core.Intervention] != 1 {
				t.Fatal("instance, source control or saved player")
			}
		})
	}
}

func TestRONLastCounterEventInvalid(t *testing.T) {
	for _, event := range []string{
		`WhenAPlayerRemovesTheLastCounterOfTypeFromAPermanent`,
		`WhenAPlayerRemovesTheLastCounterOfTypeFromAPermanent()`,
		strings.Replace(lastCounterEvent, "SinglePlayer(You)", "AnyPlayer", 1),
		strings.Replace(lastCounterEvent, "SinglePlayer(You)", "SinglePlayer(Trigger_ThatPlayer)", 1),
		strings.Replace(lastCounterEvent, "You)", "You())", 1),
		strings.Replace(lastCounterEvent, "InterventionCounter", "InterventionCounter()", 1),
		strings.Replace(lastCounterEvent, "InterventionCounter", "UnknownCounter", 1),
		strings.Replace(lastCounterEvent, "SinglePermanent(ThisPermanent)", "AnyPermanent", 1),
		strings.Replace(lastCounterEvent, "ThisPermanent", "HostPermanent", 1),
		strings.Replace(lastCounterEvent, "ThisPermanent", "ThisPermanent()", 1),
	} {
		assertConditionalRejected(t, `TriggerA(`+event+`,ActionList([GainLife(Integer(1))]))`)
	}
	assertConditionalRejected(t, `Activated(PayMana([]),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`)
	assertConditionalRejected(t, `TriggerA(`+lastCounterEvent+`,ActionList([TapPermanent(Trigger_ThatPlayer)]))`)
}
