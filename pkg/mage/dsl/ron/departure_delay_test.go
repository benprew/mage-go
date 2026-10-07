package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDepartureDelayAnnouncementAndResolution(t *testing.T) {
	for _, tc := range []struct {
		name, set string
		mana      int
	}{{"War Barge", "DRK", 3}, {"Sandals of Abdallah", "ARN", 2}} {
		for _, change := range []string{"shroud", "return", "phase", "phase-cycle", "source-departure", "tapped"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
				card := mage.NewCreature("Target", "", 2, 3)
				card.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(card, b.PlayerID())
				a.ManaPool().Add(core.Colorless, tc.mana)
				if change == "tapped" {
					g.TapPermanent(source)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); tc.name == "Sandals of Abdallah" {
						if err == nil || a.ManaPool().Count(core.Colorless) != tc.mana || g.StackSize() != 0 {
							t.Fatal("tapped activation spent resources")
						}
						return
					} else if err != nil {
						t.Fatal(err)
					}
				} else if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "shroud":
					g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
				case "return":
					g.BouncePermanentToHand(g.FindPermanent(target.ID()))
					b.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				case "phase":
					g.PhaseOut(target.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "source-departure":
					g.ExilePermanent(g.FindPermanent(source.ID()))
				}
				g.ResolveStack()
				legal := change == "tapped" || change == "phase-cycle" || change == "source-departure"
				if g.FindPermanentIncludingPhased(target.ID()).HasKeyword(core.Islandwalk) != legal {
					t.Fatal("target legality or departed source changed grant")
				}
				want := 0
				if legal && (change != "source-departure" || tc.name != "War Barge") {
					want = 1
				}
				if len(g.DelayedTriggers()) != want {
					t.Fatalf("subscriptions=%d want=%d", len(g.DelayedTriggers()), want)
				}
			})
		}
	}
}

func TestRONDepartureDelayInstancesAndPendingExpiry(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "War Barge")
	second := repositoryCard(t, "DRK", "War Barge")
	firstTarget := card.Abilities()[0].(*mage.ActionDefinition).Targets()[0]
	if firstTarget == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factory instances share targets")
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	otherSource := g.PutOnBattlefield(second, b.PlayerID())
	targets := make([]uuid.UUID, 0, 3)
	for range 3 {
		targets = append(targets, g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 3), b.PlayerID()).ID())
	}
	for i, id := range targets {
		controller, permanent := a, source
		if i == 2 {
			controller, permanent = b, otherSource
		}
		controller.ManaPool().Add(core.Colorless, 3)
		if err := g.ActivateAbilityByIndex(controller.PlayerID(), permanent.ID(), 0, []uuid.UUID{id}); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if len(g.DelayedTriggers()) != 3 {
		t.Fatal("pending resolutions overwrote subscriptions")
	}
	clone := g.Clone()
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("wrong source subscription")
	}
	g.RunStepWithPriority(core.Cleanup)
	if len(g.DelayedTriggers()) != 0 {
		t.Fatal("cleanup retained subscriptions")
	}
	g.ResolveStack()
	for i, id := range targets {
		if (g.FindPermanent(id) != nil) != (i == 2) || clone.FindPermanent(id) == nil {
			t.Fatal("captured targets, pending expiry or clone isolation failed")
		}
	}
	clone.ExilePermanent(clone.FindPermanent(otherSource.ID()))
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.FindPermanent(targets[2]) != nil || clone.FindPermanent(targets[0]) == nil {
		t.Fatal("independent factory binding failed")
	}
}

func TestRONDepartureDelayStrictLoading(t *testing.T) {
	files := fstest.MapFS{}
	for _, tc := range []struct{ name, set string }{{"War Barge", "DRK"}, {"Sandals of Abdallah", "ARN"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), tc.set+".ron")
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			if node.Fields["Name"].String == tc.name {
				files[tc.set+".ron"] = &fstest.MapFile{Data: []byte(strings.Split(string(data), "\n")[node.Position.Line-1])}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"War Barge", "Sandals of Abdallah"} {
		card, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := mage.ValidateCardDefinition(name, card); err != nil {
			t.Fatal(err)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Unsupported",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("partial collection loaded")
	}
	if _, err := registry.CreateCard("War Barge"); err != nil {
		t.Fatal("failed collection changed registry")
	}
}

func TestRONDepartureDelayResolvingContexts(t *testing.T) {
	for _, context := range []string{"spell", "aura", "trigger", "nested"} {
		t.Run(context, func(t *testing.T) {
			g, a, b := ronGame()
			watched := mage.NewCreature("Watched", "", 2, 3)
			watched.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(watched, b.PlayerID())
			types, ref := "Instant", "Ref_TargetPermanent"
			if context == "aura" {
				types, ref = "Enchantment", "HostPermanent"
			}
			if context == "trigger" {
				types, ref = "Artifact", "ThisPermanent"
			}
			action := `CreateTriggerUntil(WhenAPermanentDies(SinglePermanent(` + ref + `)),ActionList([GainLife(Integer(1))]),UntilEndOfTurn)`
			if context == "trigger" {
				action = strings.Replace(action, "WhenAPermanentDies", "WhenAPermanentLeavesTheBattlefield", 1)
			}
			if context == "nested" {
				action = `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + action + `]))`
			}
			rules := `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `])))`
			if context == "aura" {
				rules = `EnchantPermanent(IsCardtype(Creature)),Activated(PayMana([ManaCostGeneric(0)]),ActionList([` + action + `]))`
			}
			if context == "trigger" {
				rules = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([` + action + `]))`
			}
			subtypes := ""
			if context == "aura" {
				subtypes = "Aura"
			}
			nodes, err := Parse(strings.NewReader(`Card(Name:"Context",Typeline:(Supertypes:[],Cardtypes:[`+types+`],Subtypes:[`+subtypes+`]),Rules:[`+rules+`])`), "departure-context.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			switch context {
			case "aura":
				g.PutOnBattlefield(card, a.PlayerID())
				g.Attach(card.ID(), target.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			case "trigger":
				target = g.PutOnBattlefield(card, a.PlayerID())
			default:
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			if context == "nested" {
				g.RunStepWithPriority(core.EndStep)
			}
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("resolving context lost subscription")
			}
			if context == "aura" {
				other := g.PutOnBattlefield(mage.NewCreature("New host", "", 2, 3), b.PlayerID())
				g.Attach(card.ID(), other.ID())
			}
			g.DestroyPermanent(g.FindPermanent(target.ID()))
			g.PutTriggersOnStack()
			clone := g.Clone()
			g.ResolveStack()
			if a.Life() != 21 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("captured controller or clone isolation")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 21 {
				t.Fatal("clone lost body")
			}
		})
	}
}

func TestRONDepartureDelayCards(t *testing.T) {
	for _, tc := range []struct {
		name, set string
		mana      int
		tap       bool
	}{
		{"War Barge", "DRK", 3, false},
		{"Sandals of Abdallah", "ARN", 2, true},
	} {
		for _, change := range []string{"normal", "bounce", "exile", "phase-cycle", "expire", "control", "remove-abilities", "returned-source", "returned-target", "phased-body", "regenerate", "replaced-death", "simultaneous", "copy"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
					t.Fatal("unpaid artifact cast succeeded")
				}
				a.ManaPool().Add(core.Colorless, 4)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(card.ID()) != nil {
					t.Fatal("artifact entered before resolution")
				}
				g.ResolveStack()
				source := g.FindPermanent(card.ID())
				victimCard := mage.NewCreature("Recipient", "", 2, 4)
				victimCard.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(victimCard, b.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 4), b.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err == nil || g.StackSize() != 0 || source.Tapped {
					t.Fatal("unpaid activation changed state")
				}
				a.ManaPool().Add(core.Colorless, tc.mana)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{source.ID()}); err == nil || a.ManaPool().Count(core.Colorless) != tc.mana || source.Tapped {
					t.Fatal("noncreature target spent costs")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(source.ID()).Tapped != tc.tap || victim.HasKeyword(core.Islandwalk) || len(g.DelayedTriggers()) != 0 {
					t.Fatal("wrong payment or early effect")
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveStack()
				wantDelays := 1
				if change == "copy" {
					wantDelays = 2
				}
				if !g.FindPermanent(victim.ID()).HasKeyword(core.Islandwalk) || other.HasKeyword(core.Islandwalk) || len(g.DelayedTriggers()) != wantDelays {
					t.Fatal("wrong recipient or subscription count")
				}
				island := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
				if !mage.HasLandwalkEvasion(g.FindPermanent(victim.ID()), a.PlayerID(), g) || mage.HasLandwalkEvasion(other, a.PlayerID(), g) {
					t.Fatal("grant did not change islandwalk evasion")
				}
				g.PhaseOut(island.ID())
				if mage.HasLandwalkEvasion(g.FindPermanent(victim.ID()), a.PlayerID(), g) {
					t.Fatal("phased Island supplied evasion")
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					if change == "phase-cycle" {
						for _, id := range []uuid.UUID{source.ID(), victim.ID()} {
							game.PhaseIn(game.PhaseOut(id))
						}
						game.PutTriggersOnStack()
						if game.StackSize() != 0 {
							t.Fatal("phasing triggered departure")
						}
					}
					if change == "control" {
						game.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
					if change == "remove-abilities" {
						if err := mage.ApplyEffect(game, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), a.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
					}
					if change == "expire" {
						game.RunStepWithPriority(core.Cleanup)
						if len(game.DelayedTriggers()) != 0 || game.FindPermanent(victim.ID()).HasKeyword(core.Islandwalk) {
							t.Fatal("cleanup did not expire effects")
						}
					}
					if change == "returned-target" && tc.name == "War Barge" {
						game.BouncePermanentToHand(game.FindPermanent(victim.ID()))
						game.GetPlayer(b.PlayerID()).RemoveFromHand(victimCard.ID())
						returned := victimCard.Copy()
						returned.SetID(victimCard.ID())
						game.PutOnBattlefield(returned, b.PlayerID())
					}
					if change == "returned-source" && tc.name == "Sandals of Abdallah" {
						game.BouncePermanentToHand(game.FindPermanent(source.ID()))
						game.GetPlayer(a.PlayerID()).RemoveFromHand(card.ID())
						returned := card.Copy()
						returned.SetID(card.ID())
						game.PutOnBattlefield(returned, a.PlayerID())
					}
					watched, body := source.ID(), victim.ID()
					if tc.name == "Sandals of Abdallah" {
						watched, body = victim.ID(), source.ID()
					}
					if change == "phased-body" {
						game.PhaseOut(body)
					}
					if change == "regenerate" {
						game.AddRegenerationShield(body)
					}
					if change == "replaced-death" {
						game.AddExileIfWouldGoToGraveyardThisTurn(watched, other.ID())
					}
					switch change {
					case "bounce":
						game.BouncePermanentToHand(game.FindPermanent(watched))
					case "exile":
						game.ExilePermanent(game.FindPermanent(watched))
					case "simultaneous":
						game.DestroyPermanents([]*mage.Permanent{game.FindPermanent(source.ID()), game.FindPermanent(victim.ID())})
					default:
						game.DestroyPermanent(game.FindPermanent(watched))
					}
					if game.StackSize() != 0 {
						t.Fatal("delay skipped pending trigger timing")
					}
					game.PutTriggersOnStack()
					fires := change != "expire" && (tc.name != "Sandals of Abdallah" || change != "bounce" && change != "exile" && change != "replaced-death")
					wantStack := 0
					if fires {
						wantStack = wantDelays
					}
					if game.StackSize() != wantStack {
						t.Fatalf("stack=%d want=%d", game.StackSize(), wantStack)
					}
					if fires {
						if len(game.StackPeek().TargetSpecs) != 0 || game.StackPeek().Controller != a.PlayerID() && change != "copy" {
							t.Fatal("delay retargeted or changed controller")
						}
					}
					game.ResolveStack()
					present := !fires || change == "phased-body" || change == "returned-target" && tc.name == "War Barge" || change == "returned-source" && tc.name == "Sandals of Abdallah" || change == "regenerate" && tc.name == "Sandals of Abdallah"
					if change == "simultaneous" {
						present = false
					}
					if (game.FindPermanentIncludingPhased(body) != nil) != present {
						t.Fatalf("body presence=%v want=%v", game.FindPermanentIncludingPhased(body) != nil, present)
					}
					if game.FindPermanent(other.ID()) == nil {
						t.Fatal("delay affected another permanent")
					}
					if game == g && clone.FindPermanent(source.ID()) == nil {
						t.Fatal("clone shared execution")
					}
				}
			})
		}
	}
}

func TestRONDepartureDelayInvalid(t *testing.T) {
	valid := `CreateTriggerUntil(WhenAPermanentLeavesTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([GainLife(Integer(1))]),UntilEndOfTurn)`
	for _, action := range []string{
		strings.Replace(valid, "UntilEndOfTurn)", "UntilEndOfCombat)", 1),
		strings.Replace(valid, "UntilEndOfTurn)", "UntilEndOfTurn())", 1),
		strings.Replace(valid, "SinglePermanent(ThisPermanent)", "SinglePermanent(Ref_TargetPermanent)", 1),
		strings.Replace(valid, "SinglePermanent(ThisPermanent)", "SinglePermanent(Ref_TargetPlayer)", 1),
		strings.Replace(valid, "SinglePermanent(ThisPermanent)", "IsCardtype(Creature)", 1),
		strings.Replace(valid, "SinglePermanent(ThisPermanent)", "SinglePermanent(ThisPermanent,ThisPermanent)", 1),
		strings.Replace(valid, "WhenAPermanentLeavesTheBattlefield", "WhenAPermanentEntersTheBattlefield", 1),
		strings.Replace(valid, "GainLife(Integer(1))", "GainLife(Trigger_ThatMuch)", 1),
		strings.Replace(valid, "[GainLife(Integer(1))]", "[]", 1),
		strings.Replace(valid, "ActionList([GainLife(Integer(1))])", "Targeted([TargetPlayer(AnyPlayer)],ActionList([GainLife(Integer(1))]))", 1),
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([ManaCostGeneric(0)]),ActionList([` + action + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-departure.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid declaration: %s", action)
		}
	}
}

func TestRONDepartureDelayInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([ManaCostGeneric(0)]),ActionList([CreateTriggerUntil(WhenAPermanentDies(SinglePermanent(ThisPermanent)),ActionList([UnknownAction(ThisPermanent)]),UntilEndOfTurn)]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"WhenAPermanentDies": "delayed_departure_event", "SinglePermanent": "delayed_departure_subject", "ThisPermanent": "permanent_reference", "UntilEndOfTurn": "duration", "UnknownAction": "action"}
	for _, signature := range report.Signatures {
		if role, ok := want[signature.Name]; ok {
			if signature.Role != role {
				t.Fatalf("%s role=%s want=%s", signature.Name, signature.Role, role)
			}
			delete(want, signature.Name)
		}
	}
	if len(want) != 0 || len(report.Issues) != 1 {
		t.Fatalf("missing roles: %v", want)
	}
}
