package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedRegenerationProtection(t *testing.T) {
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
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([RegeneratePermanent(ThisPermanent)]))]))`)
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
				t.Fatal("regeneration delay remained waiting")
			}
			if _, err := Compile(record); err != nil {
				t.Fatal("complete source-choice record rejected", err)
			}
		})
	}
}

func TestRONWaitingDelayedRegenerationGameplay(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			factory := waitingCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([RegeneratePermanent(ThisPermanent)]))]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
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
					a.RemoveFromHand(card.ID())
					p = g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			for range 3 {
				refs := g.WaitingDelayedPermanentSources()
				if len(refs) != 1 || !refs[0].SameObject(saved) || len(g.WaitingRegenerationPermanentSources()) != 0 {
					t.Fatal("lost identity or query created shield")
				}
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{clone, g} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				if len(game.WaitingDelayedPermanentSources()) != 0 || len(game.WaitingRegenerationPermanentSources()) != 0 {
					t.Fatal("detection created shield or retained wait")
				}
				game.PutTriggersOnStack()
				game.ResolveStack()
				want := 1
				if state == "departed" || state == "returned" || state == "phased" {
					want = 0
				}
				if state == "copy" {
					want = 2
				}
				refs := game.WaitingRegenerationPermanentSources()
				if len(refs) != min(want, 1) {
					t.Fatalf("shield subjects=%d want=%d", len(refs), min(want, 1))
				}
				if want > 0 {
					live := game.FindPermanent(p.ID())
					game.GetCombat().AddAttackerIncarnation(live.ID(), saved.Incarnation(), b.PlayerID())
					game.DealDamageToPermanent(live, 2, uuid.Nil)
					for range want {
						game.DestroyPermanent(game.FindPermanent(p.ID()))
						live = game.FindPermanent(p.ID())
						if live == nil || !live.Tapped || live.Damage != 0 || game.GetCombat().IsAttacking(p.ID()) {
							t.Fatal("regeneration did not replace destruction")
						}
					}
					game.DestroyPermanent(live)
					if game.FindPermanent(p.ID()) != nil {
						t.Fatal("shield not consumed")
					}
				} else if state == "returned" {
					game.DestroyPermanent(game.FindPermanent(p.ID()))
					if game.FindPermanent(p.ID()) != nil {
						t.Fatal("returned object inherited shield")
					}
				}
				if game == clone && len(g.WaitingDelayedPermanentSources()) != 1 {
					t.Fatal("clone changed original waiting lifetime")
				}
			}
		})
	}
}

func TestRONWaitingDelayedRegenerationInstances(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([RegeneratePermanent(Ref_TargetPermanent)]))])))`)
	subjects := make([]*mage.Permanent, 0, 2)
	for _, player := range []*mage.BasePlayer{a, b} {
		target := g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 4), player.PlayerID())
		card := factory()
		card.SetOwner(player.PlayerID())
		source := g.PutOnBattlefield(card, player.PlayerID())
		if err := g.ActivateAbilityByIndex(player.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if player == a {
			g.BouncePermanentToHand(target)
			player.RemoveFromHand(target.ID())
			target = g.PutOnBattlefield(target.Card, player.PlayerID())
		} else {
			g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
		}
		subjects = append(subjects, target)
	}
	if len(g.WaitingDelayedPermanentSources()) != 2 {
		t.Fatal("saved targets lost or implicit source added")
	}
	for _, game := range []*mage.Game{g.Clone(), g} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		for i, subject := range subjects {
			game.DestroyPermanent(game.FindPermanent(subject.ID()))
			if (game.FindPermanent(subject.ID()) != nil) != (i == 1) {
				t.Fatal("target followed return or rechecked shroud")
			}
		}
	}
}

func TestRONWaitingDelayedRegenerationInvalid(t *testing.T) {
	for _, expression := range []string{`RegeneratePermanent`, `RegeneratePermanent()`, `RegeneratePermanent(ThisPermanent,ThisPermanent)`, `RegeneratePermanent(You)`, `RegeneratePermanent(ThisPermanent())`, `RegeneratePermanent(HostPermanent)`, `RegeneratePermanent(Ref_TargetPermanent)`, `RegeneratePermanent(Trigger_ThatPermanent)`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+expression+`]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid regeneration accepted", expression)
		}
	}
}

func TestRONWaitingDelayedRegenerationHost(t *testing.T) {
	g, a, b := ronGame()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Host source",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([RegeneratePermanent(HostPermanent)]))]))])`), "host.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	host := g.PutOnBattlefield(mage.NewCreature("First host", "", 1, 4), a.PlayerID())
	later := g.PutOnBattlefield(mage.NewCreature("Later host", "", 1, 4), b.PlayerID())
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	g.Attach(source.ID(), host.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("implicit host inferred a source reference")
	}
	g.Attach(source.ID(), later.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.HasRegenerationShield(host.ID()) || !g.HasRegenerationShield(later.ID()) {
		t.Fatal("implicit host did not use resolution attachment")
	}
}

func TestRONWaitingDelayedRegenerationRecords(t *testing.T) {
	for _, tc := range []struct {
		name   string
		color  core.Color
		amount int
	}{
		{"Drudge Skeletons", core.Black, 1}, {"Living Wall", core.Black, 1}, {"Uthden Troll", core.Red, 1}, {"Wall of Bone", core.Black, 1}, {"Wall of Brambles", core.Green, 1}, {"Will-o'-the-Wisp", core.Black, 1}, {"Regeneration", core.Green, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, "LEB", tc.name)
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			subject := source
			if tc.name == "Regeneration" {
				hostCard := mage.NewCreature("Host", "", 1, 4)
				hostCard.SetOwner(a.PlayerID())
				subject = g.PutOnBattlefield(hostCard, a.PlayerID())
				g.Attach(source.ID(), subject.ID())
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || g.StackSize() != 0 {
				t.Fatal("unpaid regeneration changed state")
			}
			a.ManaPool().Add(tc.color, tc.amount)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.HasRegenerationShield(subject.ID()) {
				t.Fatal("early regeneration shield")
			}
			g.ResolveStack()
			g.DestroyPermanent(subject)
			if live := g.FindPermanent(subject.ID()); live == nil || !live.Tapped || g.HasRegenerationShield(subject.ID()) {
				t.Fatal("record did not regenerate subject")
			}
			g.DestroyPermanent(g.FindPermanent(subject.ID()))
			if g.FindPermanent(subject.ID()) != nil {
				t.Fatal("record shield was not consumed")
			}
		})
	}
}
