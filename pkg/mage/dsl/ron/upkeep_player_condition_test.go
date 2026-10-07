package ron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONUpkeepPlayerConditionsRecords(t *testing.T) {
	for _, name := range []string{"Spiritual Sanctuary", "Land Tax", "Ghazbán Ogre"} {
		set := "LEG"
		if name == "Ghazbán Ogre" {
			set = "ARN"
		}
		n := hostUntapRecord(t, set, name)
		if name == "Spiritual Sanctuary" {
			if _, err := Compile(n); err != nil {
				t.Fatal(err)
			}
			continue
		}
		rule := n.Fields["Rules"].Args[0]
		nodes, err := Parse(strings.NewReader(`ActionList([GainLife(Integer(1))])`), "probe.ron")
		if err != nil {
			t.Fatal(err)
		}
		rule.Args[2] = nodes[0]
		if _, err := compileRule(rule, scope{permanent: true}); err != nil {
			t.Fatal(name, err)
		}
	}
}

func TestRONSpiritualSanctuary(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Spiritual Sanctuary")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid casting accepted")
	}
	if g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("casting rollback")
	}
	a.ManaPool().Add(core.White, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	plains := repositoryCard(t, "LEB", "Plains")
	plains.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(plains, b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || b.Life() != 20 {
		t.Fatal("detection or early gain", g.StackSize())
	}
	clone := g.Clone()
	g.PhaseOut(p.ID())
	g.ResolveStack()
	clone.ResolveStack()
	if b.Life() != 20 || a.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 21 {
		t.Fatal("intervening recheck or clone")
	}
}

func TestRONSpiritualSanctuaryLifecycle(t *testing.T) {
	for _, change := range []string{"none", "source-left", "source-control", "source-phased", "source-returned", "abilities", "land-control", "land-left", "land-phase-cycle", "land-returned", "copy", "two"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Spiritual Sanctuary")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			putPlains := func(player mage.Player) *mage.Permanent {
				c := repositoryCard(t, "LEB", "Plains")
				c.SetOwner(a.PlayerID())
				return g.PutOnBattlefield(c, player.PlayerID())
			}
			own := putPlains(a)
			other := putPlains(b)
			if change == "two" {
				g.PutOnBattlefield(repositoryCard(t, "LEG", "Spiritual Sanctuary"), b.PlayerID())
			}
			queueRONUpkeep(g, a)
			queueRONUpkeep(g, b)
			wantStack := 2
			if change == "two" {
				wantStack = 4
			}
			if g.StackSize() != wantStack {
				t.Fatal("distinct pending players", g.StackSize())
			}
			switch change {
			case "source-left", "source-returned":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if change == "source-returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-phased":
				g.PhaseOut(source.ID())
			case "abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			case "land-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: other.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "land-left":
				g.BouncePermanentToHand(g.FindPermanent(other.ID()))
			case "land-phase-cycle":
				g.PhaseIn(g.PhaseOut(other.ID()))
			case "land-returned":
				c := other.Card
				g.BouncePermanentToHand(g.FindPermanent(other.ID()))
				a.RemoveFromHand(c.ID())
				g.PutOnBattlefield(c, b.PlayerID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.SetActivePlayerIndex(1)
			clone := g.Clone()
			clone.PhaseOut(own.ID())
			g.ResolveStack()
			clone.ResolveStack()
			wantA, wantB := 21, 21
			if change == "land-control" || change == "land-left" {
				wantB = 20
			}
			if change == "copy" {
				wantB = 22
			}
			if change == "two" {
				wantA, wantB = 22, 22
			}
			wantCloneA := 20
			if change == "land-control" {
				wantCloneA = 21
			}
			if a.Life() != wantA || b.Life() != wantB || clone.GetPlayer(a.PlayerID()).Life() != wantCloneA || clone.GetPlayer(b.PlayerID()).Life() != wantB {
				t.Fatalf("binding lifetime: live %d/%d clone %d/%d", a.Life(), b.Life(), clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life())
			}
		})
	}
}

func TestRONUpkeepPlayerConditionsRechecks(t *testing.T) {
	for _, kind := range []string{"lands", "highest"} {
		for _, when := range []string{"false", "true", "lost", "gained"} {
			t.Run(kind+"/"+when, func(t *testing.T) {
				condition := `APlayerPassesFilter(Opponent,ControlsMorePermanentThanPlayer(You,IsCardtype(Land)))`
				if kind == "highest" {
					condition = `APlayerPassesFilter(AnyPlayer,SinglePlayer(ThePlayerWithTheMostLife))`
				}
				factory := permanentScopeFactory(t, "Enchantment", `TriggerI(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),`+condition+`,ActionList([GainLife(Integer(1))]))`)
				g, a, b := ronGame()
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				var land *mage.Permanent
				setTrue := func() {
					if kind == "highest" {
						b.SetLife(25)
					} else {
						land = g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
					}
				}
				if when == "true" || when == "lost" {
					setTrue()
				}
				queueRONUpkeep(g, b)
				if g.StackSize() != 0 {
					t.Fatal("wrong upkeep")
				}
				queueRONUpkeep(g, a)
				wantStack := 0
				if when == "true" || when == "lost" {
					wantStack = 1
				}
				if g.StackSize() != wantStack {
					t.Fatal("detection", g.StackSize())
				}
				if when == "gained" {
					setTrue()
				}
				if when == "lost" {
					if kind == "highest" {
						b.SetLife(20)
					} else {
						g.PhaseOut(land.ID())
					}
				}
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				g.ResolveStack()
				want := 20
				if when == "true" {
					want = 21
				}
				if a.Life() != want {
					t.Fatal("condition was not checked at both boundaries", a.Life())
				}
			})
		}
	}
}

func TestRONUpkeepPlayerConditionsInvalid(t *testing.T) {
	for _, condition := range []string{
		`PlayerPassesFilter(Trigger_ThatPlayer(),ControlsA(IsLandType(Plains)))`,
		`PlayerPassesFilter(Ref_TargetPlayer,ControlsA(IsLandType(Plains)))`,
		`PlayerPassesFilter(ThisPermanent,ControlsA(IsLandType(Plains)))`,
		`PlayerPassesFilter(Trigger_ThatPlayer,ControlsA())`,
		`PlayerPassesFilter(Trigger_ThatPlayer,ControlsA(And([])))`,
		`PlayerPassesFilter(Trigger_ThatPlayer,ControlsNo(IsLandType(Plains)))`,
		`APlayerPassesFilter(Opponent(),ControlsMorePermanentThanPlayer(You,IsCardtype(Land)))`,
		`APlayerPassesFilter(Opponent,ControlsMorePermanentThanPlayer(You))`,
		`APlayerPassesFilter(Opponent,ControlsMorePermanentThanPlayer(Trigger_ThatPlayer,IsCardtype(Land)))`,
		`APlayerPassesFilter(AnyPlayer,SinglePlayer(ThePlayerWithTheMostLife()))`,
		`APlayerPassesFilter(You,SinglePlayer(ThePlayerWithTheMostLife))`,
	} {
		text := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerI(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),` + condition + `,ActionList([GainLife(Integer(1))]))])`
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", condition)
		}
	}
	for _, event := range []string{`AtTheBeginningOfAPlayersUpkeep(AnyPlayer())`, `AtTheBeginningOfAPlayersUpkeep(SinglePlayer(Opponent))`, `AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You()))`, `AtTheBeginningOfAPlayersEndStep(AnyPlayer)`} {
		text := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerI(` + event + `,PlayerPassesFilter(Trigger_ThatPlayer,ControlsA(IsLandType(Plains))),ActionList([GainLife(Integer(1))]))])`
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", event)
		}
	}
}

func TestRONUpkeepPlayerConditionsInventoryAndScope(t *testing.T) {
	for _, name := range []string{"Spiritual Sanctuary", "Land Tax", "Ghazbán Ogre"} {
		set := "LEG"
		if name == "Ghazbán Ogre" {
			set = "ARN"
		}
		n := hostUntapRecord(t, set, name)
		entries := map[string]*InventorySignature{}
		walkInventory(n, name, "Card", "card", "collection", true, entries)
		for _, entry := range entries {
			if strings.Contains(entry.Context, "TriggerI[1]") && entry.Role == "unknown" {
				t.Fatal(name, "unclassified condition", entry.Name)
			}
		}
	}
	factory := permanentScopeFactory(t, "Enchantment", `TriggerI(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),PlayerPassesFilter(Trigger_ThatPlayer,ControlsA(IsLandType(Plains))),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`)
	first, second := factory(), factory()
	if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	if first.ID() == second.ID() {
		t.Fatal("shared card instance")
	}
	bad := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerI(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),PlayerPassesFilter(Trigger_ThatPlayer,ControlsA(IsLandType(Plains))),ActionList([])),Activated(PayMana([]),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))])`
	nodes, err := Parse(strings.NewReader(bad), "scope.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("event binding escaped its rule")
	}
}

func TestRONSpiritualSanctuaryNativeUpkeep(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Spiritual Sanctuary"), a.PlayerID())
	g.AddContinuousEffect(mage.TemporaryKeyword(source.ID(), core.Shroud))
	land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Plains"), b.PlayerID())
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Upkeep)
	if b.Life() != 21 || a.Life() != 20 || g.StackSize() != 0 {
		t.Fatal("native upkeep gain", a.Life(), b.Life(), g.StackSize())
	}
	g.PhaseOut(land.ID())
	g.RunStepWithPriority(core.Upkeep)
	if b.Life() != 21 {
		t.Fatal("phased Plains triggered")
	}
	g.PhaseIn([]uuid.UUID{land.ID()})
	g.PhaseOut(source.ID())
	g.RunStepWithPriority(core.Upkeep)
	if b.Life() != 21 {
		t.Fatal("phased source triggered")
	}
}

func TestRONUpkeepPlayerConditionsStrictRecords(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	count := map[string]int{}
	selected := fstest.MapFS{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range nodes {
			name := n.Fields["Name"].String
			if name != "Spiritual Sanctuary" && name != "Land Tax" && name != "Ghazbán Ogre" {
				continue
			}
			count[name]++
			factory, compileErr := Compile(n)
			if name == "Spiritual Sanctuary" {
				if compileErr != nil {
					t.Fatal(compileErr)
				}
				first, second := factory(), factory()
				if first.ID() == second.ID() {
					t.Fatal("shared factory identity")
				}
				if err := mage.ValidateCardDefinition(name, first); err != nil {
					t.Fatal(err)
				}
			} else {
				want := "SetController"
				if name == "Land Tax" {
					want = "SearchLibrary"
				}
				if compileErr == nil || !strings.Contains(compileErr.Error(), want) {
					t.Fatal("remaining body blocker", name, compileErr)
				}
			}
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Spiritual Sanctuary"`) {
				selected[filepath.Base(file)] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	if len(count) != 3 {
		t.Fatal("selected names", count)
	}
	for name, n := range count {
		if n != 1 {
			t.Fatal("reprint count", name, n)
		}
	}
	registry, err := LoadFS(selected, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("complete selected loading", err)
	}
	selected["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerI(AtTheBeginningOfAPlayersUpkeep(AnyPlayer()),PlayerPassesFilter(Trigger_ThatPlayer,ControlsA(IsLandType(Plains))),ActionList([]))])`)}
	if registry, err := LoadFS(selected, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry published")
	}
}
