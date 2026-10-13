package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONHandCardinalityBalance(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			factory, err := Compile(staticBaseRecord(t, set, "Balance"))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			for range 3 {
				g.PutOnBattlefield(mage.NewLand("A land"), a.PlayerID())
				g.PutOnBattlefield(mage.NewCreature("B creature", "", 1, 1), b.PlayerID())
			}
			g.PutOnBattlefield(mage.NewLand("B land"), b.PlayerID())
			g.PutOnBattlefield(mage.NewCreature("A creature", "", 1, 1), a.PlayerID())
			discardEvents := 0
			observer := mage.NewTriggered(core.EvtDiscard, false).SetCondition(func(_ *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
				discardEvents++
				lands := g.CountBattlefield(mage.IsLand)
				creatures := g.CountBattlefield(mage.IsCreature)
				if lands != 2 || creatures != 4 || len(a.Hand()) != 1 || len(b.Hand()) != 1 {
					t.Fatal("Balance instruction order or simultaneous hand movement")
				}
				return false
			})
			g.PutOnBattlefield(mage.NewArtifact("Observer", "", mage.WithAbility(observer)), a.PlayerID())
			for range 4 {
				a.AddToHand(mage.NewLand("A hand"))
			}
			b.AddToHand(mage.NewLand("B hand"))
			spell := factory()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.White, 2)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if len(a.Hand()) != 4 || len(b.Hand()) != 1 {
				t.Fatal("hand choices occurred during casting")
			}
			g.ResolveStack()
			if discardEvents != 3 || len(a.Hand()) != 1 || len(b.Hand()) != 1 || len(a.Graveyard()) != 6 || len(b.Graveyard()) != 2 {
				t.Fatalf("complete Balance: hands %d/%d graveyards %d/%d", len(a.Hand()), len(b.Hand()), len(a.Graveyard()), len(b.Graveyard()))
			}
		})
	}
}

func TestRONHandCardinalityInvalid(t *testing.T) {
	for _, action := range []string{
		`DiscardAllButNumberCards`, `DiscardAllButNumberCards()`,
		`DiscardAllButNumberCards(Integer(1),Integer(2))`,
		`DiscardAllButNumberCards(Integer(-1))`, `DiscardAllButNumberCards(Integer("1"))`,
		`DiscardAllButNumberCards(Integer())`, `DiscardAllButNumberCards(1)`,
		`DiscardAllButNumberCards(ValueX)`,
		`DiscardAllButNumberCards(TheLowestNumberOfCardsInHandAmongPlayers)`,
		`DiscardAllButNumberCards(TheLowestNumberOfCardsInHandAmongPlayers())`,
		`DiscardAllButNumberCards(TheLowestNumberOfCardsInHandAmongPlayers(AnyPlayer()))`,
		`DiscardAllButNumberCards(TheLowestNumberOfCardsInHandAmongPlayers(Opponent))`,
		`DiscardAllButNumberCards(TheLowestNumberOfCardsInHandAmongPlayers(AnyPlayer,AnyPlayer))`,
	} {
		record := staticBaseRecord(t, "LEA", "Balance")
		record.Fields["Rules"] = nameScopeNode(t, `[SpellActions(ActionList([EachPlayerAction(AnyPlayer,`+action+`)]))]`)
		if _, err := Compile(record); err == nil {
			t.Fatal("invalid declaration accepted", action)
		}
	}
	for _, rule := range []string{
		`SpellActions(ActionList([EachPlayerAction(Opponent,DiscardAllButNumberCards(Integer(1)))]))`,
		`SpellActions(ActionList([PlayerAction(You,DiscardAllButNumberCards(Integer(1)))]))`,
		`SpellActions(ActionList([GainLife(TheLowestNumberOfCardsInHandAmongPlayers(AnyPlayer))]))`,
		`Activated(DiscardAllButNumberCards(Integer(1)),ActionList([]))`,
		`PermanentLayerEffect(ThisPermanent,[DiscardAllButNumberCards(Integer(1))])`,
	} {
		record := staticBaseRecord(t, "LEA", "Balance")
		record.Fields["Rules"] = nameScopeNode(t, `[`+rule+`]`)
		if _, err := Compile(record); err == nil {
			t.Fatal("invalid context accepted", rule)
		}
	}
}

func TestRONHandCardinalityCopiesClonesAndDelay(t *testing.T) {
	body := `EachPlayerAction(AnyPlayer,DiscardAllButNumberCards(TheLowestNumberOfCardsInHandAmongPlayers(AnyPlayer)))`
	for _, state := range []string{"copy", "clone", "delay", "empty"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			if state == "delay" {
				factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))`)
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.BouncePermanentToHand(source)
				if _, ok := a.RemoveFromHand(source.ID()); !ok {
					t.Fatal("source return failed")
				}
				g.PutOnBattlefield(source.Card, b.PlayerID())
				for range 3 {
					a.AddToHand(mage.NewLand("Late A"))
				}
				b.AddToHand(mage.NewLand("Late B"))
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				g.PutTriggersOnStack()
				g.ResolveStack()
				if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
					t.Fatal("delay did not read current hands after source return")
				}
				return
			}
			f := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([`+body+`]))`)
			spell := f()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			for range 3 {
				a.AddToHand(mage.NewLand("A"))
			}
			if state != "empty" {
				b.AddToHand(mage.NewLand("B"))
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			if state == "clone" {
				p := clone.GetPlayer(b.PlayerID())
				p.RemoveFromHand(p.Hand()[0].ID())
			}
			g.ResolveStack()
			want := 1
			if state == "empty" {
				want = 0
			}
			if len(a.Hand()) != want || len(clone.GetPlayer(a.PlayerID()).Hand()) != 3 {
				t.Fatal("resolution or clone isolation")
			}
			clone.ResolveStack()
			if state == "clone" {
				want = 0
			}
			if len(clone.GetPlayer(a.PlayerID()).Hand()) != want {
				t.Fatal("cloned spell did not read its current hands")
			}
		})
	}
}

func TestRONHandCardinalityInventoryAndAtomicLoad(t *testing.T) {
	body := `EachPlayerAction(AnyPlayer,DiscardAllButNumberCards(TheLowestNumberOfCardsInHandAmongPlayers(AnyPlayer)))`
	fs := fstest.MapFS{"a.ron": {Data: []byte(`Card(Name:"Valid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + body + `]))])`)}}
	report, err := InspectFS(fs, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	found := false
	for _, sig := range report.Signatures {
		if sig.Name == "TheLowestNumberOfCardsInHandAmongPlayers" && sig.Role == "value" {
			found = true
		}
	}
	if !found {
		t.Fatal("hand cardinality value role missing")
	}
	fs["b.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(string(fs["a.ron"].Data), `Name:"Valid"`, `Name:"Invalid"`, 1))}
	fs["b.ron"].Data = []byte(strings.Replace(string(fs["b.ron"].Data), `TheLowestNumberOfCardsInHandAmongPlayers(AnyPlayer)`, `Integer(-1)`, 1))
	if registry, err := LoadFS(fs, "*.ron"); err == nil || registry != nil {
		t.Fatal("invalid hand cardinality published a partial registry")
	}
}
