package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedManaGameplay(t *testing.T) {
	for _, actions := range []string{`PlayerAction(ControllerOfPermanent(ThisPermanent),LoseUnspentMana)`, `PlayerActions(ControllerOfPermanent(ThisPermanent),[LoseUnspentMana])`} {
		for _, state := range []string{"live", "departed", "returned", "phased", "control", "copy"} {
			t.Run(actions+state, func(t *testing.T) {
				g, a, b := ronGame()
				f := ronDelayedCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+actions+`]))]))`)
				c := f()
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				a.ManaPool().Add(core.Blue, 2)
				b.ManaPool().AddRestricted(core.Colorless, 3, core.ArtifactSpellsOnly{})
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if len(g.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("early delay")
				}
				g.ResolveStack()
				switch state {
				case "departed", "returned":
					g.BouncePermanentToHand(p)
					if state == "returned" {
						a.RemoveFromHand(c.ID())
						g.PutOnBattlefield(c, b.PlayerID())
					}
				case "phased":
					g.PhaseOut(p.ID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				for range 3 {
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(saved) || a.ManaPool().TotalMana() != 2 || b.ManaPool().TotalMana() != 3 {
						t.Fatal("query lost source or cleared mana", got)
					}
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{clone, g} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					game.PutTriggersOnStack()
					wantStack := 1
					if state == "copy" {
						wantStack = 2
					}
					if game.StackSize() != wantStack || game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 2 {
						t.Fatal("trigger timing")
					}
					game.ResolveStack()
					wantA, wantB := 0, 3
					if state == "control" {
						wantA, wantB = 2, 0
					}
					if game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != wantA || game.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != wantB || game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
						t.Fatal("wrong pool or mana burn")
					}
					if len(game.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("wait survived trigger")
					}
					if game == clone && (len(g.WaitingDelayedPermanentSources()) != 1 || a.ManaPool().TotalMana() != 2 || b.ManaPool().TotalMana() != 3) {
						t.Fatal("clone changed original")
					}
				}
			})
		}
	}
}

func TestRONWaitingDelayedManaInvalid(t *testing.T) {
	for _, action := range []string{`PlayerAction(You,LoseUnspentMana())`, `PlayerAction(Ref_TargetPermanent,LoseUnspentMana)`, `PlayerAction(ActionPlayer,LoseUnspentMana)`, `PlayerAction(ControllerOfPermanent(Ref_TargetPermanent),LoseUnspentMana)`, `PlayerAction(Trigger_Player,LoseUnspentMana)`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + action + `]))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", action)
		}
	}
}
func TestRONWaitingDelayedManaProtection(t *testing.T) {
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
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(ControllerOfPermanent(ThisPermanent),LoseUnspentMana)]))]))`)
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
				t.Fatal("shield delay remained waiting")
			}
			if _, err := Compile(record); err != nil {
				t.Fatal("complete source-choice record rejected", err)
			}
		})
	}
}

func TestRONWaitingDelayedManaInstances(t *testing.T) {
	g, a, b := ronGame()
	factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(ControllerOfPermanent(Ref_TargetPermanent),LoseUnspentMana)]))])))`)
	first, second := factory(), factory()
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	p, q := g.PutOnBattlefield(first, a.PlayerID()), g.PutOnBattlefield(second, b.PlayerID())
	for _, source := range []*mage.Permanent{p, q} {
		if err := g.ActivateAbilityByIndex(source.ControllerID(), source.ID(), 0, []uuid.UUID{source.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	savedP, savedQ := g.CaptureDamageSource(p.ID()), g.CaptureDamageSource(q.ID())
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(first.ID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: q.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	got := g.WaitingDelayedPermanentSources()
	if len(got) != 2 || (!got[0].SameObject(savedP) && !got[1].SameObject(savedP)) || (!got[0].SameObject(savedQ) && !got[1].SameObject(savedQ)) {
		t.Fatal("separate target bindings lost", got)
	}
	a.ManaPool().Add(core.Blue, 2)
	b.ManaPool().Add(core.Red, 3)
	clone := g.Clone()
	for _, game := range []*mage.Game{clone, g} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		if game.StackSize() != 3 {
			t.Fatal("independent delays lost")
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 || game.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 3 {
			t.Fatal("returned target or stale controller used")
		}
		if game == clone && a.ManaPool().TotalMana() != 2 {
			t.Fatal("clone changed original")
		}
	}
}
