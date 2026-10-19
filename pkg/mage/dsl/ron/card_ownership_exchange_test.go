package ron

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCardOwnershipExchangeGameplay(t *testing.T) {
	record := staticBaseRecord(t, "ATQ", "Bronze Tablet")
	var exile Node
	exchange := nameScopeNode(t, `ExchangeOwnershipOfTwoCards(TheFirstCardExiledThisWay,TheSecondCardExiledThisWay)`)
	var walk func(Node)
	walk = func(n Node) {
		if n.Name == "Exile" {
			exile = n
		}
		for _, child := range n.Args {
			walk(child)
		}
	}
	for _, rule := range record.Fields["Rules"].Args {
		walk(rule)
	}
	for _, state := range []string{"normal", "clone", "phased-source", "returned-source", "illegal-target", "failed-payment"} {
		t.Run(state, func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(And([PayMana([ManaCostGeneric(4)]),TapPermanent(ThisPermanent)]),Targeted([TargetPermanent(AnyPermanent)],ActionList([])))])`), "probe.ron")
			if err != nil {
				t.Fatal(err)
			}
			rules := nodes[0].Fields["Rules"]
			rules.Args[0].Args[1].Args[1].Args[0].Args = []Node{exile, exchange}
			nodes[0].Fields["Rules"] = rules
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g, _ := mage.NewGameWithAnte(a, b, nil, nil)
			source := factory()
			source.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(source, a.PlayerID())
			target := mage.NewArtifact("target", "")
			target.SetOwner(b.PlayerID())
			q := g.PutOnBattlefield(target, b.PlayerID())
			if state == "failed-payment" {
				a.ManaPool().Add(core.Colorless, 3)
			} else {
				a.ManaPool().Add(core.Colorless, 4)
			}
			err = g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()})
			if state == "failed-payment" {
				if err == nil || g.StackSize() != 0 || g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 3 {
					t.Fatal("failed payment changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(g.GetExile()) != 0 || g.FindCardAnywhere(source.ID()).Owner() != a.PlayerID() {
				t.Fatal("exchange at announcement")
			}
			switch state {
			case "phased-source":
				g.PhaseOut(p.ID())
			case "returned-source":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(source.ID())
				g.PutOnBattlefield(source, a.PlayerID())
			case "illegal-target":
				g.PhaseOut(q.ID())
			}
			clone := g.Clone()
			g.ResolveStack()
			swapped := state == "normal" || state == "clone"
			if (g.FindCardAnywhere(source.ID()).Owner() == b.PlayerID()) != swapped || (g.FindCardAnywhere(target.ID()).Owner() == a.PlayerID()) != swapped {
				t.Fatal("wrong exchange or partial exchange")
			}
			if clone.FindCardAnywhere(source.ID()).Owner() != a.PlayerID() {
				t.Fatal("clone shared ownership")
			}
			clone.ResolveStack()
			if (clone.FindCardAnywhere(source.ID()).Owner() == b.PlayerID()) != swapped {
				t.Fatal("clone lost instance bindings")
			}
		})
	}
}

func TestRONCardOwnershipExchangeInvalid(t *testing.T) {
	producer := `Exile([Permanent(ThisPermanent),Permanent(ThisPermanent)],[])`
	for _, actions := range []string{
		producer + `,ExchangeOwnershipOfTwoCards`,
		producer + `,ExchangeOwnershipOfTwoCards(TheFirstCardExiledThisWay)`,
		producer + `,ExchangeOwnershipOfTwoCards(TheFirstCardExiledThisWay,TheSecondCardExiledThisWay,ThisExiledCard)`,
		`ExchangeOwnershipOfTwoCards(TheFirstCardExiledThisWay,TheSecondCardExiledThisWay),` + producer,
		producer + `,ExchangeOwnershipOfTwoCards(ThisPermanent,TheSecondCardExiledThisWay)`,
		producer + `,ExchangeOwnershipOfTwoCards(TheFirstCardExiledThisWay(),TheSecondCardExiledThisWay)`,
		`If(PermanentPassesFilter(ThisPermanent,IsTapped),[` + producer + `]),ExchangeOwnershipOfTwoCards(TheFirstCardExiledThisWay,TheSecondCardExiledThisWay)`,
		producer + `,ExchangeOwnershipOfTwoCards(TheCardRevealedFromHandThisWay,TheSecondCardExiledThisWay)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+actions+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", actions)
		}
	}
}

func TestRONCardOwnershipExchangeRevealContexts(t *testing.T) {
	record := staticBaseRecord(t, "LEG", "Tempest Efreet")
	var reveal, exchange Node
	var walk func(Node)
	walk = func(n Node) {
		if isRandomHandReveal(n) {
			reveal = n
		}
		if n.Name == "ExchangeOwnershipOfTwoCards" {
			exchange = n
		}
		for _, child := range n.Args {
			walk(child)
		}
	}
	for _, rule := range record.Fields["Rules"].Args {
		walk(rule)
	}
	reveal.Args = slices.Clone(reveal.Args)
	reveal.Args[0] = Node{Kind: Symbol, Name: "You", Position: reveal.Args[0].Position}
	exchange.Args = slices.Clone(exchange.Args)
	exchange.Args[1] = Node{Kind: Symbol, Name: "Ref_TargetGraveyardCard", Position: exchange.Args[1].Position}
	for _, context := range []string{"spell", "activation", "trigger", "delayed"} {
		t.Run(context, func(t *testing.T) {
			body := `Targeted([TargetGraveyardCard(And([IsCardtype(Creature),InAPlayersGraveyard(AnyPlayer)]))],ActionList([]))`
			rule, kind := `Activated(PayMana([]),`+body+`)`, "Artifact"
			if context == "spell" {
				rule, kind = `SpellActions(`+body+`)`, "Sorcery"
			}
			if context == "trigger" {
				rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),` + body + `)`
			}
			if context == "delayed" {
				rule = `Activated(PayMana([]),Targeted([TargetGraveyardCard(And([IsCardtype(Creature),InAPlayersGraveyard(AnyPlayer)]))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([]))])))`
			}
			nodes, err := Parse(strings.NewReader(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[`+kind+`],Subtypes:[]),Rules:[`+rule+`])`), "probe.ron")
			if err != nil {
				t.Fatal(err)
			}
			var fill func(*Node)
			fill = func(n *Node) {
				if n.Name == "ActionList" && len(n.Args) == 1 && len(n.Args[0].Args) == 0 {
					n.Args[0].Args = []Node{reveal, exchange}
					return
				}
				for i := range n.Args {
					fill(&n.Args[i])
				}
			}
			rules := nodes[0].Fields["Rules"]
			fill(&rules)
			nodes[0].Fields["Rules"] = rules
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g, _ := mage.NewGameWithAnte(a, b, nil, nil)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			hand := mage.NewArtifact("hand", "")
			hand.SetOwner(a.PlayerID())
			a.AddToHand(hand)
			grave := mage.NewCreature("grave", "", 2, 2)
			grave.SetOwner(b.PlayerID())
			b.AddToGraveyard(grave)
			card := factory()
			card.SetOwner(a.PlayerID())
			if context == "spell" {
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{grave.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			} else {
				p := g.PutOnBattlefield(card, a.PlayerID())
				if context == "trigger" {
					g.PutTriggersOnStack()
				} else if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{grave.ID()}); err != nil {
					t.Fatal(err)
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			if context == "delayed" {
				if len(g.PublicReveals()) != 0 || g.FindCardAnywhere(hand.ID()).Owner() != a.PlayerID() {
					t.Fatal("delay executed early")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				g.ResolveStack()
			}
			if g.FindCardAnywhere(hand.ID()).Owner() != b.PlayerID() || g.FindCardAnywhere(grave.ID()).Owner() != a.PlayerID() || len(a.Hand()) != 1 || len(b.Graveyard()) != 1 {
				t.Fatal("reveal reference exchange moved cards or lost bindings")
			}
			if clone.FindCardAnywhere(hand.ID()).Owner() != a.PlayerID() || len(clone.PublicReveals()) != 0 {
				t.Fatal("clone shared reveal or ownership")
			}
		})
	}
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Tempest Efreet", factory()); err != nil {
		t.Fatal(err)
	}
}

func TestRONCardOwnershipExchangeRoles(t *testing.T) {
	record := staticBaseRecord(t, "ATQ", "Bronze Tablet")
	entries := map[string]*InventorySignature{}
	walkInventory(record, "Bronze Tablet", "Card", "card", "collection", true, entries)
	count := 0
	for _, entry := range entries {
		if entry.Name == "TheFirstCardExiledThisWay" || entry.Name == "TheSecondCardExiledThisWay" {
			if entry.Role != "card_reference" {
				t.Fatal("wrong exchange operand role", entry.Role)
			}
			count++
		}
	}
	if count != 2 {
		t.Fatal("missing exchange operand inventory")
	}
	for _, rule := range []string{
		`Activated(ExchangeOwnershipOfTwoCards(Ref_TargetAnteCard,Ref_TargetAnteCard),ActionList([DrawACard]))`,
		`AsPermanentEnters(ThisPermanent,[ExchangeOwnershipOfTwoCards(ThisCard,ThisCard)])`,
		`PermanentLayerEffect(ThisPermanent,[ExchangeOwnershipOfTwoCards(ThisCard,ThisCard)])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted wrong role", rule)
		}
	}
}

func TestRONCardOwnershipExchangeInstancesAndCopy(t *testing.T) {
	for _, copied := range []bool{false, true} {
		t.Run(map[bool]string{false: "instances", true: "copy"}[copied], func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([Exile([Permanent(ThisPermanent),Permanent(Ref_TargetPermanent)],[]),ExchangeOwnershipOfTwoCards(TheFirstCardExiledThisWay,TheSecondCardExiledThisWay)])))`)
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g, _ := mage.NewGameWithAnte(a, b, nil, nil)
			one, two := factory(), factory()
			one.SetOwner(a.PlayerID())
			two.SetOwner(b.PlayerID())
			p, q := g.PutOnBattlefield(one, a.PlayerID()), g.PutOnBattlefield(two, b.PlayerID())
			x, y := mage.NewArtifact("x", ""), mage.NewArtifact("y", "")
			x.SetOwner(b.PlayerID())
			y.SetOwner(a.PlayerID())
			px, py := g.PutOnBattlefield(x, b.PlayerID()), g.PutOnBattlefield(y, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{px.ID()}); err != nil {
				t.Fatal(err)
			}
			if copied {
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			} else if err := g.ActivateAbilityByIndex(b.PlayerID(), q.ID(), 0, []uuid.UUID{py.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindCardAnywhere(one.ID()).Owner() != b.PlayerID() || g.FindCardAnywhere(x.ID()).Owner() != a.PlayerID() {
				t.Fatal("copied exchange repeated or lost its own bindings")
			}
			wantTwo, wantY := a.PlayerID(), b.PlayerID()
			if copied {
				wantTwo, wantY = b.PlayerID(), a.PlayerID()
			}
			if g.FindCardAnywhere(two.ID()).Owner() != wantTwo || g.FindCardAnywhere(y.ID()).Owner() != wantY {
				t.Fatal("instances shared result bindings")
			}
		})
	}
}
