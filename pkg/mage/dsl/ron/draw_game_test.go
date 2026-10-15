package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDrawGameDivineIntervention(t *testing.T) {
	factory, err := Compile(chosenDiscardRecord(t, "LEG", "Divine Intervention"))
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
	g.ResolveStack()
	if p.Counters[core.Intervention] != 2 || g.IsGameOver() {
		t.Fatal("opponent upkeep")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if p.Counters[core.Intervention] != 1 || g.IsGameOver() {
		t.Fatal("first upkeep")
	}
	queueRONUpkeep(g, a)
	g.ResolveTopOfStack()
	if g.StackSize() != 1 || g.IsGameOver() {
		t.Fatal("last removal must queue draw")
	}
	clone := g.Clone()
	g.DestroyPermanent(g.FindPermanent(p.ID()))
	g.ResolveStack()
	if !g.IsDraw() || g.Winner() != "" || !a.IsAlive() || !b.IsAlive() || clone.IsGameOver() {
		t.Fatal("draw result or pending clone")
	}
	clone.ResolveStack()
	if !clone.IsDraw() {
		t.Fatal("clone lost pending draw")
	}
}

func TestRONDrawGamePending(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "control", "copy", "countered"} {
		t.Run(state, func(t *testing.T) {
			factory, err := Compile(chosenDiscardRecord(t, "LEG", "Divine Intervention"))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if err := mage.ApplyEffect(g, mage.RemoveAllCounters(mage.This, core.Intervention), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("missing last-removal trigger")
			}
			switch state {
			case "departed", "returned":
				g.DestroyPermanent(p)
				if state == "returned" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			case "countered":
				g.CounterStackObject(g.StackPeek().ID)
			}
			g.ResolveStack()
			if g.IsDraw() != (state != "countered") || !a.IsAlive() || !b.IsAlive() {
				t.Fatal("pending draw identity or timing")
			}
			if state == "copy" && g.StackSize() != 1 {
				t.Fatal("original resolved after copied draw")
			}
		})
	}
}

func TestRONDrawGameStopsActions(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([GainLife(Integer(2)),DrawTheGame,GainLife(Integer(9))]))`)
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	g.SetStep(core.PrecombatMain)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.IsDraw() || a.Life() != 22 {
		t.Fatal("action list did not stop at draw")
	}
}

func TestRONDrawGameInvalid(t *testing.T) {
	for _, rule := range []string{
		`Activated(PayMana([]),ActionList([DrawTheGame()]))`,
		`Activated(PayMana([]),ActionList([DrawTheGame(You)]))`,
		`Activated(PayMana([]),ActionList([PlayerAction(You,DrawTheGame)]))`,
		`Activated(DrawTheGame,ActionList([GainLife(Integer(1))]))`,
		`AsPermanentEnters(ThisPermanent,[DrawTheGame])`,
		`ReplaceWouldGainLife(PlayerWouldGainLife(You),[DrawTheGame])`,
	} {
		assertConditionalRejected(t, rule)
	}
}

func TestRONDrawGameNested(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([ActionForEachPermanent(IsCardtype(Creature),[GainLife(Integer(1)),DrawTheGame,GainLife(Integer(9))]),GainLife(Integer(50))]))`)
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	for range 2 {
		g.PutOnBattlefield(mage.NewCreature("Member", "", 1, 1), a.PlayerID())
	}
	g.SetStep(core.PrecombatMain)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.IsDraw() || a.Life() != 21 {
		t.Fatal("nested body or next iteration continued")
	}
	for _, ctx := range []scope{{}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compileEffect(nameScopeNode(t, `DrawTheGame`), ctx); err == nil {
			t.Fatal("invalid context accepted")
		}
	}
}
