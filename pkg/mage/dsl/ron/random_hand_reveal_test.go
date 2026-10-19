package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRandomHandRevealGameplay(t *testing.T) {
	for _, state := range []string{"present", "departed", "returned", "phased", "empty"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,RevealACardFromHandAtRandom),PlayerAction(Ref_TargetPlayer,DiscardTheCardRevealedThisWay)])))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			var hand []mage.Card
			if state != "empty" {
				hand = randomRONHand(t, b, 3)
			}
			g.SetRandomResults([]int{2, 7})
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil || len(g.PublicReveals()) != 0 {
				t.Fatal("failed payment revealed a card")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if len(g.PublicReveals()) != 0 || len(b.Hand()) != len(hand) {
				t.Fatal("reveal ran during announcement")
			}
			switch state {
			case "phased":
				g.PhaseOut(p.ID())
			case "departed", "returned":
				g.DestroyPermanent(p)
				if state == "returned" {
					g.PutOnBattlefield(p.Card, a.PlayerID())
				}
			}
			clone := g.Clone()
			clone.SetRandomResults([]int{0, 9})
			g.ResolveStack()
			clone.ResolveStack()
			if state == "empty" {
				if len(g.PublicReveals()) != 0 || g.RandIntn(10) != 2 {
					t.Fatal("empty reveal consumed randomness")
				}
				return
			}
			if len(b.Hand()) != 2 || len(b.Graveyard()) != 1 || b.Graveyard()[0].ID() != hand[2].ID() || len(g.PublicReveals()) != 1 || g.PublicReveals()[0].CardID != hand[2].ID() || g.RandIntn(10) != 7 {
				t.Fatal("wrong player, result identity or random consumption")
			}
			cb := clone.GetPlayer(b.PlayerID())
			if len(cb.Graveyard()) != 1 || cb.Graveyard()[0].ID() != hand[0].ID() || clone.RandIntn(10) != 9 {
				t.Fatal("clone shared results")
			}
		})
	}
}

func TestRONRandomHandRevealInvalid(t *testing.T) {
	for _, actions := range []string{
		`PlayerAction(You,DiscardTheCardRevealedThisWay)`,
		`PlayerAction(You,DiscardTheCardRevealedThisWay),PlayerAction(You,RevealACardFromHandAtRandom)`,
		`PlayerAction(You,RevealACardFromHandAtRandom())`,
		`PlayerAction(You,RevealACardFromHandAtRandom(Integer(1)))`,
		`PlayerAction(You,RevealACardFromHandAtRandom),PlayerAction(You,DiscardTheCardRevealedThisWay())`,
		`PlayerAction(ThisPermanent,RevealACardFromHandAtRandom)`,
		`PlayerActions(You,[RevealACardFromHandAtRandom])`,
		`PlayerMayCost(You,PayLife(Integer(1))),Unless(CostWasPaid,[PlayerAction(You,RevealACardFromHandAtRandom)]),PlayerAction(You,DiscardTheCardRevealedThisWay)`,
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

func TestRONRandomHandRevealRealOperands(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"DRK", "Wand of Ith"}, {"LEG", "Tempest Efreet"}} {
		record := staticBaseRecord(t, tc.set, tc.name)
		var producer Node
		var visit func(Node)
		visit = func(n Node) {
			if isRandomHandReveal(n) {
				producer = n
			}
			for _, child := range n.Args {
				visit(child)
			}
			for _, child := range n.Fields {
				visit(child)
			}
		}
		visit(record)
		if producer.Name == "" {
			t.Fatal("real record has no producer")
		}
		g, a, b := ronGame()
		hand := randomRONHand(t, b, 2)
		target := &playerTargetPlan{name: "Ref_TargetPlayer"}
		binding := mage.DeclarePlayerTarget(target.name)
		build, err := compileRandomHandReveal(producer, nil, scope{playerTarget: target})
		if err != nil {
			t.Fatal(err)
		}
		g.SetRandomResults([]int{1})
		body := build(instanceBindings{players: map[*playerTargetPlan]*mage.PlayerTargetBinding{target: binding}})
		card := mage.NewArtifact("Producer", "", mage.WithAbility(mage.Activate(mage.Costs(), body, mage.WithTargets(binding))))
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if len(g.PublicReveals()) != 1 || g.PublicReveals()[0].CardID != hand[1].ID() || len(b.Hand()) != 2 {
			t.Fatal("real producer used wrong player or moved card")
		}
		if _, err := Compile(record); (err == nil) != (tc.name == "Wand of Ith" || tc.name == "Tempest Efreet") {
			t.Fatalf("unexpected complete record support for %s: %v", tc.name, err)
		}
	}
}

func TestRONRandomHandRevealReferences(t *testing.T) {
	for _, text := range []string{`TheCardRevealedFromHandThisWay()`, `TheCardDrawnThisWay`, `ThisPermanent`, `TheCardRevealedFromHandThisWay(Integer(1))`} {
		if _, err := compileRevealedHandCardRef(nameScopeNode(t, text), scope{handReveal: &handRevealPlan{name: "result"}}); err == nil {
			t.Fatalf("accepted %s", text)
		}
	}
	if _, err := compileRevealedHandCardRef(nameScopeNode(t, `TheCardRevealedFromHandThisWay`), scope{}); err == nil {
		t.Fatal("unbound reference accepted")
	}
	plan := &handRevealPlan{name: "result"}
	if _, err := compileRevealedHandCardRef(nameScopeNode(t, `TheCardRevealedFromHandThisWay`), scope{handReveal: plan}); err != nil {
		t.Fatal(err)
	}
}

func TestRONRandomHandRevealDistinctProducers(t *testing.T) {
	g, a, b := ronGame()
	first := randomRONHand(t, a, 2)
	second := randomRONHand(t, b, 2)
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(You,RevealACardFromHandAtRandom),PlayerAction(You,DiscardTheCardRevealedThisWay),PlayerAction(Ref_TargetPlayer,RevealACardFromHandAtRandom),PlayerAction(You,DiscardTheCardRevealedThisWay),PlayerAction(Ref_TargetPlayer,DiscardTheCardRevealedThisWay)])))`)
	for range 2 {
		card := factory()
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
	}
	g.SetRandomResults([]int{1, 0, 0, 0, 9})
	g.ResolveStack()
	reveals := g.PublicReveals()
	if len(a.Hand()) != 0 || len(b.Hand()) != 0 || len(a.Graveyard()) != 2 || len(b.Graveyard()) != 2 || len(reveals) != 4 || g.RandIntn(10) != 9 {
		t.Fatal("distinct instances or producer scopes failed")
	}
	for i, id := range []uuid.UUID{first[1].ID(), second[0].ID(), first[0].ID(), second[1].ID()} {
		if reveals[i].CardID != id {
			t.Fatal("producer read another result")
		}
	}
}
