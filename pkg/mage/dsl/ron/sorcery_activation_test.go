package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSorceryActivationTiming(t *testing.T) {
	for _, scenario := range []string{"precombat", "postcombat", "upkeep", "combat", "opponent", "stack", "unpaid", "wrong-controller", "phase"} {
		t.Run(scenario, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(And([PayMana([ManaCostGeneric(2)]), TapPermanent(ThisPermanent)]), ActionList([GainLife(Integer(1))]), ActivateOnlyAsASorcery)`)
			g, a, b := ronGame()
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(core.Colorless, 2)
			payer := a.PlayerID()
			switch scenario {
			case "postcombat":
				g.SetStep(core.PostcombatMain)
			case "upkeep":
				g.SetStep(core.Upkeep)
			case "combat":
				g.SetStep(core.DeclareBlockers)
			case "opponent":
				g.SetActivePlayerIndex(1)
			case "stack":
				spell := repositoryCard(t, "LEB", "Giant Growth")
				spell.SetOwner(b.PlayerID())
				b.AddToHand(spell)
				b.ManaPool().Add(core.Green, 1)
				target := g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 4), b.PlayerID())
				if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			case "unpaid":
				a.ManaPool().Clear()
			case "wrong-controller":
				payer = b.PlayerID()
			case "phase":
				g.PhaseOut(source.ID())
			}
			beforeStack, beforeMana := g.StackSize(), a.ManaPool().TotalMana()
			err := g.ActivateAbilityByIndex(payer, source.ID(), 0, nil)
			legal := scenario == "precombat" || scenario == "postcombat"
			if (err == nil) != legal {
				t.Fatalf("legal=%v err=%v", legal, err)
			}
			if !legal {
				if g.StackSize() != beforeStack || a.ManaPool().TotalMana() != beforeMana || g.FindPermanentIncludingPhased(source.ID()).Tapped || a.Life() != 20 {
					t.Fatal("failed activation changed resources")
				}
				return
			}
			if a.Life() != 20 || g.StackSize() != 1 || a.ManaPool().TotalMana() != 0 || !g.FindPermanent(source.ID()).Tapped {
				t.Fatal("payment or resolution timing")
			}
			clone := g.Clone()
			g.SetStep(core.Upkeep)
			g.SetActivePlayerIndex(1)
			g.ResolveStack()
			if a.Life() != 21 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("resolution timing or clone isolation")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 21 {
				t.Fatal("pending clone lost ability")
			}
		})
	}
}

func TestRONSorceryActivationRepositoryPrerequisites(t *testing.T) {
	for _, tc := range []struct {
		set, name, gap string
		x              int
	}{
		{"DRK", "Gaea's Touch", "unsupported optional action (PutACardFromHandOnBattlefield)", 0},
		{"LEB", "Illusionary Mask", "unsupported optional payment cost (ChooseACardInHand)", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := hostUntapRecord(t, tc.set, tc.name)
			if _, err := Compile(node); err == nil || !strings.Contains(err.Error(), tc.gap) {
				t.Fatalf("complete record gap: %v", err)
			}
			var rule Node
			for _, candidate := range node.Fields["Rules"].Args {
				if candidate.Name == "ActivatedWithModifiers" {
					rule = candidate
					break
				}
			}
			if rule.Name == "" {
				t.Fatal("missing repository ability")
			}
			// A neutral body isolates timing from the unsupported action contract.
			body, err := Parse(strings.NewReader(`ActionList([GainLife(Integer(1))])`), "probe.ron")
			if err != nil {
				t.Fatal(err)
			}
			rule.Args = append([]Node(nil), rule.Args...)
			rule.Args[1] = body[0]
			option, err := compileRule(rule, scope{permanent: true})
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := mage.NewArtifact("Timing probe", "", option())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(mage.NewArtifact("Other probe", "", option()), a.PlayerID())
			if card.Abilities()[0] == other.Card.Abilities()[0] {
				t.Fatal("shared ability instance")
			}
			g.SetXValue(tc.x)
			a.ManaPool().Add(core.Colorless, tc.x)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), other.ID(), 0, nil); err == nil {
				t.Fatal("nonempty stack allowed another instance")
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.SetActivePlayerIndex(1)
			g.SetStep(core.EndStep)
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 21 {
				t.Fatal("copy repeated timing or used wrong controller")
			}
			g.SetActivePlayerIndex(0)
			g.SetStep(core.PostcombatMain)
			g.SetXValue(tc.x)
			a.ManaPool().Add(core.Colorless, tc.x)
			err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if (err == nil) != (tc.name == "Illusionary Mask") {
				t.Fatal("combined once-per-turn restriction", err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), other.ID(), 0, nil); err != nil && tc.name == "Gaea's Touch" {
				t.Fatal("other instance inherited use", err)
			}
		})
	}
}

func TestRONSorceryActivationInvalid(t *testing.T) {
	for _, modifier := range []string{`ActivateOnlyAsASorcery()`, `ActivateOnlyAsASorcery(You)`, `And([ActivateOnlyAsASorcery,ActivateOnlyAsASorcery()])`} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[ActivatedWithModifiers(PayMana([]),ActionList([GainLife(Integer(1))]),%s)])`, modifier)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", modifier)
		}
	}
	for _, rule := range []string{`ActivateOnlyAsASorcery`, `Activated(ActivateOnlyAsASorcery,ActionList([]))`, `Activated(PayMana([]),ActionList([ActivateOnlyAsASorcery]))`, `SpellActions(ActionList([ActivateOnlyAsASorcery]))`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted context", rule)
		}
	}
}

func TestRONSorceryActivationStrictInventory(t *testing.T) {
	src := `Card(Name:"Timing",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[ActivatedWithModifiers(PayMana([]),ActionList([GainLife(Integer(1))]),And([ActivateOnlyAsASorcery,ActivateOnlyOnceEachTurn]))])`
	files := fstest.MapFS{"a.ron": {Data: []byte(src)}, "b.ron": {Data: []byte(src)}}
	if _, err := LoadFS(files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 2 || report.UniqueCards != 1 {
		t.Fatalf("inventory: %v %+v", err, report)
	}
	for _, signature := range report.Signatures {
		if signature.Name == "ActivateOnlyAsASorcery" && signature.Role == "activation_modifier" {
			return
		}
	}
	t.Fatal("missing modifier role")
}

func TestRONSorceryActivationManaPlanning(t *testing.T) {
	for _, scenario := range []string{"precombat", "postcombat", "opponent", "upkeep", "stack"} {
		t.Run(scenario, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Land", `ActivatedWithModifiers(TapPermanent(ThisPermanent),ActionList([AddMana(ManaProduceU)]),ActivateOnlyAsASorcery)`)
			g, a, _ := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			switch scenario {
			case "postcombat":
				g.SetStep(core.PostcombatMain)
			case "opponent":
				g.SetActivePlayerIndex(1)
			case "upkeep":
				g.SetStep(core.Upkeep)
			case "stack":
				probe := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([GainLife(Integer(1))]))`)(), a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), probe.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			legal := scenario == "precombat" || scenario == "postcombat"
			if g.CanAfford(a.PlayerID(), core.ParseManaCost("{U}"), nil) != legal {
				t.Fatal("solver ignored timing")
			}
			if g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("query changed state")
			}
			before := g.StackSize()
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if (err == nil) != legal {
				t.Fatal("explicit mana activation disagrees", err)
			}
			if g.StackSize() != before || a.ManaPool().TotalMana() != map[bool]int{true: 1, false: 0}[legal] || g.FindPermanent(source.ID()).Tapped != legal {
				t.Fatal("mana execution or rollback")
			}
		})
	}
}

func TestRONSorceryActivationModal(t *testing.T) {
	g, a, _ := ronModalGame(0)
	g.SetActivePlayerIndex(0)
	g.SetStep(core.PrecombatMain)
	factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([ManaCostGeneric(1)]),Modal_ChooseOne([ActionList([GainLife(Integer(2))]),ActionList([GainLife(Integer(3))])]),ActivateOnlyAsASorcery)`)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	a.ManaPool().Add(core.Colorless, 1)
	g.SetStep(core.Upkeep)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("modal ability ignored timing")
	}
	if a.prompts != 0 || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
		t.Fatal("timing check followed choices or payment")
	}
	g.SetStep(core.PostcombatMain)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.SetStep(core.Upkeep)
	g.ResolveStack()
	if a.Life() != 22 || a.prompts != 1 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("mode or resolution timing")
	}
}

func TestRONSorceryActivationCombinedTiming(t *testing.T) {
	for _, modifiers := range []string{
		`And([ActivateOnlyAsASorcery,ActivateOnlyIf(PlayerPassesFilter(You,IsTheirTurn))])`,
		`And([ActivateOnlyIf(PlayerPassesFilter(You,IsTheirTurn)),ActivateOnlyAsASorcery])`,
		`And([ActivateOnlyAsASorcery,And([ActivateOnlyOnceEachTurn,ActivateOnlyIf(PlayerPassesFilter(You,IsTheirTurn))])])`,
	} {
		factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([]),ActionList([GainLife(Integer(1))]),`+modifiers+`)`)
		g, a, _ := ronGame()
		source := g.PutOnBattlefield(factory(), a.PlayerID())
		g.SetStep(core.Upkeep)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
			t.Fatal("conjunction erased sorcery timing", modifiers)
		}
		g.SetStep(core.PrecombatMain)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
			t.Fatal("conjunction erased empty-stack restriction")
		}
		g.ResolveStack()
		if a.Life() != 21 {
			t.Fatal("combined timing resolution")
		}
	}
}
