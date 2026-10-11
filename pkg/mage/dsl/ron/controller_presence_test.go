package ron

import (
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONControllerPresenceDetection(t *testing.T) {
	for _, scenario := range []string{"own", "opponent", "token", "non-dwarf", "dwarf-phased", "source-phased", "removed", "source-control", "dwarf-control", "subtype-layer", "source-is-dwarf"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Goblins of the Flarg")
			card.SetOwner(b.PlayerID())
			var source *mage.Permanent
			if scenario == "source-phased" || scenario == "removed" {
				source = g.PutOnBattlefield(card, a.PlayerID())
				if scenario == "source-phased" {
					g.PhaseOut(source.ID())
				} else {
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
					g.ApplyContinuousEffects()
				}
			}
			dwarfCard := mage.Card(mage.NewCreature("Dwarf", "", 1, 1, mage.WithSubTypes("Dwarf")))
			if scenario == "token" {
				dwarfCard = mage.NewToken("Dwarf token", 1, 1, []core.CardType{core.TypeCreature}, []string{"Dwarf"})
			}
			if scenario == "non-dwarf" || scenario == "subtype-layer" || scenario == "source-is-dwarf" {
				dwarfCard = mage.NewCreature("Human", "", 1, 1, mage.WithSubTypes("Human"))
			}
			controller := a.PlayerID()
			if scenario == "opponent" || scenario == "source-control" || scenario == "dwarf-control" {
				controller = b.PlayerID()
			}
			dwarfCard.SetOwner(b.PlayerID())
			dwarf := g.PutOnBattlefield(dwarfCard, controller)
			if scenario == "dwarf-phased" {
				g.PhaseOut(dwarf.ID())
			}
			if source == nil {
				source = g.PutOnBattlefield(card, a.PlayerID())
			}
			switch scenario {
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: dwarf.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "dwarf-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: dwarf.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "subtype-layer", "source-is-dwarf":
				subject := dwarf
				if scenario == "source-is-dwarf" {
					g.BouncePermanentToHand(dwarf)
					subject = source
				}
				if err := mage.Until(core.EndOfTurn, mage.AddSubtypeFamily(mage.This, mage.SubtypeCreature, "Dwarf")).Apply(&mage.EffectContext{Game: g, SourceID: subject.ID(), Controller: subject.ControllerID()}); err != nil {
					t.Fatal(err)
				}
				g.ApplyContinuousEffects()
			}
			g.CheckStateBasedActions()
			if g.FindPermanentIncludingPhased(source.ID()) == nil {
				t.Fatal("state trigger bypassed stack")
			}
			g.CheckStateTriggers()
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			want := 1
			if slices.Contains([]string{"opponent", "non-dwarf", "dwarf-phased", "source-phased", "removed"}, scenario) {
				want = 0
			}
			if g.StackSize() != want {
				t.Fatalf("trigger count %d, want %d", g.StackSize(), want)
			}
			g.ResolveStack()
			if (g.FindPermanentIncludingPhased(source.ID()) == nil) != (want == 1) {
				t.Fatal("sacrifice outcome")
			}
			if want == 1 && !slices.ContainsFunc(b.Graveyard(), func(c mage.Card) bool { return c.ID() == card.ID() }) {
				t.Fatal("sacrifice did not use owner destination")
			}
		})
	}
}

func TestRONControllerPresencePending(t *testing.T) {
	for _, scenario := range []string{"dwarf-leaves", "counter", "control-after", "departed", "returned", "phase-cycle", "phased-resolution", "removed-after", "copy-own", "copy-opponent", "clone", "two", "exile", "regeneration"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			dwarf := g.PutOnBattlefield(mage.NewCreature("Dwarf", "", 1, 1, mage.WithSubTypes("Dwarf")), a.PlayerID())
			card := repositoryCard(t, "DRK", "Goblins of the Flarg")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.CheckStateTriggers()
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("pending or active trigger repeated")
			}
			switch scenario {
			case "dwarf-leaves":
				g.BouncePermanentToHand(dwarf)
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("countered trigger did not rearm")
				}
			case "control-after":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: dwarf.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "departed", "returned":
				g.BouncePermanentToHand(source)
				if scenario == "returned" {
					g.BouncePermanentToHand(dwarf)
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "phased-resolution":
				group := g.PhaseOut(source.ID())
				g.ResolveTopOfStack()
				if g.FindPermanentIncludingPhased(source.ID()) == nil {
					t.Fatal("sacrificed phased source")
				}
				g.PhaseIn(group)
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("phase-in did not rearm")
				}
			case "removed-after":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			case "copy-own", "copy-opponent":
				controller := a.PlayerID()
				if scenario == "copy-opponent" {
					controller = b.PlayerID()
				}
				if g.CopyStackObjectDirect(g.StackPeek(), controller, false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
				if (g.FindPermanent(source.ID()) == nil) != (scenario == "copy-own") {
					t.Fatal("copy sacrifice permission")
				}
			case "clone":
				clone := g.Clone()
				clone.BouncePermanentToHand(clone.FindPermanent(dwarf.ID()))
				clone.ResolveStack()
				if clone.FindPermanent(source.ID()) != nil || g.FindPermanent(source.ID()) == nil {
					t.Fatal("clone resolution isolation")
				}
			case "two":
				other := g.PutOnBattlefield(repositoryCard(t, "DRK", "Goblins of the Flarg"), a.PlayerID())
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 2 {
					t.Fatal("instance arming collision")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(other.ID()) != nil || g.FindPermanent(source.ID()) == nil {
					t.Fatal("wrong instance sacrificed")
				}
			case "exile":
				g.AddExileIfWouldGoToGraveyardThisTurn(source.ID(), dwarf.ID())
			case "regeneration":
				g.AddRegenerationShield(source.ID())
			}
			g.ResolveStack()
			survives := scenario == "control-after" || scenario == "returned"
			if (g.FindPermanent(source.ID()) != nil) != survives {
				t.Fatal("pending controller or source visit changed")
			}
			if scenario == "exile" && g.FindExiledCard(source.ID()) == nil {
				t.Fatal("sacrifice replacement bypassed")
			}
		})
	}
}

func TestRONControllerPresenceCastingAndMountainwalk(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Goblins of the Flarg")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	if g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("failed cast changed state")
	}
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	source := g.FindPermanent(card.ID())
	if source == nil || source.CurrentPower(g) != 1 || source.CurrentToughness(g) != 1 || !source.HasSubType("Goblin") || !source.HasSubType("Warrior") || a.ManaPool().TotalMana() != 0 {
		t.Fatal("printed creature or payment")
	}
	own := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), a.PlayerID())
	if mage.HasLandwalkEvasion(g.FindPermanent(source.ID()), b.PlayerID(), g) {
		t.Fatal("Mountainwalk used attacker's land")
	}
	mountain := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
	if !mage.HasLandwalkEvasion(g.FindPermanent(source.ID()), b.PlayerID(), g) {
		t.Fatal("defender Mountain did not prevent block")
	}
	group := g.PhaseOut(mountain.ID())
	if mage.HasLandwalkEvasion(g.FindPermanent(source.ID()), b.PlayerID(), g) {
		t.Fatal("phased Mountain prevented block")
	}
	g.PhaseIn(group)
	if !mage.HasLandwalkEvasion(g.FindPermanent(source.ID()), b.PlayerID(), g) {
		t.Fatal("phase-in did not restore Mountainwalk")
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
	g.ApplyContinuousEffects()
	if mage.HasLandwalkEvasion(g.FindPermanent(source.ID()), b.PlayerID(), g) {
		t.Fatal("removed Mountainwalk prevented block")
	}
	if g.FindPermanent(own.ID()) == nil {
		t.Fatal("land was moved")
	}
}

func TestRONControllerPresenceInvalid(t *testing.T) {
	valid := `TriggerA(WhenAPlayerControlsAPermanent(SinglePlayer(You),IsCreatureType(Dwarf)),ActionList([SacrificePermanent(ThisPermanent)]))`
	for _, player := range []string{"You", "AnyPlayer", "SinglePlayer", "SinglePlayer()", "SinglePlayer(You,You)", "SinglePlayer(You())", "SinglePlayer(Opponent)", "SinglePlayer(Trigger_ThatPlayer)", "SinglePlayer(ActionPlayer)", "SinglePlayer(Ref_TargetPlayer)"} {
		assertConditionalRejected(t, strings.Replace(valid, "SinglePlayer(You)", player, 1))
	}
	for _, filter := range []string{"IsCreatureType", "IsCreatureType()", "IsCreatureType(Dwarf,Dwarf)", "IsCreatureType(Dwarf())", "SinglePermanent(ThisPermanent)", "Other(ThisPermanent)", "IsColor(Unknown)", "And([])", "Or([])", "And([IsNonToken,Or([])])", "Or([And([])])"} {
		assertConditionalRejected(t, strings.Replace(valid, "IsCreatureType(Dwarf)", filter, 1))
	}
	for _, event := range []string{"WhenAPlayerControlsAPermanent", "WhenAPlayerControlsAPermanent()", "WhenAPlayerControlsAPermanent(SinglePlayer(You))", "WhenAPlayerControlsAPermanent(SinglePlayer(You),IsCreatureType(Dwarf),AnyPermanent)"} {
		assertConditionalRejected(t, `TriggerA(`+event+`,ActionList([SacrificePermanent(ThisPermanent)]))`)
	}
	for _, body := range []string{`PlayerActions(Trigger_ThatPlayer,[GainLife(Integer(1))])`, `SacrificePermanent(Trigger_ThatPermanent)`, `ControllersSacrificeEachPermanent(Trigger_ThosePermanents)`} {
		assertConditionalRejected(t, strings.Replace(valid, "SacrificePermanent(ThisPermanent)", body, 1))
	}
	assertConditionalRejected(t, `FromStack(`+valid+`)`)
}

func TestRONControllerPresenceRecords(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var source string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Goblins of the Flarg"`) {
			source = line
		}
	}
	if source == "" {
		t.Fatal("missing record")
	}
	files := fstest.MapFS{"DRK.ron": &fstest.MapFile{Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Goblins of the Flarg")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Goblins of the Flarg")
	if err != nil {
		t.Fatal(err)
	}
	if first.Abilities()[1].AbilityID() == second.Abilities()[1].AbilityID() {
		t.Fatal("shared state declaration")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(source, "SinglePlayer(You)", "SinglePlayer(You())", 1))}
	if partial, err := LoadFS(files, "*.ron"); err == nil || partial != nil {
		t.Fatal("partial registry published")
	}
	if LoadInto(registry, files, "*.ron") == nil {
		t.Fatal("invalid replacement accepted")
	}
	if _, err := registry.CreateCard("Goblins of the Flarg"); err != nil {
		t.Fatal("failed load erased registry")
	}
	for _, node := range nodes {
		if node.Fields["Name"].String == "Mana Vortex" {
			if _, err := Compile(node); err != nil {
				t.Fatalf("self-cast record rejected: %v", err)
			}
		}
	}
}

func TestRONControllerPresenceFilterComposition(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerControlsAPermanent(SinglePlayer(You),And([IsNonToken,Or([IsCreatureType(Dwarf),IsCreatureType(Goblin)])])),ActionList([GainLife(Integer(1))]))`)
	g, a, _ := ronGame()
	g.PutOnBattlefield(factory(), a.PlayerID())
	token := g.PutOnBattlefield(mage.NewToken("Dwarf", 1, 1, []core.CardType{core.TypeCreature}, []string{"Dwarf"}), a.PlayerID())
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("nested filter included token")
	}
	g.PutOnBattlefield(mage.NewCreature("Goblin", "", 1, 1, mage.WithSubTypes("Goblin")), a.PlayerID())
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("nested filter did not match")
	}
	g.ResolveTopOfStack()
	if a.Life() != 21 || g.FindPermanent(token.ID()) == nil {
		t.Fatal("action body or collection")
	}
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("resolved presence trigger did not rearm")
	}
}

func TestRONControllerPresencePhaseRestoration(t *testing.T) {
	for _, subject := range []string{"source", "dwarf"} {
		t.Run(subject, func(t *testing.T) {
			g, a, _ := ronGame()
			dwarf := g.PutOnBattlefield(mage.NewCreature("Dwarf", "", 1, 1, mage.WithSubTypes("Dwarf")), a.PlayerID())
			group := g.PhaseOut(dwarf.ID())
			source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Goblins of the Flarg"), a.PlayerID())
			if subject == "source" {
				sourceGroup := g.PhaseOut(source.ID())
				g.PhaseIn(group)
				group = sourceGroup
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("phased state detected")
			}
			g.PhaseIn(group)
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("restored state did not trigger")
			}
			g.ResolveStack()
			if g.FindPermanent(source.ID()) != nil {
				t.Fatal("restored state did not sacrifice")
			}
		})
	}
}

func TestRONControllerPresenceInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var source string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Goblins of the Flarg"`) {
			source = line
		}
	}
	report, err := InspectFS(fstest.MapFS{"DRK.ron": &fstest.MapFile{Data: []byte(source)}}, "*.ron")
	if err != nil || report.Compiled != 1 || report.UniqueCards != 1 || len(report.Issues) != 0 {
		t.Fatalf("inventory: %v %v", report, err)
	}
	roles := map[string]string{"SinglePlayer": "player_filter", "You": "player_reference", "IsCreatureType": "permanent_filter"}
	for name, role := range roles {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role for %s", role, name)
		}
	}
}

func TestRONControllerPresenceTargets(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerControlsAPermanent(SinglePlayer(You),IsCreatureType(Dwarf)),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([DestroyPermanent(Ref_TargetPermanent),GainLife(Integer(1))])))`)
	for _, scenario := range []string{"legal", "returned", "phased", "shroud"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := opponentTargetGame()
			g.PutOnBattlefield(factory(), a.PlayerID())
			card := mage.NewCreature("Victim", "", 1, 1)
			card.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(card, b.PlayerID())
			a.requested = []uuid.UUID{victim.ID()}
			g.PutOnBattlefield(mage.NewCreature("Dwarf", "", 1, 1, mage.WithSubTypes("Dwarf")), a.PlayerID())
			if len(a.candidates) != 0 {
				t.Fatal("state detection chose targets before placement")
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || len(a.candidates) != 1 || g.FindPermanent(victim.ID()) == nil {
				t.Fatal("target placement timing")
			}
			switch scenario {
			case "returned":
				g.BouncePermanentToHand(victim)
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "phased":
				g.PhaseOut(victim.ID())
			case "shroud":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
			}
			g.ResolveTopOfStack()
			if scenario == "legal" {
				if g.FindPermanent(victim.ID()) != nil || a.Life() != 21 {
					t.Fatal("legal target body")
				}
			} else if g.FindPermanentIncludingPhased(victim.ID()) == nil || a.Life() != 20 {
				t.Fatal("illegal target body resolved")
			}
		})
	}
}
