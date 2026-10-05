package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRONFixedLandwalkLivonya(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Livonya Silone")
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	walker := g.PutOnBattlefield(card, a.PlayerID())
	blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 2, 2), b.PlayerID())
	g.PutOnBattlefield(mage.NewCreature("Legendary creature", "", 2, 2, mage.WithSuperTypes(core.SuperLegendary)), b.PlayerID())
	if !walker.HasKeyword(core.FirstStrike) || !walker.HasKeyword(core.LegendaryLandwalk) || walker.CurrentPower(g) != 4 || walker.CurrentToughness(g) != 4 {
		t.Fatal("printed characteristics")
	}
	if mage.HasLandwalkEvasion(walker, b.PlayerID(), g) || !mage.CanBlock(blocker, walker, g) {
		t.Fatal("legendary nonland supplied evasion")
	}
	landCard := mage.NewLand("Legendary land", mage.WithSuperTypes(core.SuperLegendary))
	landCard.SetOwner(a.PlayerID())
	land := g.PutOnBattlefield(landCard, b.PlayerID())
	if !mage.HasLandwalkEvasion(walker, b.PlayerID(), g) || mage.HasLandwalkEvasion(walker, a.PlayerID(), g) {
		t.Fatal("defender control or block legality")
	}
	clone := g.Clone()
	clone.PhaseOut(land.ID())
	if mage.HasLandwalkEvasion(clone.FindPermanent(walker.ID()), b.PlayerID(), clone) || !mage.HasLandwalkEvasion(g.FindPermanent(walker.ID()), b.PlayerID(), g) {
		t.Fatal("phasing or clone isolation")
	}
	clone.PhaseIn([]uuid.UUID{land.ID()})
	if !mage.HasLandwalkEvasion(clone.FindPermanent(walker.ID()), b.PlayerID(), clone) {
		t.Fatal("phase-in lost evasion")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: land.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	if mage.HasLandwalkEvasion(g.FindPermanent(walker.ID()), b.PlayerID(), g) || !mage.HasLandwalkEvasion(g.FindPermanent(walker.ID()), a.PlayerID(), g) {
		t.Fatal("land control change")
	}
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	if mage.HasLandwalkEvasion(g.FindPermanent(walker.ID()), a.PlayerID(), g) {
		t.Fatal("departed land supplied evasion")
	}
	g.PutOnBattlefield(landCard, b.PlayerID())
	removal := mage.RemoveAllAbilities(walker.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(blocker.ID())
	g.AddContinuousEffect(removal)
	if mage.HasLandwalkEvasion(g.FindPermanent(walker.ID()), b.PlayerID(), g) || g.FindPermanent(walker.ID()).HasKeyword(core.FirstStrike) {
		t.Fatal("ability removal")
	}
	g.RemoveFromBattlefield(g.FindPermanent(blocker.ID()))
	if !mage.HasLandwalkEvasion(g.FindPermanent(walker.ID()), b.PlayerID(), g) {
		t.Fatal("ability restoration")
	}
}

func TestRONFixedLandwalkAssassin(t *testing.T) {
	for _, change := range []string{"none", "lost-keyword", "gained-keyword", "return", "phase", "phase-cycle", "shroud", "control", "source-leaves", "source-return", "source-phase", "source-control", "regenerate", "indestructible", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Merfolk Assassin")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			targetCard := repositoryCard(t, "LEG", "Devouring Deep")
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			if source.CurrentPower(g) != 1 || source.CurrentToughness(g) != 2 || !source.HasSubType("Merfolk") || !source.HasSubType("Assassin") {
				t.Fatal("printed characteristics")
			}
			if change == "gained-keyword" {
				target.RevokeBaseAttr(core.Islandwalk)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
					t.Fatal("target without islandwalk accepted")
				}
				target.GrantBaseAttr(core.Islandwalk)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source.ID()).Tapped || g.FindPermanent(target.ID()) == nil || g.StackSize() != 1 {
				t.Fatal("activation timing")
			}
			pending := g.Clone()
			switch change {
			case "lost-keyword":
				g.MutablePermanent(target.ID()).RevokeBaseAttr(core.Islandwalk)
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "phase":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "shroud":
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "source-leaves":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			case "source-phase":
				g.PhaseOut(source.ID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "regenerate":
				g.AddRegenerationShield(target.ID())
			case "indestructible":
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Indestructible)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.AddRegenerationShield(target.ID())
			}
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{target.ID()})
			survives := change == "lost-keyword" || change == "return" || change == "phase" || change == "shroud" || change == "regenerate" || change == "indestructible"
			if (g.FindPermanent(target.ID()) != nil) != survives {
				t.Fatalf("target survival: want %v", survives)
			}
			if pending.FindPermanent(target.ID()) == nil {
				t.Fatal("live resolution changed clone")
			}
			pending.ResolveStack()
			if pending.FindPermanent(target.ID()) != nil {
				t.Fatal("clone lost saved target")
			}
			if change == "regenerate" && (g.HasRegenerationShield(target.ID()) || !g.FindPermanent(target.ID()).Tapped) {
				t.Fatal("regeneration contract")
			}
		})
	}
}

func TestRONFixedLandwalkAssassinRollback(t *testing.T) {
	for _, reason := range []string{"sick", "tapped", "phased-source", "wrong-controller", "wrong-keyword", "noncreature", "phased-target", "no-target", "extra-target"} {
		t.Run(reason, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Merfolk Assassin"), a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			target := g.PutOnBattlefield(repositoryCard(t, "LEG", "Devouring Deep"), b.PlayerID())
			ids := []uuid.UUID{target.ID()}
			controller := a.PlayerID()
			tapped := false
			switch reason {
			case "sick":
				source.GrantBaseAttr(core.AttrSummonSick)
			case "tapped":
				source.Tapped = true
				tapped = true
			case "phased-source":
				g.PhaseOut(source.ID())
			case "wrong-controller":
				controller = b.PlayerID()
			case "wrong-keyword":
				target.RevokeBaseAttr(core.Islandwalk)
				target.GrantBaseAttr(core.Forestwalk)
			case "noncreature":
				target = g.PutOnBattlefield(mage.NewArtifact("Landwalk artifact", "", mage.WithKeyword(core.Islandwalk)), b.PlayerID())
				ids = []uuid.UUID{target.ID()}
			case "phased-target":
				g.PhaseOut(target.ID())
			case "no-target":
				target.RevokeBaseAttr(core.Islandwalk)
				ids = nil
			case "extra-target":
				ids = []uuid.UUID{target.ID(), source.ID()}
			}
			a.ManaPool().Add(core.Blue, 3)
			if err := g.ActivateAbilityByIndex(controller, source.ID(), 0, ids); err == nil {
				t.Fatal("invalid activation accepted")
			}
			if g.FindPermanentIncludingPhased(source.ID()).Tapped != tapped || g.StackSize() != 0 || a.ManaPool().TotalMana() != 3 || g.FindPermanentIncludingPhased(target.ID()) == nil {
				t.Fatal("failed activation spent resources")
			}
		})
	}
}

func TestRONFixedLandwalkFilters(t *testing.T) {
	for _, walk := range []string{"IsLandType(Plains)", "IsLandType(Island)", "IsLandType(Swamp)", "IsLandType(Mountain)", "IsLandType(Forest)", "IsLandType(Desert)", "IsSupertype(Legendary)"} {
		for _, predicate := range []string{"HasAbility", "DoesntHaveAbility"} {
			t.Run(predicate+walk, func(t *testing.T) {
				rule := fmt.Sprintf(`Activated(PayMana([]),Targeted([TargetPermanent(And([IsCardtype(Creature),%s(Landwalk(%s))]))],ActionList([TapPermanent(Ref_TargetPermanent)])))`, predicate, walk)
				factory := permanentScopeFactory(t, "Artifact", rule)
				g, a, b := ronGame()
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				nodes, err := Parse(strings.NewReader("Landwalk("+walk+")"), "keyword.ron")
				if err != nil {
					t.Fatal(err)
				}
				kw, err := compileLandwalkKeyword(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 2), b.PlayerID())
				if predicate == "HasAbility" {
					victim.GrantBaseAttr(kw)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				if predicate == "HasAbility" {
					g.MutablePermanent(victim.ID()).RevokeBaseAttr(kw)
				} else {
					g.MutablePermanent(victim.ID()).GrantBaseAttr(kw)
				}
				g.ResolveStack()
				if g.FindPermanent(victim.ID()).Tapped {
					t.Fatal("keyword change did not invalidate target")
				}
			})
		}
	}
}

func TestRONFixedLandwalkInvalid(t *testing.T) {
	for _, kw := range []string{"Landwalk", "Landwalk()", "Landwalk(IsSupertype(Legendary()))", "Landwalk(IsSupertype(Basic))", "Landwalk(IsSupertype(Snow))", "Landwalk(IsSupertype(Legendary),IsLandType(Forest))", "Landwalk(IsLandType(Unknown))", "Landwalk(IsLandTypeVariable(TheChosenLandType))", "Flying()"} {
		for _, predicate := range []string{"HasAbility", "DoesntHaveAbility"} {
			rule := fmt.Sprintf(`Activated(PayMana([]),Targeted([TargetPermanent(%s(%s))],ActionList([TapPermanent(Ref_TargetPermanent)])))`, predicate, kw)
			src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatalf("accepted %s", rule)
			}
		}
	}
	walk := "Landwalk(IsSupertype(Legendary))"
	src := landwalkRuleSource("And([IsCardtype(Creature),HasAbility("+walk+")])", "[CanBeBlockedAsThoughItDidntHave("+walk+")]")
	nodes, err := Parse(strings.NewReader(src), "invalid-override.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err == nil {
		t.Fatal("accepted unsupported legendary blocking override")
	}
}

func TestRONFixedLandwalkInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[Landwalk(IsSupertype(Legendary)),Activated(PayMana([]),Targeted([TargetPermanent(HasAbility(Landwalk(IsLandType(Island))))],ActionList([UnsupportedAction])))])`
	report, err := InspectFS(fstest.MapFS{"inventory.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"Landwalk": "keyword", "IsSupertype": "landwalk_filter", "Legendary": "supertype", "IsLandType": "landwalk_filter", "Island": "land_subtype", "HasAbility": "permanent_filter"}
	for _, entry := range report.Signatures {
		if want, ok := roles[entry.Name]; ok && (entry.Name != "Landwalk" || entry.Role != "rule") && entry.Role != want {
			t.Errorf("%s role %s, want %s", entry.Name, entry.Role, want)
		}
	}
	if report.Compiled != 0 {
		t.Fatal("inventory weakened strict compilation")
	}
}

func TestRONFixedLandwalkNativeCombat(t *testing.T) {
	for _, legendaryLand := range []bool{false, true} {
		t.Run(fmt.Sprint(legendaryLand), func(t *testing.T) {
			a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			walker := g.PutOnBattlefield(repositoryCard(t, "LEG", "Livonya Silone"), a.PlayerID())
			walker.RevokeBaseAttr(core.AttrSummonSick)
			blocker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), b.PlayerID())
			if legendaryLand {
				g.PutOnBattlefield(mage.NewLand("Legendary land", mage.WithSuperTypes(core.SuperLegendary)), b.PlayerID())
			}
			a.SetAttackers(1, []string{"Livonya Silone"})
			b.SetBlockers(1, map[string]string{"Grizzly Bears": "Livonya Silone"})
			g.RunStepWithPriority(core.DeclareAttackers)
			g.RunStepWithPriority(core.DeclareBlockers)
			if g.GetCombat().IsBlocking(blocker.ID()) == legendaryLand {
				t.Fatal("native landwalk block legality")
			}
			g.ExecuteCombatDamage()
			if legendaryLand {
				if b.Life() != 16 || g.FindPermanent(blocker.ID()) == nil {
					t.Fatal("unblocked first-strike damage")
				}
			} else if b.Life() != 20 || g.FindPermanent(blocker.ID()) != nil || g.FindPermanent(walker.ID()).Damage != 0 {
				t.Fatal("first strike did not kill blocker before normal damage")
			}
		})
	}
}

func TestRONFixedLandwalkGrantRemoval(t *testing.T) {
	for _, operation := range []string{"AddAbility([Landwalk(IsSupertype(Legendary))])", "LosesAbility(Landwalk(IsSupertype(Legendary)))"} {
		t.Run(operation, func(t *testing.T) {
			g, a, b := ronGame()
			rule := `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[` + operation + `],UntilEndOfTurn)])))`
			source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
			grant := strings.HasPrefix(operation, "AddAbility")
			var victimCard mage.Card = mage.NewCreature("Victim", "", 2, 2)
			if !grant {
				victimCard = repositoryCard(t, "LEG", "Livonya Silone")
			}
			victimCard.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.ResolveStack()
			clone := g.Clone()
			if g.FindPermanent(victim.ID()).HasKeyword(core.LegendaryLandwalk) != grant {
				t.Fatal("temporary legendary keyword modification")
			}
			g.RunStepWithPriority(core.Cleanup)
			if g.FindPermanent(victim.ID()).HasKeyword(core.LegendaryLandwalk) == grant || clone.FindPermanent(victim.ID()).HasKeyword(core.LegendaryLandwalk) != grant {
				t.Fatal("expiry or clone isolation")
			}
		})
	}
	g, a, b := ronGame()
	rule := `EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([Landwalk(IsSupertype(Legendary))])])`
	source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 2), b.PlayerID())
	if !g.FindPermanent(victim.ID()).HasKeyword(core.LegendaryLandwalk) {
		t.Fatal("static legendary keyword grant")
	}
	g.PhaseOut(source.ID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(victim.ID()).HasKeyword(core.LegendaryLandwalk) {
		t.Fatal("phased grant remained")
	}
}

func TestRONFixedLandwalkInstancesAndRetargetedCopy(t *testing.T) {
	a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetTurn(2)
	g.SetStep(core.PrecombatMain)
	card := repositoryCard(t, "DRK", "Merfolk Assassin")
	firstSource := g.PutOnBattlefield(card, a.PlayerID())
	firstSource.RevokeBaseAttr(core.AttrSummonSick)
	secondSource := g.PutOnBattlefield(repositoryCard(t, "DRK", "Merfolk Assassin"), b.PlayerID())
	secondSource.RevokeBaseAttr(core.AttrSummonSick)
	first := g.PutOnBattlefield(repositoryCard(t, "LEG", "Devouring Deep"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEG", "Devouring Deep"), b.PlayerID())
	third := g.PutOnBattlefield(repositoryCard(t, "LEG", "Devouring Deep"), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), firstSource.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	b.targets = []uuid.UUID{second.ID()}
	if g.CopyStackObjectDirect(original, b.PlayerID(), true) == nil {
		t.Fatal("copy failed")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), secondSource.ID(), 0, []uuid.UUID{third.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(third.ID()) != nil || g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil {
		t.Fatal("independent instance changed another target")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(second.ID()) != nil || g.FindPermanent(first.ID()) == nil || original.Targets[0] != first.ID() {
		t.Fatal("copy overwrote original target")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(first.ID()) != nil {
		t.Fatal("original lost target")
	}
}
