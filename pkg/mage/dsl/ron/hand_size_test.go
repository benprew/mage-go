package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func handSizeFactory(t *testing.T, rules string) mage.CardFactory {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Hand test",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rules+`])`), "hand.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Hand test", factory()); err != nil {
		t.Fatal(err)
	}
	return factory
}

func fillRONHand(player *mage.BasePlayer, count int) {
	for range count {
		card := mage.NewCreature("Hand card", "", 1, 1)
		card.SetOwner(player.PlayerID())
		player.AddToHand(card)
	}
}

func TestRONHandSizeIvoryTower(t *testing.T) {
	for _, set := range []string{"ATQ", "3ED"} {
		for _, size := range []int{0, 3, 4, 7} {
			t.Run(fmt.Sprintf("%s/%d", set, size), func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, set, "Ivory Tower")
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("opponent upkeep triggered")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() != 1 || a.Life() != 20 {
					t.Fatal("upkeep timing")
				}
				fillRONHand(a, size)
				fillRONHand(b, 10)
				g.DestroyPermanent(g.FindPermanent(p.ID()))
				clone := g.Clone()
				clone.ResolveStack()
				if a.Life() != 20 {
					t.Fatal("clone changed original")
				}
				g.ResolveStack()
				want := 20 + max(0, size-4)
				if a.Life() != want || b.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != want {
					t.Fatal("hand size must use the resolving controller and current hand")
				}
			})
		}
	}
}

func TestRONHandSizeStormWorld(t *testing.T) {
	for _, change := range []string{"none", "control", "return", "phase-before", "phase-after", "copy", "prevent", "large-hand", "lifelink"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Storm World")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if !card.HasSuperType(core.SuperWorld) {
				t.Fatal("missing world supertype")
			}
			if change == "phase-before" {
				g.PhaseOut(p.ID())
			}
			if change == "lifelink" {
				p.GrantBaseAttr(core.Lifelink)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			if change == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.PutTriggersOnStack()
			if change == "phase-before" {
				if g.StackSize() != 0 {
					t.Fatal("phased source triggered")
				}
				return
			}
			if g.StackSize() != 1 || g.StackPeek().Controller != a.PlayerID() || b.Life() != 20 {
				t.Fatal("event controller or timing")
			}
			fillRONHand(a, 8)
			fillRONHand(b, 1)
			switch change {
			case "return", "lifelink":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "phase-after":
				g.PhaseOut(p.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "prevent":
				g.AddPreventionShield(b.PlayerID(), 2)
			case "large-hand":
				fillRONHand(b, 5)
			}
			g.SetActivePlayerIndex(0)
			g.SetStep(core.EndStep)
			clone := g.Clone()
			clone.ResolveStack()
			if b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			want := 17
			switch change {
			case "copy":
				want = 14
			case "prevent":
				want = 19
			case "large-hand":
				want = 20
			}
			wantA := 20
			if change == "lifelink" {
				wantA = 23
			}
			if b.Life() != want || a.Life() != wantA || clone.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatalf("saved event player, damage pipeline or LKI: A=%d B=%d", a.Life(), b.Life())
			}
		})
	}
}

func TestRONHandSizeIndependentEvents(t *testing.T) {
	factory := handSizeFactory(t, `TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([PermanentDealsDamage(ThisPermanent,Minus(Integer(4),TheNumberOfCardsInPlayersHand(Trigger_ThatPlayer)),Player(Trigger_ThatPlayer))]))`)
	g, a, b := ronGame()
	for range 2 {
		g.PutOnBattlefield(factory(), a.PlayerID())
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatal("independent events were lost")
	}
	fillRONHand(a, 3)
	fillRONHand(b, 1)
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 14 {
		t.Fatal("event bindings shared state across instances or events")
	}
}

func TestRONHandSizeSignedIntermediateAndOrder(t *testing.T) {
	factory := handSizeFactory(t, `Activated(PayMana([]),ActionList([DrawNumberCards(Integer(1)),GainLife(Minus(TheNumberOfCardsInPlayersHand(You),Minus(Integer(2),Integer(5))))]))`)
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	a.AddToLibrary(mage.NewCreature("Draw", "", 1, 1))
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 24 || len(a.Hand()) != 1 {
		t.Fatal("values must keep signed intermediate results and read after the draw")
	}
}

func TestRONHandSizeSpellPlayer(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Hand spell",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostR],Rules:[SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([SpellDealsDamage(ThisSpell,TheNumberOfCardsInPlayersHand(Ref_TargetPlayer),Player(Ref_TargetPlayer))])))])`), "spell.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Hand spell", factory()); err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	spell := factory()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	fillRONHand(b, 2)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil {
		t.Fatal("unpaid spell accepted")
	}
	if len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	fillRONHand(b, 1)
	g.AddPreventionShield(b.PlayerID(), 1)
	g.ResolveStack()
	if b.Life() != 18 || a.Life() != 20 || len(a.Graveyard()) != 1 {
		t.Fatal("typed spell player or resolution value bypassed the damage pipeline")
	}
}

func TestRONHandSizeTargetAndSequence(t *testing.T) {
	factory := handSizeFactory(t, `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([DrawNumberCards(Integer(1)),PermanentDealsDamage(ThisPermanent,Minus(TheNumberOfCardsInPlayersHand(Ref_TargetPlayer),Minus(Integer(4),Integer(2))),Player(Ref_TargetPlayer))])))`)
	for _, illegal := range []bool{false, true} {
		g, a, b := ronGame()
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		fillRONHand(b, 4)
		a.AddToLibrary(mage.NewCreature("Draw", "", 1, 1))
		target := b.PlayerID()
		if illegal {
			target = p.ID()
		}
		err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target})
		if (err != nil) != illegal {
			t.Fatalf("target declaration: %v", err)
		}
		g.ResolveStack()
		want := 18
		if illegal {
			want = 20
		}
		if b.Life() != want || len(a.Hand()) != map[bool]int{false: 1, true: 0}[illegal] {
			t.Fatal("target legality or nested subtraction")
		}
	}
}

func TestRONHandSizeInvalidDeclarations(t *testing.T) {
	for _, amount := range []string{
		`TheNumberOfCardsInPlayersHand`, `TheNumberOfCardsInPlayersHand()`,
		`TheNumberOfCardsInPlayersHand(You,You)`, `TheNumberOfCardsInPlayersHand(You())`,
		`TheNumberOfCardsInPlayersHand(ThisPermanent)`, `TheNumberOfCardsInPlayersHand(Trigger_ThatPlayer)`,
		`TheNumberOfCardsInPlayersHand(Ref_TargetPlayer)`, `TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer)`,
		`Minus`, `Minus(Integer(1))`, `Minus(Integer(1),Integer(2),Integer(3))`,
		`Minus(Integer(1),TheNumberOfCardsInPlayersHand(Ref_TargetPlayer))`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([GainLife(` + amount + `)]))])`
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", amount)
		}
	}
	for _, players := range []string{`AnyPlayer()`, `SinglePlayer(You())`, `SinglePlayer(Trigger_ThatPlayer)`, `SinglePlayer(Ref_TargetPlayer)`, `AnyPlayer,AnyPlayer`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(AtTheBeginningOfAPlayersUpkeep(`+players+`),ActionList([GainLife(Integer(1))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted upkeep %s", players)
		}
	}
	for _, rules := range []string{
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([])),Activated(PayMana([]),ActionList([GainLife(TheNumberOfCardsInPlayersHand(Trigger_ThatPlayer))]))`,
		`Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([]))),Activated(PayMana([]),ActionList([GainLife(TheNumberOfCardsInPlayersHand(Ref_TargetPlayer))]))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([PermanentDealsDamage(ThisPermanent,Integer(1),Player(Trigger_ThatPlayer()))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rules+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted binding leak or invalid recipient", rules)
		}
	}
}

func TestRONHandSizeCopperTabletNative(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, set, "Copper Tablet")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(card, a.PlayerID())
			g.RunStepWithPriority(core.Upkeep)
			if a.Life() != 19 || b.Life() != 20 || g.StackSize() != 0 {
				t.Fatal("native controller upkeep damage")
			}
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.Upkeep)
			if a.Life() != 19 || b.Life() != 19 {
				t.Fatal("native opponent upkeep damage")
			}
			g.PutOnBattlefield(repositoryCard(t, set, "Copper Tablet"), b.PlayerID())
			if a.Life() != 19 || b.Life() != 19 {
				t.Fatal("entry repeated the upkeep boundary")
			}
			g.RunStepWithPriority(core.Cleanup)
			if a.Life() != 19 || b.Life() != 19 {
				t.Fatal("cleanup triggered upkeep damage")
			}
		})
	}
}

func TestRONHandSizeIvoryCopyController(t *testing.T) {
	g, a, b := ronGame()
	g.PutOnBattlefield(repositoryCard(t, "ATQ", "Ivory Tower"), a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	fillRONHand(a, 5)
	fillRONHand(b, 7)
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 23 {
		t.Fatal("You must use each copy's controller")
	}
}

func TestRONHandSizeInventoryRoles(t *testing.T) {
	source := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([GainLife(Minus(TheNumberOfCardsInPlayersHand(Trigger_ThatPlayer),Integer(4)))]))])`
	report, err := InspectFS(fstest.MapFS{"test.ron": &fstest.MapFile{Data: []byte(source)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v", err)
	}
	for _, name := range []string{"Minus", "TheNumberOfCardsInPlayersHand", "Trigger_ThatPlayer"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name {
				want := "value"
				if name == "Trigger_ThatPlayer" {
					want = "player_reference"
				}
				if signature.Role != want {
					t.Fatalf("%s role: %s", name, signature.Role)
				}
				found = true
			}
		}
		if !found {
			t.Fatal("missing nested signature", name)
		}
	}
}

func TestRONHandSizeAcceptedRecordsLoad(t *testing.T) {
	files := fstest.MapFS{}
	for _, entry := range []struct{ set, name string }{
		{"ATQ", "Ivory Tower"}, {"3ED", "Ivory Tower"}, {"LEG", "Storm World"},
		{"LEA", "Copper Tablet"}, {"LEB", "Copper Tablet"}, {"2ED", "Copper Tablet"}, {"CED", "Copper Tablet"}, {"CEI", "Copper Tablet"},
	} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + entry.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for record := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(record, `Name: "`+entry.name+`"`) {
				files[entry.set+entry.name+".ron"] = &fstest.MapFile{Data: []byte(record)}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 3 {
		t.Fatalf("complete definitions and reprints: %v", err)
	}
}
