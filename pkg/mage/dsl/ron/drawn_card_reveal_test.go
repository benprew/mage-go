package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDrawnCardRevealSindbad(t *testing.T) {
	record := staticBaseRecord(t, "ARN", "Sindbad")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"land", "nonland", "empty", "replaced", "departed", "returned", "phased"} {
		t.Run(state, func(t *testing.T) {
			g, a, _ := ronGame()
			source := factory()
			source.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(source, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || p.Tapped || len(g.PublicReveals()) != 0 {
				t.Fatal("summoning sickness payment mutated state")
			}
			g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
			var drawn mage.Card = mage.NewArtifact("Nonland", "")
			if state == "land" {
				drawn = mage.NewLand("Forest")
			}
			drawn.SetOwner(a.PlayerID())
			if state != "empty" && state != "replaced" {
				a.AddToLibrary(drawn)
			}
			if state == "replaced" {
				replacement := g.PutOnBattlefield(mage.NewArtifact("Replacement", ""), a.PlayerID())
				g.AddEmptyLibraryDrawReplacement(replacement.ID(), a.PlayerID(), func(game *mage.Game, _id uuid.UUID) {})
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if len(a.Hand()) != 0 || len(g.PublicReveals()) != 0 {
				t.Fatal("draw or reveal ran at announcement")
			}
			switch state {
			case "departed", "returned":
				g.DestroyPermanent(p)
				if state == "returned" {
					g.PutOnBattlefield(p.Card, a.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				player := game.GetPlayer(a.PlayerID())
				if state == "empty" || state == "replaced" {
					if len(game.PublicReveals()) != 0 || len(player.Graveyard()) > 0 || len(player.Hand()) != 0 {
						t.Fatal("failed draw reused a card")
					}
					continue
				}
				if len(game.PublicReveals()) != 1 || game.PublicReveals()[0].CardID != drawn.ID() {
					t.Fatal("wrong draw revealed")
				}
				if state == "land" {
					if len(player.Hand()) != 1 {
						t.Fatal("land discarded")
					}
				} else if len(player.Hand()) != 0 || len(player.Graveyard()) == 0 || player.Graveyard()[len(player.Graveyard())-1].ID() != drawn.ID() {
					t.Fatal("nonland not discarded")
				}
			}
		})
	}
}

func TestRONDrawnCardRevealInvalid(t *testing.T) {
	for _, actions := range []string{
		`RevealTheCardDrawnThisWay`,
		`DiscardTheCardDrawnThisWay`,
		`RevealTheCardDrawnThisWay,DrawACard`,
		`DrawACard,RevealTheCardDrawnThisWay()`,
		`DrawACard,PlayerAction(You,RevealTheCardDrawnThisWay)`,
		`PlayerActions(You,[DrawACard]),RevealTheCardDrawnThisWay`,
		`DrawACard,If(ACardWasRevealedThisWay(IsCardtype(Land)),[RevealTheCardDrawnThisWay]),If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(1))])`,
		`DrawACard,DiscardTheCardDrawnThisWay()`,
		`DrawNumberCards(Integer(2)),RevealTheCardDrawnThisWay`,
		`MayAction(DrawACard),RevealTheCardDrawnThisWay`,
		`DrawACard,If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(1))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+actions+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", actions)
		}
	}
}

func TestRONDrawnCardRevealContexts(t *testing.T) {
	for _, kind := range []string{"spell", "trigger", "delayed"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b := ronGame()
			land := mage.NewLand("Forest")
			land.SetOwner(b.PlayerID())
			b.AddToLibrary(land)
			actions := `PlayerAction(Ref_TargetPlayer,DrawACard),RevealTheCardDrawnThisWay,If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(2))]),PlayerAction(Ref_TargetPlayer,DiscardTheCardDrawnThisWay)`
			typ, rule := "Instant", `SpellActions(Targeted([TargetPlayer(Opponent)],ActionList([`+actions+`])))`
			if kind == "trigger" {
				typ, rule = "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),Targeted([TargetPlayer(Opponent)],ActionList([`+actions+`])))`
			}
			if kind == "delayed" {
				typ, rule = "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(Opponent)],ActionList([PlayerAction(Ref_TargetPlayer,DrawACard),RevealTheCardDrawnThisWay,CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(2))]),PlayerAction(Ref_TargetPlayer,DiscardTheCardDrawnThisWay)]))])))`
			}
			card := permanentScopeFactory(t, typ, rule)()
			card.SetOwner(a.PlayerID())
			before := a.Life()
			switch kind {
			case "spell":
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
			case "trigger":
				g.PutOnBattlefield(card, a.PlayerID())
				g.PutTriggersOnStack()
			case "delayed":
				p := g.PutOnBattlefield(card, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			}
			if len(g.PublicReveals()) != 0 {
				t.Fatal("reveal ran before resolution")
			}
			g.ResolveStack()
			if kind == "delayed" {
				if a.Life() != before || len(b.Hand()) != 1 {
					t.Fatal("delay ran early")
				}
				b.RemoveFromHand(land.ID())
				b.AddToHand(land)
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					game.PutTriggersOnStack()
					game.ResolveStack()
					if game.GetPlayer(a.PlayerID()).Life() != before+2 || len(game.GetPlayer(b.PlayerID()).Hand()) != 1 {
						t.Fatal("delay lost facts or followed returned visit")
					}
				}
			} else if a.Life() != before+2 || len(b.Hand()) != 0 || len(b.Graveyard()) != 1 {
				t.Fatal("explicit player draw or discard failed")
			}
			if len(g.PublicReveals()) != 1 || g.PublicReveals()[0].CardID != land.ID() {
				t.Fatal("context lost result")
			}
		})
	}
}

func TestRONDrawnCardRevealDistinctProducers(t *testing.T) {
	g, a, _ := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([DrawACard,RevealTheCardDrawnThisWay,DrawACard,RevealTheCardDrawnThisWay,Unless(ACardWasRevealedThisWay(IsCardtype(Land)),[DiscardTheCardDrawnThisWay])]))`)
	for range 2 {
		card := factory()
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	cards := []mage.Card{mage.NewLand("First"), mage.NewArtifact("Second", ""), mage.NewArtifact("Third", ""), mage.NewLand("Fourth")}
	for _, card := range cards {
		card.SetOwner(a.PlayerID())
		a.AddToLibrary(card)
	}
	clone := g.Clone()
	clone.GetPlayer(a.PlayerID()).SetLibrary([]mage.Card{cards[3], cards[2], cards[1], cards[0]})
	g.ResolveStack()
	clone.ResolveStack()
	if len(a.Hand()) != 3 || len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != cards[1].ID() {
		t.Fatal("producer shadowing failed")
	}
	if len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 1 || clone.GetPlayer(a.PlayerID()).Graveyard()[0].ID() != cards[2].ID() {
		t.Fatal("clone shared results")
	}
	for i, reveal := range g.PublicReveals() {
		if reveal.CardID != cards[i].ID() {
			t.Fatal("factory read another result")
		}
	}
}
