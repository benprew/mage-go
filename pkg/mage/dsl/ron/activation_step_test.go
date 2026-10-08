package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONActivationSteps(t *testing.T) {
	for _, condition := range []string{"IsDuringCombat", "IsDuringEndOfCombatStep", "IsDuringDeclareBlockersStep", "IsDuringUpkeep"} {
		for step := core.Untap; step <= core.Cleanup; step++ {
			for _, active := range []int{0, 1} {
				t.Run(fmt.Sprintf("%s/%s/%d", condition, step, active), func(t *testing.T) {
					rule := fmt.Sprintf(`ActivatedWithModifiers(PayMana([ManaCostGeneric(1)]),ActionList([GainLife(Integer(1))]),ActivateOnlyIf(%s))`, condition)
					card := permanentScopeFactory(t, "Artifact", rule)()
					g, a, _ := ronGame()
					card.SetOwner(a.PlayerID())
					p := g.PutOnBattlefield(card, a.PlayerID())
					g.SetStep(step)
					g.SetActivePlayerIndex(active)
					a.ManaPool().Add(core.Colorless, 1)
					want := step >= core.BeginCombat && step <= core.EndCombat
					switch condition {
					case "IsDuringEndOfCombatStep":
						want = step == core.EndCombat
					case "IsDuringDeclareBlockersStep":
						want = step == core.DeclareBlockers
					case "IsDuringUpkeep":
						want = step == core.Upkeep
					}
					ability := p.Card.Abilities()[0].(mage.ActivatedAbility)
					if ability.CanActivate(a.PlayerID(), g) != want {
						t.Fatal("query ignored step restriction")
					}
					err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil)
					if (err == nil) != want {
						t.Fatalf("activation = %v, allowed = %v", err, want)
					}
					if !want {
						if a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 || a.Life() != 20 {
							t.Fatal("rejection changed resources")
						}
						return
					}
					if a.ManaPool().TotalMana() != 0 || g.StackSize() != 1 || a.Life() != 20 {
						t.Fatal("announcement did not pay before resolution")
					}
					g.SetStep(core.PrecombatMain)
					g.SetActivePlayerIndex(1 - active)
					g.ResolveStack()
					if a.Life() != 21 {
						t.Fatal("resolution repeated timing check")
					}
				})
			}
		}
	}
}

func TestRONActivationStepsInvalid(t *testing.T) {
	for _, condition := range []string{"IsDuringCombat()", "IsDuringCombat(You)", "IsDuringEndOfCombatStep()", "IsDuringDeclareBlockersStep(Integer(1))", "IsDuringUpkeep()", "IsDuringMainPhase", "And([IsDuringCombat,IsDuringUpkeep])", "IsDuringCombat(Ref_TargetPermanent)", "IsDuringCombat(Unknown)"} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[ActivatedWithModifiers(PayMana([]),ActionList([GainLife(Integer(1))]),ActivateOnlyIf(%s))])`, condition)
		nodes, err := Parse(strings.NewReader(src), "invalid-step.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", condition)
		}
	}
}

func TestRONActivationStepsConjunctionAndMana(t *testing.T) {
	for _, mana := range []bool{false, true} {
		body := "GainLife(Integer(1))"
		if mana {
			body = "AddMana(ManaProduceU)"
		}
		rule := fmt.Sprintf(`ActivatedWithModifiers(TapPermanent(ThisPermanent),ActionList([%s]),And([ActivateOnlyIf(IsDuringUpkeep),ActivateOnlyIf(PlayerPassesFilter(You,IsTheirTurn)),ActivateOnlyOnceEachTurn]))`, body)
		g, a, _ := ronGame()
		card := permanentScopeFactory(t, "Land", rule)()
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		for _, state := range []struct {
			active int
			step   core.PhaseStep
			want   bool
		}{{0, core.Draw, false}, {1, core.Upkeep, false}, {0, core.Upkeep, true}} {
			g.SetActivePlayerIndex(state.active)
			g.SetStep(state.step)
			if mana && g.CanAfford(a.PlayerID(), core.ParseManaCost("{U}"), nil) != state.want {
				t.Fatal("mana solver lost combined restrictions")
			}
			if g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("query mutated resources")
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil)
			if (err == nil) != state.want {
				t.Fatalf("combined activation: %v", err)
			}
		}
		g.UntapPermanent(g.FindPermanent(p.ID()))
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
			t.Fatal("step restriction replaced use limit")
		}
		g.SetStep(core.Draw)
		g.ResolveStack()
		if mana {
			if a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
				t.Fatal("restricted mana ability timing")
			}
		} else if a.Life() != 21 {
			t.Fatal("restricted activation resolution")
		}
	}
}

func TestRONActivationStepsJadeStatue(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase-cycle", "control", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEB", "Jade Statue")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid statue cast")
			}
			if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 3 || g.StackSize() != 0 {
				t.Fatal("cast rollback")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("premature entry")
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p.HasType(core.TypeCreature) || !p.HasType(core.TypeArtifact) {
				t.Fatal("printed types")
			}
			a.ManaPool().Add(core.Colorless, 1)
			g.SetStep(core.BeginCombat)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("unpaid animation")
			}
			if a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 || p.HasType(core.TypeCreature) {
				t.Fatal("activation rollback")
			}
			a.ManaPool().Add(core.Colorless, 1)
			g.SetActivePlayerIndex(1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()).HasType(core.TypeCreature) {
				t.Fatal("premature animation")
			}
			pending := g.Clone()
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.SetStep(core.DeclareBlockers)
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{p.ID()})
			g.ApplyContinuousEffects()
			current := g.FindPermanent(p.ID())
			want := change != "return" && change != "phase"
			if current.HasType(core.TypeCreature) != want || current.HasSubType("Golem") != want || !current.HasType(core.TypeArtifact) {
				t.Fatal("animation identity or types")
			}
			if want && (current.CurrentPower(g) != 3 || current.CurrentToughness(g) != 6) {
				t.Fatal("animation base values")
			}
			if pending.FindPermanent(p.ID()).HasType(core.TypeCreature) {
				t.Fatal("pending clone changed")
			}
			pending.ResolveStack()
			if !pending.FindPermanent(p.ID()).HasType(core.TypeCreature) {
				t.Fatal("clone lost animation")
			}
			g.RunStepWithPriority(core.EndCombat)
			if g.FindPermanent(p.ID()).HasType(core.TypeCreature) || g.FindPermanent(p.ID()).HasSubType("Golem") || !g.FindPermanent(p.ID()).HasType(core.TypeArtifact) {
				t.Fatal("native combat expiry")
			}
			if !pending.FindPermanent(p.ID()).HasType(core.TypeCreature) {
				t.Fatal("expiry changed clone")
			}
		})
	}
}

func TestRONActivationStepsDesert(t *testing.T) {
	for _, change := range []string{"none", "source-leaves", "source-return", "source-phase", "source-control", "target-return", "target-phase", "target-phase-cycle", "target-shroud", "target-control", "not-attacking", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ARN", "Desert")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.PlayLand(a.PlayerID(), card.ID()); err != nil {
				t.Fatal(err)
			}
			source := g.FindPermanent(card.ID())
			if !source.HasType(core.TypeLand) || !source.HasSubType("Desert") {
				t.Fatal("land metadata")
			}
			if !g.CanAfford(a.PlayerID(), core.ParseManaCost("{1}"), nil) || source.Tapped {
				t.Fatal("mana query")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 || !g.FindPermanent(source.ID()).Tapped {
				t.Fatal("mana ability")
			}
			g.UntapPermanent(g.FindPermanent(source.ID()))
			targetCard := mage.NewCreature("Attacker", "", 4, 5)
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			target.RevokeBaseAttr(core.AttrSummonSick)
			g.SetActivePlayerIndex(1)
			g.SetStep(core.DeclareAttackers)
			g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{target.ID()})
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err == nil {
				t.Fatal("early Desert activation")
			}
			if g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
				t.Fatal("timing rollback")
			}
			g.SetStep(core.EndCombat)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err == nil {
				t.Fatal("wrong controller")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{source.ID()}); err == nil {
				t.Fatal("noncreature target")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source.ID()).Tapped || g.FindPermanent(target.ID()).Damage != 0 {
				t.Fatal("damage announcement")
			}
			pending := g.Clone()
			switch change {
			case "source-leaves":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			case "source-phase":
				g.PhaseOut(source.ID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "target-return":
				g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "target-phase":
				g.PhaseOut(target.ID())
			case "target-phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "target-shroud":
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
			case "target-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "not-attacking":
				mage.ApplyEffect(g, mage.RemovePermanentFromCombat(mage.This), target.ID(), b.PlayerID(), nil)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.SetStep(core.Upkeep)
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{target.ID()})
			damage := 1
			switch change {
			case "target-return", "target-phase", "target-phase-cycle", "target-shroud", "target-control", "not-attacking":
				damage = 0
			case "copy":
				damage = 2
			}
			if g.FindPermanent(target.ID()).Damage != damage {
				t.Fatalf("damage = %d, want %d", g.FindPermanent(target.ID()).Damage, damage)
			}
			if pending.FindPermanent(target.ID()).Damage != 0 {
				t.Fatal("clone shared damage")
			}
			pending.ResolveStack()
			if pending.FindPermanent(target.ID()).Damage != 1 {
				t.Fatal("pending clone lost damage")
			}
		})
	}
}

func TestRONActivationStepsCombatDurationTargets(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase-cycle", "control", "copy"} {
		t.Run(change, func(t *testing.T) {
			rule := `ActivatedWithModifiers(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AdjustPT(1,2),AddAbility([Flying])],UntilEndOfCombat)])),ActivateOnlyIf(IsDuringCombat))`
			g, a, b := ronGame()
			sourceCard := permanentScopeFactory(t, "Artifact", rule)()
			sourceCard.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			targetCard := mage.NewCreature("Subject", "", 2, 3)
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			g.SetStep(core.BeginCombat)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "phase":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{target.ID()})
			g.ApplyContinuousEffects()
			affected := change != "return" && change != "phase"
			power := 2
			if affected {
				power++
			}
			if change == "copy" {
				power++
			}
			if g.FindPermanent(target.ID()).CurrentPower(g) != power || g.FindPermanent(target.ID()).HasKeyword(core.Flying) != affected {
				t.Fatal("combat-duration subject or source independence")
			}
			g.RunStepWithPriority(core.EndCombat)
			if g.FindPermanent(target.ID()).CurrentPower(g) != 2 || g.FindPermanent(target.ID()).HasKeyword(core.Flying) {
				t.Fatal("combat-duration target expiry")
			}
		})
	}
}

func TestRONActivationStepsCombatDurationInvalid(t *testing.T) {
	for _, rule := range []string{
		`Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(1,1)],UntilEndOfCombat())]))`,
		`Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(1,1)],UntilEndOfCombat(You))]))`,
		`SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPlayer,[AdjustPT(1,1)],UntilEndOfCombat)])))`,
		`Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CreatePermanentLayerEffectUntil(Trigger_ThatPermanent,[AdjustPT(1,1)],UntilEndOfCombat)]))]))`,
	} {
		kind := "Artifact"
		if strings.HasPrefix(rule, "SpellActions") {
			kind = "Instant"
		}
		src := fmt.Sprintf(`Card(Name:"Invalid duration",Typeline:(Supertypes:[],Cardtypes:[%s],Subtypes:[]),Rules:[%s])`, kind, rule)
		nodes, err := Parse(strings.NewReader(src), "invalid-duration.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONActivationStepsJadeCombatAndInstances(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEB", "Jade Statue")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	otherCard := repositoryCard(t, "LEB", "Jade Statue")
	otherCard.SetOwner(b.PlayerID())
	other := g.PutOnBattlefield(otherCard, b.PlayerID())
	g.SetStep(core.BeginCombat)
	g.SetActivePlayerIndex(1)
	a.ManaPool().Add(core.Colorless, 4)
	b.ManaPool().Add(core.Colorless, 2)
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), other.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(source.ID()).HasType(core.TypeCreature) || !g.FindPermanent(other.ID()).HasType(core.TypeCreature) {
		t.Fatal("factory instances share source")
	}
	g.ResolveStack()
	for _, id := range []uuid.UUID{source.ID(), other.ID()} {
		if g.FindPermanent(id).CurrentPower(g) != 3 || g.FindPermanent(id).CurrentToughness(g) != 6 {
			t.Fatal("repeated base-setting animation accumulated")
		}
	}
	attackerCard := mage.NewCreature("Attacker", "", 4, 5)
	attackerCard.SetOwner(b.PlayerID())
	attacker := g.PutOnBattlefield(attackerCard, b.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	g.SetStep(core.DeclareAttackers)
	g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
	g.SetStep(core.DeclareBlockers)
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: source.ID(), AttackerID: attacker.ID()}})
	g.ExecuteCombatDamage()
	if a.Life() != 20 || g.FindPermanent(source.ID()).Damage != 4 || g.FindPermanent(attacker.ID()).Damage != 3 {
		t.Fatal("animated Statue did not block with 3/6 values")
	}
	g.RunStepWithPriority(core.EndCombat)
	if g.FindPermanent(source.ID()).HasType(core.TypeCreature) || g.FindPermanent(other.ID()).HasType(core.TypeCreature) {
		t.Fatal("independent animations did not expire")
	}
}

func TestRONActivationStepsInventory(t *testing.T) {
	for _, condition := range []string{"IsDuringCombat", "IsDuringEndOfCombatStep", "IsDuringDeclareBlockersStep", "IsDuringUpkeep"} {
		src := fmt.Sprintf(`Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[ActivatedWithModifiers(PayMana([]),ActionList([UnsupportedAction]),ActivateOnlyIf(%s))])`, condition)
		nodes, err := Parse(strings.NewReader(src), "inventory-step.ron")
		if err != nil {
			t.Fatal(err)
		}
		entries := map[string]*InventorySignature{}
		walkInventory(nodes[0], "Inventory", "Card", "card", "collection", false, entries)
		found := false
		for _, entry := range entries {
			if entry.Name == condition && entry.Role == "activation_condition" && entry.Kind == "symbol" {
				found = true
			}
		}
		if !found {
			t.Fatal("step role lost below unsupported body")
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("reporting accepted unsupported body")
		}
	}
}

func TestRONActivationStepsCombatKeywordRemoval(t *testing.T) {
	g, a, _ := ronGame()
	rule := `Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(Flying)],UntilEndOfCombat)]))`
	card := permanentScopeFactory(t, "Artifact", rule)()
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	source.GrantBaseAttr(core.Flying)
	g.SetStep(core.BeginCombat)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(source.ID()).HasKeyword(core.Flying) {
		t.Fatal("combat-duration keyword removal failed")
	}
	g.RunStepWithPriority(core.EndCombat)
	if !g.FindPermanent(source.ID()).HasKeyword(core.Flying) {
		t.Fatal("combat-duration removal did not expire")
	}
}
