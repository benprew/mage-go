package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedCombatProtection(t *testing.T) {
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
		for _, branch := range []string{"IsAttacking", "IsBlocking"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "copy", "clone"} {
				t.Run(tc.name+"/"+branch+"/"+state, func(t *testing.T) {
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
					body := `If(PermanentPassesFilter(ThisPermanent,` + branch + `),[PermanentDealsDamage(ThisPermanent,Integer(2),Player(You))])`
					factory := waitingCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))`)
					sourceCard := factory()
					sourceCard.SetOwner(a.PlayerID())
					source := g.PutOnBattlefield(sourceCard, a.PlayerID())
					g.ChangeObjectColor(source.ID(), tc.color)
					if branch == "IsAttacking" {
						g.GetCombat().AddAttackerIncarnation(source.ID(), g.CaptureDamageSource(source.ID()).Incarnation(), b.PlayerID())
					} else {
						attacker := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 1, 10), b.PlayerID())
						g.GetCombat().AddAttackerIncarnation(attacker.ID(), g.CaptureDamageSource(attacker.ID()).Incarnation(), a.PlayerID())
						g.GetCombat().AddBlockerIncarnation(source.ID(), g.CaptureDamageSource(source.ID()).Incarnation(), attacker.ID())
					}
					original := g.CaptureDamageSource(source.ID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("delay exists before resolution")
					}
					g.ResolveStack()
					switch state {
					case "departed", "returned":
						g.BouncePermanentToHand(source)
						if state == "returned" {
							a.RemoveFromHand(sourceCard.ID())
							g.PutOnBattlefield(sourceCard, b.PlayerID())
						}
					case "phased":
						g.PhaseOut(source.ID())
					case "cycle":
						g.PhaseIn(g.PhaseOut(source.ID()))
					}
					binding := mage.BindDamageSource("waiting source")
					bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
					payment := make([]mage.Cost, len(costs))
					for i, cost := range costs {
						payment[i] = cost()
					}
					consumer := mage.NewArtifact("RON consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), mage.Sequence(plan.waitingDelayedPermanentChoice(bindings), shield(bindings)))))
					if err := mage.ValidateCardDefinition(consumer.Name(), consumer); err != nil {
						t.Fatal(err)
					}
					p := g.PutOnBattlefield(consumer, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || g.StackSize() != 0 || chooser.calls != 0 {
						t.Fatal("unpaid choice changed state")
					}
					a.ManaPool().Add(core.White, 2)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if chooser.calls != 0 {
						t.Fatal("choice ran at announcement")
					}
					if state == "clone" {
						clone := g.Clone()
						clone.ResolveStack()
						clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						clone.PutTriggersOnStack()
						clone.ResolveStack()
						if clone.GetPlayer(a.PlayerID()).Life() != 20 || chooser.calls != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
							t.Fatal("clone lost source or shared lifetime")
						}
					}
					g.ResolveStack()
					if chooser.calls != 1 || !chooser.source.SameObject(original) {
						t.Fatal("conditional delayed body lost source choice")
					}
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("fired ability remained waiting")
					}
					g.PutTriggersOnStack()
					g.ResolveStack()
					want := 20
					if state == "copy" {
						want = 18
					}
					if a.Life() != want || b.Life() != 20 {
						t.Fatal("delayed damage or shield consumption failed")
					}
					g.DealDamageToPlayerFromSource(chooser, 1, original)
					if state == "departed" || state == "returned" || state == "phased" || state == "cycle" {
						if a.Life() != want {
							t.Fatal("false combat branch consumed shield")
						}
						g.DealDamageToPlayerFromSource(chooser, 1, original)
					}
					if a.Life() != want-1 {
						t.Fatal("shield was not consumed")
					}
					if _, err := Compile(record); err != nil {
						t.Fatal("complete source-choice record rejected", err)
					}
				})
			}
		}
	}
}

func waitingCombatFactory(t *testing.T, rules string) mage.CardFactory {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Combat source",Typeline:(Supertypes:[],Cardtypes:[Artifact,Creature],Subtypes:[]),CardPT:(Power:1,Toughness:10),Rules:[`+rules+`])`), "waiting-combat.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Combat source", factory()); err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONWaitingDelayedCombatInvalid(t *testing.T) {
	for _, body := range []string{
		`If(PermanentPassesFilter(ThisPermanent(),IsAttacking),[GainLife(Integer(1))])`,
		`If(PermanentPassesFilter(Ref_TargetPermanent,IsBlocking),[GainLife(Integer(1))])`,
		`If(PermanentPassesFilter(HostPermanent,IsAttacking),[GainLife(Integer(1))])`,
		`If(PermanentPassesFilter(Trigger_ThatPermanent,IsBlocking),[GainLife(Integer(1))])`,
		`If(PermanentPassesFilter(ThisPermanent,IsBlocking),[])`,
		`If(PermanentPassesFilter(ThisPermanent,IsBlocking()),[GainLife(Integer(1))])`,
		`Unless(PermanentPassesFilter(ThisPermanent,IsAttacking),[GainLife(Integer(1))])`,
		`IfElse(PermanentPassesFilter(ThisPermanent,IsBlocking),[GainLife(Integer(1))],[GainLife(Integer(2))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+body+`]))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid combat branch accepted", body)
		}
	}
}

func TestRONWaitingDelayedCombatBodyInstances(t *testing.T) {
	for _, combat := range []string{"IsAttacking", "IsBlocking"} {
		t.Run(combat, func(t *testing.T) {
			g, a, b := ronGame()
			factory := waitingCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([If(PermanentPassesFilter(Ref_TargetPermanent,`+combat+`),[If(PermanentPassesFilter(Ref_TargetPermanent,`+combat+`),[PermanentDealsDamage(ThisPermanent,Integer(2),Player(You))])])]))])))`)
			q := g.PutOnBattlefield(mage.NewCreature("Subject", "", 1, 10), b.PlayerID())
			qSource := g.CaptureDamageSource(q.ID())
			cards := []mage.Card{factory(), factory()}
			if cards[0].Abilities()[0].(*mage.ActionDefinition).Targets()[0] == cards[1].Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("factory shared a target declaration")
			}
			sources := make([]mage.DamageSource, 1, 3)
			sources[0] = qSource
			for i, player := range []*mage.BasePlayer{a, b} {
				cards[i].SetOwner(player.PlayerID())
				p := g.PutOnBattlefield(cards[i], player.PlayerID())
				sources = append(sources, g.CaptureDamageSource(p.ID()))
				if err := g.ActivateAbilityByIndex(player.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			}
			got := g.WaitingDelayedPermanentSources()
			if len(got) != 3 {
				t.Fatal("body-only self declarations missing beside shared target", len(got))
			}
			for _, source := range sources {
				found := false
				for _, ref := range got {
					found = found || ref.SameObject(source)
				}
				if !found {
					t.Fatal("lost exact body or target reference")
				}
			}
			if combat == "IsAttacking" {
				g.GetCombat().AddAttackerIncarnation(q.ID(), qSource.Incarnation(), a.PlayerID())
			} else {
				attacker := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 1, 10), a.PlayerID())
				g.GetCombat().AddAttackerIncarnation(attacker.ID(), g.CaptureDamageSource(attacker.ID()).Incarnation(), b.PlayerID())
				g.GetCombat().AddBlockerIncarnation(q.ID(), qSource.Incarnation(), attacker.ID())
			}
			clone := g.Clone()
			clone.RemoveFromCombat(q.ID())
			for _, game := range []*mage.Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
			}
			if a.Life() != 18 || b.Life() != 18 || clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("body lost independent controller or resolution-time combat state")
			}
		})
	}
}
