package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDelayedStepRukhEgg(t *testing.T) {
	for _, duringEnd := range []bool{false, true} {
		t.Run(fmt.Sprint(duringEnd), func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ARN", "Rukh Egg")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Red, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if duringEnd {
				g.RunStepWithPriority(core.EndStep)
			}
			g.DestroyPermanent(g.FindPermanent(card.ID()))
			if len(g.DelayedTriggers()) != 0 {
				t.Fatal("delay created before death trigger resolution")
			}
			g.ResolveStack()
			if len(g.DelayedTriggers()) != 1 || len(g.AllBattlefield()) != 0 {
				t.Fatal("death trigger did not create delay")
			}
			clone := g.Clone()
			g.RunStepWithPriority(core.Cleanup)
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("cleanup consumed delay")
			}
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.EndStep)
			if len(g.DelayedTriggers()) != 0 || len(g.AllBattlefield()) != 1 {
				t.Fatal("next end step did not create token")
			}
			token := g.AllBattlefield()[0]
			if token.ControllerID() != a.PlayerID() || !token.IsToken || token.CurrentPower(g) != 4 || token.CurrentToughness(g) != 4 || !token.HasKeyword(core.Flying) || !token.HasSubType("Bird") || !slices.Contains(token.Colors(), core.Red) {
				t.Fatal("wrong token")
			}
			if b.Life() != 20 || len(clone.AllBattlefield()) != 0 || len(clone.DelayedTriggers()) != 1 {
				t.Fatal("clone shared delay")
			}
			g.RunStepWithPriority(core.EndStep)
			if len(g.AllBattlefield()) != 1 {
				t.Fatal("delay repeated")
			}
			clone.RunStepWithPriority(core.EndStep)
			if len(clone.AllBattlefield()) != 1 {
				t.Fatal("clone lost delay")
			}
		})
	}
}

func TestRONDelayedStepTargetCapture(t *testing.T) {
	for _, change := range []string{"normal", "return", "phase", "phase-cycle", "control", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep, ActionList([DestroyPermanent(Ref_TargetPermanent)]))])))`)
			first, second := factory(), factory()
			if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("shared target declaration")
			}
			targetCard := repositoryCard(t, "LEB", "Llanowar Elves")
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			other := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
			first.SetOwner(a.PlayerID())
			a.AddToHand(first)
			if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("missing delay")
			}
			switch change {
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(target.ID()))
				b.RemoveFromHand(targetCard.ID())
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "phase":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			if g.StackSize() != 0 {
				t.Fatal("delay did not wait for priority")
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || len(g.StackPeek().TargetSpecs) != 0 {
				t.Fatal("captured target became a new target")
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.FindPermanentIncludingPhased(target.ID()) == nil {
				t.Fatal("clone changed live target")
			}
			g.ResolveStack()
			wantPresent := change == "return" || change == "phase"
			if (g.FindPermanentIncludingPhased(target.ID()) != nil) != wantPresent || g.FindPermanent(other.ID()) == nil {
				t.Fatal("wrong captured incarnation")
			}
		})
	}
}

func TestRONDelayedStepSourceCapture(t *testing.T) {
	for _, change := range []string{"normal", "return-before", "return-after", "phase", "phase-cycle", "control", "removed"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]), ActionList([CreateFutureTrigger(AtTheNextEndOfCombat, ActionList([SacrificePermanent(ThisPermanent), PermanentDealsDamage(ThisPermanent, Integer(2), Player(You))]))]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || len(g.DelayedTriggers()) != 0 {
				t.Fatal("unpaid activation created delay")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if change == "return-before" {
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			}
			g.ResolveStack()
			if change == "return-after" {
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			}
			if change == "phase" {
				g.PhaseOut(p.ID())
			}
			if change == "phase-cycle" {
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			if change == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if change == "removed" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			}
			g.AddPreventionShield(a.PlayerID(), 1)
			g.RunStepWithPriority(core.EndCombat)
			wantPresent := change == "return-before" || change == "return-after" || change == "phase" || change == "control"
			if (g.FindPermanentIncludingPhased(p.ID()) != nil) != wantPresent || a.Life() != 19 || b.Life() != 20 {
				t.Fatal("source incarnation, controller or LKI damage lost")
			}
			g.RunStepWithPriority(core.EndCombat)
			if a.Life() != 19 {
				t.Fatal("delay repeated")
			}
		})
	}
}

func TestRONDelayedStepUpkeepAndNested(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]), ActionList([CreateFutureTrigger(AtTheBeginningOfPlayersNextUpkeep(You), ActionList([GainLife(Integer(1)), CreateFutureTrigger(AtTheBeginningOfTheNextEndStep, ActionList([GainLife(Integer(2))]))]))]))`)
	card := factory()
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Upkeep)
	if a.Life() != 20 || len(g.DelayedTriggers()) != 2 {
		t.Fatal("opponent upkeep consumed delay")
	}
	g.SetActivePlayerIndex(0)
	g.RunStepWithPriority(core.Upkeep)
	if a.Life() != 22 || b.Life() != 20 || len(g.DelayedTriggers()) != 2 {
		t.Fatal("independent upkeep delays or nested registration lost")
	}
	g.RunStepWithPriority(core.EndStep)
	if a.Life() != 26 || len(g.DelayedTriggers()) != 0 {
		t.Fatal("nested delay lost controller")
	}
}

func TestRONDelayedStepInvalid(t *testing.T) {
	for _, action := range []string{
		`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep(), ActionList([GainLife(Integer(1))]))`,
		`CreateFutureTrigger(AtTheNextEndOfCombat(Integer(1)), ActionList([GainLife(Integer(1))]))`,
		`CreateFutureTrigger(AtNextEndOfCombatThisTurn, ActionList([GainLife(Integer(1))]))`,
		`CreateFutureTrigger(AtTheBeginningOfPlayersNextUpkeep(Opponent), ActionList([GainLife(Integer(1))]))`,
		`CreateFutureTrigger(AtTheBeginningOfPlayersNextUpkeep(You()), ActionList([GainLife(Integer(1))]))`,
		`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep, ActionList([]))`,
		`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep, Targeted([TargetPermanent(AnyPermanent)], ActionList([DestroyPermanent(Ref_TargetPermanent)])))`,
		`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep, ActionList([DestroyPermanent(Ref_TargetPermanent)]))`,
		`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep, ActionList([TapPermanent(Trigger_ThatCreature)]))`,
		`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep, ActionList([GainLife(Integer(1))]), UntilEndOfTurn)`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([%s]))])`, action)
		nodes, err := Parse(strings.NewReader(src), "bad-delay.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONDelayedStepXAndIndependentTargets(t *testing.T) {
	g, a, b := ronGame()
	nodes, err := Parse(strings.NewReader(`Card(Name: "Delayed X", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), ManaCost: [ManaCostX], Rules: [SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep, ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [AdjustPTX(PlusX, Zero, ValueX)], UntilEndOfTurn)]))])))])`), "delay-x.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	first := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
	for i, target := range []*mage.Permanent{first, second} {
		spell := factory()
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		a.ManaPool().Add(core.Colorless, i+2)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, i+2); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if first.CurrentPower(g) != 1 || second.CurrentPower(g) != 1 || len(g.DelayedTriggers()) != 2 {
		t.Fatal("delayed modifications applied early")
	}
	g.RunStepWithPriority(core.EndStep)
	if first.CurrentPower(g) != 3 || second.CurrentPower(g) != 4 {
		t.Fatal("delay shared X or target state")
	}
	g.RunStepWithPriority(core.Cleanup)
	if first.CurrentPower(g) != 1 || second.CurrentPower(g) != 1 {
		t.Fatal("delayed boost did not expire")
	}
}

func TestRONDelayedStepAttackEventAndCounter(t *testing.T) {
	for _, counter := range []bool{false, true} {
		t.Run(fmt.Sprint(counter), func(t *testing.T) {
			g, a, b := ronGame()
			card := permanentScopeFactory(t, "Artifact", `TriggerA(WhenACreatureAttacks(SinglePermanent(ThisPermanent)), ActionList([CreateFutureTrigger(AtTheNextEndOfCombat, ActionList([SacrificePermanent(Trigger_ThatCreature), PermanentDealsDamage(Trigger_ThatCreature, Integer(5), Player(You))]))]))`)()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtDeclaredAttacker, SourceID: p.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
			g.ResolveStack()
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("attack binding not captured")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if counter {
				g.GetStack().RemoveByID(g.StackPeek().ID)
			}
			g.ResolveStack()
			wantLife := 15
			if counter {
				wantLife = 20
			}
			if a.Life() != wantLife || (g.FindPermanent(p.ID()) != nil) != counter || len(g.DelayedTriggers()) != 0 {
				t.Fatal("counter or source event reference lost")
			}
			g.RunStepWithPriority(core.EndCombat)
			if a.Life() != wantLife {
				t.Fatal("countered delay rearmed")
			}
		})
	}
}

func TestRONDelayedStepRukhEggNonDeathsAndCopies(t *testing.T) {
	for _, operation := range []string{"exile", "bounce", "phase", "regenerate", "copy", "two"} {
		t.Run(operation, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ARN", "Rukh Egg")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			switch operation {
			case "exile":
				g.ExilePermanent(p)
			case "bounce":
				g.BouncePermanentToHand(p)
			case "phase":
				g.PhaseOut(p.ID())
			case "regenerate":
				g.AddRegenerationShield(p.ID())
				g.DestroyPermanent(p)
			default:
				g.DestroyPermanent(p)
				g.PutTriggersOnStack()
				if operation == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if operation == "two" {
					other := repositoryCard(t, "ARN", "Rukh Egg")
					other.SetOwner(a.PlayerID())
					q := g.PutOnBattlefield(other, a.PlayerID())
					g.DestroyPermanent(q)
				}
			}
			g.ResolveStack()
			want := 0
			if operation == "copy" || operation == "two" {
				want = 2
			}
			if len(g.DelayedTriggers()) != want {
				t.Fatal("wrong death delay count")
			}
			g.RunStepWithPriority(core.EndStep)
			tokens := g.FilterBattlefield(mage.IsToken)
			if len(tokens) != want {
				t.Fatal("wrong token count")
			}
			counts := map[uuid.UUID]int{}
			for _, token := range tokens {
				counts[token.ControllerID()]++
			}
			if operation == "two" && (counts[a.PlayerID()] != 1 || counts[b.PlayerID()] != 1) {
				t.Fatal("death controller confused with owner")
			}
			if operation == "copy" && counts[b.PlayerID()] != 2 {
				t.Fatal("copied death trigger lost controller")
			}
		})
	}
}

func TestRONDelayedStepInventory(t *testing.T) {
	src := `Card(Name: "Inventory", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([CreateFutureTrigger(AtTheBeginningOfPlayersNextUpkeep(You), ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep, ActionList([GainLife(Integer(2))]))]))]))])`
	nodes, err := Parse(strings.NewReader(src), "delay-inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", false, entries)
	for name, role := range map[string]string{"AtTheBeginningOfPlayersNextUpkeep": "delayed_step_event", "AtTheBeginningOfTheNextEndStep": "delayed_step_event", "You": "player_reference", "GainLife": "action", "Integer": "value"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("%s missing %s role below rejected parent", name, role)
		}
	}
}

func TestRONDelayedStepRejectUnsavedSpellEvent(t *testing.T) {
	src := `Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [TriggerA(WhenAPlayerCastsASpell(AnyPlayer, AnySpell), ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep, ActionList([CounterSpell(Trigger_ThatSpell)]))]))])`
	nodes, err := Parse(strings.NewReader(src), "delay-spell-event.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("unsaved event spell escaped into delayed body")
	}
}

func TestRONDelayedStepCapturedEventPlayer(t *testing.T) {
	g, a, b := ronGame()
	seedRONDrawLibrary(a)
	seedRONDrawLibrary(b)
	watcher := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerCastsASpell(AnyPlayer, AnySpell), ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep, ActionList([PlayerAction(Trigger_ThatPlayer, DrawACard)]))]))`)()
	watcher.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(watcher, a.PlayerID())
	for _, player := range []mage.Player{a, b} {
		spell := repositoryCard(t, "LEB", "Lightning Bolt")
		spell.SetOwner(player.PlayerID())
		player.AddToHand(spell)
		player.ManaPool().Add(core.Red, 1)
		if err := g.CastSpellByID(player.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	g.DestroyPermanent(g.FindPermanent(p.ID()))
	g.RunStepWithPriority(core.EndStep)
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 || a.Life() != 20 || b.Life() != 14 {
		t.Fatal("delays shared the last cast event player")
	}
}

func TestRONDelayedStepIllegalBeforeCreation(t *testing.T) {
	for _, change := range []string{"return", "phase"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEB", "Llanowar Elves")
			card.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(card, b.PlayerID())
			spell := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep, ActionList([DestroyPermanent(Ref_TargetPermanent)]))])))`)()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if change == "phase" {
				g.PhaseOut(target.ID())
			} else {
				g.BouncePermanentToHand(g.FindPermanent(target.ID()))
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			}
			g.ResolveStack()
			if len(g.DelayedTriggers()) != 0 {
				t.Fatal("all-illegal spell created delay")
			}
		})
	}
}
