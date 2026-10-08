package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedCollectionActionProtection(t *testing.T) {
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
		for _, action := range []string{"TapEachPermanent", "UntapEachPermanent", "PutEachPermanentIntoItsOwnersHand", "DestroyEachPermanent", "DestroyEachPermanentNoRegen", "damage"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control", "copy", "clone"} {
				t.Run(tc.name+"/"+action+"/"+state, func(t *testing.T) {
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
					_, a, b := ronGame()
					chooser := &ronSourceChoicePlayer{BasePlayer: a}
					g := mage.NewGame(chooser, b)
					body := action + `(SinglePermanent(ThisPermanent))`
					if action == "damage" {
						body = `PermanentDealsDamage(ThisPermanent,Integer(1),EachPermanent(SinglePermanent(ThisPermanent)))`
					}
					factory := collectionActionSourceFactory(t, action == "damage", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))])))`)
					card := factory()
					card.SetOwner(a.PlayerID())
					p := g.PutOnBattlefield(card, a.PlayerID())
					p.Tapped = action == "UntapEachPermanent"
					g.ChangeObjectColor(p.ID(), tc.color)
					saved := g.CaptureDamageSource(p.ID())
					donor := g.PutOnBattlefield(mage.NewCreature("Donor", "", 1, 4), b.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{donor.ID()}); err != nil {
						t.Fatal(err)
					}
					if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("delay exists at announcement")
					}
					g.ResolveStack()
					switch state {
					case "departed", "returned":
						g.BouncePermanentToHand(p)
						if state == "returned" {
							a.RemoveFromHand(card.ID())
							p = g.PutOnBattlefield(card, b.PlayerID())
							p.Tapped = action == "UntapEachPermanent"
						}
					case "phased":
						g.PhaseOut(p.ID())
					case "cycle":
						g.PhaseIn(g.PhaseOut(p.ID()))
					case "control":
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
					before := p.Tapped
					binding := mage.BindDamageSource("tap source")
					bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
					payment := make([]mage.Cost, len(costs))
					for i, cost := range costs {
						payment[i] = cost()
					}
					consumer := g.PutOnBattlefield(mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), mage.Sequence(plan.waitingDelayedPermanentChoice(bindings), shield(bindings))))), a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), consumer.ID(), 0, nil); err == nil || g.StackSize() != 0 || chooser.calls != 0 {
						t.Fatal("failed payment changed state")
					}
					a.ManaPool().Add(core.White, 2)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), consumer.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if chooser.calls != 0 {
						t.Fatal("choice ran at announcement")
					}
					for range 3 {
						got := g.WaitingDelayedPermanentSources()
						if len(got) != 2 || !collectionActionHasSource(got, saved) || p.Tapped != before {
							t.Fatal("query lost source or executed tap action")
						}
					}
					var clone *mage.Game
					if state == "clone" {
						clone = g.Clone()
						clone.ResolveStack()
						if chooser.calls != 0 {
							t.Fatal("clone changed chooser")
						}
					}
					g.ResolveStack()
					if chooser.calls != 1 || !chooser.source.SameObject(saved) {
						t.Fatal("tap reference missing from source choice")
					}
					for _, game := range []*mage.Game{g, clone} {
						if game == nil {
							continue
						}
						game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 2, saved)
						if game.GetPlayer(a.PlayerID()).Life() != 20 {
							t.Fatal("chosen source not prevented")
						}
						game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						if len(game.WaitingDelayedPermanentSources()) != 0 {
							t.Fatal("detected delay still waits")
						}
						game.PutTriggersOnStack()
						game.ResolveStack()
						live := game.FindPermanentIncludingPhased(p.ID())
						present := state != "departed" && state != "returned" && state != "phased"
						leaves := action == "PutEachPermanentIntoItsOwnersHand" || strings.HasPrefix(action, "Destroy")
						if present && leaves {
							if live != nil {
								t.Fatal("collection did not move its original subject")
							}
						} else if state != "departed" {
							if live == nil {
								t.Fatal("collection moved an absent or returned subject")
							}
							wantTap := before
							if present && action == "TapEachPermanent" {
								wantTap = true
							}
							if present && action == "UntapEachPermanent" {
								wantTap = false
							}
							wantDamage := 0
							if present && action == "damage" {
								wantDamage = 1
								if state == "copy" {
									wantDamage = 2
								}
							}
							if live.Tapped != wantTap || live.Damage != wantDamage {
								t.Fatal("collection affected wrong state", live.Tapped, live.Damage)
							}
						}
						game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 1, saved)
						if game.GetPlayer(a.PlayerID()).Life() != 19 {
							t.Fatal("shield not consumed")
						}
						if game == g && clone != nil && len(clone.WaitingDelayedPermanentSources()) != 2 {
							t.Fatal("clone shares lifetime")
						}
					}
					if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ChooseADamageSource") {
						t.Fatal("unfinished aggregation accepted", err)
					}
				})
			}
		}
	}
}

func TestRONWaitingDelayedCollectionActionInvalid(t *testing.T) {
	for _, action := range []string{"TapEachPermanent", "UntapEachPermanent", "PutEachPermanentIntoItsOwnersHand", "DestroyEachPermanent", "DestroyEachPermanentNoRegen", "damage"} {
		for _, args := range []string{"", "SinglePermanent(ThisPermanent),AnyPermanent", "SinglePermanent(ThisPermanent())", "SinglePermanent(You)", "SinglePermanent(Ref_TargetPermanent)", "SinglePermanent(Trigger_ThatPermanent)", "SinglePermanent(HostPermanent)", "SinglePermanent(TheChosenPermanent)", "And([])", "Or([])"} {
			body := action + `(` + args + `)`
			if action == "damage" {
				body = `PermanentDealsDamage(ThisPermanent,Integer(1),EachPermanent(` + args + `))`
			}
			nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))])`), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid delayed tap accepted", action, args)
			}
		}
	}
}

func TestRONWaitingDelayedCollectionActionInstances(t *testing.T) {
	for _, action := range []string{"TapEachPermanent", "UntapEachPermanent"} {
		t.Run(action, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`(Or([SinglePermanent(ThisPermanent),SinglePermanent(Ref_TargetPermanent)]))]))])))`)
			sources := make([]*mage.Permanent, 0, 2)
			targets := make([]*mage.Permanent, 0, 2)
			for _, player := range []*mage.BasePlayer{a, b} {
				card := factory()
				card.SetOwner(player.PlayerID())
				source := g.PutOnBattlefield(card, player.PlayerID())
				targetCard := mage.NewCreature("Target", "", 1, 3)
				targetCard.SetOwner(player.PlayerID())
				target := g.PutOnBattlefield(targetCard, player.PlayerID())
				source.Tapped = action == "UntapEachPermanent"
				target.Tapped = source.Tapped
				if err := g.ActivateAbilityByIndex(player.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				sources = append(sources, source)
				targets = append(targets, target)
			}
			if len(g.WaitingDelayedPermanentSources()) != 4 {
				t.Fatal("factories shared source or target binding")
			}
			returnedCard := targets[0].Card
			g.BouncePermanentToHand(targets[0])
			a.RemoveFromHand(returnedCard.ID())
			returned := g.PutOnBattlefield(returnedCard, b.PlayerID())
			returned.Tapped = action == "UntapEachPermanent"
			g.MutablePermanent(targets[1].ID()).GrantBaseAttr(core.Shroud)
			clone := g.Clone()
			for _, game := range []*mage.Game{clone, g} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				for _, source := range sources {
					if game.FindPermanent(source.ID()).Tapped != (action == "TapEachPermanent") {
						t.Fatal("per-instance source action missing")
					}
				}
				if game.FindPermanent(returned.ID()).Tapped != (action == "UntapEachPermanent") {
					t.Fatal("target action followed returned incarnation")
				}
				if game.FindPermanent(targets[1].ID()).Tapped != (action == "TapEachPermanent") {
					t.Fatal("saved delayed subject was retargeted")
				}
				if len(game.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("resolved delay still supplies sources")
				}
				if game == clone && len(g.WaitingDelayedPermanentSources()) != 4 {
					t.Fatal("clone consumed original delays")
				}
			}
		})
	}
}

func collectionActionHasSource(sources []mage.DamageSource, saved mage.DamageSource) bool {
	for _, source := range sources {
		if source.SameObject(saved) {
			return true
		}
	}
	return false
}

func TestRONWaitingDelayedCollectionActionDamageComposition(t *testing.T) {
	for _, tc := range []struct {
		set                  string
		first, second, other int
	}{
		{`And([Ref_TargetPermanents,Ref_TargetPermanents])`, 1, 1, 0},
		{`And([Ref_TargetPermanents,HasAbility(Flying),IsCardtype(Creature)])`, 1, 0, 0},
		{`Or([Ref_TargetPermanents,HasAbility(Flying)])`, 1, 1, 1},
		{`And([And([Ref_TargetPermanents]),HasAbility(Flying)])`, 1, 0, 0},
		{`Or([SinglePermanent(ThisPermanent),Ref_TargetPermanents])`, 1, 1, 0},
	} {
		t.Run(tc.set, func(t *testing.T) {
			g, a, b := ronGame()
			factory := collectionActionSourceFactory(t, true, `Activated(PayMana([]),Targeted([NumberTargetPermanents(Integer(2),IsCardtype(Creature))],ActionList([PermanentDealsDamage(ThisPermanent,Integer(1),EachPermanent(`+tc.set+`))])))`)
			card := factory()
			source := g.PutOnBattlefield(card, a.PlayerID())
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 4, mage.WithKeyword(core.Flying)), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 4), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 4, mage.WithKeyword(core.Flying)), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID(), second.ID()}); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{clone, g} {
				game.ResolveStack()
				for id, want := range map[uuid.UUID]int{first.ID(): tc.first, second.ID(): tc.second, other.ID(): tc.other} {
					if p := game.FindPermanent(id); p == nil || p.Damage != want {
						t.Fatal("wrong collection damage", id, want)
					}
				}
			}
		})
	}
}

func TestRONWaitingDelayedCollectionActionLateAttachment(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapEachPermanent(AttachedToAPermanent(SinglePermanent(ThisPermanent)))]))]))`)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if got := g.WaitingDelayedPermanentSources(); len(got) != 1 {
		t.Fatal("empty attachment collection lost source")
	}
	aura := g.PutOnBattlefield(mage.NewEnchantment("Late Aura", "", mage.WithSubTypes("Aura")), b.PlayerID())
	g.Attach(aura.ID(), p.ID())
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if !game.FindPermanent(aura.ID()).Tapped || game.FindPermanent(p.ID()).Tapped {
			t.Fatal("attachment membership was not read at resolution")
		}
	}
}

func collectionActionSourceFactory(t *testing.T, creature bool, rules string) mage.CardFactory {
	t.Helper()
	metadata := `Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[])`
	if creature {
		metadata = `Typeline:(Supertypes:[],Cardtypes:[Artifact,Creature],Subtypes:[]),CardPT:(Power:1,Toughness:4)`
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Collection source",`+metadata+`,Rules:[`+rules+`])`), "collection-source.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory
}
