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

func TestRONDiscardPaymentFactLandsEdge(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "LEG", "Land's Edge", "edge.ron"))
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"land", "nonland", "opponent", "source-left", "source-return", "phase-cycle", "source-phased", "moved", "copy", "clone", "invalid", "empty", "phase"} {
		t.Run(mode, func(t *testing.T) {
			a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A"), invalid: mode == "invalid"}
			b := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("B")}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			sourceCard := factory()
			sourceCard.SetOwner(a.PlayerID())
			a.AddToHand(sourceCard)
			a.ManaPool().Add(core.Red, 3)
			if err := g.CastSpellByID(a.PlayerID(), sourceCard.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(sourceCard.ID())
			if source == nil {
				t.Fatal("cast did not enter")
			}
			payer := a
			if mode == "opponent" {
				payer = b
			}
			var paid mage.Card = mage.NewLand("Paid land")
			if mode == "nonland" {
				paid = mage.NewArtifact("Paid artifact", "")
			}
			paid.SetOwner(payer.PlayerID())
			if mode != "empty" {
				payer.AddToHand(paid)
			}
			if mode == "phase" {
				g.PhaseOut(source.ID())
			}
			err := g.ActivateAbilityByIndex(payer.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()})
			if mode == "invalid" || mode == "empty" || mode == "phase" {
				if err == nil || g.StackSize() != 0 || b.Life() != 20 || len(payer.Graveyard()) != 0 {
					t.Fatal("failed activation spent payment")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(payer.Hand()) != 0 || len(payer.Graveyard()) != 1 {
				t.Fatal("wrong player paid")
			}
			if mode == "source-left" {
				g.RemoveFromBattlefield(source)
			}
			if mode == "source-return" {
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(sourceCard, b.PlayerID())
			}
			if mode == "source-phased" {
				g.PhaseOut(source.ID())
			}
			if mode == "phase-cycle" {
				g.PhaseIn(g.PhaseOut(source.ID()))
			}
			if mode == "moved" {
				if _, ok := g.MoveFromGraveyard(payer.PlayerID(), paid.ID(), core.ZoneHand); !ok {
					t.Fatal("move failed")
				}
			}
			if mode == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			want := 18
			if mode == "nonland" {
				want = 20
			}
			if mode == "copy" {
				want = 16
			}
			if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatal("clone result")
			}
			g.ResolveStack()
			if b.Life() != want {
				t.Fatalf("life=%d want=%d", b.Life(), want)
			}
		})
	}
}

func TestRONDiscardPaymentFactInvalid(t *testing.T) {
	branch := `If(DiscardedCardPassesFilter(IsCardtype(Land)),[GainLife(Integer(2))])`
	for _, tc := range []struct{ cost, body string }{
		{`PayMana([])`, branch}, {`And([DiscardACard,DiscardACard])`, branch},
		{`DiscardNumberCards(Integer(2))`, branch}, {`DiscardACard`, `If(DiscardedCardPassesFilter(),[])`},
		{`DiscardACard`, `If(DiscardedCardPassesFilter(IsTapped),[])`},
		{`DiscardACard`, `If(DiscardedCardPassesFilter(And([])),[])`},
		{`DiscardACard`, `If(DiscardedCardPassesFilter(IsCardtype(Land),IsCardtype(Creature)),[])`},
		{`DiscardACard()`, branch},
		{`DiscardACard`, `MayCost(DiscardACard),If(CostWasPaid,[` + branch + `])`},
		{`DiscardACardAtRandom`, branch}, {`And([DiscardACard,DiscardACardAtRandom])`, branch},
		{`DiscardCard(TheLastCardDrawnThisTurn)`, branch},
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(` + tc.cost + `,ActionList([` + tc.body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", src)
		}
	}
	src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + branch + `]))])`
	nodes, err := Parse(strings.NewReader(src), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err == nil || !strings.Contains(err.Error(), "discard") {
		t.Fatal("unbound spell branch", err)
	}
}

func TestRONDiscardPaymentFactIndependentAndTargets(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "LEG", "Land's Edge", "edge.ron"))
	if err != nil {
		t.Fatal(err)
	}
	a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	if !source.Card.HasSuperType(core.SuperWorld) {
		t.Fatal("world supertype missing")
	}
	walker := g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 8), b.PlayerID())
	creature := g.PutOnBattlefield(mage.NewCreature("Creature", "", 2, 3), b.PlayerID())
	land := mage.NewLand("First land")
	a.AddToHand(land)
	if err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{creature.ID()}); err == nil || len(a.Hand()) != 1 {
		t.Fatal("illegal target spent cost")
	}
	if err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{walker.ID()}); err != nil {
		t.Fatal(err)
	}
	a.AddToHand(mage.NewArtifact("Second payment", ""))
	if err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if b.Life() != 20 {
		t.Fatal("nonland inherited earlier land result")
	}
	g.ResolveStack()
	if g.FindPermanent(walker.ID()).Counters[core.Loyalty] != 6 || b.Life() != 20 {
		t.Fatal("saved land result or planeswalker damage lost")
	}
	a.AddToHand(mage.NewLand("Third payment"))
	if err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{walker.ID()}); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(walker.ID())
	g.ResolveStack()
	if len(a.Hand()) != 0 || g.FindPermanentIncludingPhased(walker.ID()).Counters[core.Loyalty] != 6 {
		t.Fatal("target recheck or retained payment failed")
	}
	other := g.PutOnBattlefield(factory(), b.PlayerID())
	g.CheckStateBasedActions()
	if g.FindPermanent(source.ID()) != nil || g.FindPermanent(other.ID()) == nil {
		t.Fatal("later world permanent did not replace earlier one")
	}
}

func TestRONDiscardPaymentFactFiltersAndDelay(t *testing.T) {
	rule := `Activated(DiscardACard,ActionList([If(DiscardedCardPassesFilter(And([IsCardtype(Land),IsNonColor(Blue)])),[GainLife(Integer(2))]),If(DiscardedCardPassesFilter(IsNonCardtype(Land)),[GainLife(Integer(5))]),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([If(DiscardedCardPassesFilter(IsCardtype(Land)),[GainLife(Integer(3))])]))]))`
	for _, land := range []bool{false, true} {
		a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A")}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
		var card mage.Card = mage.NewArtifact("Paid", "")
		if land {
			card = mage.NewLand("Paid")
		}
		a.AddToHand(card)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		want := 25
		if land {
			want = 22
		}
		if a.Life() != want {
			t.Fatal("separate predicate result")
		}
		g.RemoveFromBattlefield(source)
		g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
		g.PutTriggersOnStack()
		g.ResolveStack()
		if a.Life() != 25 {
			t.Fatalf("saved delayed predicate: %d", a.Life())
		}
	}
}

func TestRONDiscardPaymentFactLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, `Card(Name: "Land's Edge",`) {
			record = line
			break
		}
	}
	if record == "" {
		t.Fatal("record missing")
	}
	files := fstest.MapFS{"edge.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Land's Edge")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Land's Edge")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() == second.ID() || first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("factory shares ability state")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.Replace(record, "Land's Edge", "Invalid edge", 1), "DiscardACard,", "PayMana([]),"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Land's Edge") {
		t.Fatal("invalid collection changed registry", err)
	}
}

func TestRONDiscardPaymentFactInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(DiscardACard,ActionList([UnknownAction,If(DiscardedCardPassesFilter(IsCardtype(Land)),[GainLife(Integer(1))])]))])`
	report, err := InspectFS(fstest.MapFS{"inventory.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, signature := range report.Signatures {
		if signature.Name == "IsCardtype" && signature.Role == "card_filter" {
			return
		}
	}
	t.Fatal("discarded-card predicate was not inventoried below unsupported action")
}
