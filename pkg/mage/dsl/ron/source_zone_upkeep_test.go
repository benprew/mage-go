package ron

import (
	"fmt"
	"maps"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func sourceZoneFixture(zone, body string) string {
	return fmt.Sprintf(`Card(Name:"Zone watcher",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s(TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([%s])))])`, zone, body)
}

func TestRONSourceZoneUpkeepGameplay(t *testing.T) {
	for _, zone := range []string{"FromGraveyard", "FromExile"} {
		for _, returned := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/returned=%t", zone, returned), func(t *testing.T) {
				body := `PutGraveyardCardIntoHand(ThisGraveyardCard)`
				if zone == "FromExile" {
					body = `PutExiledCardIntoOwnersGraveyard(ThisExiledCard)`
				}
				body += `,PlayerAction(Trigger_ThatPlayer,GainLife(Integer(2))),GainLife(Integer(1))`
				registry, err := LoadFS(fstest.MapFS{"zone.ron": {Data: []byte(sourceZoneFixture(zone, body))}}, "*.ron")
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				card, err := registry.CreateCard("Zone watcher")
				if err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				if zone == "FromGraveyard" {
					a.AddToGraveyard(card)
				} else {
					g.ExileCard(card, card.ID())
				}
				queueRONUpkeep(g, b)
				if g.StackSize() != 0 {
					t.Fatal("opponent upkeep triggered")
				}
				queueRONUpkeep(g, a)
				if g.StackSize() != 1 || a.Life() != 20 {
					t.Fatal("source zone did not queue an independent trigger")
				}
				if returned {
					if zone == "FromGraveyard" {
						a.RemoveFromGraveyard(card.ID())
						a.AddToGraveyard(card)
					} else {
						g.RemoveFromExile(card.ID())
						g.ExileCard(card, card.ID())
					}
				}
				g.ResolveStack()
				if a.Life() != 23 || b.Life() != 20 {
					t.Fatal("event player or controller lost")
				}
				if zone == "FromGraveyard" && (len(a.Hand()) == 1) == returned {
					t.Fatal("graveyard source identity")
				}
				if zone == "FromExile" && (len(a.Graveyard()) == 1) == returned {
					t.Fatal("exile source identity")
				}
			})
		}
	}
}

func TestRONSourceZoneUpkeepInvalid(t *testing.T) {
	for _, rule := range []string{
		`FromGraveyard()`, `FromExile(TriggerA())`,
		`FromGraveyard(TriggerA(AtTheBeginningOfAPlayersUpkeep(Opponent),ActionList([GainLife(Integer(1))])))`,
		`FromExile(TriggerA(AtTheBeginningOfAPlayersEndStep(SinglePlayer(You)),ActionList([GainLife(Integer(1))])))`,
		`FromGraveyard(TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([TapPermanent(ThisPermanent)])))`,
		`FromExile(TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([PutGraveyardCardIntoHand(ThisGraveyardCard)])))`,
		`FromGraveyard(TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([PutExiledCardIntoOwnersGraveyard(ThisExiledCard)])))`,
		`FromGraveyard(TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([PutGraveyardCardIntoHand(ThisGraveyardCard())])))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([PutGraveyardCardIntoHand(ThisGraveyardCard)]))`,
		`FromGraveyard(FromExile(TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([GainLife(Integer(1))]))))`,
		`FromExile(TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([SpellDealsDamage(ThisSpell,Integer(1),Player(You))])))`,
		`FromGraveyard(TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([GainLife(ValueX)])))`,
		`FromGraveyard(TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,GainLife(Integer(1)))]))))`,
	} {
		t.Run(rule, func(t *testing.T) {
			src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s])`, rule)
			if _, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil {
				t.Fatal("invalid declaration accepted")
			}
		})
	}
	registry := mage.NewRegistry()
	registry.Register("Existing", func() mage.Card { return mage.NewArtifact("Existing", "") })
	bad := sourceZoneFixture("FromGraveyard", `PutGraveyardCardIntoHand(ThisExiledCard)`)
	if err := LoadInto(registry, fstest.MapFS{"bad.ron": {Data: []byte(bad)}}, "*.ron"); err == nil || !registry.CardRegistered("Existing") || registry.CardRegistered("Zone watcher") {
		t.Fatal("invalid collection changed registry")
	}
}

func TestRONSourceZoneUpkeepRecordPrerequisites(t *testing.T) {
	for _, tc := range []struct{ set, name, wrapper, body string }{
		{"LEA", "Nether Shadow", "FromGraveyard", `PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))`},
		{"LEB", "Nether Shadow", "FromGraveyard", `PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))`},
		{"2ED", "Nether Shadow", "FromGraveyard", `PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))`},
		{"3ED", "Nether Shadow", "FromGraveyard", `PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))`},
		{"CED", "Nether Shadow", "FromGraveyard", `PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))`},
		{"CEI", "Nether Shadow", "FromGraveyard", `PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))`},
		{"LEG", "All Hallow's Eve", "FromExile", ""},
	} {
		t.Run(tc.set+"/"+tc.name, func(t *testing.T) {
			node := repositoryRecord(t, tc.set, tc.name, tc.set+".ron")
			_, compileErr := Compile(node)
			if (compileErr == nil) != (tc.wrapper == "FromGraveyard") {
				t.Fatalf("complete record support changed: %v", compileErr)
			}
			var wrapper Node
			for _, rule := range node.Fields["Rules"].Args {
				if rule.Name == tc.wrapper {
					wrapper = rule
				}
			}
			if wrapper.Name == "" || wrapper.Args[0].Name != "TriggerI" {
				t.Fatal("record source-zone requirement changed")
			}
			trigger := wrapper.Args[0]
			var body Node
			if tc.body == "" {
				body = trigger.Args[2].Args[0].Args[1].Args[1].Args[0]
				if body.Name != "PutExiledCardIntoOwnersGraveyard" {
					t.Fatal("record exile move changed")
				}
			} else {
				nodes, err := Parse(strings.NewReader(tc.body), "probe.ron")
				if err != nil {
					t.Fatal(err)
				}
				body = nodes[0]
			}
			trigger.Name = "TriggerA"
			trigger.Args = []Node{trigger.Args[0], {Kind: Tuple, Name: "ActionList", Args: []Node{{Kind: List, Args: []Node{body}}}}}
			wrapper.Args = []Node{trigger}
			fields := maps.Clone(node.Fields)
			fields["Rules"] = Node{Kind: List, Args: []Node{wrapper}}
			if tc.wrapper == "FromExile" {
				spell, err := Parse(strings.NewReader(`SpellActions(ActionList([GainLife(Integer(0))]))`), "probe.ron")
				if err != nil {
					t.Fatal(err)
				}
				fields["Rules"] = Node{Kind: List, Args: []Node{wrapper, spell[0]}}
			}
			node.Fields = fields
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			if tc.wrapper == "FromGraveyard" {
				a.AddToGraveyard(card)
			} else {
				g.ExileCard(card, card.ID())
			}
			queueRONUpkeep(g, a)
			if g.StackSize() != 1 {
				t.Fatal("record upkeep did not trigger")
			}
			g.ResolveStack()
			if tc.wrapper == "FromGraveyard" && a.Life() != 21 {
				t.Fatal("record upkeep player missing")
			}
			if tc.wrapper == "FromExile" && (g.FindExiledCard(card.ID()) != nil || len(a.Graveyard()) != 1) {
				t.Fatal("record exile source action failed")
			}
		})
	}
}

func TestRONSourceZoneUpkeepDelayedFrames(t *testing.T) {
	for _, zone := range []string{"FromGraveyard", "FromExile"} {
		t.Run(zone, func(t *testing.T) {
			move := `PutGraveyardCardIntoHand(ThisGraveyardCard)`
			if zone == "FromExile" {
				move = `PutExiledCardIntoOwnersGraveyard(ThisExiledCard)`
			}
			body := `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + move + `,PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`
			registry, err := LoadFS(fstest.MapFS{"zone.ron": {Data: []byte(sourceZoneFixture(zone, body))}}, "*.ron")
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			for _, owner := range []mage.Player{a, b} {
				card, err := registry.CreateCard("Zone watcher")
				if err != nil {
					t.Fatal(err)
				}
				card.SetOwner(owner.PlayerID())
				if zone == "FromGraveyard" {
					owner.AddToGraveyard(card)
				} else {
					g.ExileCard(card, card.ID())
				}
				queueRONUpkeep(g, owner)
			}
			g.ResolveStack()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 2 {
				t.Fatal("delayed source frames shared")
			}
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 21 {
				t.Fatal("delayed event players shared")
			}
			for _, owner := range []mage.Player{a, b} {
				if zone == "FromGraveyard" && len(owner.Hand()) != 1 {
					t.Fatal("delayed graveyard source lost")
				}
				if zone == "FromExile" && len(owner.Graveyard()) != 1 {
					t.Fatal("delayed exile source lost")
				}
			}
		})
	}
}

func TestRONSourceZoneUpkeepInventory(t *testing.T) {
	for _, zone := range []string{"FromGraveyard", "FromExile"} {
		body, source := `PutGraveyardCardIntoHand(ThisGraveyardCard)`, "ThisGraveyardCard"
		if zone == "FromExile" {
			body, source = `PutExiledCardIntoOwnersGraveyard(ThisExiledCard)`, "ThisExiledCard"
		}
		report, err := InspectFS(fstest.MapFS{"zone.ron": {Data: []byte(sourceZoneFixture(zone, body))}}, "*.ron")
		if err != nil {
			t.Fatal(err)
		}
		for name, role := range map[string]string{"TriggerA": "rule", "AtTheBeginningOfAPlayersUpkeep": "event", source: "card_reference"} {
			found := false
			for _, signature := range report.Signatures {
				if signature.Name == name && signature.Role == role {
					found = true
					if signature.Support != "unverified" {
						t.Fatal("syntax inventory claimed gameplay coverage")
					}
				}
			}
			if !found {
				t.Fatalf("missing %s role %s", name, role)
			}
		}
	}
}
