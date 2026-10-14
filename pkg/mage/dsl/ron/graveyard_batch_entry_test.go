package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const graveyardBatchAction = `PutEachCardFromEachPlayersGraveyardOntoTheBattlefield(IsCardtype(Creature),AnyPlayer,[EntersNormally])`

func TestRONGraveyardBatchEntryInvalid(t *testing.T) {
	for _, action := range []string{
		`PutEachCardFromEachPlayersGraveyardOntoTheBattlefield`,
		`PutEachCardFromEachPlayersGraveyardOntoTheBattlefield()`,
		`PutEachCardFromEachPlayersGraveyardOntoTheBattlefield(IsCardtype(Creature),AnyPlayer)`,
		`PutEachCardFromEachPlayersGraveyardOntoTheBattlefield(IsCardtype(Artifact),AnyPlayer,[EntersNormally])`,
		`PutEachCardFromEachPlayersGraveyardOntoTheBattlefield(IsCardtype(Creature()),AnyPlayer,[EntersNormally])`,
		`PutEachCardFromEachPlayersGraveyardOntoTheBattlefield(IsCardtype(Creature),AnyPlayer(),[EntersNormally])`,
		`PutEachCardFromEachPlayersGraveyardOntoTheBattlefield(IsCardtype(Creature),You,[EntersNormally])`,
		`PutEachCardFromEachPlayersGraveyardOntoTheBattlefield(IsCardtype(Creature),AnyPlayer,[])`,
		`PutEachCardFromEachPlayersGraveyardOntoTheBattlefield(IsCardtype(Creature),AnyPlayer,[EntersNormally()])`,
		`PutEachCardFromEachPlayersGraveyardOntoTheBattlefield(IsCardtype(Creature),AnyPlayer,[EntersTapped])`,
		`PutEachCardFromEachPlayersGraveyardOntoTheBattlefield(IsCardtype(Creature),AnyPlayer,[EntersNormally,EntersNormally])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([` + action + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
	for _, rule := range []string{
		`Activated(` + graveyardBatchAction + `,ActionList([DrawACard]))`,
		`AsPermanentEnters([` + graveyardBatchAction + `])`,
		`PermanentLayerEffect(ThisPermanent,` + graveyardBatchAction + `)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted context %s", rule)
		}
	}
}

func TestRONGraveyardBatchEntryInvalidScope(t *testing.T) {
	nodes, err := Parse(strings.NewReader(graveyardBatchAction), "scope.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{}, {resolving: true, drawReplacementBody: true}, {resolving: true, enteringSource: true}} {
		if _, err := compileEffect(nodes[0], ctx); err == nil {
			t.Fatal("accepted nonresolving, replacement or entry context")
		}
	}
}

func TestRONGraveyardBatchEntryGameplay(t *testing.T) {
	for _, context := range []string{"spell", "spell-copy", "activation", "delayed"} {
		t.Run(context, func(t *testing.T) {
			rule, kind := `SpellActions(ActionList([`+graveyardBatchAction+`]))`, "Instant"
			if context == "activation" {
				rule, kind = `Activated(PayMana([]),ActionList([`+graveyardBatchAction+`]))`, "Artifact"
			}
			if context == "delayed" {
				rule = `SpellActions(ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + graveyardBatchAction + `]))]))`
			}
			registry := mage.NewRegistry()
			src := `Card(Name:"Return",Typeline:(Supertypes:[],Cardtypes:[` + kind + `],Subtypes:[]),Rules:[` + rule + `])`
			if err := LoadInto(registry, fstest.MapFS{"return.ron": {Data: []byte(src)}}, "*.ron"); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			creature := mage.NewCreature("Returned", "", 1, 1)
			creature.SetOwner(b.PlayerID())
			b.AddToGraveyard(creature)
			card, err := registry.CreateCard("Return")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			if context == "activation" {
				p := g.PutOnBattlefield(card, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ExilePermanent(p)
			} else {
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
			}
			if context == "spell-copy" {
				g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
			}
			if g.FindPermanent(creature.ID()) != nil {
				t.Fatal("action ran during announcement")
			}
			if context == "delayed" {
				g.ResolveStack()
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
			}
			clone := g.Clone()
			clone.ResolveStack()
			if clone.FindPermanent(creature.ID()) == nil || g.FindPermanent(creature.ID()) != nil {
				t.Fatal("pending clone shared execution")
			}
			g.ResolveStack()
			if p := g.FindPermanent(creature.ID()); p == nil || p.ControllerID() != b.PlayerID() || len(b.Graveyard()) != 0 {
				t.Fatal("RON owner-controlled entry failed")
			}
		})
	}
}

func TestRONGraveyardBatchEntryAllHallowsEve(t *testing.T) {
	node := repositoryRecord(t, "LEG", "All Hallow's Eve", "LEG.ron")

	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	registry := mage.NewRegistry()
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, `Card(Name: "All Hallow's Eve"`) {
			if err := LoadInto(registry, fstest.MapFS{"eve.ron": {Data: []byte(line)}}, "*.ron"); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := Compile(node); err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	first := mage.NewCreature("First", "", 2, 2)
	second := mage.NewCreature("Second", "", 2, 2, mage.WithAbility(mage.EntersWithNCounters(core.P1P1, 2)))
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	a.AddToGraveyard(first)
	b.AddToGraveyard(second)
	card, err := registry.CreateCard("All Hallow's Eve")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	queueRONUpkeep(g, b)
	g.ResolveStack()
	if g.FindExiledCard(card.ID()).Counters[core.Scream] != 2 {
		t.Fatal("opponent upkeep counted down")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if g.FindExiledCard(card.ID()).Counters[core.Scream] != 1 || g.FindPermanent(first.ID()) != nil {
		t.Fatal("first upkeep returned cards")
	}
	observed := 0
	watcher := mage.NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, reader mage.GameReader, _, _ uuid.UUID) bool {
		if e.FromZone == core.ZoneGraveyard && e.ToZone == core.ZoneBattlefield {
			observed++
			if reader.FindPermanent(first.ID()) == nil || reader.FindPermanent(second.ID()) == nil || len(reader.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
				t.Error("countdown or simultaneous observation failed")
			}
		}
		return false
	})
	g.PutOnBattlefield(mage.NewArtifact("Watcher", "", mage.WithAbility(watcher)), a.PlayerID())
	queueRONUpkeep(g, a)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.FindPermanent(first.ID()) == nil || g.FindPermanent(first.ID()) != nil {
		t.Fatal("pending upkeep clone failed")
	}
	g.ResolveStack()
	if observed != 4 || g.FindExiledCard(card.ID()) != nil || len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != card.ID() || g.FindPermanent(second.ID()).Counters[core.P1P1] != 2 {
		t.Fatal("full card resolution failed")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if g.StackSize() != 0 {
		t.Fatal("departed source triggered again")
	}
}
