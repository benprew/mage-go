package ron

import (
	"maps"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage/core"
)

const exileCounterRemovalInstruction = `RemoveCounters([ACounterOfTypeFromCardInExile(ScreamCounter,ThisExiledCard)])`

func TestRONExileCounterRemovalGameplay(t *testing.T) {
	body := exileCounterRemovalInstruction + `,If(ExiledCardPassesFilter(ThisExiledCard,HasNoCountersOfType(ScreamCounter)),[GainLife(Integer(2))]),GainLife(Integer(1))`
	registry, err := LoadFS(fstest.MapFS{"source.ron": {Data: []byte(sourceZoneFixture("FromExile", body))}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"zero", "one", "wide", "changed", "depart", "return", "hidden", "hidden-after", "copy", "clone", "two-instances"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card, err := registry.CreateCard("Zone watcher")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			g.ExileCard(card, card.ID())
			count := 1
			if scenario == "zero" {
				count = 0
			}
			if scenario == "wide" || scenario == "copy" {
				count = 300
				if scenario == "copy" {
					count = 2
				}
			}
			g.FindExiledCard(card.ID()).Counters[core.Scream] = count
			g.FindExiledCard(card.ID()).Counters[core.Charge] = 2
			if scenario == "hidden" {
				g.FindExiledCard(card.ID()).FaceDown = true
				g.RevealExiledCardTo(card.ID(), a.PlayerID())
			}
			secondID := card.ID()
			if scenario == "two-instances" {
				second, err := registry.CreateCard("Zone watcher")
				if err != nil {
					t.Fatal(err)
				}
				second.SetOwner(a.PlayerID())
				g.ExileCard(second, second.ID())
				g.FindExiledCard(second.ID()).Counters[core.Scream] = 3
				secondID = second.ID()
			}
			queueRONUpkeep(g, b)
			if g.StackSize() != 0 {
				t.Fatal("wrong upkeep player")
			}
			queueRONUpkeep(g, a)
			clone := g.Clone()
			if scenario == "hidden-after" {
				g.FindExiledCard(card.ID()).FaceDown = true
				g.RevealExiledCardTo(card.ID(), a.PlayerID())
			}
			if scenario == "changed" {
				g.FindExiledCard(card.ID()).Counters[core.Scream] = 2
				count = 2
			}
			if scenario == "depart" || scenario == "return" {
				g.RemoveFromExile(card.ID())
				if scenario == "return" {
					g.ExileCard(card, card.ID())
					g.FindExiledCard(card.ID()).Counters[core.Scream] = 1
				}
			}
			if scenario == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy missing")
			}
			g.ResolveStack()
			wantLife := 21
			wantOpponentLife := 20
			wantCount := max(count-1, 0)
			switch scenario {
			case "zero", "one", "clone":
				wantLife = 23
			case "hidden":
				wantLife, wantCount = 20, 1
			case "return", "hidden-after":
				wantCount = 1
			case "two-instances":
				wantLife = 24
			case "copy":
				wantLife, wantOpponentLife, wantCount = 23, 21, 0
			}
			if a.Life() != wantLife || b.Life() != wantOpponentLife {
				t.Fatalf("life = %d/%d, want %d/%d", a.Life(), b.Life(), wantLife, wantOpponentLife)
			}
			if object := g.FindExiledCard(card.ID()); object != nil {
				if object.Counters[core.Scream] != wantCount {
					t.Fatalf("counters = %d, want %d", object.Counters[core.Scream], wantCount)
				}
				if scenario != "return" && object.Counters[core.Charge] != 2 {
					t.Fatal("removal changed another counter type")
				}
			}
			if scenario == "two-instances" && g.FindExiledCard(secondID).Counters[core.Scream] != 2 {
				t.Fatal("factory instances share source bindings")
			}
			if scenario == "clone" {
				if clone.FindExiledCard(card.ID()).Counters[core.Scream] != 1 {
					t.Fatal("resolution changed clone counters")
				}
				clone.ResolveStack()
				if clone.FindExiledCard(card.ID()).Counters[core.Scream] != 0 || clone.GetPlayer(a.PlayerID()).Life() != 23 {
					t.Fatal("clone lost its saved removal body")
				}
			}
		})
	}
}

func TestRONExileCounterRemovalInventory(t *testing.T) {
	report, err := InspectFS(fstest.MapFS{"source.ron": {Data: []byte(sourceZoneFixture("FromExile", exileCounterRemovalInstruction))}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"ACounterOfTypeFromCardInExile": "counter_removal_instruction", "ScreamCounter": "card_counter_type", "ThisExiledCard": "card_reference"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
				if signature.Support != "unverified" {
					t.Fatal("inventory claimed gameplay coverage")
				}
			}
		}
		if !found {
			t.Fatalf("missing %s in %s", name, role)
		}
	}
}

func TestRONExileCounterRemovalNested(t *testing.T) {
	body := `If(ExiledCardPassesFilter(ThisExiledCard,HasACounterOfType(ScreamCounter)),[` + exileCounterRemovalInstruction + `]),If(ExiledCardPassesFilter(ThisExiledCard,HasNoCountersOfType(ScreamCounter)),[PutExiledCardIntoOwnersGraveyard(ThisExiledCard)]),GainLife(Integer(1))`
	registry, err := LoadFS(fstest.MapFS{"source.ron": {Data: []byte(sourceZoneFixture("FromExile", body))}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	card, err := registry.CreateCard("Zone watcher")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	g.ExileCard(card, card.ID())
	g.FindExiledCard(card.ID()).Counters[core.Scream] = 1
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if g.FindExiledCard(card.ID()) != nil || len(a.Graveyard()) != 1 || a.Life() != 21 {
		t.Fatal("nested removal lost its source or stopped later instructions")
	}
}

func TestRONExileCounterRemovalAllHallowsEve(t *testing.T) {
	node := repositoryRecord(t, "LEG", "All Hallow's Eve", "LEG.ron")
	if _, err := Compile(node); err != nil {
		t.Fatalf("full record failed: %v", err)
	}
	probes, err := Parse(strings.NewReader(`SpellActions(ActionList([GainLife(Integer(0))])) GainLife(Integer(2))`), "probe.ron")
	if err != nil {
		t.Fatal(err)
	}
	wrapper := node.Fields["Rules"].Args[1]
	trigger := wrapper.Args[0]
	body := trigger.Args[2]
	branch := body.Args[0].Args[1]
	branch.Args = append([]Node(nil), branch.Args...)
	branch.Args[1].Args = []Node{branch.Args[1].Args[0], probes[1]}
	body.Args = []Node{{Kind: List, Args: []Node{body.Args[0].Args[0], branch}}}
	trigger.Args = []Node{trigger.Args[0], trigger.Args[1], body}
	wrapper.Args = []Node{trigger}
	node.Fields = maps.Clone(node.Fields)
	node.Fields["Rules"] = Node{Kind: List, Args: []Node{probes[0], wrapper}}
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	g.ExileCard(card, card.ID())
	g.FindExiledCard(card.ID()).Counters[core.Scream] = 2
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if object := g.FindExiledCard(card.ID()); object == nil || object.Counters[core.Scream] != 1 || a.Life() != 20 {
		t.Fatal("first upkeep did not remove exactly one counter")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if g.FindExiledCard(card.ID()) != nil || len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != card.ID() || a.Life() != 22 {
		t.Fatal("second upkeep did not run the zero branch after removal")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("departed exile source armed another trigger")
	}
}

func TestRONExileCounterRemovalInvalid(t *testing.T) {
	for _, body := range []string{
		`RemoveCounters([])`,
		`RemoveCounters([ACounterOfTypeFromCardInExile(ScreamCounter,ThisExiledCard),ACounterOfTypeFromCardInExile(ScreamCounter,ThisExiledCard)])`,
		`RemoveCounters([ACounterOfTypeFromCardInExile(ScreamCounter)])`,
		`RemoveCounters([ACounterOfTypeFromCardInExile(UnknownCounter,ThisExiledCard)])`,
		`RemoveCounters([ACounterOfTypeFromCardInExile(PTCounter(+2,+2),ThisExiledCard)])`,
		`RemoveCounters([ACounterOfTypeFromCardInExile(ScreamCounter,ThisPermanent)])`,
		`RemoveCounters([ACounterOfTypeFromCardInExile(ScreamCounter,ThisGraveyardCard)])`,
		`RemoveCounters([ACounterOfTypeFromCardInExile(ScreamCounter,Ref_TargetCard)])`,
	} {
		if _, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(sourceZoneFixture("FromExile", body))}}, "*.ron"); err == nil {
			t.Fatalf("invalid declaration accepted: %s", body)
		}
	}
	for _, source := range []string{
		sourceZoneFixture("FromGraveyard", exileCounterRemovalInstruction),
		`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([` + exileCounterRemovalInstruction + `]))])`,
		`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(` + exileCounterRemovalInstruction + `,ActionList([GainLife(Integer(1))]))])`,
	} {
		if _, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(source)}}, "*.ron"); err == nil {
			t.Fatal("invalid removal context accepted")
		}
	}
	registry, err := LoadFS(fstest.MapFS{"good.ron": {Data: []byte(sourceZoneFixture("FromExile", exileCounterRemovalInstruction))}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if err := LoadInto(registry, fstest.MapFS{"bad.ron": {Data: []byte(sourceZoneFixture("FromGraveyard", exileCounterRemovalInstruction))}}, "*.ron"); err == nil {
		t.Fatal("invalid collection accepted")
	}
	if _, err := registry.CreateCard("Zone watcher"); err != nil {
		t.Fatal("invalid collection changed the registry")
	}
}
