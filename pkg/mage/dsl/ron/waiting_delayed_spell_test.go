package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedSpellFrame(t *testing.T) {
	for _, change := range []string{"live", "resolved", "recast", "copy", "source-leaves", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetStep(core.PrecombatMain)
			g.SetTurn(2)
			watcher := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerCastsASpell(AnyPlayer, AnySpell), ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep, ActionList([CounterSpell(Trigger_ThatSpell)]))]))`)()
			watcher.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(watcher, a.PlayerID())
			card := repositoryCard(t, "LEB", "Lightning Bolt")
			card.SetOwner(b.PlayerID())
			b.AddToHand(card)
			if err := g.CastSpellByID(b.PlayerID(), card.ID(), []uuid.UUID{a.PlayerID()}, 0); err == nil || g.StackSize() != 0 || len(g.DelayedTriggers()) != 0 {
				t.Fatal("unpaid cast changed state")
			}
			b.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(b.PlayerID(), card.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			saved := g.CaptureSpellDamageSource(g.StackPeek().ID)
			if len(g.WaitingDelayedSpellSources()) != 0 {
				t.Fatal("delay before trigger resolution")
			}
			g.PutTriggersOnStack()
			g.ResolveTopOfStack()
			if got := g.WaitingDelayedSpellSources(); len(got) != 1 || !got[0].SameObject(saved) {
				t.Fatal("RON event frame")
			}
			if change == "source-leaves" {
				g.BouncePermanentToHand(p)
			}
			var recast uuid.UUID
			if change == "resolved" || change == "recast" {
				g.ResolveTopOfStack()
			}
			if change == "recast" {
				g.BouncePermanentToHand(p)
				b.RemoveFromGraveyard(card.ID())
				b.AddToHand(card)
				b.ManaPool().Add(core.Red, 1)
				if err := g.CastSpellByID(b.PlayerID(), card.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
				recast = g.StackPeek().ID
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				if len(game.WaitingDelayedSpellSources()) != 0 || len(game.StackBoundSpellSources()) != 1 {
					t.Fatal("waiting to stack transition")
				}
				if change == "copy" && game.CopyStackObjectDirect(game.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
				game.ResolveTopOfStack()
				if change == "copy" {
					game.ResolveTopOfStack()
				}
				if change == "recast" && game.GetStack().FindByID(recast) == nil {
					t.Fatal("recast countered")
				}
				game.ResolveStack()
				want := 20
				if change == "resolved" {
					want = 17
				}
				if change == "recast" {
					want = 14
				}
				if game.GetPlayer(a.PlayerID()).Life() != want || game.StackSize() != 0 {
					t.Fatal("native delayed counter/gameplay")
				}
			}
		})
	}
}

func TestRONWaitingDelayedSpellInvalid(t *testing.T) {
	for _, ref := range []string{`Trigger_ThatSpell()`, `Trigger_ThatPlayer`, `Ref_TargetSpell`, `ThisSpell`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenAPlayerCastsASpell(AnyPlayer,AnySpell),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CounterSpell(` + ref + `)]))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid delayed reference", ref)
		}
	}
}

func TestRONWaitingDelayedSpellDepartureFrame(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.PrecombatMain)
	watcher := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerCastsASpell(AnyPlayer, AnySpell), ActionList([CreateTriggerUntil(WhenAPermanentLeavesTheBattlefield(SinglePermanent(ThisPermanent)), ActionList([CounterSpell(Trigger_ThatSpell)]), UntilEndOfTurn)]))`)()
	watcher.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(watcher, a.PlayerID())
	card := repositoryCard(t, "LEA", "Lightning Bolt")
	card.SetOwner(b.PlayerID())
	b.AddToHand(card)
	b.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(b.PlayerID(), card.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.CaptureSpellDamageSource(g.StackPeek().ID)
	g.PutTriggersOnStack()
	g.ResolveTopOfStack()
	if got := g.WaitingDelayedSpellSources(); len(got) != 1 || !got[0].SameObject(original) {
		t.Fatal("departure frame missing")
	}
	g.BouncePermanentToHand(p)
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 20 || len(g.WaitingDelayedSpellSources()) != 1 {
		t.Fatal("persistent delayed counter")
	}
	g.RunStepWithPriority(core.Cleanup)
	if len(g.WaitingDelayedSpellSources()) != 0 {
		t.Fatal("expired frame")
	}
}
