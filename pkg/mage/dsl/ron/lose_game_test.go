package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLoseGameLich(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		for _, paid := range []bool{true, false} {
			t.Run(set+map[bool]string{true: "/paid", false: "/unpaid"}[paid], func(t *testing.T) {
				record := staticBaseRecord(t, set, "Lich")
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				a := &requiredNumberPlayer{&optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}}}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				p := g.PutOnBattlefield(factory(), a.PlayerID())
				if paid {
					for range 2 {
						g.PutOnBattlefield(mage.NewCreature("Payment", "", 1, 1), a.PlayerID())
					}
				}
				g.DealDamageToPlayer(a, 2, p.ID())
				g.PutTriggersOnStack()
				if !a.IsAlive() || g.StackSize() != 1 {
					t.Fatal("loss before resolution")
				}
				clone := g.Clone()
				g.ResolveStack()
				if a.IsAlive() != paid || !b.IsAlive() || a.Life() != -2 || len(a.Graveyard()) != map[bool]int{true: 2, false: 0}[paid] {
					t.Fatal("payment branch or explicit loss")
				}
				if !clone.GetPlayer(a.PlayerID()).IsAlive() {
					t.Fatal("loss shared with clone")
				}
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).IsAlive() != paid {
					t.Fatal("clone lost payment branch")
				}
				if _, err := Compile(staticBaseRecord(t, set, "Lich")); err != nil {
					t.Fatalf("full record: %v", err)
				}
			})
		}
	}
}

func TestRONLoseGameScopes(t *testing.T) {
	for _, body := range []string{`LoseTheGame`, `PlayerAction(Ref_TargetPlayer,LoseTheGame)`, `PlayerActions(Ref_TargetPlayer,[LoseTheGame])`} {
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([`+body+`])))`)
		a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		wantA := body != "LoseTheGame"
		if a.IsAlive() != wantA || b.IsAlive() == wantA || a.Life() != 20 || b.Life() != 20 {
			t.Fatal("wrong player scope")
		}
	}
}

func TestRONLoseGameInvalid(t *testing.T) {
	for _, rule := range []string{
		`Activated(PayMana([]),ActionList([LoseTheGame()]))`,
		`Activated(PayMana([]),ActionList([LoseTheGame(You)]))`,
		`Activated(PayMana([]),ActionList([PlayerAction(ActionPlayer,LoseTheGame)]))`,
		`Activated(PayMana([]),ActionList([PlayerAction(Trigger_ThatPlayer,LoseTheGame)]))`,
		`AsPermanentEnters(ThisPermanent,[LoseTheGame])`,
		`ReplaceWouldGainLife(PlayerWouldGainLife(You),[LoseTheGame])`,
	} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+rule+`])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONLoseGameQueuedIdentity(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "control", "copy"} {
		t.Run(state, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([LoseTheGame]))`)
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			queueRONUpkeep(g, a)
			switch state {
			case "departed", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveTopOfStack()
			if a.IsAlive() != (state == "copy") || b.IsAlive() != (state != "copy") {
				t.Fatal("queued controller or copied controller")
			}
			if !clone.GetPlayer(a.PlayerID()).IsAlive() || !clone.GetPlayer(b.PlayerID()).IsAlive() {
				t.Fatal("loss shared")
			}
			clone.ResolveTopOfStack()
			if clone.GetPlayer(a.PlayerID()).IsAlive() != (state == "copy") || clone.GetPlayer(b.PlayerID()).IsAlive() != (state != "copy") {
				t.Fatal("cloned queued controller")
			}
		})
	}
}

func TestRONLoseGameWaiting(t *testing.T) {
	g, a, b := ronGame()
	factory := waitingCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(ControllerOfPermanent(ThisPermanent),LoseTheGame)]))]))`)
	card := factory()
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, b.PlayerID())
	for range 3 {
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || !a.IsAlive() || !b.IsAlive() {
			t.Fatal("query changed state or lost exact source")
		}
	}
	clone := g.Clone()
	for _, game := range []*mage.Game{clone, g} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).IsAlive() || !game.GetPlayer(b.PlayerID()).IsAlive() {
			t.Fatal("loss used returned source")
		}
		if game == clone && !a.IsAlive() {
			t.Fatal("clone changed original")
		}
	}
}

func TestRONLoseGameSpell(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerActions(Ref_TargetPlayer,[LoseTheGame])])))`)
	for range 2 {
		g, a, b := ronGame()
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		g.SetStep(core.PrecombatMain)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		if !a.IsAlive() || !b.IsAlive() {
			t.Fatal("announcement caused loss")
		}
		g.ResolveStack()
		if !a.IsAlive() || b.IsAlive() || a.Life() != 20 || b.Life() != 20 {
			t.Fatal("spell player binding")
		}
	}
}
