package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedDamageFrames(t *testing.T) {
	for _, name := range []string{"Lightning Bolt", "Prodigal Sorcerer"} {
		for _, event := range []string{"WhenAPlayerIsDealtDamage(SinglePlayer(You))", "WhenAPermanentIsDealtDamage(SinglePermanent(ThisPermanent))"} {
			for _, change := range []string{"live", "departed", "returned", "phased", "copy", "source-leaves", "prevented"} {
				if name == "Lightning Bolt" && change == "phased" {
					continue
				}
				t.Run(name+"/"+event+"/"+change, func(t *testing.T) {
					_, a, b := ronGame()
					chooser := &ronSourceChoicePlayer{BasePlayer: a}
					g := mage.NewGame(chooser, b)
					g.SetStep(core.PrecombatMain)
					src := `Card(Name:"Observer",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:9),Rules:[TriggerA(` + event + `,ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(Trigger_AmountOfDamageDealt)]))]))])`
					nodes, err := Parse(strings.NewReader(src), "observer.ron")
					if err != nil {
						t.Fatal(err)
					}
					factory, err := Compile(nodes[0])
					if err != nil {
						t.Fatal(err)
					}
					watcher := factory()
					watcher.SetOwner(a.PlayerID())
					p := g.PutOnBattlefield(watcher, a.PlayerID())
					target := a.PlayerID()
					if strings.HasPrefix(event, "WhenAPermanent") {
						target = p.ID()
					}
					if change == "prevented" {
						g.AddPreventionShield(target, 3)
					}
					card := repositoryCard(t, "LEA", name)
					card.SetOwner(b.PlayerID())
					var source *mage.Permanent
					var saved mage.DamageSource
					if name == "Lightning Bolt" {
						b.AddToHand(card)
						if err := g.CastSpellByID(b.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err == nil || g.StackSize() != 0 {
							t.Fatal("unpaid cast")
						}
						b.ManaPool().Add(core.Red, 1)
						if err := g.CastSpellByID(b.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
							t.Fatal(err)
						}
						saved = g.CaptureSpellDamageSource(g.StackPeek().ID)
					} else {
						source = g.PutOnBattlefield(card, b.PlayerID())
						if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{target}); err == nil || source.Tapped {
							t.Fatal("sickness rollback")
						}
						source.RevokeBaseAttr(core.AttrSummonSick)
						if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{target}); err != nil {
							t.Fatal(err)
						}
						saved = g.CaptureDamageSource(source.ID())
					}
					if len(g.WaitingDelayedDamageSources()) != 0 {
						t.Fatal("delay created early")
					}
					g.ResolveTopOfStack()
					if change == "prevented" {
						if len(g.WaitingDelayedDamageSources()) != 0 || g.StackSize() != 0 {
							t.Fatal("prevented event")
						}
						return
					}
					g.PutTriggersOnStack()
					if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("trigger copy")
					}
					g.ResolveStack()
					got := g.WaitingDelayedDamageSources()
					if len(got) != 1 || !got[0].SameObject(saved) {
						t.Fatal("RON damage event frame missing")
					}
					if source != nil {
						switch change {
						case "departed", "returned":
							g.BouncePermanentToHand(g.FindPermanent(source.ID()))
							if change == "returned" {
								b.RemoveFromHand(card.ID())
								g.PutOnBattlefield(card, a.PlayerID())
								g.ChangeObjectColor(card.ID(), core.Red)
							}
						case "phased":
							g.PhaseOut(source.ID())
						}
					} else if change == "returned" {
						b.RemoveFromGraveyard(card.ID())
						b.AddToHand(card)
						b.ManaPool().Add(core.Red, 1)
						if err := g.CastSpellByID(b.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
							t.Fatal(err)
						}
						g.ResolveStack()
					}
					if change == "source-leaves" {
						g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					}
					filterName := "Circle of Protection: Red"
					if name == "Prodigal Sorcerer" {
						filterName = "Circle of Protection: Blue"
					}
					filter, _ := chosenSourceFilter(t, "LEA", filterName)
					binding := mage.BindDamageSource("chosen")
					fixture := mage.NewArtifact("Choice", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.ChooseDamageSourceFromWaitingDelayedDamageBindings(mage.You, filter, binding))))
					if err := mage.ValidateCardDefinition(fixture.Name(), fixture); err != nil {
						t.Fatal(err)
					}
					choice := g.PutOnBattlefield(fixture, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), choice.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					g.ResolveTopOfStack()
					if chooser.calls != 1 || !chooser.source.SameObject(saved) {
						t.Fatal("exact filtered waiting source")
					}
					clone := g.Clone()
					for _, game := range []*mage.Game{g, clone} {
						game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						game.PutTriggersOnStack()
						if len(game.WaitingDelayedDamageSources()) != 0 || len(game.StackBoundDamageSources()) != 1 {
							t.Fatal("waiting to stack transition")
						}
						game.ResolveStack()
						want := 20
						amount := 1
						if name == "Lightning Bolt" {
							amount = 3
						}
						if strings.HasPrefix(event, "WhenAPermanent") {
							want += amount
						}
						if change == "copy" {
							want += amount
						}
						if game.GetPlayer(a.PlayerID()).Life() != want {
							t.Fatalf("delayed gameplay life=%d want=%d", game.GetPlayer(a.PlayerID()).Life(), want)
						}
					}
				})
			}
		}
	}
}

func TestRONWaitingDelayedDamageInvalid(t *testing.T) {
	for _, event := range []string{`WhenAPlayerIsDealtDamage(SinglePlayer(Opponent))`, `WhenAPermanentIsDealtDamage(SinglePermanent(HostPermanent))`, `WhenAPlayerIsDealtDamage(SinglePermanent(ThisPermanent))`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(` + event + `,ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(Trigger_AmountOfDamageDealt)]))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid event", event)
		}
	}
	src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(Trigger_AmountOfDamageDealt)]))]))])`
	nodes, err := Parse(strings.NewReader(src), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err == nil {
		t.Fatal("unbound event value")
	}
}
