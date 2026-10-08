package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedToken(t *testing.T) {
	for _, operand := range []string{"count", "player", "players", "zero"} {
		for _, state := range []string{"live", "phased", "returned", "copy", "clone"} {
			t.Run(operand+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				action := `CreateTokens([NumberTokens(ManaValueOfPermanent(ThisPermanent),` + saprolingRON + `)],[])`
				if operand == "zero" {
					action = `CreateTokens([NumberTokens(Minus(PowerOfPermanent(ThisPermanent),PowerOfPermanent(ThisPermanent)),` + saprolingRON + `)],[])`
				}
				if operand == "player" {
					action = `PlayerAction(ControllerOfPermanent(ThisPermanent),CreateTokens([` + saprolingRON + `],[]))`
				}
				if operand == "players" {
					action = `PlayerActions(ControllerOfPermanent(ThisPermanent),[CreateTokens([` + saprolingRON + `],[])])`
				}
				nodes, err := Parse(strings.NewReader(`Card(Name:"Token source",Typeline:(Supertypes:[],Cardtypes:[Artifact,Creature],Subtypes:[]),ManaCost:[ManaCostGeneric(2)],CardPT:(Power:2,Toughness:4),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`]))]))])`), "waiting-token.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				card := factory()
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if len(g.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("delay created before resolution")
				}
				g.ResolveStack()
				switch state {
				case "phased":
					g.PhaseOut(p.ID())
				case "returned":
					g.BouncePermanentToHand(p)
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
					g.ChangeObjectColor(card.ID(), core.Red)
				default:
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				for range 2 {
					sources := g.WaitingDelayedPermanentSources()
					if len(sources) != 1 || !sources[0].SameObject(saved) || len(g.FilterBattlefield(mage.IsToken)) != 0 {
						t.Fatal("token delay lost source or query created tokens")
					}
				}
				game := g
				if state == "clone" {
					game = g.Clone()
				}
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				if len(game.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("fired delay remains waiting")
				}
				want := 0
				if operand == "count" {
					want = 2
				}
				if operand == "player" || operand == "players" {
					want = 1
				}
				if state == "copy" {
					want *= 2
				}
				tokens := game.FilterBattlefield(mage.IsToken)
				if len(tokens) != want {
					t.Fatalf("tokens %d want %d", len(tokens), want)
				}
				owner := b.PlayerID()
				if state == "phased" || state == "returned" || operand == "count" {
					owner = a.PlayerID()
				}
				owners := map[uuid.UUID]int{}
				for _, token := range tokens {
					owners[token.Card.Owner()]++
					if token.ControllerID() != token.Card.Owner() {
						t.Fatal("wrong token controller")
					}
				}
				if operand == "count" && state == "copy" {
					if owners[a.PlayerID()] != 2 || owners[b.PlayerID()] != 2 {
						t.Fatal("copy lost resolving player")
					}
				} else if owners[owner] != want {
					t.Fatal("wrong lexical token player")
				}
				if state == "clone" && (len(g.WaitingDelayedPermanentSources()) != 1 || len(g.FilterBattlefield(mage.IsToken)) != 0) {
					t.Fatal("clone shared delay or token state")
				}
			})
		}
	}
}

func TestRONWaitingDelayedTokenInvalid(t *testing.T) {
	for _, action := range []string{
		`PlayerAction(ActionPlayer,CreateTokens([` + saprolingRON + `],[]))`,
		`PlayerActions(You,[CreateTokens([NumberTokens(ManaValueOfPermanent(Ref_TargetPermanent),` + saprolingRON + `)],[])])`,
		`PlayerAction(You,CreateTokens([` + saprolingRON + `],[Tapped]))`,
		`PlayerActions(You,[CreateTokens([NumberTokens(TheNumberOfCountersRemovedThisWay,` + saprolingRON + `)],[])])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`]))]))])`), "invalid-token.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid token scope accepted")
		}
	}
}

func TestRONWaitingDelayedTokenRukhEgg(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ARN", "Rukh Egg")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.DestroyPermanent(p)
	g.ResolveStack()
	if len(g.DelayedTriggers()) != 1 || len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("constant token creation inferred a source read")
	}
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	tokens := g.FilterBattlefield(mage.IsToken)
	if len(tokens) != 1 || tokens[0].ControllerID() != a.PlayerID() || tokens[0].Card.Owner() != a.PlayerID() || tokens[0].CurrentPower(g) != 4 || tokens[0].CurrentToughness(g) != 4 || !tokens[0].HasKeyword(core.Flying) || !tokens[0].HasSubType("Bird") {
		t.Fatal("Rukh Egg lost token or delayed controller")
	}
	if len(g.DelayedTriggers()) != 0 || len(clone.DelayedTriggers()) != 1 || len(clone.FilterBattlefield(mage.IsToken)) != 0 {
		t.Fatal("Rukh Egg shared clone lifetime")
	}
}

func TestRONWaitingDelayedTokenProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEB", "Circle of Protection: Black", core.Black}, {"LEA", "Circle of Protection: Blue", core.Blue}, {"LEA", "Circle of Protection: Green", core.Green}, {"LEA", "Circle of Protection: Red", core.Red}, {"LEA", "Circle of Protection: White", core.White}, {"ATQ", "Circle of Protection: Artifacts", core.Colorless}, {"LEG", "Greater Realm of Preservation", core.Black},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := staticBaseRecord(t, tc.set, tc.name)
			rule := record.Fields["Rules"].Args[0]
			actions := rule.Args[1].Args[0].Args
			plan, err := compileChosenDamageSourceChoice(actions[0])
			if err != nil {
				t.Fatal(err)
			}
			shield, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
			if err != nil {
				t.Fatal(err)
			}
			costs, _, err := compileCosts(rule.Args[0])
			if err != nil {
				t.Fatal(err)
			}
			a := &ronSourceChoicePlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CreateTokens([NumberTokens(PowerOfPermanent(ThisPermanent),TokenWithPT(PT(1,1),Colorless,[],[Creature],SubtypesList([Saproling]),[]))],[])]))]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.ChangeObjectColor(p.ID(), tc.color)
			saved := g.CaptureDamageSource(p.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.BouncePermanentToHand(p)
			binding := mage.BindDamageSource("waiting token")
			bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
			payment := make([]mage.Cost, len(costs))
			for i, cost := range costs {
				payment[i] = cost()
			}
			consumer := mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), mage.Sequence(plan.waitingDelayedPermanentChoice(bindings), shield(bindings)))))
			if err := mage.ValidateCardDefinition(consumer.Name(), consumer); err != nil {
				t.Fatal(err)
			}
			q := g.PutOnBattlefield(consumer, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err == nil || a.calls != 0 || g.StackSize() != 0 {
				t.Fatal("unpaid choice changed state")
			}
			a.ManaPool().Add(core.White, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.calls != 0 {
				t.Fatal("early source selection")
			}
			g.ResolveStack()
			if a.calls != 1 || !a.source.SameObject(saved) {
				t.Fatal("waiting token body omitted departed source")
			}
			g.DealDamageToPlayerFromSource(a, 2, saved)
			g.DealDamageToPlayerFromSource(a, 1, saved)
			if a.Life() != 19 {
				t.Fatal("shield prevention or consumption failed")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if len(g.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("token delay remained waiting")
			}
			if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ChooseADamageSource") {
				t.Fatal("general aggregation was accepted", err)
			}
		})
	}
}
