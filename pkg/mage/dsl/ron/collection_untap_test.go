package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCollectionUntapMeekstone(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, set, "Meekstone")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(card, b.PlayerID())
			small := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
			large := g.PutOnBattlefield(repositoryCard(t, "LEB", "Hill Giant"), a.PlayerID())
			g.TapPermanent(small)
			g.TapPermanent(large)
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(1), mage.Fixed(0))), small.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.RunStepWithPriority(core.Untap)
			if !g.FindPermanent(small.ID()).Tapped || !g.FindPermanent(large.ID()).Tapped {
				t.Fatal("final power restriction")
			}
			if !g.UntapPermanent(g.FindPermanent(large.ID())) {
				t.Fatal("other untap blocked")
			}
			g.DestroyPermanent(g.FindPermanent(source.ID()))
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(small.ID()).Tapped {
				t.Fatal("departed source remained active")
			}
		})
	}
}

func TestRONCollectionUntapArena(t *testing.T) {
	for _, scenario := range []string{"resolve", "departed", "returned", "phase", "clone", "control", "copy", "late-entry"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			legend := g.PutOnBattlefield(repositoryCard(t, "LEG", "Jasmine Boreal"), a.PlayerID())
			ordinary := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
			land := g.PutOnBattlefield(repositoryCard(t, "LEG", "Karakas"), a.PlayerID())
			phased := g.PutOnBattlefield(repositoryCard(t, "LEG", "Jasmine Boreal"), b.PlayerID())
			g.PhaseOut(phased.ID())
			card := repositoryCard(t, "LEG", "Arena of the Ancients")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(card, b.PlayerID())
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || legend.Tapped || ordinary.Tapped {
				t.Fatal("entry trigger timing")
			}
			current := source.ID()
			switch scenario {
			case "departed":
				g.DestroyPermanent(source)
			case "returned":
				g.BouncePermanentToHand(source)
				current = g.PutOnBattlefield(card, b.PlayerID()).ID()
				g.PutTriggersOnStack()
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: source.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "late-entry":
				legend = g.PutOnBattlefield(repositoryCard(t, "LEG", "Lady Orca"), a.PlayerID())
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.FindPermanent(legend.ID()).Tapped {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if !g.FindPermanent(legend.ID()).Tapped || g.FindPermanent(ordinary.ID()).Tapped || g.FindPermanent(land.ID()).Tapped || g.FindPermanentIncludingPhased(phased.ID()).Tapped {
				t.Fatalf("tap selection: legend=%v ordinary=%v land=%v phased=%v", g.FindPermanent(legend.ID()).Tapped, g.FindPermanent(ordinary.ID()).Tapped, g.FindPermanent(land.ID()).Tapped, g.FindPermanentIncludingPhased(phased.ID()).Tapped)
			}
			g.TapPermanent(ordinary)
			g.TapPermanent(land)
			g.RunStepWithPriority(core.Untap)
			blocked := scenario != "departed" && scenario != "phase"
			if g.FindPermanent(legend.ID()).Tapped != blocked || g.FindPermanent(ordinary.ID()).Tapped || g.FindPermanent(land.ID()).Tapped {
				t.Fatal("untap restriction selection")
			}
			if scenario == "returned" && g.FindPermanent(current) == nil {
				t.Fatal("returned source lost")
			}
		})
	}
}

func TestRONCollectionUntapKarakas(t *testing.T) {
	for _, scenario := range []string{"legal", "nonlegendary", "land", "phase", "return", "copy", "mana"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "LEG", "Karakas")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			targetCard := repositoryCard(t, "LEG", "Jasmine Boreal")
			targetCard.SetOwner(b.PlayerID())
			if scenario == "nonlegendary" {
				targetCard = repositoryCard(t, "LEB", "Grizzly Bears")
			}
			if scenario == "land" {
				targetCard = repositoryCard(t, "LEG", "Karakas")
			}
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			if scenario == "phase" {
				g.PhaseOut(target.ID())
			}
			index := 1
			targets := []uuid.UUID{target.ID()}
			if scenario == "mana" {
				index = 0
				targets = nil
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, targets)
			if scenario == "nonlegendary" || scenario == "land" || scenario == "phase" {
				if err == nil || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("illegal target changed payment")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "mana" {
				if g.StackSize() != 0 || a.ManaPool().Count(core.White) != 1 {
					t.Fatal("mana ability metadata")
				}
				return
			}
			if !g.FindPermanent(source.ID()).Tapped || g.FindPermanent(target.ID()) == nil {
				t.Fatal("activation timing")
			}
			current := target.ID()
			if scenario == "return" {
				g.BouncePermanentToHand(target)
				b.RemoveFromHand(targetCard.ID())
				current = g.PutOnBattlefield(targetCard, b.PlayerID()).ID()
			}
			if scenario == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.FindPermanent(current) == nil {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if (g.FindPermanent(current) != nil) != (scenario == "return") {
				t.Fatal("target incarnation or owner destination")
			}
			if scenario != "return" && len(b.Hand()) != 1 {
				t.Fatal("wrong owner hand")
			}
		})
	}
}

func TestRONCollectionUntapInvalid(t *testing.T) {
	for _, rule := range []string{
		`EachPermanentRuleEffect(AnyPermanent, [DoesntUntapDuringControllersUntap()])`,
		`EachPermanentRuleEffect(AnyPermanent, [])`,
		`EachPermanentRuleEffect(AnyPermanent, [DoesntUntapDuringControllersUntap, CantBeBlocked])`,
		`EachPermanentRuleEffect(SinglePermanent(Ref_TargetPermanent), [DoesntUntapDuringControllersUntap])`,
		`EachPermanentRuleEffect(ControlledByAPlayer(SinglePlayer(You)), [DoesntUntapDuringControllersUntap])`,
		`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), ActionList([TapEachPermanent(Ref_TargetPermanent)]))`,
		`EachPermanentRuleEffect(IsSupertype(Legendary()), [DoesntUntapDuringControllersUntap])`,
		`EachPermanentRuleEffect(IsSupertype(Unknown), [DoesntUntapDuringControllersUntap])`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", rule)
		}
	}
}

func TestRONCollectionTapRiptide(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEB", "Water Elemental"), b.PlayerID())
	phased := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
	g.PhaseOut(phased.ID())
	ordinary := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
	events, intermediate := 0, 0
	observer := mage.NewArtifact("Observer", "", mage.WithAbility(mage.NewTriggered(core.EvtTapped, false, mage.GainLifeFor(mage.You, mage.Fixed(1))).SetCondition(func(event *core.GameEvent, game mage.GameReader, _, _ uuid.UUID) bool {
		if event.SourceID == first.ID() || event.SourceID == second.ID() {
			events++
			if !game.FindPermanent(first.ID()).Tapped || !game.FindPermanent(second.ID()).Tapped {
				intermediate++
			}
		}
		return false
	})))
	g.PutOnBattlefield(observer, a.PlayerID())
	spell := repositoryCard(t, "DRK", "Riptide")
	if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if first.Tapped || second.Tapped {
		t.Fatal("tap before resolution")
	}
	clone := g.Clone()
	g.ResolveStack()
	if events != 2 || intermediate != 0 || !g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped || g.FindPermanent(ordinary.ID()).Tapped || g.FindPermanent(land.ID()).Tapped || g.FindPermanentIncludingPhased(phased.ID()).Tapped {
		t.Fatal("simultaneous tap selection")
	}
	if clone.FindPermanent(first.ID()).Tapped {
		t.Fatal("clone changed with original")
	}
	clone.ResolveStack()
	if !clone.FindPermanent(second.ID()).Tapped {
		t.Fatal("clone lost collection action")
	}
}

func TestRONCollectionUntapDeclarationsAndInventory(t *testing.T) {
	rule := `EachPermanentRuleEffect(And([IsSupertype(Legendary), IsCardtype(Creature)]), [DoesntUntapDuringControllersUntap])`
	factory := permanentScopeFactory(t, "Artifact", rule)
	first, second := factory(), factory()
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared factory ability")
	}
	for _, card := range []mage.Card{first, second} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	src := `Card(Name: "Inventory", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [` + rule + `, TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), ActionList([TapEachPermanent(IsCardtype(Creature))]))])`
	report, err := InspectFS(fstest.MapFS{"inventory.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"DoesntUntapDuringControllersUntap": "permanent_rule_operation", "IsSupertype": "permanent_filter", "IsCardtype": "permanent_filter", "TapEachPermanent": "action"} {
		if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == name && s.Role == role }) {
			t.Fatalf("missing %s role for %s", role, name)
		}
	}
}
