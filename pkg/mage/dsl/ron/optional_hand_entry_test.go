package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronHandEntryPlayer struct {
	*mage.BasePlayer
	accept  bool
	choices int
}

func (p *ronHandEntryPlayer) ChooseMayAbility(string) bool { return p.accept }
func (p *ronHandEntryPlayer) ChooseCardFromHand(cards []mage.Card, _ string, _ mage.GameReader) mage.Card {
	p.choices++
	if len(cards) > 0 {
		return cards[0]
	}
	return nil
}

func TestRONOptionalHandEntryGaeasTouch(t *testing.T) {
	for _, scenario := range []string{"accept", "decline", "nonbasic", "wrong-type", "source-left", "source-phase", "copy", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronHandEntryPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "decline"}
			b := &ronHandEntryPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := repositoryCard(t, "DRK", "Gaea's Touch")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Green, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			name := "Forest"
			if scenario == "nonbasic" {
				name = "Tropical Island"
			}
			if scenario == "wrong-type" {
				name = "Llanowar Elves"
			}
			land := repositoryCard(t, "LEB", name)
			land.SetOwner(a.PlayerID())
			a.AddToHand(land)
			other := repositoryCard(t, "LEB", "Forest")
			other.SetOwner(b.PlayerID())
			b.AddToHand(other)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.choices != 0 || g.FindPermanent(land.ID()) != nil {
				t.Fatal("choice at announcement")
			}
			if scenario == "source-left" {
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			}
			if scenario == "source-phase" {
				g.PhaseOut(source.ID())
			}
			if scenario == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			}
			clone := g.Clone()
			if scenario == "clone" {
				player := clone.GetPlayer(a.PlayerID()).(*mage.SearchPlayer)
				clone.SetPlayerAt(0, &ronHandEntryPlayer{BasePlayer: player.BasePlayer, accept: true})
				clone.ResolveStack()
				if len(a.Hand()) != 1 || clone.FindPermanent(land.ID()) == nil {
					t.Fatal("clone entry or isolation")
				}
			}
			g.ResolveStack()
			want := scenario != "decline" && scenario != "nonbasic" && scenario != "wrong-type"
			if (g.FindPermanent(land.ID()) != nil) != want {
				t.Fatal("wrong filtered entry")
			}
			if (g.FindPermanent(other.ID()) != nil) != (scenario == "copy") {
				t.Fatal("wrong copied controller")
			}
			if scenario == "accept" || scenario == "decline" || scenario == "nonbasic" || scenario == "wrong-type" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
					t.Fatal("second activation allowed")
				}
				a.ManaPool().Clear()
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(source.ID()) != nil || g.StackSize() != 0 || a.ManaPool().Count(core.Green) != 2 {
					t.Fatal("sacrifice mana ability")
				}
			}
		})
	}
}

func TestRONOptionalHandEntryInvalid(t *testing.T) {
	for _, action := range []string{
		`PutACardFromHandOnBattlefield(IsCardtype(Creature),[])`,
		`PutACardFromHandOnBattlefield(IsCardtype(Creature),[EntersNormally()])`,
		`PutACardFromHandOnBattlefield(IsCardtype(Creature),[EntersTapped])`,
		`PutACardFromHandOnBattlefield(And([]),[EntersNormally])`,
		`PutACardFromHandOnBattlefield(ControlledByAPlayer(SinglePlayer(You)),[EntersNormally])`,
		`PutACardFromHandOnBattlefield(IsCreatureType(Forest),[EntersNormally])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([MayAction(` + action + `)]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", action)
		}
	}
}

func TestRONOptionalHandEntryTimingAndRollback(t *testing.T) {
	for _, scenario := range []string{"upkeep", "opponent", "nonempty-stack", "source-phased", "next-turn", "new-instance"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronHandEntryPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := repositoryCard(t, "DRK", "Gaea's Touch")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast allowed")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 || a.choices != 0 {
				t.Fatal("cast did not roll back")
			}
			a.ManaPool().Add(core.Green, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			forest := repositoryCard(t, "LEB", "Forest")
			forest.SetOwner(a.PlayerID())
			a.AddToHand(forest)
			switch scenario {
			case "upkeep":
				g.SetStep(core.Upkeep)
			case "opponent":
				g.SetActivePlayerIndex(1)
			case "source-phased":
				g.PhaseOut(source.ID())
			case "nonempty-stack":
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			case "next-turn":
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.RunStepWithPriority(core.Cleanup)
				g.SetTurn(2)
				g.SetStep(core.PrecombatMain)
			case "new-instance":
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				source = g.PutOnBattlefield(card, a.PlayerID())
			}
			beforeMana, beforeStack, beforeChoices := a.ManaPool().TotalMana(), g.StackSize(), a.choices
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			legal := scenario == "next-turn" || scenario == "new-instance"
			if (err == nil) != legal {
				t.Fatalf("legal=%v error=%v", legal, err)
			}
			if !legal && (a.ManaPool().TotalMana() != beforeMana || g.StackSize() != beforeStack || a.choices != beforeChoices) {
				t.Fatal("illegal activation changed resources")
			}
			if legal {
				g.ResolveStack()
			}
		})
	}
}

func TestRONOptionalHandEntryShapesAndStrictLoad(t *testing.T) {
	for _, filter := range []string{`IsPermanent`, `IsCardtype(Creature)`, `And([IsCreatureType(Goblin),IsPermanent])`, `And([IsSupertype(Basic),IsLandType(Forest)])`, `Or([IsCardtype(Creature),IsCardtype(Artifact)])`} {
		factory := permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([MayAction(PutACardFromHandOnBattlefield(`+filter+`,[EntersNormally]))]))`)
		if err := mage.ValidateCardDefinition("Filter", factory()); err != nil {
			t.Fatal(err)
		}
	}
	eureka := repositoryCard(t, "LEG", "Eureka")
	if err := mage.ValidateCardDefinition(eureka.Name(), eureka); err != nil {
		t.Fatal(err)
	}
	wizard := repositoryCard(t, "DRK", "Goblin Wizard")
	if err := mage.ValidateCardDefinition(wizard.Name(), wizard); err != nil {
		t.Fatal(err)
	}
	egg := repositoryCard(t, "LEG", "Triassic Egg")
	if err := mage.ValidateCardDefinition(egg.Name(), egg); err != nil {
		t.Fatal(err)
	}

}

func TestRONOptionalHandEntryStrictCollection(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	node := hostUntapRecord(t, "DRK", "Gaea's Touch")
	record := strings.Split(string(data), "\n")[node.Position.Line-1]
	files := fstest.MapFS{"one.ron": &fstest.MapFile{Data: []byte(record)}, "two.ron": &fstest.MapFile{Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatalf("load: %v", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 2 || report.UniqueCards != 1 {
		t.Fatalf("inventory: %v", err)
	}
	roles := map[string]bool{}
	for _, sig := range report.Signatures {
		if sig.Name == "IsSupertype" && sig.Role == "hand_entry_filter" {
			roles["filter"] = true
		}
		if sig.Name == "EntersNormally" && sig.Role == "entry_operation" {
			roles["entry"] = true
		}
	}
	if len(roles) != 2 {
		t.Fatal("missing hand entry inventory roles")
	}
	files["unsupported.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Unsupported",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if partial, err := LoadFS(files, "*.ron"); err == nil || partial != nil {
		t.Fatal("partial registry published")
	}
}
