package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLampDrawSelection(t *testing.T) {
	for _, set := range []string{"ARN", "3ED"} {
		t.Run(set, func(t *testing.T) {
			factory, err := Compile(staticBaseRecord(t, set, "Aladdin's Lamp"))
			if err != nil {
				t.Fatal(err)
			}
			if err = mage.ValidateCardDefinition("Aladdin's Lamp", factory()); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			g.SetTurn(3)
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0) == nil {
				t.Fatal("unpaid cast")
			}
			a.ManaPool().Add(core.Colorless, 10)
			if err = g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			for _, name := range []string{"First", "Second", "Third", "Tail"} {
				a.AddToLibrary(mage.NewLand(name))
			}
			a.ManaPool().Add(core.Colorless, 3)
			if g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil) == nil || p.Tapped || a.ManaPool().TotalMana() != 3 {
				t.Fatal("zero X or rollback")
			}
			g.SetXValue(3)
			if err = g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if len(g.LibraryLooks(a.PlayerID())) != 0 {
				t.Fatal("look during announcement")
			}
			g.SetXValue(99)
			g.DestroyPermanent(p)
			g.ResolveStack()
			clone := g.Clone()
			g.PlayerDrawCard(a)
			if len(a.Hand()) != 1 || len(a.Library()) != 3 || len(g.LibraryLooks(a.PlayerID())) != 3 || a.Library()[0].Name() != "Tail" {
				t.Fatal("replacement selection, draw or saved X")
			}
			if len(g.LibraryLooks(b.PlayerID())) != 0 || len(g.PublicReveals()) != 0 {
				t.Fatal("private cards revealed")
			}
			clone.PlayerDrawCard(clone.GetPlayer(a.PlayerID()))
			if len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(a.Hand()) != 1 {
				t.Fatal("clone isolation")
			}
			g.PlayerDrawCard(a)
			if len(a.Hand()) != 2 || len(g.LibraryLooks(a.PlayerID())) != 3 {
				t.Fatal("replacement used twice")
			}
		})
	}
}

func TestRONLampDrawSelectionInvalid(t *testing.T) {
	valid := `ActivatedWithModifiers(PayManaX([ManaCostX],ValueX),ActionList([LookAtTheTopNumberCardsOfLibrary(ValueX,[PutAGenericCardOnTopOfLibrary,PutTheRemainingCardsOnTheBottomOfLibraryInARandomOrder])]),XCantBeZero)`
	for _, rule := range []string{
		strings.Replace(valid, "XCantBeZero)", "XCantBeZero())", 1),
		strings.Replace(valid, "XCantBeZero)", "And([XCantBeZero,XCantBeZero]))", 1),
		strings.Replace(valid, "PayManaX([ManaCostX],ValueX)", "PayMana([])", 1),
		strings.Replace(valid, "ValueX,[", "Integer(-1),[", 1),
		strings.Replace(valid, "ValueX,[", "Ref_TargetPlayer,[", 1),
		strings.Replace(valid, "PutAGenericCardOnTopOfLibrary,", "", 1),
		strings.Replace(valid, "PutAGenericCardOnTopOfLibrary,", "PutAGenericCardOnTopOfLibrary(),", 1),
		strings.Replace(valid, "PutTheRemainingCardsOnTheBottomOfLibraryInARandomOrder", "LeaveRemainingCardsOnTopOfLibraryInSameOrder", 1),
		`AsPermanentEnters(ThisPermanent,[LookAtTheTopNumberCardsOfLibrary(Integer(2),[PutAGenericCardOnTopOfLibrary,PutTheRemainingCardsOnTheBottomOfLibraryInARandomOrder])])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration", rule)
		}
	}
}

func TestRONLampDrawSelectionLifecycle(t *testing.T) {
	for _, state := range []string{"phase-before", "phase-after", "return", "control", "cleanup", "copy", "two-sources", "two-players", "empty", "short", "before-resolution", "tapped", "underpaid"} {
		t.Run(state, func(t *testing.T) {
			factory, err := Compile(staticBaseRecord(t, "ARN", "Aladdin's Lamp"))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			g.SetTurn(3)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			count := 8
			if state == "empty" {
				count = 0
			}
			if state == "short" {
				count = 1
			}
			for range count {
				a.AddToLibrary(mage.NewLand("A"))
				b.AddToLibrary(mage.NewLand("B"))
			}
			mana := 3
			if state == "underpaid" {
				mana = 2
			}
			a.ManaPool().Add(core.Colorless, mana)
			g.SetXValue(3)
			if err = g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				if state != "underpaid" {
					t.Fatal(err)
				}
				if p.Tapped || a.ManaPool().TotalMana() != 2 || g.StackSize() != 0 {
					t.Fatal("underpayment changed state")
				}
				return
			}
			if state == "tapped" && g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil) == nil {
				t.Fatal("tapped source activated")
			}
			if state == "before-resolution" {
				g.PlayerDrawCard(a)
				if len(g.LibraryLooks(a.PlayerID())) != 0 || len(a.Hand()) != 1 {
					t.Fatal("early replacement")
				}
			}
			want := min(3, count)
			switch state {
			case "phase-before":
				g.PhaseOut(p.ID())
			case "return":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				want = 6
			case "two-sources", "two-players":
				player := a
				if state == "two-players" {
					player = b
				}
				second := g.PutOnBattlefield(factory(), player.PlayerID())
				player.ManaPool().Add(core.Colorless, 2)
				g.SetXValue(2)
				if err = g.ActivateAbilityByIndex(player.PlayerID(), second.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if state == "two-sources" {
					want = 5
				}
			}
			g.ResolveStack()
			if state == "phase-after" {
				g.PhaseOut(p.ID())
			}
			if state == "cleanup" {
				g.RunStepWithPriority(core.Cleanup)
				want = 0
			}
			g.PlayerDrawCard(b)
			if state == "two-players" {
				if len(g.LibraryLooks(b.PlayerID())) != 2 {
					t.Fatal("second instance lost player")
				}
			} else if len(g.LibraryLooks(b.PlayerID())) != 0 {
				t.Fatal("wrong drawing player")
			}
			g.PlayerDrawCard(a)
			if len(g.LibraryLooks(a.PlayerID())) != want {
				t.Fatal("wrong look count", len(g.LibraryLooks(a.PlayerID())), want)
			}
			expectedHand := 1
			if state == "empty" {
				expectedHand = 0
			}
			if state == "before-resolution" {
				expectedHand = 2
			}
			if len(a.Hand()) != expectedHand || len(a.Library()) != count-expectedHand {
				t.Fatal("draw or library count", len(a.Hand()), len(a.Library()))
			}
			if g.StackSize() != 0 {
				t.Fatal("replacement used stack")
			}
		})
	}
}

func TestRONLampDrawSelectionWaitingAndScope(t *testing.T) {
	action := `LookAtTheTopNumberCardsOfLibrary(PowerOfPermanent(ThisPermanent),[PutAGenericCardOnTopOfLibrary,PutTheRemainingCardsOnTheBottomOfLibraryInARandomOrder])`
	factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`]))]))`)
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	for range 5 {
		a.AddToLibrary(mage.NewLand("A"))
	}
	saved := g.CaptureDamageSource(p.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.BouncePermanentToHand(p)
	for range 2 {
		sources := g.WaitingDelayedPermanentSources()
		if len(sources) != 1 || !sources[0].SameObject(saved) || len(g.LibraryLooks(a.PlayerID())) != 0 {
			t.Fatal("waiting query executed or lost source")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if len(g.LibraryLooks(a.PlayerID())) != 2 || len(a.Library()) != 5 || len(a.Hand()) != 1 {
		t.Fatal("delayed LKI or pure library ordering")
	}
}

func TestRONLampDrawSelectionInventory(t *testing.T) {
	record := staticBaseRecord(t, "ARN", "Aladdin's Lamp")
	entries := make(map[string]*InventorySignature)
	walkInventory(record, "Aladdin's Lamp", "card", "card", "card", true, entries)
	for name, role := range map[string]string{"LookAtTheTopNumberCardsOfLibrary": "action", "ValueX": "value", "PutAGenericCardOnTopOfLibrary": "library_ordering_operation", "PutTheRemainingCardsOnTheBottomOfLibraryInARandomOrder": "library_ordering_operation", "XCantBeZero": "activation_modifier"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing semantic role", name, role)
		}
	}
}

func TestRONLampDrawSelectionPlayerScope(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,LookAtTheTopNumberCardsOfLibrary(Integer(2),[PutAGenericCardOnTopOfLibrary,PutTheRemainingCardsOnTheBottomOfLibraryInARandomOrder]))])))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	for range 3 {
		a.AddToLibrary(mage.NewLand("A"))
		b.AddToLibrary(mage.NewLand("B"))
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.LibraryLooks(a.PlayerID())) != 0 || len(g.LibraryLooks(b.PlayerID())) != 2 || len(b.Library()) != 3 || len(b.Hand()) != 0 {
		t.Fatal("fixed count or player scope")
	}
}
