package ron

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

var ronLaces = []struct {
	name  string
	color core.Color
}{
	{"Purelace", core.White}, {"Thoughtlace", core.Blue}, {"Deathlace", core.Black}, {"Chaoslace", core.Red}, {"Lifelace", core.Green},
}

func TestRONLaceCards(t *testing.T) {
	for _, tc := range ronLaces {
		for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
			for mode := range 2 {
				t.Run(fmt.Sprintf("%s/%s/%d", tc.name, set, mode), func(t *testing.T) {
					g, a, b := ronModalGame(mode)
					victimName := "Air Elemental"
					if tc.color == core.Blue {
						victimName = "Shivan Dragon"
					}
					victim := repositoryCard(t, "LEB", victimName)
					initialColors := victim.ManaCost().Colors()
					initialCost := victim.ManaCost().String()
					victim.SetOwner(b.PlayerID())
					var target uuid.UUID
					if mode == 0 {
						target = castRONSpell(t, g, b, victim).ID
					} else {
						target = g.PutOnBattlefield(victim, b.PlayerID()).ID()
					}
					card := repositoryCard(t, set, tc.name)
					if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
						t.Fatal(err)
					}
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					a.ManaPool().Add(tc.color, 1)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
						t.Fatal(err)
					}
					if a.ManaPool().TotalMana() != 0 || !slices.Equal(g.EffectiveColors(target), initialColors) {
						t.Fatal("payment or early color change")
					}
					obj := g.StackPeek()
					if len(obj.Targets) != 1 || len(obj.ModalTargets) != 0 {
						t.Fatal("wrong target group")
					}
					clone := g.Clone()
					clone.ResolveTopOfStack()
					if !slices.Equal(g.EffectiveColors(target), initialColors) {
						t.Fatal("clone changed live game")
					}
					g.ResolveTopOfStack()
					if !slices.Equal(g.EffectiveColors(target), []core.Color{tc.color}) {
						t.Fatal("wrong changed color")
					}
					g.ResolveStack()
					p := g.FindPermanent(victim.ID())
					if p == nil || !slices.Equal(p.Colors(), []core.Color{tc.color}) || p.Card.ManaCost().String() != initialCost {
						t.Fatal("color carry or mana symbols")
					}
					g.RunStepWithPriority(core.Cleanup)
					if !slices.Equal(g.FindPermanent(p.ID()).Colors(), []core.Color{tc.color}) {
						t.Fatal("color expired at cleanup")
					}
					g.PhaseOut(p.ID())
					g.PhaseIn([]uuid.UUID{p.ID()})
					if !slices.Equal(g.FindPermanent(p.ID()).Colors(), []core.Color{tc.color}) {
						t.Fatal("phasing lost color")
					}
					g.ExilePermanent(g.FindPermanent(p.ID()))
					returned, ok := g.RemoveFromExile(p.ID())
					if !ok {
						t.Fatal("missing exiled card")
					}
					q := g.PutOnBattlefield(returned, b.PlayerID())
					if !slices.Equal(q.Colors(), initialColors) {
						t.Fatal("color followed new incarnation")
					}
					if a.prompts != 0 {
						t.Fatal("branch chosen at resolution")
					}
				})
			}
		}
	}
}

func TestRONLaceInvalid(t *testing.T) {
	for _, body := range []string{
		`Targeted_Modal`, `Targeted_Modal()`, `Targeted_Modal([])`,
		`Targeted_Modal([ActionList([]),ActionList([])])`,
		`Targeted_Modal([Targeted([TargetSpell(AnySpell)],ActionList([CreateSpellEffect(Ref_TargetPermanent,[SetColor(SimpleColorList([Red]))])])),Targeted([TargetPermanent(AnyPermanent)],ActionList([]))])`,
		`Targeted_Modal([Targeted([TargetSpell(AnySpell)],ActionList([CreateSpellEffect(Ref_TargetSpell,[SetColor(SimpleColorList([Colorless]))])])),Targeted([TargetPermanent(AnyPermanent)],ActionList([]))])`,
		`Targeted_Modal([Targeted([TargetSpell(AnySpell)],ActionList([CreateSpellEffect(Ref_TargetSpell,[SetColor(SimpleColorList([Red,Blue]))])])),Targeted([TargetPermanent(AnyPermanent)],ActionList([]))])`,
		`ActionList([CreateSpellEffect(Ref_TargetSpell,[SetColor(SimpleColorList([Red]))])])`,
		`Targeted([TargetPermanent(AnyPermanent)],ActionList([CreatePermanentLayerEffect(Ref_TargetPermanent,[SetColor(SimpleColorList([Colorless]))])]))`,
	} {
		t.Run(body, func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(`+body+`)])`), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid declaration accepted")
			}
		})
	}
}

func castLace(t *testing.T, g *mage.Game, a *ronModePlayer, name string, target uuid.UUID) *mage.StackObject {
	t.Helper()
	card := repositoryCard(t, "LEB", name)
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green} {
		a.ManaPool().Add(color, 1)
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
		t.Fatal(err)
	}
	return g.StackPeek()
}

func TestRONLaceRetargetedCopy(t *testing.T) {
	for _, fromSpell := range []bool{false, true} {
		t.Run(fmt.Sprint(fromSpell), func(t *testing.T) {
			a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B")}
			g := mage.NewGame(a, b)
			g.SetActivePlayerIndex(1)
			g.SetTurn(2)
			g.SetStep(core.PrecombatMain)
			victim := repositoryCard(t, "LEB", "Air Elemental")
			stack := castRONSpell(t, g, b.BasePlayer, victim)
			p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			first, second := p.ID(), stack.ID
			if fromSpell {
				first, second = second, first
			}
			original := castLace(t, g, a, "Deathlace", first)
			b.targets = []uuid.UUID{second}
			copied := g.CopyStackObjectDirect(original, b.PlayerID(), true)
			if copied == nil || copied.Targets[0] != second || original.Targets[0] != first {
				t.Fatal("copy did not keep separate union targets")
			}
			clone := g.Clone()
			g.ResolveTopOfStack()
			if !slices.Equal(g.EffectiveColors(second), []core.Color{core.Black}) || slices.Equal(g.EffectiveColors(first), []core.Color{core.Black}) {
				t.Fatal("copy changed the original target")
			}
			g.ResolveTopOfStack()
			if !slices.Equal(g.EffectiveColors(first), []core.Color{core.Black}) {
				t.Fatal("original lost its binding")
			}
			clone.ResolveStack()
			if !slices.Equal(clone.FindPermanent(p.ID()).Colors(), []core.Color{core.Black}) || !slices.Equal(clone.FindPermanent(victim.ID()).Colors(), []core.Color{core.Black}) {
				t.Fatal("clone lost copied bindings")
			}
			if a.prompts != 0 || b.prompts != 0 {
				t.Fatal("union target used a mode choice")
			}
		})
	}
}

func TestRONLaceLegality(t *testing.T) {
	for _, scenario := range []string{"phased", "returned", "shroud", "protection", "control", "unpaid", "player", "two-targets", "stack-ability", "wrong-stack-id"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronModalGame(0)
			victim := repositoryCard(t, "LEB", "Air Elemental")
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, b.PlayerID())
			card := repositoryCard(t, "LEB", "Deathlace")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			ids := make([]uuid.UUID, 1, 2)
			ids[0] = p.ID()
			invalid := scenario == "unpaid" || scenario == "player" || scenario == "two-targets" || scenario == "stack-ability" || scenario == "wrong-stack-id"
			if scenario != "unpaid" {
				a.ManaPool().Add(core.Black, 1)
			}
			switch scenario {
			case "player":
				ids[0] = b.PlayerID()
			case "two-targets":
				ids = append(ids, p.ID())
			case "stack-ability":
				sol := g.PutOnBattlefield(repositoryCard(t, "LEB", "Rod of Ruin"), b.PlayerID())
				b.ManaPool().Add(core.Colorless, 3)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), sol.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				ids[0] = g.StackPeek().ID
			case "wrong-stack-id":
				spell := castRONSpell(t, g, b, repositoryCard(t, "LEB", "Air Elemental"))
				ids[0] = spell.SourceID
			}
			mana, stackSize := a.ManaPool().TotalMana(), g.StackSize()
			err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, 0)
			if invalid {
				if err == nil || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != mana || g.StackSize() != stackSize {
					t.Fatal("invalid announcement changed resources")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "phased":
				g.PhaseOut(p.ID())
			case "returned":
				g.ExilePermanent(p)
				returned, ok := g.RemoveFromExile(p.ID())
				if !ok {
					t.Fatal("exile failed")
				}
				g.PutOnBattlefield(returned, b.PlayerID())
			case "shroud":
				g.FindPermanent(p.ID()).GrantBaseAttr(core.Shroud)
			case "protection":
				if err := mage.ApplyEffect(g, mage.GrantProtectionTarget(core.Black), uuid.New(), b.PlayerID(), []uuid.UUID{p.ID()}); err != nil {
					t.Fatal(err)
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveStack()
			if scenario == "phased" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			want := core.Blue
			if scenario == "control" {
				want = core.Black
			}
			if !slices.Equal(g.FindPermanent(p.ID()).Colors(), []core.Color{want}) {
				t.Fatal("resolution legality or control changed color")
			}
		})
	}
}

func TestRONLaceOrderingAndProtection(t *testing.T) {
	g, a, b := ronModalGame(0)
	p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
	castLace(t, g, a, "Deathlace", p.ID())
	castLace(t, g, a, "Chaoslace", p.ID())
	g.ResolveTopOfStack()
	if !slices.Equal(g.FindPermanent(p.ID()).Colors(), []core.Color{core.Red}) {
		t.Fatal("top lace did not resolve")
	}
	g.ResolveTopOfStack()
	if !slices.Equal(g.FindPermanent(p.ID()).Colors(), []core.Color{core.Black}) {
		t.Fatal("later timestamp did not replace colors")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Green)), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(g.FindPermanent(p.ID()).Colors(), []core.Color{core.Green}) {
		t.Fatal("later temporary color failed")
	}
	g.RunStepWithPriority(core.Cleanup)
	if !slices.Equal(g.FindPermanent(p.ID()).Colors(), []core.Color{core.Black}) {
		t.Fatal("cleanup lost indefinite color")
	}
	protected := g.PutOnBattlefield(repositoryCard(t, "LEB", "White Knight"), b.PlayerID())
	bolt := castRONSpell(t, g, b, repositoryCard(t, "LEB", "Lightning Bolt"), protected.ID())
	castLace(t, g, a, "Deathlace", bolt.ID)
	g.ResolveStack()
	if g.FindPermanent(protected.ID()) == nil || g.FindPermanent(protected.ID()).Damage != 0 {
		t.Fatal("recolored damage spell bypassed protection")
	}
}

func TestRONLaceRegistryAndInventory(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), set+".ron")
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			for _, lace := range ronLaces {
				if node.Fields["Name"].String == lace.name {
					files[set+lace.name+".ron"] = &fstest.MapFile{Data: []byte(strings.Split(string(data), "\n")[node.Position.Line-1])}
				}
			}
		}
	}
	if len(files) != 30 {
		t.Fatal("missing reprints")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 5 {
		t.Fatal("reprints counted twice")
	}
	for _, lace := range ronLaces {
		first, err := registry.CreateCard(lace.name)
		if err != nil {
			t.Fatal(err)
		}
		second, err := registry.CreateCard(lace.name)
		if err != nil {
			t.Fatal(err)
		}
		a, b := first.Abilities()[0].(*mage.ActionDefinition), second.Abilities()[0].(*mage.ActionDefinition)
		if a.Targets()[0] == b.Targets()[0] {
			t.Fatal("factories share declaration identity")
		}
		if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
			t.Fatal(err)
		}
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 30 || report.UniqueCards != 5 || len(report.Issues) != 0 {
		t.Fatal("wrong accepted counts")
	}
	for _, wanted := range []struct{ name, role string }{{"Ref_TargetSpell", "spell_reference"}, {"Ref_TargetPermanent", "permanent_reference"}, {"SetColor", "resolving_modification"}, {"Red", "color"}} {
		found := false
		for _, s := range report.Signatures {
			if s.Name == wanted.name && s.Role == wanted.role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing role %s for %s", wanted.role, wanted.name)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted_Modal([]))])`)}
	if partial, err := LoadFS(files, "*.ron"); partial != nil || err == nil {
		t.Fatal("published partial registry")
	}
	if _, err := registry.CreateCard("Deathlace"); err != nil {
		t.Fatal("rejected collection changed accepted registry")
	}
}

func TestRONLaceShapeBoundaries(t *testing.T) {
	spell := `Targeted([TargetSpell(AnySpell)],ActionList([CreateSpellEffect(Ref_TargetSpell,[SetColor(SimpleColorList([Red]))])]))`
	permanent := `Targeted([TargetPermanent(AnyPermanent)],ActionList([CreatePermanentLayerEffect(Ref_TargetPermanent,[SetColor(SimpleColorList([Red]))])]))`
	for _, tc := range []struct {
		name, body, kind string
		valid            bool
	}{
		{"reversed", `Targeted_Modal([` + permanent + `,` + spell + `])`, "Instant", true},
		{"sorcery", `Targeted_Modal([` + spell + `,` + permanent + `])`, "Sorcery", true},
		{"permanent-context", `Targeted_Modal([` + spell + `,` + permanent + `])`, "Artifact", false},
		{"one-branch", `Targeted_Modal([` + spell + `])`, "Instant", false},
		{"duplicate-spell", `Targeted_Modal([` + spell + `,` + spell + `])`, "Instant", false},
		{"duplicate-permanent", `Targeted_Modal([` + permanent + `,` + permanent + `])`, "Instant", false},
		{"different-colors", `Targeted_Modal([` + spell + `,` + strings.Replace(permanent, "[Red]", "[Green]", 1) + `])`, "Instant", false},
		{"cross-reference", `Targeted_Modal([` + spell + `,` + strings.Replace(permanent, "Ref_TargetPermanent", "Ref_TargetSpell", 1) + `])`, "Instant", false},
		{"empty-color", `Targeted_Modal([` + strings.Replace(spell, "[Red]", "[]", 1) + `,` + permanent + `])`, "Instant", false},
		{"filtered-spell", `Targeted_Modal([` + strings.Replace(spell, "AnySpell", "IsCardtype(Creature)", 1) + `,` + permanent + `])`, "Instant", false},
		{"extra-action", `Targeted_Modal([` + strings.Replace(spell, `ActionList([`, `ActionList([GainLife(Integer(1)),`, 1) + `,` + permanent + `])`, "Instant", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Shape",Typeline:(Supertypes:[],Cardtypes:[`+tc.kind+`],Subtypes:[]),Rules:[SpellActions(`+tc.body+`)])`), "shape.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if (err == nil) != tc.valid {
				t.Fatalf("compilation = %v, valid = %v", err, tc.valid)
			}
			if tc.valid {
				card := factory()
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
