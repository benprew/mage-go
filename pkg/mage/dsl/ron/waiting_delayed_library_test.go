package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedLibraryGameplay(t *testing.T) {
	for _, action := range []string{
		`LookAtTheTopNumberCardsOfPlayersLibrary(ControllerOfPermanent(ThisPermanent),Integer(2),[LeaveRemainingCardsOnTopOfLibraryInSameOrder])`,
		`LookAtTheTopNumberCardsOfPlayersLibrary(You,PowerOfPermanent(ThisPermanent),[LeaveRemainingCardsOnTopOfLibraryInSameOrder])`,
		`MayHavePlayerAction(ControllerOfPermanent(ThisPermanent),Shuffle)`,
	} {
		for _, state := range []string{"live", "departed", "returned", "phased", "control", "copy"} {
			t.Run(action+state, func(t *testing.T) {
				a := &ronLibraryViewer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
				b := &ronLibraryViewer{BasePlayer: mage.NewBasePlayer("B")}
				g := mage.NewGame(a, b)
				for range 4 {
					a.AddToLibrary(mage.NewLand("A land"))
					b.AddToLibrary(mage.NewLand("B land"))
				}
				f := ronDelayedCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+action+`]))]))`)
				c := f()
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
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
					if len(got) != 1 || !got[0].SameObject(saved) || len(g.LibraryLooks(a.PlayerID())) != 0 || a.offers != 0 {
						t.Fatal("lost source or ran library action")
					}
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{clone, g} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					game.PutTriggersOnStack()
					game.ResolveStack()
					if strings.HasPrefix(action, "Look") {
						want := 2
						if state == "copy" {
							want = 4
						}
						looks := game.LibraryLooks(a.PlayerID())
						if len(looks) != want || len(game.LibraryLooks(b.PlayerID())) != 0 {
							t.Fatal("wrong private count", len(looks))
						}
						owner := a.PlayerID()
						if state == "control" && strings.Contains(action, "ControllerOf") {
							owner = b.PlayerID()
						}
						for _, look := range looks {
							if look.Owner != owner {
								t.Fatal("wrong library")
							}
						}
					}
					if len(game.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("wait survived trigger")
					}
					if game == clone && len(g.WaitingDelayedPermanentSources()) != 1 {
						t.Fatal("clone consumed original")
					}
				}
				if strings.HasPrefix(action, "May") {
					want := 1
					if state == "copy" {
						want = 2
					}
					if a.offers != want {
						t.Fatal("wrong shuffle offers", a.offers)
					}
					if state == "control" {
						if b.shuffles != want || a.shuffles != 0 {
							t.Fatal("wrong shuffle player")
						}
					} else if a.shuffles != want || b.shuffles != 0 {
						t.Fatal("wrong shuffle player")
					}
				}
			})
		}
	}
}
func TestRONWaitingDelayedLibraryInvalid(t *testing.T) {
	for _, count := range []string{`Integer(-1)`, `PowerOfPermanent(Ref_TargetPermanent)`, `PowerOfPermanent(You)`, `TheNumberOfCardsInPlayersHand(ActionPlayer)`, `ValueX`, `PowerOfPermanent(Trigger_ThatPermanent)`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([LookAtTheTopNumberCardsOfPlayersLibrary(You,` + count + `,[LeaveRemainingCardsOnTopOfLibraryInSameOrder])]))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid count accepted", count)
		}
	}
}
func TestRONWaitingDelayedLibraryProtection(t *testing.T) {
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
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([LookAtTheTopNumberCardsOfPlayersLibrary(ControllerOfPermanent(ThisPermanent),Minus(ManaValueOfPermanent(ThisPermanent),ManaValueOfPermanent(ThisPermanent)),[LeaveRemainingCardsOnTopOfLibraryInSameOrder])]))]))`)
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
			if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ChooseADamageSource") {
				t.Fatal("general aggregation was accepted", err)
			}
		})
	}
}
func TestRONWaitingDelayedLibraryInstances(t *testing.T) {
	g, a, b := ronGame()
	for range 12 {
		a.AddToLibrary(repositoryCard(t, "LEB", "Island"))
		b.AddToLibrary(repositoryCard(t, "LEB", "Island"))
	}
	factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([LookAtTheTopNumberCardsOfPlayersLibrary(You,PowerOfPermanent(Ref_TargetPermanent),[PutTheRemainingCardsOnTopOfLibraryInAnyOrder])]))])))`)
	first, second := factory(), factory()
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	p, q := g.PutOnBattlefield(first, a.PlayerID()), g.PutOnBattlefield(second, b.PlayerID())
	for _, subject := range []*mage.Permanent{p, q} {
		if err := g.ActivateAbilityByIndex(subject.ControllerID(), subject.ID(), 0, []uuid.UUID{subject.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(1), mage.Fixed(0))), q.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(first.ID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.MutablePermanent(q.ID()).GrantBaseAttr(core.Shroud)
	clone := g.Clone()
	for _, game := range []*mage.Game{clone, g} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if len(game.LibraryLooks(a.PlayerID())) != 2 || len(game.LibraryLooks(b.PlayerID())) != 6 {
			t.Fatal("shared target or amount read before resolution")
		}
		if game == clone && (len(g.LibraryLooks(a.PlayerID())) != 0 || len(g.WaitingDelayedPermanentSources()) != 2) {
			t.Fatal("clone shared library bindings")
		}
	}
}
