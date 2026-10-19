package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONExplicitCardMovementInvalid(t *testing.T) {
	reveal := `PlayerAction(You,RevealACardFromHandAtRandom)`
	for _, body := range []string{
		`PutCardFromHandIntoPlayersHand(TheCardRevealedFromHandThisWay,You)`,
		reveal + `,PutCardFromHandIntoPlayersHand`,
		reveal + `,PutCardFromHandIntoPlayersHand(TheCardRevealedFromHandThisWay)`,
		reveal + `,PutCardFromHandIntoPlayersHand(TheCardRevealedFromHandThisWay(),You)`,
		reveal + `,PutCardFromHandIntoPlayersHand(ThisPermanent,You)`,
		reveal + `,PutCardFromHandIntoPlayersHand(TheCardRevealedFromHandThisWay,ActionPlayer)`,
		`If(PermanentPassesFilter(ThisPermanent,IsTapped),[` + reveal + `]),PutCardFromHandIntoPlayersHand(TheCardRevealedFromHandThisWay,You)`,
		`PutCardFromAnywhereIntoPlayersGraveyard(ThisCard)`,
		`PutCardFromAnywhereIntoPlayersGraveyard(ThisCard(),You)`,
		`PutCardFromAnywhereIntoPlayersGraveyard(ThisPermanent,You)`,
		`PutCardFromAnywhereIntoPlayersGraveyard(ThisCard,Ref_TargetPlayer)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid movement", body)
		}
	}
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[PutCardFromAnywhereIntoPlayersGraveyard(ThisCard,You)])`,
		`PermanentLayerEffect(ThisPermanent,[PutCardFromAnywhereIntoPlayersGraveyard(ThisCard,You)])`,
		`Activated(PutCardFromAnywhereIntoPlayersGraveyard(ThisCard,You),ActionList([]))`,
		`ReplaceWouldDraw(PlayerWouldDrawACard(You),[PutCardFromAnywhereIntoPlayersGraveyard(ThisCard,You)])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid context", rule)
		}
	}
}

func TestRONExplicitCardMovementInstancesAndDelay(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		body := `PlayerAction(Ref_TargetPlayer,RevealACardFromHandAtRandom),ExchangeOwnershipOfTwoCards(TheCardRevealedFromHandThisWay,ThisPermanent),PutCardFromHandIntoPlayersHand(TheCardRevealedFromHandThisWay,You),PutCardFromAnywhereIntoPlayersGraveyard(ThisCard,Ref_TargetPlayer)`
		if delayed {
			body = `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + body + `]))`
		}
		factory := permanentScopeFactory(t, "Artifact", `Activated(SacrificePermanent(ThisPermanent),Targeted([TargetPlayer(Opponent)],ActionList([`+body+`])))`)
		a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
		g, _ := mage.NewGameWithAnte(a, b, nil, nil)
		one, two := factory(), factory()
		one.SetOwner(a.PlayerID())
		two.SetOwner(b.PlayerID())
		p, q := g.PutOnBattlefield(one, a.PlayerID()), g.PutOnBattlefield(two, b.PlayerID())
		x, y := mage.NewArtifact("x", ""), mage.NewArtifact("y", "")
		x.SetOwner(a.PlayerID())
		y.SetOwner(b.PlayerID())
		a.AddToHand(x)
		b.AddToHand(y)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if err := g.ActivateAbilityByIndex(b.PlayerID(), q.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if delayed {
			for range 2 {
				g.WaitingDelayedGraveyardSources()
				g.WaitingDelayedPermanentSources()
			}
			if len(g.PublicReveals()) != 0 || g.FindCardAnywhere(one.ID()).Owner() != a.PlayerID() {
				t.Fatal("waiting query executed movement")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
		}
		if g.FindCardAnywhere(one.ID()).Owner() != b.PlayerID() || g.FindCardAnywhere(two.ID()).Owner() != a.PlayerID() || len(a.Graveyard()) != 1 || len(b.Graveyard()) != 1 || len(a.Hand()) != 1 || len(b.Hand()) != 1 {
			t.Fatal("instances lost separate bindings or destinations")
		}
	}
}

func TestRONExplicitCardMovementTempest(t *testing.T) {
	for _, state := range []string{"decline", "paid", "cannot-pay", "empty", "returned", "exiled", "hand", "library", "battlefield", "phased", "copy", "stolen", "clone", "sick", "tapped", "bad-target"} {
		t.Run(state, func(t *testing.T) {
			factory, err := Compile(staticBaseRecord(t, "LEG", "Tempest Efreet"))
			if err != nil {
				t.Fatal(err)
			}
			a := mage.NewBasePlayer("A")
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: state == "paid" || state == "cannot-pay"}
			g, err := mage.NewGameWithAnte(a, b, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			g.SetTurn(2)
			g.SetStep(core.PrecombatMain)
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(core.Red, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil {
				t.Fatal("paid cast did not enter")
			}
			if state != "sick" {
				p.RevokeBaseAttr(core.AttrSummonSick)
			}
			if state == "tapped" {
				p.Tapped = true
			}
			payer := a.PlayerID()
			opponent := b.PlayerID()
			if state == "stolen" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				p = g.FindPermanent(p.ID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				payer, opponent = b.PlayerID(), a.PlayerID()
			}
			revealed := mage.NewArtifact("revealed", "")
			revealed.SetOwner(opponent)
			if state != "empty" {
				g.GetPlayer(opponent).AddToHand(revealed)
			}
			if state == "cannot-pay" {
				b.SetLife(9)
			}
			target := opponent
			if state == "bad-target" {
				target = payer
			}
			err = g.ActivateAbilityByIndex(payer, p.ID(), 0, []uuid.UUID{target})
			if state == "sick" || state == "tapped" || state == "bad-target" {
				if err == nil || g.FindPermanent(p.ID()) == nil || g.StackSize() != 0 {
					t.Fatal("illegal activation paid a cost")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()) != nil || len(a.Graveyard()) != 1 || len(g.PublicReveals()) != 0 {
				t.Fatal("source payment or timing incorrect")
			}
			switch state {
			case "returned", "exiled", "hand", "library", "battlefield", "phased":
				moved, ok := a.RemoveFromGraveyard(card.ID())
				if !ok {
					t.Fatal("source missing")
				}
				switch state {
				case "returned":
					a.AddToGraveyard(moved)
				case "exiled":
					g.ExileCard(moved, uuid.Nil)
				case "hand":
					a.AddToHand(moved)
				case "library":
					a.SetLibrary([]mage.Card{moved})
				case "battlefield", "phased":
					q := g.PutOnBattlefield(moved, a.PlayerID())
					if state == "phased" {
						g.PhaseOut(q.ID())
					}
				}
			}
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), payer, false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			g.ResolveStack()
			if state == "paid" {
				if b.Life() != 10 || len(g.PublicReveals()) != 0 || len(b.Hand()) != 1 {
					t.Fatal("paid branch performed unpaid actions")
				}
				return
			}
			if state == "phased" {
				if g.FindPermanentIncludingPhased(card.ID()) == nil {
					t.Fatal("phased source moved")
				}
				return
			}
			if g.CardZone(card.ID()) != core.ZoneGraveyard {
				t.Fatal("source did not move from anywhere")
			}
			if state != "empty" && len(g.PublicReveals()) == 0 {
				t.Fatal("opponent did not reveal")
			}
			swapped := state == "decline" || state == "cannot-pay" || state == "clone" || state == "copy"
			wantOwner := a.PlayerID()
			if swapped {
				wantOwner = opponent
			}
			if g.FindCardAnywhere(card.ID()).Owner() != wantOwner {
				t.Fatal("incorrect source owner")
			}
			if len(g.GetPlayer(wantOwner).Graveyard()) != 1 {
				t.Fatal("source is in wrong player's graveyard")
			}
			if state != "empty" {
				wantHandOwner := opponent
				if swapped {
					wantHandOwner = payer
				}
				if len(g.GetPlayer(wantHandOwner).Hand()) != 1 || g.FindCardAnywhere(revealed.ID()).Owner() != wantHandOwner {
					t.Fatal("revealed hand card moved to wrong player")
				}
			}
			if len(clone.PublicReveals()) != 0 {
				t.Fatal("clone shares reveals")
			}
			clone.ResolveStack()
			if clone.FindCardAnywhere(card.ID()).Owner() != wantOwner || clone.CardZone(card.ID()) != core.ZoneGraveyard {
				t.Fatal("clone lost bindings")
			}
		})
	}
}
