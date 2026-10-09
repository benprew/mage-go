package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLinkedExileProducers(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEG", "Knowledge Vault"}, {"DRK", "Safe Haven"}} {
		for _, state := range []string{"present", "departed", "returned", "phased", "control", "empty"} {
			t.Run(tc.name+state, func(t *testing.T) {
				record := staticBaseRecord(t, tc.set, tc.name)
				rules := record.Fields["Rules"]
				rules.Args = rules.Args[:1]
				record.Fields["Rules"] = rules
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				card := factory()
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				top := mage.NewCreature("Subject", "", 2, 2)
				top.SetOwner(b.PlayerID())
				var targets []uuid.UUID
				if tc.name == "Knowledge Vault" {
					if state != "empty" {
						a.SetLibrary([]mage.Card{top})
					}
				} else {
					target := g.PutOnBattlefield(top, a.PlayerID())
					targets = []uuid.UUID{target.ID()}
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, targets); err == nil || p.Tapped || len(g.GetExile()) != 0 {
					t.Fatal("unpaid exile changed state")
				}
				a.ManaPool().Add(core.Colorless, 2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, targets); err != nil {
					t.Fatal(err)
				}
				if !p.Tapped || len(g.GetExile()) != 0 {
					t.Fatal("wrong announcement timing")
				}
				clone := g.Clone()
				switch state {
				case "departed", "returned":
					g.DestroyPermanent(p)
					if state == "returned" {
						a.RemoveFromGraveyard(card.ID())
						g.PutOnBattlefield(card, b.PlayerID())
					}
				case "phased":
					g.PhaseOut(p.ID())
				case "control":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), p.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				if tc.name == "Safe Haven" && state == "empty" {
					g.PhaseOut(targets[0])
				}
				g.ResolveStack()
				exiled := g.FindExiledCard(top.ID())
				if state == "empty" {
					if exiled != nil {
						t.Fatal("absent subject moved")
					}
				} else {
					if exiled == nil || exiled.ExiledBy != p.ID() || exiled.Card.Owner() != b.PlayerID() || exiled.Visit == uuid.Nil {
						t.Fatal("wrong exile provenance")
					}
					if tc.name == "Knowledge Vault" && (!exiled.FaceDown || exiled.VisibleTo(a.PlayerID()) || exiled.VisibleTo(b.PlayerID())) {
						t.Fatal("private exile revealed")
					}
				}
				if len(clone.GetExile()) != 0 {
					t.Fatal("move changed pending clone")
				}
				clone.ResolveStack()
				if tc.name == "Safe Haven" && state == "empty" && clone.FindExiledCard(top.ID()) == nil {
					t.Fatal("clone shared target phasing")
				}
			})
		}
	}
}

func TestRONLinkedExileInvalid(t *testing.T) {
	for _, action := range []string{
		`Exile([TheTopCardOfLibrary()], [ExilesFaceDown])`,
		`Exile([TheTopCardOfLibrary], [ExilesFaceDown()])`,
		`Exile([TheTopCardOfLibrary], [ExilesFaceDown,ExilesFaceDown])`,
		`Exile([TheTopCardOfLibrary], [EntersTapped])`,
		`Exile([TheTopCardOfLibrary,Permanent(ThisPermanent)], [])`,
		`Exile([Permanent(Ref_TargetPermanent)], [])`,
		`Exile([Permanent(ThisPermanent)], [ExilesFaceDown])`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+action+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
	for _, rule := range []string{
		`Activated(Exile([TheTopCardOfLibrary], [ExilesFaceDown]),ActionList([]))`,
		`AsPermanentEnters(ThisPermanent,[Exile([TheTopCardOfLibrary], [ExilesFaceDown])])`,
		`PermanentLayerEffect(ThisPermanent,[Exile([TheTopCardOfLibrary], [ExilesFaceDown])])`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([Exile([TheTopCardOfLibrary],[ExilesFaceDown])]))])`)
	if _, err := Compile(n); err == nil {
		t.Fatal("library link without permanent source accepted")
	}
}

func TestRONLinkedExileContexts(t *testing.T) {
	for _, kind := range []string{"activation", "trigger", "delayed"} {
		for _, down := range []string{"", "ExilesFaceDown"} {
			t.Run(kind+down, func(t *testing.T) {
				body := `Exile([TheTopCardOfLibrary],[` + down + `])`
				rule := `Activated(PayMana([]),ActionList([` + body + `]))`
				if kind == "trigger" {
					rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([` + body + `]))`
				}
				if kind == "delayed" {
					rule = `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + body + `]))]))`
				}
				g, a, b := ronGame()
				card := permanentScopeFactory(t, "Artifact", rule)()
				card.SetOwner(a.PlayerID())
				first, current := mage.NewLand("First"), mage.NewLand("Current")
				first.SetOwner(a.PlayerID())
				current.SetOwner(a.PlayerID())
				a.SetLibrary([]mage.Card{first})
				p := g.PutOnBattlefield(card, a.PlayerID())
				if kind == "trigger" {
					g.PutTriggersOnStack()
				} else if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if kind == "delayed" {
					g.ResolveStack()
					if len(g.WaitingDelayedPermanentSources()) != 1 || len(g.GetExile()) != 0 {
						t.Fatal("waiting query moved card or lost source")
					}
					g.BouncePermanentToHand(p)
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
					g.PutTriggersOnStack()
				}
				a.SetLibrary([]mage.Card{current, first})
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				exiled := g.FindExiledCard(current.ID())
				if exiled == nil || exiled.ExiledBy != card.ID() || exiled.FaceDown != (down != "") || len(a.Library()) != 1 || a.Library()[0].ID() != first.ID() || len(b.Library()) != 0 || len(g.PublicReveals()) != 0 || a.LastDrawnCardID() != uuid.Nil {
					t.Fatal("wrong timing, source, player, or visibility")
				}
				if clone.FindExiledCard(current.ID()) == nil {
					t.Fatal("pending clone lost producer")
				}
			})
		}
	}
}
