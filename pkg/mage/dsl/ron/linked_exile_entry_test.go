package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLinkedExileEntrySafeHaven(t *testing.T) {
	factory, err := Compile(staticBaseRecord(t, "DRK", "Safe Haven"))
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"pay", "decline", "phased", "phase-cycle", "control", "returned", "copied", "illegal-target", "token"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "decline"}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			g.SetTurn(2)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			other := factory()
			other.SetOwner(a.PlayerID())
			q := g.PutOnBattlefield(other, a.PlayerID())
			creature := mage.NewCreature("Subject", "", 2, 2, mage.WithAbility(mage.EnterTappedAsEnters()))
			if scenario == "token" {
				creature = mage.NewToken("Subject", 2, 2, []core.CardType{core.TypeCreature}, nil)
			}
			creature.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(creature, a.PlayerID())
			separate := mage.NewCreature("Separate", "", 2, 2)
			separate.SetOwner(a.PlayerID())
			s := g.PutOnBattlefield(separate, a.PlayerID())
			for i, pair := range [][2]*mage.Permanent{{p, target}, {q, s}} {
				a.ManaPool().Add(core.Colorless, 2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), pair[0].ID(), 0, []uuid.UUID{pair[1].ID()}); err != nil {
					t.Fatal(err)
				}
				if i == 0 && scenario == "illegal-target" {
					g.PhaseOut(target.ID())
				}
				g.ResolveStack()
			}
			if scenario == "illegal-target" {
				if g.FindExiledCard(creature.ID()) != nil || g.FindPermanentIncludingPhased(target.ID()) == nil {
					t.Fatal("illegal target exiled")
				}
				return
			}
			qPhased := g.PhaseOut(q.ID())
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.PhaseIn(qPhased)
			switch scenario {
			case "phased":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "returned":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "copied":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.SetPlayerAt(0, &counterPaymentClonePayer{Player: clone.GetPlayer(a.PlayerID()), accept: a.accept})
			clone.SetPlayerAt(1, &counterPaymentClonePayer{Player: clone.GetPlayer(b.PlayerID()), accept: b.accept})
			clone.ResolveStack()
			g.ResolveStack()
			paid := scenario == "pay" || scenario == "phase-cycle" || scenario == "copied" || scenario == "token"
			for _, game := range []*mage.Game{g, clone} {
				returned := game.FindPermanent(creature.ID())
				if scenario == "token" {
					if returned != nil || game.FindExiledCard(creature.ID()) != nil {
						t.Fatal("token returned")
					}
				} else if paid {
					if returned == nil || returned.ControllerID() != b.PlayerID() || !returned.Tapped || game.FindExiledCard(creature.ID()) != nil {
						t.Fatalf("owner entry failed gameOriginal=%v returned=%v exile=%v source=%v", game == g, returned, game.FindExiledCard(creature.ID()), game.FindPermanent(p.ID()))
					}
				} else if returned != nil || game.FindExiledCard(creature.ID()) == nil {
					t.Fatal("failed sacrifice returned cards")
				}
				if game.FindExiledCard(separate.ID()) == nil || game.FindPermanent(q.ID()) == nil {
					t.Fatal("another source lost its linked cards")
				}
				if paid && game.FindPermanent(p.ID()) != nil {
					t.Fatal("paid source remains")
				}
				if !paid && game.FindPermanentIncludingPhased(p.ID()) == nil {
					t.Fatal("failed payment removed source")
				}
			}
		})
	}
}

func TestRONLinkedExileEntryInvalid(t *testing.T) {
	producer := `Activated(PayMana([]),ActionList([Exile([TheTopCardOfLibrary],[])]))`
	body := `PutEachExiledCardOntoTheBattlefield(WasExiledWithPermanent(ThisPermanent),[EntersUnderOwnersControl])`
	for _, rule := range []string{
		`Activated(PayMana([]),ActionList([PutEachExiledCardOntoTheBattlefield(WasExiledWithPermanent(ThisPermanent),[])]))`,
		`Activated(PayMana([]),ActionList([PutEachExiledCardOntoTheBattlefield(WasExiledWithPermanent(ThisPermanent),[EntersUnderOwnersControl,EntersTapped])]))`,
		`Activated(PayMana([]),ActionList([PutEachExiledCardOntoTheBattlefield(WasExiledWithPermanent(You),[EntersUnderOwnersControl])]))`,
		`Activated(PayMana([]),ActionList([PutEachExiledCardOntoTheBattlefield(WasExiledWithPermanent(Ref_TargetPermanent),[EntersUnderOwnersControl])]))`,
		`Activated(PayMana([]),ActionList([PutEachExiledCardOntoTheBattlefield(TheOtherPermanentsExiledThisWay,[EntersUnderOwnersControl])]))`,
		`Activated(` + body + `,ActionList([]))`, `AsPermanentEnters(ThisPermanent,[` + body + `])`, `PermanentLayerEffect(ThisPermanent,[` + body + `])`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+producer+`,`+rule+`])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	for _, prefix := range []string{"", producer + "," + producer + ","} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+prefix+`Activated(PayMana([]),ActionList([`+body+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatal("missing or ambiguous link accepted")
		}
	}
}

func TestRONLinkedExileEntryContexts(t *testing.T) {
	producer := `Activated(PayMana([]),ActionList([Exile([TheTopCardOfLibrary],[ExilesFaceDown])]))`
	for _, kind := range []string{"activation", "departure", "delayed"} {
		t.Run(kind, func(t *testing.T) {
			body := `PutEachExiledCardOntoTheBattlefield(WasExiledWithPermanent(ThisPermanent),[EntersUnderOwnersControl])`
			consumer := `Activated(PayMana([]),ActionList([` + body + `]))`
			if kind == "departure" {
				consumer = `TriggerA(WhenAPermanentLeavesTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([PutEachExiledCardOntoTheBattlefield(WasExiledWithPermanent(Trigger_ThatPermanent),[EntersUnderOwnersControl])]))`
			}
			if kind == "delayed" {
				consumer = `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + body + `]))]))`
			}
			factory := permanentScopeFactory(t, "Artifact", producer+","+consumer)
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			top := mage.NewCreature("Top", "", 1, 1)
			top.SetOwner(b.PlayerID())
			a.SetLibrary([]mage.Card{top})
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if kind != "departure" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "delayed" {
				g.ResolveStack()
				for range 2 {
					if len(g.WaitingDelayedPermanentSources()) != 1 || g.FindPermanent(top.ID()) != nil {
						t.Fatal("waiting query executed entry")
					}
				}
			}
			if kind != "activation" {
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			}
			if kind == "delayed" {
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			}
			g.ResolveStack()
			if entered := g.FindPermanent(top.ID()); entered == nil || entered.ControllerID() != b.PlayerID() || g.FindExiledCard(top.ID()) != nil {
				t.Fatal("saved source context or owner lost")
			}
		})
	}
}

func TestRONLinkedExileEntryCopyReplacement(t *testing.T) {
	g, a, b := ronGame()
	payer := &ronEntryCopyPlayer{Player: a, accept: true}
	g.SetPlayerAt(0, payer)
	subject := repositoryCard(t, "LEB", "Clone")
	subject.SetOwner(a.PlayerID())
	donor := g.PutOnBattlefield(repositoryCard(t, "FEM", "Icatian Javelineers"), b.PlayerID())
	producer := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([Exile([TheTopCardOfLibrary],[])])),Activated(PayMana([]),ActionList([PutEachExiledCardOntoTheBattlefield(WasExiledWithPermanent(ThisPermanent),[EntersUnderOwnersControl])]))`)
	source := producer()
	source.SetOwner(a.PlayerID())
	q := g.PutOnBattlefield(source, a.PlayerID())
	a.SetLibrary([]mage.Card{subject})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if payer.offers != 0 {
		t.Fatal("exile made an entry choice")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	copied := g.FindPermanent(subject.ID())
	if copied == nil || copied.Name() != donor.Name() || copied.Counters[core.Javelin] != 1 || payer.offers != 1 || payer.choices != 1 {
		t.Fatal("entry replacement did not acquire fresh copied rules")
	}
	g.BouncePermanentToHand(copied)
	if a.Hand()[0].Name() != "Clone" || g.LKI(subject.ID()).Name != "Icatian Javelineers" {
		t.Fatal("printed departure or copy LKI lost")
	}
}
