package ron

import (
	"maps"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const persistentUpkeep = `CreateTrigger(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([GainLife(Integer(2))]))`

func TestRONPersistentStepTomb(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := chosenDiscardRecord(t, set, "Cyclopean Tomb")
			record.Fields = maps.Clone(record.Fields)
			rules := record.Fields["Rules"]
			trigger := rules.Args[1]
			creation := trigger.Args[1].Args[0].Args[0]
			if creation.Name != "CreateTrigger" {
				t.Fatal("missing real subscription")
			}
			probe, err := Parse(strings.NewReader(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenAPermanentIsPutIntoAPlayersGraveyard(SinglePermanent(ThisPermanent),AnyPlayer),ActionList([`+persistentUpkeep+`]))])`), "probe")
			if err != nil {
				t.Fatal(err)
			}
			creation.Args = append([]Node(nil), creation.Args...)
			creation.Args[1] = probe[0].Fields["Rules"].Args[0].Args[1].Args[0].Args[0].Args[1]
			trigger.Args = append([]Node(nil), trigger.Args...)
			trigger.Args[1].Args = []Node{{Kind: List, Args: []Node{creation}}}
			rules.Args = []Node{trigger}
			record.Fields["Rules"] = rules
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.DestroyPermanent(p)
			if len(g.DelayedTriggers()) != 0 {
				t.Fatal("subscription installed before resolution")
			}
			g.ResolveStack()
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("no persistent subscription")
			}
			clone := g.Clone()
			for i := range 3 {
				g.RunStepWithPriority(core.Cleanup)
				g.SetActivePlayerIndex(1)
				g.RunStepWithPriority(core.Upkeep)
				if a.Life() != 20+i*2 || b.Life() != 20 {
					t.Fatal("wrong upkeep matched")
				}
				g.SetActivePlayerIndex(0)
				g.RunStepWithPriority(core.Upkeep)
				if a.Life() != 22+i*2 || len(g.DelayedTriggers()) != 1 {
					t.Fatal("persistent body did not repeat")
				}
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone shared state")
			}
			clone.RunStepWithPriority(core.Upkeep)
			if clone.GetPlayer(a.PlayerID()).Life() != 22 {
				t.Fatal("clone lost subscription")
			}
		})
	}
}

func TestRONPersistentStepInvalid(t *testing.T) {
	for _, action := range []string{
		`CreateTrigger`, `CreateTrigger()`,
		strings.Replace(persistentUpkeep, "SinglePlayer(You)", "AnyPlayer", 1),
		strings.Replace(persistentUpkeep, "You)", "You())", 1),
		strings.Replace(persistentUpkeep, "SinglePlayer(You)", "SinglePlayer(Opponent)", 1),
		strings.Replace(persistentUpkeep, "[GainLife(Integer(2))]", "[]", 1),
		strings.Replace(persistentUpkeep, "AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You))", "AtTheBeginningOfTheNextEndStep", 1),
		strings.Replace(persistentUpkeep, "GainLife(Integer(2))", "GainLife(Trigger_ThatMuch)", 1),
	} {
		records, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([`+action+`]))])`), "invalid")
		if err == nil {
			_, err = Compile(records[0])
		}
		if err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONPersistentStepCapturedTarget(t *testing.T) {
	for _, change := range []string{"phase", "return", "control", "copy", "counter", "pending-clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			action := strings.Replace(persistentUpkeep, "GainLife(Integer(2))", "TapPermanent(Ref_TargetPermanent)", 1)
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+action+`])))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			targetCard := mage.NewCreature("Subject", "", 2, 2)
			targetCard.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(targetCard, b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanentIncludingPhased(p.ID()).Tapped {
				t.Fatal("body ran at registration")
			}
			switch change {
			case "phase":
				g.PhaseOut(p.ID())
			case "return":
				g.BouncePermanentToHand(p)
				b.RemoveFromHand(targetCard.ID())
				p = g.PutOnBattlefield(targetCard, b.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: card.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || len(g.StackPeek().TargetSpecs) != 0 {
				t.Fatal("saved reference became a target")
			}
			if change == "counter" {
				g.GetStack().RemoveByID(g.StackPeek().ID)
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.FindPermanentIncludingPhased(p.ID()).Tapped {
				t.Fatal("pending clone changed original")
			}
			g.ResolveStack()
			want := change != "phase" && change != "return" && change != "counter"
			if g.FindPermanentIncludingPhased(p.ID()).Tapped != want {
				t.Fatal("wrong saved visit")
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			g.UntapPermanent(g.FindPermanent(p.ID()))
			g.RunStepWithPriority(core.Cleanup)
			g.SetActivePlayerIndex(0)
			g.RunStepWithPriority(core.Upkeep)
			if g.FindPermanentIncludingPhased(p.ID()).Tapped != (change != "return") || len(g.DelayedTriggers()) != 1 {
				t.Fatal("subscription lost its identity or stopped")
			}
		})
	}
}

func TestRONPersistentStepBoundaries(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([`+persistentUpkeep+`]))])`), "probe")
	if err != nil {
		t.Fatal(err)
	}
	action := nodes[0].Fields["Rules"].Args[0].Args[1].Args[0].Args[0]
	for _, ctx := range []scope{{}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compilePersistentStep(action, ctx); err == nil {
			t.Fatal("invalid context accepted")
		}
	}
	if got := inventoryArgumentRole(action, "action", 0); got != "persistent_step_event" {
		t.Fatal(got)
	}
	if got := inventoryArgumentRole(action.Args[0], "persistent_step_event", 0); got != "player_filter" {
		t.Fatal(got)
	}
	record := chosenDiscardRecord(t, "LEA", "Cyclopean Tomb")
	if _, err := Compile(record); err != nil {
		t.Fatalf("complete Tomb rejected: %v", err)
	}
}

func TestRONPersistentStepPayment(t *testing.T) {
	g, a, _ := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(2)]),ActionList([`+persistentUpkeep+`]))`)
	card := factory()
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("unpaid activation succeeded")
	}
	if len(g.DelayedTriggers()) != 0 || g.StackSize() != 0 {
		t.Fatal("failed payment installed a subscription")
	}
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if len(g.DelayedTriggers()) != 0 {
		t.Fatal("payment installed subscription before resolution")
	}
	g.ResolveStack()
	if len(g.DelayedTriggers()) != 1 {
		t.Fatal("paid ability did not install subscription")
	}
}
