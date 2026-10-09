package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLibraryTopMovePetra(t *testing.T) {
	for _, name := range []string{"Plains", "Island"} {
		for _, state := range []string{"present", "departed", "returned", "phased", "empty"} {
			t.Run(name+state, func(t *testing.T) {
				factory, err := Compile(staticBaseRecord(t, "LEG", "Petra Sphinx"))
				if err != nil {
					t.Fatal(err)
				}
				g, a, base := ronGame()
				b := &libraryNameTester{BasePlayer: base, name: name}
				g.SetPlayerAt(1, b)
				top, next := mage.NewLand("Plains"), mage.NewLand("Next")
				top.SetOwner(b.PlayerID())
				next.SetOwner(b.PlayerID())
				if state != "empty" {
					b.SetLibrary([]mage.Card{top, next})
				}
				card := factory()
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				if b.calls != 0 || len(g.PublicReveals()) != 0 || len(b.Hand()) != 0 || len(b.Graveyard()) != 0 {
					t.Fatal("announcement ran actions")
				}
				switch state {
				case "departed", "returned":
					g.DestroyPermanent(p)
					if state == "returned" {
						g.PutOnBattlefield(card, b.PlayerID())
					}
				case "phased":
					g.PhaseOut(p.ID())
				}
				g.ResolveStack()
				if b.calls != 1 {
					t.Fatal("wrong player chose name")
				}
				hand, grave, library, reveals := 0, 0, 0, 0
				if state != "empty" {
					library, reveals = 1, 1
					if name == "Plains" {
						hand = 1
					} else {
						grave = 1
					}
				}
				if len(b.Hand()) != hand || len(b.Graveyard()) != grave || len(b.Library()) != library || len(g.PublicReveals()) != reveals || len(a.Hand()) != 0 {
					t.Fatal("wrong Petra branch or player")
				}
				if hand == 1 && b.Hand()[0].ID() != top.ID() || grave == 1 && b.Graveyard()[0].ID() != top.ID() || library == 1 && b.Library()[0].ID() != next.ID() {
					t.Fatal("wrong card moved")
				}
				if b.LastDrawnCardID() != uuid.Nil {
					t.Fatal("hand move was a draw")
				}
			})
		}
	}
}

func TestRONLibraryTopMoveScopes(t *testing.T) {
	for _, action := range []string{"PutTopOfLibraryInHand", "PutTopOfLibraryInGraveyard"} {
		for _, body := range []string{action, `PlayerAction(You,` + action + `)`, `PlayerActions(You,[PlayerActions(ActionPlayer,[` + action + `])])`} {
			g, a, b := ronGame()
			for _, player := range []*mage.BasePlayer{a, b} {
				c := mage.NewLand("Top")
				c.SetOwner(player.PlayerID())
				player.AddToLibrary(c)
			}
			card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),ActionList([`+body+`]))`)()
			p := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || len(a.Library()) != 1 {
				t.Fatal("failed payment moved card")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			clone.GetPlayer(a.PlayerID()).SetLibrary(nil)
			g.ResolveStack()
			clone.ResolveStack()
			if len(a.Library()) != 0 || len(b.Library()) != 1 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 0 || len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 0 {
				t.Fatal("scope or clone crossed")
			}
		}
	}
}

func TestRONLibraryTopMoveInvalid(t *testing.T) {
	for _, action := range []string{"PutTopOfLibraryInHand", "PutTopOfLibraryInGraveyard"} {
		for _, rule := range []string{`SpellActions(ActionList([` + action + `()]))`, `SpellActions(ActionList([` + action + `(You)]))`, `SpellActions(ActionList([PlayerAction(Ref_TargetPlayer,` + action + `)]))`, `Activated(` + action + `,ActionList([]))`, `PermanentLayerEffect(ThisPermanent,[` + action + `])`, `AsPermanentEnters(ThisPermanent,[` + action + `])`} {
			n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)
			if _, err := Compile(n); err == nil {
				t.Fatalf("accepted %s", rule)
			}
		}
	}
}

func TestRONLibraryTopMoveContexts(t *testing.T) {
	for _, action := range []string{"PutTopOfLibraryInHand", "PutTopOfLibraryInGraveyard"} {
		for _, kind := range []string{"spell", "trigger", "delayed"} {
			t.Run(action+kind, func(t *testing.T) {
				g, a, b := ronGame()
				top := mage.NewLand("First")
				top.SetOwner(a.PlayerID())
				a.AddToLibrary(top)
				body := `PlayerAction(You,` + action + `)`
				typ, rule := "Instant", `SpellActions(ActionList([`+body+`]))`
				if kind == "trigger" {
					typ, rule = "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([`+body+`]))`
				}
				if kind == "delayed" {
					typ, rule = "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))`
				}
				card := permanentScopeFactory(t, typ, rule)()
				card.SetOwner(a.PlayerID())
				switch kind {
				case "spell":
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
						t.Fatal(err)
					}
				case "trigger":
					g.PutOnBattlefield(card, a.PlayerID())
					g.PutTriggersOnStack()
				case "delayed":
					p := g.PutOnBattlefield(card, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					g.PutTriggersOnStack()
				}
				if len(a.Library()) != 1 {
					t.Fatal("moved before resolution")
				}
				second := mage.NewLand("Current")
				second.SetOwner(a.PlayerID())
				a.SetLibrary([]mage.Card{second, top})
				g.ResolveStack()
				if len(a.Library()) != 1 || a.Library()[0].ID() != top.ID() || len(b.Hand()) != 0 || len(b.Graveyard()) != 0 {
					t.Fatal("move used announcement top or other player")
				}
				cards := a.Hand()
				if action == "PutTopOfLibraryInGraveyard" {
					cards = a.Graveyard()
					if kind == "spell" {
						cards = cards[:len(cards)-1]
					}
				}
				if len(cards) != 1 || cards[0].ID() != second.ID() {
					t.Fatal("resolution did not move current top")
				}
			})
		}
	}
}
