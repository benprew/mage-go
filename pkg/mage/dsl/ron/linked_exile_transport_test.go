package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLinkedExileTransportKnowledgeVault(t *testing.T) {
	factory, err := Compile(staticBaseRecord(t, "LEG", "Knowledge Vault"))
	if err != nil {
		t.Fatal(err)
	}
	for _, sacrifice := range []bool{true, false} {
		g, a, b := ronGame()
		card := factory()
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		first, second := mage.NewLand("First"), mage.NewLand("Second")
		first.SetOwner(a.PlayerID())
		second.SetOwner(b.PlayerID())
		a.SetLibrary([]mage.Card{first, second})
		for range 2 {
			p.Tapped = false
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
		}
		old := mage.NewLand("Old hand")
		old.SetOwner(a.PlayerID())
		a.AddToHand(old)
		clone := g.Clone()
		if sacrifice {
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
		} else {
			g.DestroyPermanent(p)
		}
		if sacrifice {
			if len(a.Hand()) != 1 || a.Hand()[0].ID() != first.ID() || len(b.Hand()) != 1 {
				t.Fatal("return did not follow discard")
			}
			if len(g.GetExile()) != 0 {
				t.Fatal("sacrifice return lost source identity")
			}
		}
		g.PutTriggersOnStack()
		g.ResolveStack()
		if !sacrifice && (len(g.GetExile()) != 0 || len(a.Graveyard()) != 2 || len(b.Graveyard()) != 1 || len(a.Hand()) != 1) {
			t.Fatal("departure transport failed")
		}
		if len(clone.GetExile()) != 2 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 {
			t.Fatal("clone changed")
		}
	}
}

func TestRONLinkedExileTransportInvalid(t *testing.T) {
	producer := `Activated(PayMana([]),ActionList([Exile([TheTopCardOfLibrary],[])]))`
	for _, body := range []string{
		`ReturnEachExiledCardToOwnersHand(WasExiledWithPermanent())`,
		`ReturnEachExiledCardToOwnersHand(WasExiledWithPermanent(You))`,
		`ReturnEachExiledCardToOwnersHand(WasExiledWithPermanent(Ref_TargetPermanent))`,
		`ReturnEachExiledCardToOwnersHand(WasExiledWithPermanent(Trigger_ThatPermanent))`,
		`PutExiledCardsIntoOwnersGraveyards(WasExiledWithPermanent(ThisPermanent),You)`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+producer+`,Activated(PayMana([]),ActionList([`+body+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	for _, prefix := range []string{"", producer + "," + producer + ","} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+prefix+`Activated(PayMana([]),ActionList([ReturnEachExiledCardToOwnersHand(WasExiledWithPermanent(ThisPermanent))]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatal("accepted missing or ambiguous producer")
		}
	}
}

func TestRONLinkedExileTransportSourceIdentity(t *testing.T) {
	factory, err := Compile(staticBaseRecord(t, "LEG", "Knowledge Vault"))
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"phased", "phase-cycle", "control", "returned", "copied", "exile-replacement"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			source := factory()
			source.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(source, a.PlayerID())
			other := factory()
			other.SetOwner(a.PlayerID())
			q := g.PutOnBattlefield(other, a.PlayerID())
			first, second := mage.NewLand("First"), mage.NewLand("Second")
			first.SetOwner(a.PlayerID())
			second.SetOwner(a.PlayerID())
			a.SetLibrary([]mage.Card{first, second})
			for _, subject := range []*mage.Permanent{p, q} {
				a.ManaPool().Add(core.Colorless, 2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), subject.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "phased":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "returned":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(source.ID())
				g.PutOnBattlefield(source, a.PlayerID())
			case "copied":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "exile-replacement":
				g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), b.PlayerID())
			}
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			paid := state == "phase-cycle" || state == "copied" || state == "exile-replacement"
			for _, game := range []*mage.Game{g, clone} {
				if game.FindExiledCard(second.ID()) == nil {
					t.Fatal("another source's exile was consumed")
				}
				if paid {
					if game.FindExiledCard(first.ID()) != nil || len(game.GetPlayer(a.PlayerID()).Hand()) != 1 || len(game.GetPlayer(b.PlayerID()).Hand()) != 0 {
						t.Fatal("wrong paid transport")
					}
				} else if state != "returned" && game.FindExiledCard(first.ID()) == nil {
					t.Fatal("failed sacrifice returned cards")
				}
				game.PutTriggersOnStack()
				game.ResolveStack()
				if state == "returned" && (game.FindExiledCard(first.ID()) != nil || game.FindPermanent(p.ID()) == nil || len(game.GetPlayer(a.PlayerID()).Hand()) != 0) {
					t.Fatal("old departure lost its linked incarnation")
				}
				if state == "phased" && game.FindExiledCard(first.ID()) == nil {
					t.Fatal("phasing caused departure")
				}
			}
		})
	}
}

func TestRONLinkedExileTransportContexts(t *testing.T) {
	producer := `Activated(PayMana([]),ActionList([Exile([TheTopCardOfLibrary],[ExilesFaceDown])]))`
	for _, kind := range []string{"activation", "trigger", "delayed"} {
		t.Run(kind, func(t *testing.T) {
			body := `ReturnEachExiledCardToOwnersHand(WasExiledWithPermanent(ThisPermanent))`
			consumer := `Activated(PayMana([]),ActionList([` + body + `]))`
			if kind == "trigger" {
				consumer = `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([` + body + `]))`
			}
			if kind == "delayed" {
				consumer = `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + body + `]))]))`
			}
			factory := permanentScopeFactory(t, "Artifact", producer+","+consumer)
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			top := mage.NewLand("Top")
			top.SetOwner(a.PlayerID())
			a.SetLibrary([]mage.Card{top})
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if kind == "trigger" {
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
			} else {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "delayed" {
				g.ResolveStack()
				if len(g.WaitingDelayedPermanentSources()) != 1 || len(a.Hand()) != 0 {
					t.Fatal("waiting query executed transport")
				}
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				g.PutTriggersOnStack()
			}
			g.ResolveStack()
			if g.FindExiledCard(top.ID()) != nil || len(a.Hand()) != 1 || a.Hand()[0].ID() != top.ID() || len(b.Hand()) != 0 {
				t.Fatal("context lost saved source or owner")
			}
		})
	}
}

func TestRONLinkedExileTransportSafeHavenProducer(t *testing.T) {
	record := staticBaseRecord(t, "DRK", "Safe Haven")
	rules := record.Fields["Rules"]
	rules.Args = append(rules.Args[:1], nameScopeNode(t, `Activated(PayMana([]),ActionList([ReturnEachExiledCardToOwnersHand(WasExiledWithPermanent(ThisPermanent))]))`))
	record.Fields["Rules"] = rules
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	creature := mage.NewCreature("Subject", "", 2, 2)
	creature.SetOwner(b.PlayerID())
	target := g.PutOnBattlefield(creature, a.PlayerID())
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.GetExile()) != 0 || len(b.Hand()) != 1 || b.Hand()[0].ID() != creature.ID() {
		t.Fatal("permanent producer did not share the declaration")
	}
}

func TestRONLinkedExileTransportInvalidContexts(t *testing.T) {
	producer := `Activated(PayMana([]),ActionList([Exile([TheTopCardOfLibrary],[])]))`
	body := `ReturnEachExiledCardToOwnersHand(WasExiledWithPermanent(ThisPermanent))`
	for _, rule := range []string{
		`Activated(` + body + `,ActionList([]))`,
		`AsPermanentEnters(ThisPermanent,[` + body + `])`,
		`PermanentLayerEffect(ThisPermanent,[` + body + `])`,
		`Activated(PayMana([]),ActionList([ReturnEachExiledCardToOwnersHand(WasExiledWithPermanent)]))`,
		`Activated(PayMana([]),ActionList([ReturnEachExiledCardToOwnersHand(SinglePermanent(ThisPermanent))]))`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+producer+`,`+rule+`])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}
