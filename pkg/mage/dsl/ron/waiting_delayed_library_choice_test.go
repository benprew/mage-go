package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedLibraryChoiceProtection(t *testing.T) {
	for name, build := range map[string]func(mage.ResolutionValue) mage.ResolvingEffect{"scry": mage.ScryAtResolution, "surveil": mage.SurveilAtResolution} {
		for _, tc := range []struct {
			set, name string
			color     core.Color
		}{
			{"LEB", "Circle of Protection: Black", core.Black}, {"LEA", "Circle of Protection: Blue", core.Blue}, {"LEA", "Circle of Protection: Green", core.Green}, {"LEA", "Circle of Protection: Red", core.Red}, {"LEA", "Circle of Protection: White", core.White}, {"ATQ", "Circle of Protection: Artifacts", core.Colorless}, {"LEG", "Greater Realm of Preservation", core.Black},
		} {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
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
				factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([GainLife(Integer(0))]))`)
				card := factory()
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				g.ChangeObjectColor(p.ID(), tc.color)
				saved := g.CaptureDamageSource(p.ID())
				nodes, err := Parse(strings.NewReader(`ManaValueOfPermanent(ThisPermanent)`), "amount.ron")
				if err != nil {
					t.Fatal(err)
				}
				amount, err := compileInstanceValue(nodes[0], scope{permanent: true})
				if err != nil {
					t.Fatal(err)
				}
				if err := mage.ApplyEffect(g, mage.AtNext(mage.EachEndStep, build(amount(instanceBindings{}))), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				g.BouncePermanentToHand(p)
				binding := mage.BindDamageSource("waiting shield")
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
					t.Fatal("waiting shield body omitted departed source")
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
					t.Fatal("shield delay remained waiting")
				}
				if _, err := Compile(record); err != nil {
					t.Fatal("complete source-choice record rejected", err)
				}
			})
		}
	}
}

func TestRONWaitingDelayedLibraryChoiceValues(t *testing.T) {
	for name, build := range map[string]func(mage.ResolutionValue) mage.ResolvingEffect{"scry": mage.ScryAtResolution, "surveil": mage.SurveilAtResolution} {
		for _, value := range []string{`PowerOfPermanent(ThisPermanent)`, `Plus(PowerOfPermanent(ThisPermanent),Integer(1))`, `Minus(PowerOfPermanent(ThisPermanent),PowerOfPermanent(ThisPermanent))`} {
			t.Run(name+"/"+value, func(t *testing.T) {
				a := &ronLibraryViewer{BasePlayer: mage.NewBasePlayer("A")}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				for range 4 {
					a.AddToLibrary(repositoryCard(t, "LEB", "Island"))
				}
				nodes, err := Parse(strings.NewReader(value), "amount.ron")
				if err != nil {
					t.Fatal(err)
				}
				amount, err := compileInstanceValue(nodes[0], scope{permanent: true})
				if err != nil {
					t.Fatal(err)
				}
				factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),ActionList([GainLife(Integer(0))]))`)
				for _, player := range []*mage.BasePlayer{a.BasePlayer, b} {
					card := factory()
					card.SetOwner(player.PlayerID())
					p := g.PutOnBattlefield(card, player.PlayerID())
					if err := mage.ApplyEffect(g, mage.AtNext(mage.EachEndStep, build(amount(instanceBindings{}))), p.ID(), player.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					g.BouncePermanentToHand(p)
				}
				if len(g.WaitingDelayedPermanentSources()) != 2 || a.orders != 0 {
					t.Fatal("lost per-instance source or early choice")
				}
				clone := g.Clone()
				clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 2 {
					t.Fatal("clone shares waiting frame")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				g.ResolveStack()
				if name == "scry" {
					want := 2
					if strings.HasPrefix(value, "Plus") {
						want = 3
					}
					if strings.HasPrefix(value, "Minus") {
						want = 0
					}
					if len(a.seenOrder) != want {
						t.Fatal("wrong lowered amount", len(a.seenOrder), want)
					}
				}
			})
		}
	}
}

func TestRONWaitingDelayedLibraryChoiceInvalid(t *testing.T) {
	for _, value := range []string{`PowerOfPermanent(You)`, `PowerOfPermanent(Ref_TargetPermanent)`, `PowerOfPermanent(Trigger_ThatPermanent)`, `PowerOfPermanent(ThisPermanent,ThisPermanent)`, `PowerOfPermanent(ThisPermanent())`, `Plus(Integer(1))`, `ValueX`, `TheNumberOfCardsInPlayersHand(ActionPlayer)`} {
		nodes, err := Parse(strings.NewReader(value), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileInstanceValue(nodes[0], scope{permanent: true}); err == nil {
			t.Fatal("invalid value accepted", value)
		}
	}
}
