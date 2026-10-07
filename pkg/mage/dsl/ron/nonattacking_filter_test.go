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

func TestRONNonattackingArcadesCombat(t *testing.T) {
	a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
	g := mage.NewGame(a, b)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	card := repositoryCard(t, "LEG", "Arcades Sabboth")
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if card.Power() != 7 || card.Toughness() != 7 || !card.HasType(core.TypeCreature) {
		t.Fatal("metadata")
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	if g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("cast rollback")
	}
	for _, color := range []core.Color{core.Green, core.White, core.Blue} {
		a.ManaPool().Add(color, 2)
	}
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("early entry")
	}
	g.ResolveStack()
	source := g.FindPermanent(card.ID())
	if !source.HasKeyword(core.Flying) {
		t.Fatal("missing flying")
	}
	checkConditionalPT(t, g, source.ID(), 7, 9)
	ally := g.PutOnBattlefield(mage.NewCreature("Ally", "", 2, 3), a.PlayerID())
	ally.RevokeBaseAttr(core.AttrSummonSick)
	foe := g.PutOnBattlefield(mage.NewCreature("Foe", "", 2, 3), b.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
	checkConditionalPT(t, g, ally.ID(), 2, 5)
	checkConditionalPT(t, g, foe.ID(), 2, 3)
	checkConditionalPT(t, g, land.ID(), 0, 0)
	a.SetAttackers(1, []string{"Ally"})
	g.RunStepWithPriority(core.DeclareAttackers)
	if !g.GetCombat().IsAttacking(ally.ID()) {
		t.Fatal("native attack rejected")
	}
	checkConditionalPT(t, g, ally.ID(), 2, 3)
	g.MutablePermanent(ally.ID()).Tapped = false
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, ally.ID(), 2, 3)
	clone := g.Clone()
	g.RemoveFromCombat(ally.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, ally.ID(), 2, 5)
	checkConditionalPT(t, clone, ally.ID(), 2, 3)
	g.MutablePermanent(ally.ID()).Tapped = true
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, ally.ID(), 2, 3)
	g.MutablePermanent(ally.ID()).Tapped = false
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, ally.ID(), 2, 5)
	g.RunStepWithPriority(core.EndCombat)
	g.RunStepWithPriority(core.Cleanup)
	checkConditionalPT(t, g, ally.ID(), 2, 5)
}

func TestRONNonattackingArcadesIdentity(t *testing.T) {
	for _, scenario := range []string{"source-phase", "recipient-phase", "source-return", "recipient-return", "control", "removed", "two"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Arcades Sabboth")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			allyCard := mage.NewCreature("Ally", "", 2, 3)
			allyCard.SetOwner(a.PlayerID())
			ally := g.PutOnBattlefield(allyCard, a.PlayerID())
			foe := g.PutOnBattlefield(mage.NewCreature("Foe", "", 2, 3), b.PlayerID())
			g.GetCombat().AddAttacker(ally.ID(), b.PlayerID())
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, ally.ID(), 2, 3)
			clone := g.Clone()
			switch scenario {
			case "source-phase":
				phased := g.PhaseOut(source.ID())
				g.RemoveFromCombat(ally.ID())
				g.ApplyContinuousEffects()
				checkConditionalPT(t, g, ally.ID(), 2, 3)
				g.PhaseIn(phased)
			case "recipient-phase":
				g.PhaseIn(g.PhaseOut(ally.ID()))
			case "source-return":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				g.RemoveFromCombat(ally.ID())
				g.ApplyContinuousEffects()
				checkConditionalPT(t, g, ally.ID(), 2, 3)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "recipient-return":
				g.BouncePermanentToHand(g.FindPermanent(ally.ID()))
				a.RemoveFromHand(allyCard.ID())
				ally = g.PutOnBattlefield(allyCard, a.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.RemoveFromCombat(ally.ID())
			case "removed":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				g.RemoveFromCombat(ally.ID())
			case "two":
				g.PutOnBattlefield(repositoryCard(t, "LEG", "Arcades Sabboth"), a.PlayerID())
				g.RemoveFromCombat(ally.ID())
			}
			g.ApplyContinuousEffects()
			want := 5
			if scenario == "control" || scenario == "removed" {
				want = 3
			}
			if scenario == "two" {
				want = 7
			}
			checkConditionalPT(t, g, ally.ID(), 2, want)
			foeT := 3
			if scenario == "control" {
				foeT = 5
			}
			checkConditionalPT(t, g, foe.ID(), 2, foeT)
			checkConditionalPT(t, clone, allyCard.ID(), 2, 3)
		})
	}
}

func TestRONNonattackingArcadesActivation(t *testing.T) {
	for _, scenario := range []string{"normal", "copy", "return", "phase", "control"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Arcades Sabboth")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 2, nil); err == nil {
				t.Fatal("unpaid activation accepted")
			}
			if g.StackSize() != 0 {
				t.Fatal("failed payment changed stack")
			}
			a.ManaPool().Add(core.White, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 2, nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 {
				t.Fatal("missing W payment")
			}
			checkConditionalPT(t, g, source.ID(), 7, 9)
			clone := g.Clone()
			switch scenario {
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveStack()
			if scenario == "phase" {
				g.PhaseIn([]uuid.UUID{source.ID()})
			}
			want := 10
			if scenario == "copy" {
				want = 11
			}
			if scenario == "return" || scenario == "phase" {
				want = 9
			}
			checkConditionalPT(t, g, source.ID(), 7, want)
			checkConditionalPT(t, clone, source.ID(), 7, 9)
			clone.ResolveStack()
			checkConditionalPT(t, clone, source.ID(), 7, 10)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, source.ID(), 7, 9)
		})
	}
}

func TestRONNonattackingConditionsAndDeclarations(t *testing.T) {
	for _, aura := range []bool{false, true} {
		subject, typeline := "ThisPermanent", "[Creature]"
		subtypes := "[]"
		enchant := ""
		if aura {
			subject, typeline, subtypes, enchant = "HostPermanent", "[Enchantment]", "[Aura]", "EnchantPermanent(IsCardtype(Creature)),"
		}
		text := fmt.Sprintf(`Card(Name: "Condition", Typeline: (Supertypes: [], Cardtypes: %s, Subtypes: %s), CardPT: (Power: 2, Toughness: 3), Rules: [%s If(PermanentPassesFilter(%s, And([IsntAttacking, Or([IsUntapped, IsBlocking])])), [PermanentLayerEffect(%s, [AdjustPT(0, 2)])])])`, typeline, subtypes, enchant, subject, subject)
		registry, err := LoadFS(fstest.MapFS{"condition.ron": {Data: []byte(text)}}, "*.ron")
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		card, err := registry.CreateCard("Condition")
		if err != nil {
			t.Fatal(err)
		}
		p := g.PutOnBattlefield(card, a.PlayerID())
		if aura {
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 3), a.PlayerID())
			g.Attach(p.ID(), host.ID())
			p = host
		}
		checkConditionalPT(t, g, p.ID(), 2, 5)
		g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
		g.ApplyContinuousEffects()
		checkConditionalPT(t, g, p.ID(), 2, 3)
		g.RemoveFromCombat(p.ID())
		g.ApplyContinuousEffects()
		checkConditionalPT(t, g, p.ID(), 2, 5)
	}
	valid := `Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []), CardPT: (Power: 1, Toughness: 1), Rules: [If(PermanentPassesFilter(ThisPermanent, IsntAttacking), [PermanentLayerEffect(ThisPermanent, [AdjustPT(0, 2)])])])`
	for _, text := range []string{
		strings.ReplaceAll(valid, "IsntAttacking", "IsntAttacking()"),
		strings.ReplaceAll(valid, "IsntAttacking", "IsntAttacking(Integer(1))"),
		strings.ReplaceAll(valid, "ThisPermanent", "HostPermanent"),
		strings.ReplaceAll(valid, "ThisPermanent", "Ref_TargetPermanent"),
		strings.ReplaceAll(valid, "IsntAttacking", "And([])"),
	} {
		if registry, err := LoadFS(fstest.MapFS{"invalid.ron": {Data: []byte(text)}}, "*.ron"); err == nil || registry != nil {
			t.Fatalf("accepted invalid declaration: %s", text)
		}
	}
	report, err := InspectFS(fstest.MapFS{"condition.ron": {Data: []byte(valid)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v", err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "IsntAttacking" && signature.Role == "permanent_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing filter role")
	}
}

func TestRONNonattackingTargetLegality(t *testing.T) {
	text := `Card(Name: "Bonus", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), ManaCost: [ManaCostW], Rules: [SpellActions(Targeted([TargetPermanent(And([IsCardtype(Creature), IsntAttacking]))], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [AdjustPT(0, 2)], UntilEndOfTurn)])))])`
	registry, err := LoadFS(fstest.MapFS{"bonus.ron": {Data: []byte(text)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"legal", "attacking", "becomes-attacking", "returned", "phased"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			targetCard := mage.NewCreature("Target", "", 2, 3)
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			spell, err := registry.CreateCard("Bonus")
			if err != nil {
				t.Fatal(err)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.White, 1)
			if scenario == "attacking" {
				g.GetCombat().AddAttacker(target.ID(), a.PlayerID())
			}
			err = g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0)
			if scenario == "attacking" {
				if err == nil || g.StackSize() != 0 || a.ManaPool().Count(core.White) != 1 || len(a.Hand()) != 1 {
					t.Fatal("illegal announcement spent resources")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "becomes-attacking" {
				g.GetCombat().AddAttacker(target.ID(), a.PlayerID())
			}
			if scenario == "returned" {
				g.BouncePermanentToHand(target)
				b.RemoveFromHand(targetCard.ID())
				g.PutOnBattlefield(targetCard, b.PlayerID())
			}
			if scenario == "phased" {
				g.PhaseOut(target.ID())
			}
			g.ResolveStack()
			if scenario == "phased" {
				g.PhaseIn([]uuid.UUID{target.ID()})
			}
			want := 3
			if scenario == "legal" {
				want = 5
			}
			checkConditionalPT(t, g, target.ID(), 2, want)
			if a.ManaPool().Count(core.White) != 0 {
				t.Fatal("resolution refunded cost")
			}
		})
	}
}
