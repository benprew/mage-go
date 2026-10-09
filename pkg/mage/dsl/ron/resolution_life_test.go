package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONResolutionLifeWand(t *testing.T) {
	factory, err := Compile(staticBaseRecord(t, "DRK", "Wand of Ith"))
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"land-pay", "land-decline", "nonland-pay", "nonland-decline", "short", "zero-pay", "zero-decline", "empty", "departed", "returned", "phased", "control", "copy", "self"} {
		t.Run(state, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: !strings.Contains(state, "decline")}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast succeeded")
			}
			a.ManaPool().Add(core.Colorless, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil || p.Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("complete record entry or cost differs")
			}
			payer := b
			if state == "self" {
				payer = a
			}
			var held mage.Card = mage.NewArtifact("Nonland", "{X}{X}{3}{U}")
			amount := 4
			if strings.HasPrefix(state, "land") {
				held = mage.NewLand("Land")
				amount = 1
			}
			if strings.HasPrefix(state, "zero") {
				held = mage.NewArtifact("Zero", "")
				amount = 0
			}
			held.SetOwner(payer.PlayerID())
			if state != "empty" {
				payer.AddToHand(held)
			}
			if state == "short" {
				payer.SetLife(3)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{payer.PlayerID()}); err == nil || p.Tapped || payer.offers != 0 || len(g.PublicReveals()) != 0 {
				t.Fatal("failed activation changed state")
			}
			a.ManaPool().Add(core.Colorless, 3)
			g.SetActivePlayerIndex(1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{payer.PlayerID()}); err == nil || p.Tapped || a.ManaPool().TotalMana() != 3 {
				t.Fatal("wrong-turn activation spent resources")
			}
			g.SetActivePlayerIndex(0)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{payer.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if !p.Tapped || a.ManaPool().TotalMana() != 0 || payer.offers != 0 || len(g.PublicReveals()) != 0 {
				t.Fatal("wrong payment or resolution timing")
			}
			switch state {
			case "departed", "returned":
				g.DestroyPermanent(p)
				if state == "returned" {
					a.RemoveFromGraveyard(card.ID())
					g.PutOnBattlefield(p.Card, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("ability copy failed")
				}
			}
			before := payer.Life()
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				if game == clone {
					for i, original := range []*ronPaymentPlayer{a, b} {
						game.SetPlayerAt(i, &ronPaymentPlayer{BasePlayer: game.GetPlayer(original.PlayerID()).(*mage.SearchPlayer).BasePlayer, accept: original.accept})
					}
				}
				game.ResolveStack()
				player := game.GetPlayer(payer.PlayerID())
				count := 1
				if state == "copy" {
					count = 2
				}
				if state == "empty" {
					if len(game.PublicReveals()) != 0 || player.Life() != before || len(player.Graveyard()) != 0 {
						t.Fatal("empty reveal reused a result")
					}
					continue
				}
				if len(game.PublicReveals()) != count || game.PublicReveals()[0].CardID != held.ID() {
					t.Fatal("wrong public reveal")
				}
				paid := payer.accept && state != "short"
				if paid {
					if player.Life() != before-count*amount || len(player.Hand()) != 1 || len(player.Graveyard()) != 0 {
						t.Fatal("paid reveal discarded or spent wrong life")
					}
				} else if player.Life() != before || len(player.Hand()) != 0 || len(player.Graveyard()) != 1 || player.Graveyard()[0].ID() != held.ID() {
					t.Fatal("unpaid reveal lost exact card")
				}
				if payer != a && game.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("source controller paid target cost")
				}
			}
			wantOffers := 1
			if state == "empty" || state == "short" {
				wantOffers = 0
			}
			if state == "copy" {
				wantOffers = 2
			}
			if payer.offers != wantOffers {
				t.Fatalf("offers=%d want=%d", payer.offers, wantOffers)
			}
		})
	}
}

func TestRONResolutionLifeInvalid(t *testing.T) {
	for _, rule := range []string{
		`Activated(PayLife(ManaValueOfTheCardRevealedThisWay),ActionList([DrawACard]))`,
		`Activated(PayMana([]),ActionList([MayCost(PayLife(ManaValueOfTheCardRevealedThisWay)),Unless(CostWasPaid,[DrawACard])]))`,
		`Activated(PayMana([]),ActionList([MayCost(PayLife(ManaValueOfTheCardRevealedThisWay)),Unless(CostWasPaid,[DrawACard]),PlayerAction(You,RevealACardFromHandAtRandom)]))`,
		`Activated(PayMana([]),ActionList([PlayerAction(You,RevealACardFromHandAtRandom),MayCost(PayLife(ManaValueOfTheCardRevealedThisWay())),Unless(CostWasPaid,[DrawACard])]))`,
		`Activated(PayMana([]),ActionList([PlayerAction(You,RevealACardFromHandAtRandom),MayCost(PayLife(ManaValueOfTheCardRevealedThisWay,ManaValueOfTheCardRevealedThisWay)),Unless(CostWasPaid,[DrawACard])]))`,
		`Activated(PayMana([]),ActionList([PlayerAction(You,RevealACardFromHandAtRandom),MustCost(PayLife(ManaValueOfTheCardRevealedThisWay)),Unless(CostWasPaid,[DrawACard])]))`,
		`Activated(PayMana([]),ActionList([PlayerAction(You,RevealACardFromHandAtRandom),MayCost(Or([PayLife(ManaValueOfTheCardRevealedThisWay),PayMana([])])),Unless(CostWasPaid,[DrawACard])]))`,
		`Activated(PayMana([]),ActionList([PlayerAction(You,RevealACardFromHandAtRandom),MayCost(PayLife(ManaValueOfTheCardRevealedThisWay)),Unless(CostWasPaid,[])]))`,
		`AsPermanentEnters(ThisPermanent,[PayLife(ManaValueOfTheCardRevealedThisWay)])`,
		`Activated(PayMana([]),ActionList([DrawACard,MayCost(PayLife(ManaValueOfTheCardRevealedThisWay)),Unless(CostWasPaid,[DrawACard])]))`,
		`Activated(PayMana([]),ActionList([MayCost(PayLife(Integer(1))),Unless(CostWasPaid,[PlayerAction(You,RevealACardFromHandAtRandom)]),GainLife(ManaValueOfTheCardRevealedThisWay)]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONResolutionLifeContexts(t *testing.T) {
	for _, kind := range []string{"spell", "trigger", "delayed"} {
		for _, producer := range []string{"hand", "drawn", "library"} {
			t.Run(kind+"/"+producer, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				g.SetStep(core.PrecombatMain)
				held := mage.NewArtifact("Held", "{X}{2}{U}")
				held.SetOwner(b.PlayerID())
				production := `PlayerAction(Ref_TargetPlayer,RevealACardFromHandAtRandom)`
				if producer == "hand" {
					b.AddToHand(held)
				} else {
					b.AddToLibrary(held)
				}
				if producer == "drawn" {
					production = `PlayerAction(Ref_TargetPlayer,DrawACard),RevealTheCardDrawnThisWay`
				}
				if producer == "library" {
					production = `PlayerActions(Ref_TargetPlayer,[RevealTopCardOfLibrary])`
				}
				payment := `PlayerMayCost(Ref_TargetPlayer,PayLife(ManaValueOfTheCardRevealedThisWay)),If(CostWasPaid,[GainLife(Plus(ManaValueOfTheCardRevealedThisWay,Integer(1)))])`
				actions := production + `,` + payment
				typ, rule := "Instant", `SpellActions(Targeted([TargetPlayer(Opponent)],ActionList([`+actions+`])))`
				if kind == "trigger" {
					typ, rule = "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),Targeted([TargetPlayer(Opponent)],ActionList([`+actions+`])))`
				}
				if kind == "delayed" {
					typ, rule = "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(Opponent)],ActionList([`+production+`,CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+payment+`]))])))`
				}
				card := permanentScopeFactory(t, typ, rule)()
				card.SetOwner(a.PlayerID())
				var p *mage.Permanent
				switch kind {
				case "spell":
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
						t.Fatal(err)
					}
				case "trigger":
					p = g.PutOnBattlefield(card, a.PlayerID())
					g.PutTriggersOnStack()
				case "delayed":
					p = g.PutOnBattlefield(card, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
						t.Fatal(err)
					}
				}
				if b.offers != 0 || len(g.PublicReveals()) != 0 {
					t.Fatal("payment or reveal ran before resolution")
				}
				g.ResolveStack()
				if kind == "delayed" {
					if b.offers != 0 || a.Life() != 20 || b.Life() != 20 {
						t.Fatal("delayed payment ran early")
					}
					if producer == "library" {
						b.SetLibrary(nil)
					} else {
						b.RemoveFromHand(held.ID())
						b.AddToHand(held)
					}
					g.DestroyPermanent(p)
					clone := g.Clone()
					clone.SetPlayerAt(1, &ronPaymentPlayer{BasePlayer: clone.GetPlayer(b.PlayerID()).(*mage.SearchPlayer).BasePlayer, accept: false})
					for _, game := range []*mage.Game{g, clone} {
						game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						game.PutTriggersOnStack()
						game.ResolveStack()
					}
					if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
						t.Fatal("clone payment changed original outcomes")
					}
				}
				if a.Life() != 24 || b.Life() != 17 || b.offers != 1 || len(g.PublicReveals()) != 1 {
					t.Fatalf("context lost saved value: A=%d B=%d offers=%d", a.Life(), b.Life(), b.offers)
				}
			})
		}
	}
}

func TestRONResolutionLifeDistinctProducers(t *testing.T) {
	g, a, b := ronGame()
	bpayer := &ronPaymentPlayer{BasePlayer: b, accept: true}
	g.SetPlayerAt(1, bpayer)
	first := mage.NewArtifact("First", "{2}")
	first.SetOwner(a.PlayerID())
	a.AddToHand(first)
	second := mage.NewArtifact("Second", "{7}")
	second.SetOwner(b.PlayerID())
	b.AddToHand(second)
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(Opponent)],ActionList([PlayerAction(You,RevealACardFromHandAtRandom),GainLife(ManaValueOfTheCardRevealedThisWay),PlayerAction(Ref_TargetPlayer,RevealACardFromHandAtRandom),PlayerMayCost(Ref_TargetPlayer,PayLife(ManaValueOfTheCardRevealedThisWay)),Unless(CostWasntPaid,[GainLife(ManaValueOfTheCardRevealedThisWay)])])))`)
	for range 2 {
		card := factory()
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if a.Life() != 38 || b.Life() != 6 || bpayer.offers != 2 || len(g.PublicReveals()) != 4 {
		t.Fatal("lexical producers or factories shared result state")
	}
}
