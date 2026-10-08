package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedHandActionProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEB", "Circle of Protection: Black", core.Black},
		{"LEA", "Circle of Protection: Blue", core.Blue},
		{"LEA", "Circle of Protection: Green", core.Green},
		{"LEA", "Circle of Protection: Red", core.Red},
		{"LEA", "Circle of Protection: White", core.White},
		{"ATQ", "Circle of Protection: Artifacts", core.Colorless},
		{"LEG", "Greater Realm of Preservation", core.Black},
	} {
		for name, action := range map[string]string{
			"reveal":   `PlayerAction(ControllerOfPermanent(ThisPermanent),RevealHand)`,
			"matching": `PlayerAction(ControllerOfPermanent(ThisPermanent),DiscardCards(IsCardtype(Creature)))`,
			"hand":     `PlayerAction(ControllerOfPermanent(ThisPermanent),DiscardHand)`,
			"chosen":   `PlayerAction(ControllerOfPermanent(ThisPermanent),DiscardACard)`,
			"random":   `PlayerAction(ControllerOfPermanent(ThisPermanent),DiscardACardAtRandom)`,
			"amount":   `DiscardNumberCardsAtRandom(TheNumberOfCardsOfTypeInPlayersHand(IsCardtype(Creature),ControllerOfPermanent(ThisPermanent)))`,
		} {
			for _, state := range []string{"live", "departed", "returned", "phased", "control", "copy", "clone"} {
				t.Run(tc.name+"/"+name+"/"+state, func(t *testing.T) {
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
					a := &ronDelayedPaymentPlayer{ronSourceChoicePlayer: &ronSourceChoicePlayer{BasePlayer: mage.NewBasePlayer("A")}, accept: true}
					b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
					g := mage.NewGame(a, b)
					for _, player := range []*mage.BasePlayer{a.BasePlayer, b.BasePlayer} {
						for range 3 {
							addRONHandCard(player, mage.NewCreature("Hand", "", 1, 1))
						}
						addRONHandCard(player, mage.NewLand("Land"))
					}
					factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`]))]))`)
					card := factory()
					card.SetOwner(a.PlayerID())
					source := g.PutOnBattlefield(card, a.PlayerID())
					g.ChangeObjectColor(source.ID(), tc.color)
					original := g.CaptureDamageSource(source.ID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("delay created at announcement")
					}
					g.ResolveStack()
					switch state {
					case "departed", "returned":
						g.BouncePermanentToHand(source)
						if state == "returned" {
							a.RemoveFromHand(card.ID())
							g.PutOnBattlefield(card, b.PlayerID())
						}
					case "phased":
						g.PhaseOut(source.ID())
					case "control":
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
					binding := mage.BindDamageSource("hand source")
					bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
					payment := make([]mage.Cost, len(costs))
					for i, cost := range costs {
						payment[i] = cost()
					}
					consumer := g.PutOnBattlefield(mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), mage.Sequence(plan.waitingDelayedPermanentChoice(bindings), shield(bindings))))), a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), consumer.ID(), 0, nil); err == nil || g.StackSize() != 0 || a.calls != 0 {
						t.Fatal("failed payment changed state")
					}
					a.ManaPool().Add(core.White, 2)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), consumer.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if a.calls != 0 || len(g.PublicReveals()) != 0 {
						t.Fatal("choice or reveal ran at announcement")
					}
					beforeA, beforeB := len(a.Hand()), len(b.Hand())
					for range 3 {
						got := g.WaitingDelayedPermanentSources()
						if len(got) != 1 || !got[0].SameObject(original) || a.offers != 0 || b.offers != 0 || len(g.PublicReveals()) != 0 || len(a.Hand()) != beforeA || len(b.Hand()) != beforeB {
							t.Fatal("query lost source or ran hand action")
						}
					}
					clone := g.Clone()
					if state == "clone" {
						clone.ResolveStack()
						clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						clone.PutTriggersOnStack()
						clone.ResolveStack()
						if len(g.WaitingDelayedPermanentSources()) != 1 || len(a.Hand()) != beforeA || a.calls != 0 {
							t.Fatal("clone changed original")
						}
						if len(clone.WaitingDelayedPermanentSources()) != 0 {
							t.Fatal("clone kept detected delay")
						}
					}
					g.ResolveStack()
					if a.calls != 1 || !a.source.SameObject(original) {
						t.Fatal("hand operand missing from source choice")
					}
					addRONHandCard(a.BasePlayer, mage.NewCreature("Late", "", 1, 1))
					addRONHandCard(b.BasePlayer, mage.NewCreature("Late", "", 1, 1))
					beforeA++
					beforeB++
					g.SetRandomResults([]int{0, 0, 0, 0, 0, 0, 0, 0})
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("detected delay still waits")
					}
					g.PutTriggersOnStack()
					g.ResolveStack()
					targetA := state != "control" || name == "amount"
					count := 1
					if state == "copy" {
						count = 2
					}
					moved := 0
					revealed := 0
					switch name {
					case "reveal":
						revealed = beforeA * count
						if !targetA {
							revealed = beforeB * count
						}
					case "matching", "amount":
						moved = 4
						if name == "amount" && state == "copy" {
							moved = 5
						}
					case "hand":
						moved = beforeA
						if !targetA {
							moved = beforeB
						}
					case "chosen", "random":
						moved = count
					}
					wantA, wantB := beforeA, beforeB
					if targetA {
						wantA -= moved
					} else {
						wantB -= moved
					}
					if len(a.Hand()) != wantA || len(b.Hand()) != wantB || len(g.PublicReveals()) != revealed {
						t.Fatal("wrong current/departure player or late hand", len(a.Hand()), len(b.Hand()), len(g.PublicReveals()), wantA, wantB, revealed)
					}
					g.DealDamageToPlayerFromSource(a, 2, original)
					if a.Life() != 20 {
						t.Fatal("chosen source not prevented")
					}
					g.DealDamageToPlayerFromSource(a, 1, original)
					if a.Life() != 19 {
						t.Fatal("shield not consumed")
					}
					if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ChooseADamageSource") {
						t.Fatal("incomplete aggregation accepted", err)
					}
				})
			}
		}
	}
}

func TestRONWaitingDelayedHandActionInvalid(t *testing.T) {
	for _, action := range []string{
		`PlayerAction(ControllerOfPermanent(),RevealHand)`,
		`PlayerAction(ControllerOfPermanent(Ref_TargetPermanent),DiscardHand)`,
		`PlayerAction(Trigger_ThatPlayer,DiscardACardAtRandom)`,
		`PlayerAction(ThisPermanent,DiscardCards(IsCardtype(Creature)))`,
		`PlayerAction(ControllerOfPermanent(ThisPermanent),DiscardCards(And([])))`,
		`DiscardNumberCardsAtRandom(PowerOfPermanent(Ref_TargetPermanent))`,
		`DiscardNumberCardsAtRandom(TheNumberOfCardsInPlayersHand(Trigger_ThatPlayer))`,
		`DiscardNumberCardsAtRandom(TheNumberOfCountersRemovedThisWay)`,
		`DiscardNumberCardsAtRandom(Integer(-1))`,
		`DiscardNumberCardsAtRandom(Plus(Integer(1)))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + action + `]))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid hand declaration accepted", action)
		}
	}
}

func TestRONWaitingDelayedHandActionValues(t *testing.T) {
	for _, tc := range []struct {
		value  string
		amount int
	}{
		{`PowerOfPermanent(ThisPermanent)`, 2},
		{`ManaValueOfPermanent(ThisPermanent)`, 2},
		{`Plus(Integer(1),TheNumberOfCardsOfTypeInPlayersHand(IsCardtype(Creature),ControllerOfPermanent(ThisPermanent)))`, 3},
		{`Minus(Integer(0),TheNumberOfCardsInPlayersHand(ControllerOfPermanent(ThisPermanent)))`, 0},
		{`Minus(TheNumberOfCardsInPlayersHand(ControllerOfPermanent(ThisPermanent)),TheNumberOfCardsInPlayersHand(ControllerOfPermanent(ThisPermanent)))`, 0},
	} {
		t.Run(tc.value, func(t *testing.T) {
			g, a, b := ronGame()
			action := `DiscardNumberCardsAtRandom(` + tc.value + `)`
			nodes, err := Parse(strings.NewReader(`Card(Name:"Source",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),ManaCost:[ManaCostGeneric(1),ManaCostU],CardPT:(Power:2,Toughness:4),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`]))]))])`), "values.ron")
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
			original := g.CaptureDamageSource(p.ID())
			for range 2 {
				addRONHandCard(a, mage.NewCreature("Hand", "", 1, 1))
			}
			addRONHandCard(a, mage.NewLand("Land"))
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.BouncePermanentToHand(p)
			a.RemoveFromHand(card.ID())
			card.SetBasePT(9, 9)
			g.PutOnBattlefield(card, b.PlayerID())
			got := g.WaitingDelayedPermanentSources()
			if len(got) != 1 || !got[0].SameObject(original) {
				t.Fatal("dynamic amount lost original source")
			}
			addRONHandCard(a, mage.NewLand("Late land"))
			g.SetRandomResults([]int{0, 0, 0, 7})
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				if len(game.GetPlayer(a.PlayerID()).Hand()) != 4-tc.amount || len(game.GetPlayer(a.PlayerID()).Graveyard()) != tc.amount || len(game.GetPlayer(b.PlayerID()).Hand()) != 0 {
					t.Fatal("wrong LKI value or dynamic amount", len(game.GetPlayer(a.PlayerID()).Hand()), tc.amount)
				}
				wantNext := 0
				if tc.amount == 3 {
					wantNext = 7
				}
				if game.RandIntn(10) != wantNext {
					t.Fatal("wrong random consumption")
				}
			}
		})
	}
}

func TestRONWaitingDelayedHandActionInstancesAndReplacement(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(map[bool]string{false: "graveyard", true: "library"}[replace], func(t *testing.T) {
			a := &ronDiscardPlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			for _, player := range []*mage.BasePlayer{a.BasePlayer, b} {
				for range 4 {
					addRONHandCard(player, mage.NewLand("Hand"))
				}
			}
			if replace {
				g.PutOnBattlefield(mage.NewArtifact("Replacement", "", mage.WithStaticAbility(mage.DiscardToLibraryReplacement())), a.PlayerID())
			}
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(ControllerOfPermanent(ThisPermanent),DiscardNumberCardsAtRandom(PowerOfPermanent(Ref_TargetPermanent)))]))])))`)
			donors := make([]*mage.Permanent, 0, 2)
			for i, player := range []*mage.BasePlayer{a.BasePlayer, b} {
				donorCard := mage.NewCreature("Amount", "", 2+i, 4)
				donorCard.SetOwner(player.PlayerID())
				donor := g.PutOnBattlefield(donorCard, player.PlayerID())
				donors = append(donors, donor)
				card := factory()
				card.SetOwner(player.PlayerID())
				source := g.PutOnBattlefield(card, player.PlayerID())
				if err := g.ActivateAbilityByIndex(player.PlayerID(), source.ID(), 0, []uuid.UUID{donor.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.DestroyPermanent(source)
			}
			if len(g.WaitingDelayedPermanentSources()) != 4 {
				t.Fatal("instances shared source or target")
			}
			for _, donor := range donors {
				g.DestroyPermanent(donor)
			}
			seen := 0
			watcher := mage.NewTriggered(core.EvtDiscard, false).SetCondition(func(evt *core.GameEvent, reader mage.GameReader, _, _ uuid.UUID) bool {
				seen++
				want := 2
				if evt.PlayerID == b.PlayerID() {
					want = 1
				}
				if len(reader.GetPlayer(evt.PlayerID).Hand()) != want {
					t.Error("event saw partial hand batch")
				}
				return false
			})
			g.PutOnBattlefield(mage.NewArtifact("Watcher", "", mage.WithAbility(watcher)), b.PlayerID())
			g.SetRandomResults([]int{0, 0, 0, 0, 0, 7})
			clone := g.Clone()
			for _, game := range []*mage.Game{clone, g} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				for _, player := range []*mage.BasePlayer{a.BasePlayer, b} {
					actual := game.GetPlayer(player.PlayerID())
					amount := 2
					if player == b {
						amount = 3
					}
					if len(actual.Hand()) != 4-amount {
						t.Fatal("per-instance controller or LKI target amount lost")
					}
					library := 0
					grave := 2 + amount
					if replace && game == g && player == a.BasePlayer {
						library = 2
						grave -= 2
					}
					if len(actual.Library()) != library || len(actual.Graveyard()) != grave {
						t.Fatal("discard bypassed replacement", len(actual.Library()), len(actual.Graveyard()), library, grave)
					}
				}
				if game.RandIntn(10) != 7 || len(game.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("query consumed randomness or retained detected delay")
				}
			}
			if seen != 10 {
				t.Fatal("wrong discard event count", seen)
			}
		})
	}
}
