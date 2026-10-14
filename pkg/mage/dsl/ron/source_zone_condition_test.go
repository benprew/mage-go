package ron

import (
	"fmt"
	"maps"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
)

const orderedSourceCondition = `CardIsInPlayersGraveyardWithANumberOfCardsAboveIt(ThisGraveyardCard,You,GreaterThanOrEqualTo(Integer(3)),IsCardtype(Creature))`

func zoneConditionFixture(zone, condition string) string {
	return fmt.Sprintf(`Card(Name:"Condition watcher",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s(TriggerI(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),%s,ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))])))])`, zone, condition)
}

func TestRONSourceZoneConditionGameplay(t *testing.T) {
	for _, scenario := range []string{"below-only", "present", "remove-above", "return", "two-instances"} {
		t.Run(scenario, func(t *testing.T) {
			registry, err := LoadFS(fstest.MapFS{"condition.ron": {Data: []byte(zoneConditionFixture("FromGraveyard", orderedSourceCondition))}}, "*.ron")
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			addCreature := func() mage.Card {
				c := mage.NewCreature("Filler", "", 1, 1)
				c.SetOwner(a.PlayerID())
				a.AddToGraveyard(c)
				return c
			}
			for range 3 {
				addCreature()
			}
			card, err := registry.CreateCard("Condition watcher")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToGraveyard(card)
			var above mage.Card
			if scenario != "below-only" {
				for range 3 {
					above = addCreature()
				}
			}
			if scenario == "two-instances" {
				second, err := registry.CreateCard("Condition watcher")
				if err != nil {
					t.Fatal(err)
				}
				second.SetOwner(a.PlayerID())
				a.AddToGraveyard(second)
			}
			queueRONUpkeep(g, b)
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered")
			}
			queueRONUpkeep(g, a)
			wantStack := 1
			if scenario == "below-only" {
				wantStack = 0
			}
			if g.StackSize() != wantStack {
				t.Fatalf("stack = %d", g.StackSize())
			}
			if scenario == "remove-above" {
				a.RemoveFromGraveyard(above.ID())
			}
			if scenario == "return" {
				a.RemoveFromGraveyard(card.ID())
				a.AddToGraveyard(card)
				for range 3 {
					addCreature()
				}
			}
			g.ResolveStack()
			want := 20
			if scenario == "present" || scenario == "two-instances" {
				want = 21
			}
			if a.Life() != want {
				t.Fatalf("life = %d", a.Life())
			}
		})
	}
	registry, err := LoadFS(fstest.MapFS{"condition.ron": {Data: []byte(zoneConditionFixture("FromExile", `ExiledCardPassesFilter(ThisExiledCard,IsCardtype(Artifact))`))}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	card, err := registry.CreateCard("Condition watcher")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	g.ExileCard(card, card.ID())
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("exile condition failed")
	}
}

func TestRONSourceZoneConditionRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			node := repositoryRecord(t, set, "Nether Shadow", set+".ron")
			if _, err := Compile(node); err != nil {
				t.Fatal(err)
			}
			wrapper := node.Fields["Rules"].Args[1]
			trigger := wrapper.Args[0]
			probe, err := Parse(strings.NewReader(`ActionList([GainLife(Integer(1))])`), "probe.ron")
			if err != nil {
				t.Fatal(err)
			}
			trigger.Args = append([]Node(nil), trigger.Args...)
			trigger.Args[2] = probe[0]
			wrapper.Args = []Node{trigger}
			node.Fields = maps.Clone(node.Fields)
			node.Fields["Rules"] = Node{Kind: List, Args: []Node{node.Fields["Rules"].Args[0], wrapper}}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToGraveyard(card)
			for range 3 {
				c := mage.NewCreature("Above", "", 1, 1)
				c.SetOwner(a.PlayerID())
				a.AddToGraveyard(c)
			}
			queueRONUpkeep(g, a)
			g.ResolveStack()
			if a.Life() != 21 {
				t.Fatal("unchanged record condition failed")
			}
		})
	}
}

func TestRONSourceZoneConditionInvalid(t *testing.T) {
	for _, condition := range []string{
		`CardIsInPlayersGraveyardWithANumberOfCardsAboveIt()`,
		strings.Replace(orderedSourceCondition, "ThisGraveyardCard", "ThisExiledCard", 1),
		strings.Replace(orderedSourceCondition, "ThisGraveyardCard", "ThisGraveyardCard()", 1),
		strings.Replace(orderedSourceCondition, "You", "Opponent", 1),
		strings.Replace(orderedSourceCondition, "Integer(3)", "Integer(-1)", 1),
		strings.Replace(orderedSourceCondition, "Integer(3)", "ValueX", 1),
		strings.Replace(orderedSourceCondition, "GreaterThanOrEqualTo", "LessThan", 1),
		strings.Replace(orderedSourceCondition, "IsCardtype(Creature)", "IsTapped", 1),
		`ExiledCardPassesFilter(ThisExiledCard,HasACounterOfType(ScreamCounter))`,
	} {
		if _, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(zoneConditionFixture("FromGraveyard", condition))}}, "*.ron"); err == nil {
			t.Fatalf("invalid condition accepted: %s", condition)
		}
	}
	for _, src := range []string{
		zoneConditionFixture("FromExile", orderedSourceCondition),
		zoneConditionFixture("FromExile", `ExiledCardPassesFilter(ThisExiledCard,HasACounterOfType(ScreamCounter))`),
		strings.Replace(zoneConditionFixture("FromGraveyard", orderedSourceCondition), "SinglePlayer(You)", "Opponent", 1),
		strings.Replace(zoneConditionFixture("FromGraveyard", orderedSourceCondition), "ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))])", "Targeted([TargetPlayer(AnyPlayer)],ActionList([GainLife(Integer(1))]))", 1),
		strings.Replace(zoneConditionFixture("FromGraveyard", orderedSourceCondition), "GainLife(Integer(1))", "TapPermanent(ThisPermanent)", 1),
	} {
		if _, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil {
			t.Fatalf("invalid context accepted: %s", src)
		}
	}
	registry := mage.NewRegistry()
	registry.Register("Existing", func() mage.Card { return mage.NewArtifact("Existing", "") })
	if err := LoadInto(registry, fstest.MapFS{
		"good.ron": {Data: []byte(zoneConditionFixture("FromGraveyard", orderedSourceCondition))},
		"bad.ron":  {Data: []byte(zoneConditionFixture("FromExile", orderedSourceCondition))},
	}, "*.ron"); err == nil || !registry.CardRegistered("Existing") || registry.CardRegistered("Condition watcher") {
		t.Fatal("rejected collection changed registry")
	}
}

func TestRONSourceZoneConditionInventory(t *testing.T) {
	report, err := InspectFS(fstest.MapFS{"condition.ron": {Data: []byte(zoneConditionFixture("FromGraveyard", orderedSourceCondition))}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"TriggerI": "rule", "CardIsInPlayersGraveyardWithANumberOfCardsAboveIt": "trigger_condition", "ThisGraveyardCard": "card_reference", "GreaterThanOrEqualTo": "value_comparison", "IsCardtype": "card_filter"} {
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
			t.Fatalf("missing %s role %s", name, role)
		}
	}
}
