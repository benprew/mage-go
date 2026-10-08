package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedCombatExitGameplay(t *testing.T) {
	for _, action := range []string{"RemoveCreatureFromCombat", "RemoveCreatureFromCombatAndUnblockBlockers"} {
		for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "copy", "two blockers", "other removed"} {
			t.Run(action+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				f := waitingCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`(ThisPermanent)]))]))`)
				c := f()
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 10), b.PlayerID())
				g.GetCombat().AddAttackerIncarnation(attacker.ID(), g.CaptureDamageSource(attacker.ID()).Incarnation(), a.PlayerID())
				g.GetCombat().AddBlockerIncarnation(p.ID(), saved.Incarnation(), attacker.ID())
				if state == "two blockers" || state == "other removed" {
					other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 10), a.PlayerID())
					g.GetCombat().AddBlockerIncarnation(other.ID(), g.CaptureDamageSource(other.ID()).Incarnation(), attacker.ID())
					if state == "other removed" {
						g.RemoveFromCombat(other.ID())
					}
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if len(g.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("delay created before resolution")
				}
				g.ResolveStack()
				switch state {
				case "departed", "returned":
					g.BouncePermanentToHand(p)
					if state == "returned" {
						a.RemoveFromHand(c.ID())
						p = g.PutOnBattlefield(c, a.PlayerID())
						g.GetCombat().AddBlockerIncarnation(p.ID(), g.CaptureDamageSource(p.ID()).Incarnation(), attacker.ID())
					}
				case "phased", "cycle":
					ids := g.PhaseOut(p.ID())
					if state == "cycle" {
						g.PhaseIn(ids)
					}
				}
				before := g.GetCombat().IsBlocking(p.ID())
				for range 3 {
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(saved) || g.GetCombat().IsBlocking(p.ID()) != before || !g.GetCombat().GroupFor(attacker.ID()).Blocked {
						t.Fatal("query changed combat or lost identity", got)
					}
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{clone, g} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(game.WaitingDelayedPermanentSources()) != 0 || game.GetCombat().IsBlocking(p.ID()) != before {
						t.Fatal("detection executed body or kept wait")
					}
					game.PutTriggersOnStack()
					wantStack := 1
					if state == "copy" {
						wantStack = 2
					}
					if game.StackSize() != wantStack {
						t.Fatal("lost delayed body")
					}
					game.ResolveStack()
					if game.GetCombat().IsBlocking(p.ID()) != (state == "returned") {
						t.Fatal("wrong combat membership")
					}
					wantBlocked := action == "RemoveCreatureFromCombat" || (state != "live" && state != "copy")
					if game.GetCombat().GroupFor(attacker.ID()).Blocked != wantBlocked {
						t.Fatal("wrong historical blocked status")
					}
					if game == clone && (len(g.WaitingDelayedPermanentSources()) != 1 || g.GetCombat().IsBlocking(p.ID()) != before) {
						t.Fatal("clone changed original")
					}
				}
			})
		}
	}
}

func TestRONWaitingDelayedCombatExitInvalid(t *testing.T) {
	for _, action := range []string{"RemoveCreatureFromCombat", "RemoveCreatureFromCombatAndUnblockBlockers"} {
		for _, subject := range []string{"", "You", "ThisPermanent()", "Ref_TargetPermanent", "Trigger_ThatCreature", "ActionPermanent"} {
			body := action + `(` + subject + `)`
			nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))])`), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid delayed subject accepted", body)
			}
		}
	}
}

func TestRONWaitingDelayedCombatExitInstances(t *testing.T) {
	for _, action := range []string{"RemoveCreatureFromCombat", "RemoveCreatureFromCombatAndUnblockBlockers"} {
		t.Run(action, func(t *testing.T) {
			g, a, b := ronGame()
			f := waitingCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`(Ref_TargetPermanent)]))])))`)
			p, q := g.PutOnBattlefield(f(), a.PlayerID()), g.PutOnBattlefield(f(), b.PlayerID())
			if p.Card.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == q.Card.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("shared targets")
			}
			g.GetCombat().AddAttackerIncarnation(p.ID(), g.CaptureDamageSource(p.ID()).Incarnation(), b.PlayerID())
			g.GetCombat().AddAttackerIncarnation(q.ID(), g.CaptureDamageSource(q.ID()).Incarnation(), a.PlayerID())
			for _, source := range []*mage.Permanent{p, q} {
				if err := g.ActivateAbilityByIndex(source.ControllerID(), source.ID(), 0, []uuid.UUID{source.ID()}); err != nil {
					t.Fatal(err)
				}
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			if len(g.WaitingDelayedPermanentSources()) != 2 {
				t.Fatal("lost independent targets")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 3 {
				t.Fatal("lost copied frame")
			}
			g.ResolveStack()
			if g.GetCombat().IsAttacking(p.ID()) || g.GetCombat().IsAttacking(q.ID()) {
				t.Fatal("wrong target binding")
			}
		})
	}
}

func TestRONWaitingDelayedCombatExitProtection(t *testing.T) {
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
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([RemoveCreatureFromCombatAndUnblockBlockers(ThisPermanent)]))]))`)
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
				t.Fatal("combat-exit delay remained waiting")
			}
			if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ChooseADamageSource") {
				t.Fatal("general aggregation was accepted", err)
			}
		})
	}
}
