package ron

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONChosenSourceFilterDeclarations(t *testing.T) {
	for _, expression := range []string{`AnyDamageSource()`, `AnyDamageSource(You)`, `IsColor()`, `IsColor(Red,Blue)`, `IsColor(Colorless)`, `IsColor(AnyColor)`, `IsColor(Red())`, `IsCardtype(Unknown)`, `IsCardtype(Artifact())`, `Or([])`, `And([])`, `Or([IsColor(Red),You])`, `Or(IsColor(Red))`, `SinglePermanent(ThisPermanent)`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([ChooseADamageSource(` + expression + `)]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		_, err = Compile(nodes[0])
		if err == nil || strings.Contains(err.Error(), "unsupported action (ChooseADamageSource)") {
			t.Fatal("filter did not receive declaration validation", expression, err)
		}
	}
}

func chosenSourceFilter(t *testing.T, set, name string) (mage.DamageSourceFilter, Node) {
	t.Helper()
	nodes, err := Parse(strings.NewReader(mustRONSet(t, set)), set+".ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, card := range nodes {
		if card.Fields["Name"].String != name {
			continue
		}
		var found *Node
		var walk func(Node)
		walk = func(n Node) {
			if n.Name == "ChooseADamageSource" {
				choiceNode := n
				found = &choiceNode
			}
			for _, a := range n.Args {
				walk(a)
			}
			for _, v := range n.Fields {
				walk(v)
			}
		}
		walk(card)
		if found == nil {
			t.Fatal("choice missing", name)
		}
		filter, err := compileChosenDamageSourceFilter(found.Args[0])
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(card); name == "Eye for an Eye" {
			if err == nil || strings.Contains(err.Error(), "ChooseADamageSource") {
				t.Fatal("unsupported replacement body was accepted or choice rejected", name, err)
			}
		} else if err != nil {
			t.Fatal("complete source-choice record rejected", name, err)
		}
		return filter, card
	}
	t.Fatal("missing card", name)
	return mage.DamageSourceFilter{}, Node{}
}

func TestRONChosenSourceFilterRecords(t *testing.T) {
	for _, tc := range []struct {
		set, name      string
		blue, artifact bool
		colors         []core.Color
	}{
		{"LEB", "Circle of Protection: Black", false, false, []core.Color{core.Black}}, {"LEA", "Circle of Protection: Blue", true, false, []core.Color{core.Blue}},
		{"LEA", "Circle of Protection: Green", false, false, []core.Color{core.Green}}, {"LEA", "Circle of Protection: Red", false, false, []core.Color{core.Red}},
		{"LEA", "Circle of Protection: White", false, false, []core.Color{core.White}}, {"ATQ", "Circle of Protection: Artifacts", false, true, nil},
		{"LEG", "Greater Realm of Preservation", false, false, []core.Color{core.Black, core.Red}}, {"LEA", "Jade Monolith", true, true, []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green}},
		{"LEA", "Reverse Damage", true, true, []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green}}, {"ARN", "Eye for an Eye", true, true, []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green}}, {"DRK", "Dark Sphere", true, true, []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			filter, _ := chosenSourceFilter(t, tc.set, tc.name)
			g, a, _ := ronGame()
			blue := g.PutOnBattlefield(mage.NewCreature("Blue", "{U}", 1, 1), a.PlayerID())
			artifact := g.PutOnBattlefield(mage.NewArtifact("Artifact", ""), a.PlayerID())
			if filter.Match(g.CaptureDamageSource(blue.ID())) != tc.blue || filter.Match(g.CaptureDamageSource(artifact.ID())) != tc.artifact {
				t.Fatal("record filter mismatch")
			}
			for color, cost := range map[core.Color]string{core.White: "{W}", core.Blue: "{U}", core.Black: "{B}", core.Red: "{R}", core.Green: "{G}"} {
				colored := g.PutOnBattlefield(mage.NewCreature("Color source", cost, 1, 1), a.PlayerID())
				if filter.Match(g.CaptureDamageSource(colored.ID())) != slices.Contains(tc.colors, color) {
					t.Fatal("record color filter mismatch", color)
				}
			}
		})
	}
}

func TestRONChosenSourceFilterGameplay(t *testing.T) {
	for _, scenario := range []string{"live", "recolored", "departed", "returned", "phased", "copy", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			filter, _ := chosenSourceFilter(t, "LEA", "Circle of Protection: Blue")
			_, a, b := ronGame()
			chooser := &ronSourceChoicePlayer{BasePlayer: b}
			g := mage.NewGame(a, chooser)
			card := repositoryCard(t, "LEA", "Prodigal Sorcerer")
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			event, choice, amount := mage.BindDamageSource("event"), mage.BindDamageSource("choice"), mage.BindNumber("amount")
			fixture := mage.NewEnchantment("Filter fixture", "", mage.WithAbility(mage.WhenEvent(mage.ControllerIsDealtDamage(amount).WithSource(event), mage.Sequence(
				mage.ChooseDamageSourceMatchingFrom(mage.You, []mage.DamageSourceRef{mage.FromDamageSource(event)}, filter, choice),
				mage.DealDamageFrom(mage.FromDamageSource(choice), mage.ActivePlayer, mage.Fixed(2))))))
			if err := mage.ValidateCardDefinition(fixture.Name(), fixture); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(fixture, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if chooser.calls != 0 {
				t.Fatal("choice ran at announcement")
			}
			g.ResolveTopOfStack()
			g.PutTriggersOnStack()
			original := g.CaptureDamageSource(source.ID())
			switch scenario {
			case "recolored":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Red)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "departed", "returned":
				g.RemoveFromBattlefield(source)
				if scenario == "returned" {
					returned := g.PutOnBattlefield(card, a.PlayerID())
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Red)), returned.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.StackSize() == 0 || chooser.calls != 0 {
				t.Fatal("clone changed original choice")
			}
			g.ResolveStack()
			want := 1
			if scenario == "copy" {
				want = 2
			}
			if chooser.calls != want || !chooser.source.SameObject(original) || !chooser.source.HasColor(core.Blue) || a.Life() != 20-2*want || b.Life() != 19 {
				t.Fatal("saved event filter or identity changed")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != a.Life() {
				t.Fatal("clone filter result changed")
			}
		})
	}
}

func TestRONChosenSourceFilterInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([ChooseADamageSource(Or([IsColor(Red),IsCardtype(Artifact)]))]))])`
	report, err := InspectFS(fstest.MapFS{"filter.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, sig := range report.Signatures {
		if sig.Name == "Or" || sig.Name == "IsColor" || sig.Name == "IsCardtype" {
			if sig.Role != "damage_source_filter" {
				t.Fatal("lost filter role", sig)
			}
		}
		if sig.Name == "Red" && sig.Role != "color" {
			t.Fatal("lost color role")
		}
		if sig.Name == "Artifact" && sig.Role != "card_type" {
			t.Fatal("lost type role")
		}
	}
}
