package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedDamageChoiceOperands(t *testing.T) {
	for _, operand := range []string{"source", "player"} {
		for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control", "copy", "clone"} {
			t.Run(operand+"/"+state, func(t *testing.T) {
				a := &ronSourceChoicePlayer{BasePlayer: mage.NewBasePlayer("A")}
				b := &ronSourceChoicePlayer{BasePlayer: mage.NewBasePlayer("B")}
				g := mage.NewGame(a, b)
				record := staticBaseRecord(t, "LEA", "Jade Monolith")
				rule := record.Fields["Rules"].Args[0]
				targeted := rule.Args[1]
				actions := targeted.Args[1].Args[0].Args
				choice, err := compileChosenDamageSourceChoice(actions[0])
				if err != nil {
					t.Fatal(err)
				}
				targeted.Args[1].Args[0].Args = actions[1:]
				plan, err := compileActions(targeted, scope{permanent: true, chosenDamageSource: choice})
				if err != nil {
					t.Fatal(err)
				}
				playerText := "You"
				if operand == "player" {
					playerText = "ControllerOfPermanent(ThisPermanent)"
				}
				nodes, err := Parse(strings.NewReader(playerText), "choice-player.ron")
				if err != nil {
					t.Fatal(err)
				}
				player, err := compilePlayerRef(nodes[0], scope{permanent: true})
				if err != nil {
					t.Fatal(err)
				}
				source, err := Parse(strings.NewReader("ThisPermanent"), "choice-source.ron")
				if err != nil {
					t.Fatal(err)
				}
				ref, err := compileSelfDamageSource(source[0], scope{permanent: true}, true)
				if err != nil {
					t.Fatal(err)
				}
				costs, _, err := compileCosts(rule.Args[0])
				if err != nil {
					t.Fatal(err)
				}
				factory := func() mage.Card {
					binding := mage.BindDamageSource("chosen")
					bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{choice: binding}}
					body := plan.buildWithBindings(bindings)
					refs := []mage.DamageSourceRef{ref(bindings)}
					if operand == "player" {
						refs = []mage.DamageSourceRef{mage.DamageSourceOf(mage.AttachedPermanent)}
					}
					choose := mage.ChooseDamageSourceMatchingFrom(player.build(bindings), refs, choice.filter, binding)
					body.effects = []mage.Effect{mage.AtNext(mage.EachEndStep, mage.Sequence(append([]mage.Effect{choose}, body.effects...)...))}
					payment := make([]mage.Cost, len(costs))
					for i, cost := range costs {
						payment[i] = cost()
					}
					return mage.NewCreature("Choice delay", "{U}", 1, 20, mage.WithAbility(Activated(payment, body)))
				}
				card := factory()
				card.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				p := g.PutOnBattlefield(card, a.PlayerID())
				q := g.PutOnBattlefield(mage.NewCreature("Recipient", "", 1, 20), a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err == nil || g.StackSize() != 0 {
					t.Fatal("failed payment changed state")
				}
				a.ManaPool().Add(core.Blue, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
					t.Fatal(err)
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if len(g.WaitingDelayedPermanentSources()) != 0 || a.calls != 0 {
					t.Fatal("delay or choice ran at announcement")
				}
				g.ResolveStack()
				switch state {
				case "departed", "returned":
					g.BouncePermanentToHand(p)
					if state == "returned" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, b.PlayerID())
						g.ChangeObjectColor(card.ID(), core.Red)
					}
				case "phased":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				found := false
				for _, s := range g.WaitingDelayedPermanentSources() {
					found = found || s.SameObject(saved)
				}
				if !found || a.calls != 0 || b.calls != 0 {
					t.Fatal("waiting choice omitted source or ran early")
				}
				game := g
				if state == "clone" {
					game = g.Clone()
				}
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				if len(game.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("detected delay still waiting")
				}
				game.PutTriggersOnStack()
				game.ResolveStack()
				target := game.FindPermanent(q.ID())
				game.DealDamageToPermanentFromSource(target, 2, saved)
				want := 0
				if operand == "player" {
					want = 2
				}
				if game.FindPermanent(q.ID()).Damage != want {
					t.Fatal("choice or saved redirection failed", game.FindPermanent(q.ID()).Damage, want)
				}
				if operand == "source" {
					recipient := game.GetPlayer(a.PlayerID())
					wantLife := 18
					if state == "copy" {
						wantLife = 20
						if game.GetPlayer(b.PlayerID()).Life() != 18 {
							t.Fatal("copied controller lost redirection")
						}
					}
					if recipient.Life() != wantLife {
						t.Fatal("wrong redirect player", recipient.Life(), wantLife)
					}
					if state == "copy" {
						game.DealDamageToPermanentFromSource(game.FindPermanent(q.ID()), 2, saved)
						if recipient.Life() != 18 || game.GetPlayer(b.PlayerID()).Life() != 18 || game.FindPermanent(q.ID()).Damage != 0 {
							t.Fatal("copies shared consumption")
						}
					}
					game.DealDamageToPermanentFromSource(game.FindPermanent(q.ID()), 1, saved)
					if game.FindPermanent(q.ID()).Damage != 1 {
						t.Fatal("redirection was not consumed once")
					}
					wantCalls := 1
					if state == "copy" {
						wantCalls = 2
					}
					if state != "clone" && (a.calls+b.calls != wantCalls || !a.source.SameObject(saved)) {
						t.Fatal("choice lost exact source or instance")
					}
				} else if a.calls+b.calls != 0 {
					t.Fatal("absent explicit source made a choice")
				}
				if state == "clone" && (a.calls != 0 || a.Life() != 20 || len(g.WaitingDelayedPermanentSources()) == 0) {
					t.Fatal("clone changed original")
				}
			})
		}
	}
}

func TestRONWaitingDelayedDamageChoiceProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEB", "Circle of Protection: Black", core.Black}, {"LEA", "Circle of Protection: Blue", core.Blue},
		{"LEA", "Circle of Protection: Green", core.Green}, {"LEA", "Circle of Protection: Red", core.Red},
		{"LEA", "Circle of Protection: White", core.White}, {"ATQ", "Circle of Protection: Artifacts", core.Colorless},
		{"LEG", "Greater Realm of Preservation", core.Black},
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
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			binding := mage.BindDamageSource("waiting choice")
			sourceCard := mage.NewArtifact("Choice producer", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.AtNext(mage.EachEndStep, mage.ChooseDamageSourceFrom(mage.You, []mage.DamageSourceRef{mage.This}, mage.BindDamageSource("future choice"))))))
			sourceCard.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(sourceCard, a.PlayerID())
			g.ChangeObjectColor(p.ID(), tc.color)
			saved := g.CaptureDamageSource(p.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.BouncePermanentToHand(p)
			bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
			payment := make([]mage.Cost, len(costs))
			for i, cost := range costs {
				payment[i] = cost()
			}
			card := mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), mage.Sequence(plan.waitingDelayedPermanentChoice(bindings), shield(bindings)))))
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			q := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err == nil || a.calls != 0 || g.StackSize() != 0 {
				t.Fatal("unpaid choice changed state")
			}
			a.ManaPool().Add(core.White, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.calls != 0 {
				t.Fatal("choice ran at announcement")
			}
			g.ResolveStack()
			if a.calls != 1 || !a.source.SameObject(saved) {
				t.Fatal("waiting choice body omitted departed source")
			}
			g.DealDamageToPlayerFromSource(a, 2, saved)
			g.DealDamageToPlayerFromSource(a, 1, saved)
			if a.Life() != 19 {
				t.Fatal("shield did not consume once")
			}
			if _, err := Compile(record); err != nil {
				t.Fatal("complete source-choice record rejected", err)
			}
		})
	}
}

func TestRONWaitingDelayedDamageChoiceInvalid(t *testing.T) {
	for _, expression := range []string{"ChooseADamageSource(AnyDamageSource())", "ChooseADamageSource(And([]))", "ChooseADamageSource(IsColor(Colorless))", "ChooseADamageSource(AnyDamageSource,You)"} {
		nodes, err := Parse(strings.NewReader(expression), "invalid-choice.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileChosenDamageSourceChoice(nodes[0]); err == nil {
			t.Fatal("invalid choice accepted", expression)
		}
	}
	for _, expression := range []string{"ControllerOfPermanent(Ref_TargetPermanent)", "Trigger_ThatPlayer", "ControllerOfPermanent(ThisPermanent())"} {
		nodes, err := Parse(strings.NewReader(expression), "invalid-player.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compilePlayerRef(nodes[0], scope{permanent: true}); err == nil {
			t.Fatal("invalid choice player accepted", expression)
		}
	}
}
