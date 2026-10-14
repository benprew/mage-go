package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDiscardCountReturnRecall(t *testing.T) {
	record := hostUntapRecord(t, "LEG", "Recall")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"normal", "short", "zero", "copy", "clone"} {
		t.Run(mode, func(t *testing.T) {
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			x := 2
			if mode == "zero" {
				x = 0
			}
			held := 2
			if mode == "short" {
				held = 1
			}
			for range held {
				c := mage.NewLand("Held")
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
			}
			for range 3 {
				c := mage.NewLand("Grave")
				c.SetOwner(a.PlayerID())
				a.AddToGraveyard(c)
			}
			a.ManaPool().Add(core.Blue, 1)
			a.ManaPool().Add(core.Colorless, 2*x)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, x); err != nil {
				t.Fatal(err)
			}
			if len(a.Graveyard()) != 3 || len(a.Hand()) != held {
				t.Fatal("resolution ran at cast")
			}
			if mode == "copy" {
				c := mage.NewLand("B held")
				c.SetOwner(b.PlayerID())
				b.AddToHand(c)
				for range 3 {
					c := mage.NewLand("B grave")
					c.SetOwner(b.PlayerID())
					b.AddToGraveyard(c)
				}
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
				if len(b.Hand()) != 1 || len(b.Graveyard()) != 3 || len(a.Graveyard()) != 3 {
					t.Fatal("copy shared payer or result")
				}
			}
			if mode == "clone" {
				clone := g.Clone()
				clone.ResolveStack()
				if clone.FindExiledCard(card.ID()) == nil || g.FindExiledCard(card.ID()) != nil {
					t.Fatal("clone shared resolution")
				}
			}
			g.ResolveStack()
			if len(a.Hand()) != held || len(a.Graveyard()) != 3 || g.FindExiledCard(card.ID()) == nil || a.ManaPool().TotalMana() != 0 {
				t.Fatalf("Recall result hand %d grave %d", len(a.Hand()), len(a.Graveyard()))
			}
		})
	}
}

func TestRONDiscardCountReturnInvalid(t *testing.T) {
	for _, body := range []string{
		`ReturnNumberCardsFromPlayersGraveyardToHand(NumCardsDiscardedThisWay,AnyCard,You)`,
		`DiscardNumberCards(ValueX),ReturnNumberCardsFromPlayersGraveyardToHand(NumCardsDiscardedThisWay(),AnyCard,You)`,
		`DiscardNumberCards(ValueX),ReturnNumberCardsFromPlayersGraveyardToHand(NumCardsDiscardedThisWay,AnyPermanent,You)`,
		`DiscardNumberCards(ValueX),ReturnNumberCardsFromPlayersGraveyardToHand(NumCardsDiscardedThisWay,AnyCard,Trigger_ThatPlayer)`,
		`If(PlayerPassesFilter(You,IsTheirTurn),[DiscardNumberCards(ValueX)]),GainLife(NumCardsDiscardedThisWay)`,
		`DiscardNumberCards(ValueX),ReturnNumberCardsFromPlayersGraveyardToHand`,
		`DiscardNumberCards(Integer(-1))`,
	} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),ManaCost:[ManaCostX],Rules:[SpellActions(ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func TestRONDiscardCountReturnStrictLoad(t *testing.T) {
	src := `Card(Name:"Count",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),ManaCost:[ManaCostX],Rules:[SpellActions(ActionList([DiscardNumberCards(ValueX),GainLife(NumCardsDiscardedThisWay),DiscardNumberCards(Integer(0)),GainLife(NumCardsDiscardedThisWay)]))])`
	registry := mage.NewRegistry()
	files := fstest.MapFS{"valid.ron": {Data: []byte(src)}}
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	c, err := registry.CreateCard("Count")
	if err != nil {
		t.Fatal(err)
	}
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.AddToHand(mage.NewLand("Held"))
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 1); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("separate producer results", a.Life())
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(src, "DiscardNumberCards(ValueX),", "", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection replaced registry")
	}
}

func TestRONDiscardCountReturnInstancesAndDelayed(t *testing.T) {
	factory, err := Compile(hostUntapRecord(t, "LEG", "Recall"))
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	for index, p := range []mage.Player{a, b} {
		g.SetActivePlayerIndex(index)
		c := factory()
		c.SetOwner(p.PlayerID())
		p.AddToHand(c)
		held := 2
		if p.PlayerID() == b.PlayerID() {
			held = 1
		}
		for range held {
			c := mage.NewLand("Held")
			c.SetOwner(p.PlayerID())
			p.AddToHand(c)
		}
		for range 3 {
			c := mage.NewLand("Grave")
			c.SetOwner(p.PlayerID())
			p.AddToGraveyard(c)
		}
		p.ManaPool().Add(core.Blue, 1)
		p.ManaPool().Add(core.Colorless, 4)
		if err := g.CastSpellByID(p.PlayerID(), c.ID(), nil, 2); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if len(p.Hand()) != held || len(p.Graveyard()) != 3 || g.FindExiledCard(c.ID()) == nil {
			t.Fatal("factory instance reused result")
		}
	}
	if len(b.Hand()) != 1 || len(a.Hand()) != 2 || len(a.Graveyard()) != 3 {
		t.Fatal("instances shared frame")
	}
	g.ResolveStack()
	if len(a.Hand()) != 2 || len(a.Graveyard()) != 3 || len(b.Graveyard()) != 3 {
		t.Fatal("instance count changed")
	}
	g.SetActivePlayerIndex(0)
	source := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([DiscardACard,CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(NumCardsDiscardedThisWay)]))]))`)
	artifact := g.PutOnBattlefield(source(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.ResolveStack()
	if a.Life() != 21 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("delayed count lost or shared")
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 {
		t.Fatal("clone count lost")
	}
}

func TestRONDiscardCountReturnCompleteLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, `Card(Name: "Recall",`) {
			record = line
			break
		}
	}
	if record == "" {
		t.Fatal("missing complete record")
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, fstest.MapFS{"recall.ron": {Data: []byte(record)}}, "*.ron"); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.CreateCard("Recall"); err != nil {
		t.Fatal(err)
	}
}
