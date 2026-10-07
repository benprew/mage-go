package ron

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronColorChooser struct {
	*mage.BasePlayer
	colors  []core.Color
	prompts int
	options []string
}

func (p *ronColorChooser) ChooseMode(options []string, _ string) int {
	p.prompts++
	p.options = slices.Clone(options)
	label := colorChoiceLabel(p.colors)
	return slices.Index(options, label)
}

func colorChoiceLabel(colors []core.Color) string {
	if len(colors) == 0 {
		return "colorless"
	}
	var names []string
	for _, c := range colors {
		names = append(names, c.String())
	}
	return strings.Join(names, ", ")
}

func TestRONColorChoiceRealCards(t *testing.T) {
	for _, name := range []string{"Alchor's Tomb", "Dream Coat"} {
		for _, state := range []string{"normal", "source-left", "control", "shroud", "return", "phase", "phase-cycle", "copy", "clone"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				a := &ronColorChooser{BasePlayer: mage.NewBasePlayer("A"), colors: []core.Color{core.Green}}
				b := &ronColorChooser{BasePlayer: mage.NewBasePlayer("B"), colors: []core.Color{core.Blue}}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				card := repositoryCard(t, "LEG", name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				hostCard := mage.NewCreature("Host", "{R}", 2, 8)
				hostCard.SetOwner(a.PlayerID())
				host := g.PutOnBattlefield(hostCard, a.PlayerID())
				wrong := g.PutOnBattlefield(mage.NewCreature("Other", "{R}", 2, 8), b.PlayerID())
				a.AddToHand(card)
				var castTargets, targets []uuid.UUID
				if name == "Dream Coat" {
					castTargets = []uuid.UUID{host.ID()}
				} else {
					targets = []uuid.UUID{host.ID()}
				}
				if g.CastSpellByID(a.PlayerID(), card.ID(), castTargets, 0) == nil {
					t.Fatal("unpaid cast accepted")
				}
				a.ManaPool().Add(core.Colorless, 4)
				a.ManaPool().Add(core.Blue, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), castTargets, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				source := g.FindPermanent(card.ID())
				if source == nil || a.prompts != 0 {
					t.Fatal("entry or choice timing")
				}
				a.ManaPool().Clear()
				if name == "Alchor's Tomb" {
					if g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets) == nil {
						t.Fatal("unpaid activation")
					}
					a.ManaPool().Add(core.Colorless, 2)
					if g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{wrong.ID()}) == nil || a.ManaPool().TotalMana() != 2 || g.FindPermanent(source.ID()).Tapped {
						t.Fatal("illegal target spent cost")
					}
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
					t.Fatal(err)
				}
				if a.prompts != 0 || !slices.Equal(g.FindPermanent(host.ID()).Colors(), []core.Color{core.Red}) {
					t.Fatal("choice before resolution")
				}
				if g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets) == nil {
					t.Fatal("tap or once-per-turn restriction missing")
				}
				switch state {
				case "source-left":
					g.BouncePermanentToHand(source)
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "shroud":
					g.MutablePermanent(host.ID()).GrantBaseAttr(core.Shroud)
				case "return":
					g.BouncePermanentToHand(host)
					a.RemoveFromHand(hostCard.ID())
					g.PutOnBattlefield(hostCard, a.PlayerID())
				case "phase":
					g.PhaseOut(host.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(host.ID()))
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				if state == "copy" && name == "Dream Coat" && b.prompts != 1 {
					t.Fatal("copy did not ask its controller")
				}
				if state == "clone" && !slices.Equal(clone.FindPermanent(host.ID()).Colors(), []core.Color{core.Red}) {
					t.Fatal("pending clone changed")
				}
				clone.ResolveStack()
				if state == "phase" {
					g.PhaseIn([]uuid.UUID{host.ID()})
				}
				changed := state != "return" && state != "phase" && (name == "Dream Coat" || (state != "control" && state != "shroud"))
				want := []core.Color{core.Red}
				if changed {
					want = a.colors
				}
				if !slices.Equal(g.FindPermanent(host.ID()).Colors(), want) {
					t.Fatalf("colors=%v want=%v", g.FindPermanent(host.ID()).Colors(), want)
				}
				if name == "Alchor's Tomb" && !changed && a.prompts != 0 {
					t.Fatal("illegal target made a choice")
				}
				g.RunStepWithPriority(core.Cleanup)
				if !slices.Equal(g.FindPermanent(host.ID()).Colors(), want) {
					t.Fatal("indefinite color expired")
				}
				if changed {
					g.PhaseIn(g.PhaseOut(host.ID()))
					if !slices.Equal(g.FindPermanent(host.ID()).Colors(), want) {
						t.Fatal("phase cycle lost color")
					}
					g.BouncePermanentToHand(g.FindPermanent(host.ID()))
					a.RemoveFromHand(hostCard.ID())
					g.PutOnBattlefield(hostCard, a.PlayerID())
					if !slices.Equal(g.FindPermanent(host.ID()).Colors(), []core.Color{core.Red}) {
						t.Fatal("returned object inherited color")
					}
				}
			})
		}
	}
}

func TestRONColorChoiceAllSubsets(t *testing.T) {
	colors := []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green}
	for mask := range 32 {
		t.Run(fmt.Sprint(mask), func(t *testing.T) {
			var chosen []core.Color
			for i, color := range colors {
				if mask&(1<<i) != 0 {
					chosen = append(chosen, color)
				}
			}
			a := &ronColorChooser{BasePlayer: mage.NewBasePlayer("A"), colors: chosen}
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			host := g.PutOnBattlefield(mage.NewCreature("Host", "{R}", 2, 8), a.PlayerID())
			coat := g.PutOnBattlefield(repositoryCard(t, "LEG", "Dream Coat"), a.PlayerID())
			g.Attach(coat.ID(), host.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), coat.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if !slices.Equal(g.FindPermanent(host.ID()).Colors(), chosen) || len(a.options) != 32 {
				t.Fatalf("subset %d: %v", mask, g.FindPermanent(host.ID()).Colors())
			}
		})
	}
}

func TestRONColorChoiceInvalid(t *testing.T) {
	for _, actions := range []string{
		`ChooseAColor(AnyColor)`,
		`ChooseAColor(AnyColor()), CreatePermanentLayerEffect(ThisPermanent,[SetColor(TheChosenColor)])`,
		`ChooseAColor(Colorless), CreatePermanentLayerEffect(ThisPermanent,[SetColor(TheChosenColor)])`,
		`ChooseAColor(AnyColor), CreatePermanentLayerEffect(ThisPermanent,[SetColor(TheChosenColors)])`,
		`ChooseColors(), CreatePermanentLayerEffect(ThisPermanent,[SetColor(TheChosenColors)])`,
		`ChooseColors, CreatePermanentLayerEffect(ThisPermanent,[SetColor(TheChosenColor)])`,
		`ChooseColors, CreatePermanentLayerEffect(Ref_TargetPermanent,[SetColor(TheChosenColors)])`,
		`ChooseColors, CreatePermanentLayerEffect(ThisPermanent,[SetColor(TheChosenColors())])`,
		`CreatePermanentLayerEffect(ThisPermanent,[SetColor(TheChosenColor)]), ChooseAColor(AnyColor)`,
		`ChooseColors, CreatePermanentLayerEffect(ThisPermanent,[SetColor(TheChosenColors)]), CreatePermanentLayerEffect(ThisPermanent,[SetColor(TheChosenColors)])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+actions+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", actions)
		}
	}
}

func TestRONColorChoiceInstancesAndTurns(t *testing.T) {
	a := &ronColorChooser{BasePlayer: mage.NewBasePlayer("A"), colors: []core.Color{core.Green}}
	b := &ronColorChooser{BasePlayer: mage.NewBasePlayer("B"), colors: []core.Color{core.Blue}}
	g := mage.NewGame(a, b)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	hosts := make([]*mage.Permanent, 2)
	coats := make([]*mage.Permanent, 2)
	for i, player := range []mage.Player{a, b} {
		hosts[i] = g.PutOnBattlefield(mage.NewCreature("Host", "{R}", 2, 8), player.PlayerID())
		coats[i] = g.PutOnBattlefield(repositoryCard(t, "LEG", "Dream Coat"), player.PlayerID())
		g.Attach(coats[i].ID(), hosts[i].ID())
		if err := g.ActivateAbilityByIndex(player.PlayerID(), coats[i].ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	if coats[0].RuntimeAbilities[0] == coats[1].RuntimeAbilities[0] {
		t.Fatal("factories share abilities")
	}
	clone := g.Clone()
	g.ResolveStack()
	if !slices.Equal(g.FindPermanent(hosts[0].ID()).Colors(), a.colors) || !slices.Equal(g.FindPermanent(hosts[1].ID()).Colors(), b.colors) {
		t.Fatal("pending choices overwrote each other")
	}
	a.colors = []core.Color{core.White, core.Black}
	b.colors = nil
	clone.ResolveStack()
	if len(clone.FindPermanent(hosts[0].ID()).Colors()) != 0 || len(clone.FindPermanent(hosts[1].ID()).Colors()) != 0 || !slices.Equal(g.FindPermanent(hosts[0].ID()).Colors(), []core.Color{core.Green}) {
		t.Fatalf("clone choice isolation: %v / %v", clone.FindPermanent(hosts[0].ID()).Colors(), clone.FindPermanent(hosts[1].ID()).Colors())
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Red)), hosts[0].ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(g.FindPermanent(hosts[0].ID()).Colors(), []core.Color{core.Red}) {
		t.Fatal("later timestamp failed")
	}
	g.RunStepWithPriority(core.Cleanup)
	if !slices.Equal(g.FindPermanent(hosts[0].ID()).Colors(), []core.Color{core.Green}) {
		t.Fatal("cleanup did not restore indefinite color")
	}
	g.SetTurn(2)
	g.SetStep(core.PrecombatMain)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), coats[0].ID(), 0, nil); err != nil {
		t.Fatal("next turn activation:", err)
	}
	g.ResolveStack()
	if !slices.Equal(g.FindPermanent(hosts[0].ID()).Colors(), a.colors) {
		t.Fatal("new choice did not replace old colors")
	}
}

func TestRONColorChoiceSingleColors(t *testing.T) {
	for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green} {
		t.Run(color.String(), func(t *testing.T) {
			a := &ronColorChooser{BasePlayer: mage.NewBasePlayer("A"), colors: []core.Color{color}}
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			card := repositoryCard(t, "LEG", "Alchor's Tomb")
			target := card.Abilities()[0].(*mage.ActionDefinition).Targets()[0]
			if target == repositoryCard(t, "LEG", "Alchor's Tomb").Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("shared targets")
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{source.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if !slices.Equal(g.FindPermanent(source.ID()).Colors(), []core.Color{color}) || len(a.options) != 5 {
				t.Fatal("single-color choice failed")
			}
		})
	}
}

func TestRONColorChoiceInventory(t *testing.T) {
	report, err := InspectFS(os.DirFS("../../../../cards/mtgish"), "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range report.Issues {
		if issue.Card == "Dream Coat" || issue.Card == "Alchor's Tomb" {
			t.Fatal(issue)
		}
	}
	want := map[string]string{"ChooseAColor": "action", "ChooseColors": "action", "AnyColor": "color_choice_filter", "TheChosenColor": "color_choice_reference", "TheChosenColors": "color_choice_reference"}
	seen := map[string]bool{}
	for _, signature := range report.Signatures {
		if slices.Contains(signature.Cards, "Dream Coat") || slices.Contains(signature.Cards, "Alchor's Tomb") {
			if role, ok := want[signature.Name]; ok {
				if signature.Role != role {
					t.Fatalf("%s role %s want %s", signature.Name, signature.Role, role)
				}
				seen[signature.Name] = true
			}
		}
	}
	if len(seen) != len(want) {
		t.Fatal("missing choice inventory roles", seen)
	}
}
