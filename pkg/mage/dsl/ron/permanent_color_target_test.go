package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPermanentColorTargetLowering(t *testing.T) {
	for _, syntax := range []string{
		`IsColor(Red)`, `IsNonColor(Black)`,
		`And([IsCardtype(Creature),IsNonColor(Black),IsNonCardtype(Artifact)])`,
		`Or([IsColor(Red),And([IsColor(Blue),IsCardtype(Land)])])`,
	} {
		filter, handled, err := compileInspectableTargetFilter(nameScopeNode(t, syntax))
		if err != nil || !handled {
			t.Fatalf("color lowering %s: handled=%v error=%v", syntax, handled, err)
		}
		if _, ok := mage.InspectPermanentTarget(mage.DeclarePermanentTarget("color", filter)); !ok {
			t.Fatalf("color target lost its native predicate: %s", syntax)
		}
	}
	for _, syntax := range []string{
		`IsColor()`, `IsNonColor`, `IsColor(Unknown)`, `IsColor(Red,Blue)`,
		`IsNonColor(Black,White)`, `And([IsColor(Red),Or([])])`,
	} {
		if _, _, err := compileInspectableTargetFilter(nameScopeNode(t, syntax)); err == nil {
			t.Fatalf("invalid color target accepted: %s", syntax)
		}
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid color",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(`+syntax+`)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`)); err == nil {
			t.Fatalf("invalid card accepted: %s", syntax)
		}
	}
	for _, syntax := range []string{`And([IsColor(Red),IsEnchanted])`, `Or([IsColor(Blue),Unknown])`} {
		if _, handled, _ := compileInspectableTargetFilter(nameScopeNode(t, syntax)); handled {
			t.Fatalf("partial target claimed: %s", syntax)
		}
	}
}

func TestRONPermanentColorTargetBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(And([IsCardtype(Land),IsNonColor(Black)])),TargetPermanent(IsCardtype(Land))],ActionList([DestroyPermanent(Ref_TargetPermanent1),DestroyPermanentNoRegen(Ref_TargetPermanent2)])))`)
	g, a, b := ronGame()
	one := g.PutOnBattlefield(mage.NewLand("One"), b.PlayerID())
	two := g.PutOnBattlefield(mage.NewLand("Two"), b.PlayerID())
	first := castRONSpell(t, g, a, factory(), one.ID(), one.ID())
	second := castRONSpell(t, g, a, factory(), two.ID(), two.ID())
	watch, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.IsLand))
	visits := watch.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
	for _, p := range []*mage.Permanent{one, two} {
		g.ChangeObjectColor(p.ID(), core.Black)
		g.AddRegenerationShield(p.ID())
	}
	for _, obj := range []*mage.StackObject{first, second} {
		if certain, known := g.InspectSpellDestruction(obj, visits); !known || !certain {
			t.Fatal("illegal color group blocked the independent legal group")
		}
	}
	firstInstructions, _ := mage.InspectDirectDestruction(mage.Sequence(first.Effects...))
	plan, ok := mage.InspectPermanentTarget(firstInstructions[0].Subject)
	if !ok {
		t.Fatal("first factory lost its color declaration")
	}
	if _, known := plan.Select(g, second); known {
		t.Fatal("factory instances shared a declaration")
	}
	clone := g.Clone()
	clone.MutablePermanent(two.ID()).GrantBaseAttr(core.Shroud)
	if certain, known := clone.InspectSpellDestruction(clone.StackPeek(), visits); !known || certain {
		t.Fatal("all-illegal target gate ignored the changed clone")
	}
	clone.ResolveStack()
	if clone.FindPermanent(two.ID()) == nil || clone.FindPermanent(one.ID()) != nil {
		t.Fatal("clone did not keep independent target groups")
	}
	g.ResolveStack()
	if g.FindPermanent(one.ID()) != nil || g.FindPermanent(two.ID()) != nil {
		t.Fatalf("numbered groups did not execute independently: first=%v second=%v stack=%d", g.FindPermanent(one.ID()) != nil, g.FindPermanent(two.ID()) != nil, g.StackSize())
	}
}

func TestRONPermanentColorTargetGameplay(t *testing.T) {
	for _, name := range []string{"Terror", "Blue Elemental Blast", "Red Elemental Blast", "Active Volcano", "Flash Flood"} {
		sets := []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}
		if name == "Active Volcano" || name == "Flash Flood" {
			sets = []string{"LEG"}
		}
		for _, set := range sets {
			for _, state := range []string{"normal", "changed color", "multicolor", "colorless", "artifact", "shield", "indestructible", "shroud", "phase cycle", "phase", "return", "clone", "copy"} {
				t.Run(set+"/"+name+"/"+state, func(t *testing.T) {
					mode := 0
					if name == "Blue Elemental Blast" || name == "Red Elemental Blast" {
						mode = 1
					}
					g, a, b := ronModalGame(mode)
					mana, goodColor, badColor := "{G}", core.Green, core.Black
					switch name {
					case "Blue Elemental Blast", "Flash Flood":
						mana, goodColor, badColor = "{R}", core.Red, core.Blue
					case "Red Elemental Blast", "Active Volcano":
						mana, goodColor, badColor = "{U}", core.Blue, core.Red
					}
					card := mage.NewCreature("Subject", mana, 2, 2)
					card.SetOwner(b.PlayerID())
					p := g.PutOnBattlefield(card, b.PlayerID())
					watch, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
					visits := watch.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
					obj := castRONSpell(t, g, a.BasePlayer, repositoryCard(t, set, name), p.ID())
					want := true
					switch state {
					case "changed color":
						g.ChangeObjectColor(p.ID(), badColor)
						want = false
					case "multicolor":
						g.AddContinuousEffect(mage.BecomesColors(p.ID(), []core.Color{goodColor, badColor}, core.EndOfTurn))
						want = name != "Terror"
					case "colorless":
						g.AddContinuousEffect(mage.BecomesColors(p.ID(), nil, core.EndOfTurn))
						want = name == "Terror"
					case "artifact":
						g.FindPermanent(p.ID()).GrantBaseAttr(core.AttrIsArtifact)
						want = name != "Terror"
					case "shield":
						g.AddRegenerationShield(p.ID())
						want = name == "Terror"
					case "indestructible":
						g.FindPermanent(p.ID()).GrantBaseAttr(core.Indestructible)
						want = false
					case "shroud":
						g.FindPermanent(p.ID()).GrantBaseAttr(core.Shroud)
						want = false
					case "phase", "phase cycle":
						g.PhaseOut(p.ID())
						want = state == "phase cycle"
						if want {
							g.PhaseIn([]uuid.UUID{p.ID()})
						}
					case "return":
						g.RemoveFromBattlefield(p)
						g.PutOnBattlefield(card, b.PlayerID())
						want = false
					case "copy":
						obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
					}
					g.ApplyContinuousEffects()
					for range 2 {
						if certain, known := g.InspectSpellDestruction(obj, visits); !known || certain != want {
							t.Fatalf("certainty=%v/%v want=%v", certain, known, want)
						}
					}
					if state == "clone" {
						clone := g.Clone()
						clone.ChangeObjectColor(p.ID(), badColor)
						if certain, known := clone.InspectSpellDestruction(clone.StackPeek(), visits); !known || certain {
							t.Fatal("clone reused original colors")
						}
						clone.ResolveStack()
					}
					g.ResolveStack()
					if (g.FindPermanentIncludingPhased(p.ID()) == nil) != want {
						t.Fatal("native gameplay differs from query")
					}
					if state == "shield" && !want && !g.FindPermanent(p.ID()).Tapped {
						t.Fatal("query used the regeneration shield")
					}
					if name != "Terror" && a.prompts != 1 {
						t.Fatal("inspection changed the announced mode")
					}
				})
			}
		}
	}
}

func TestRONPermanentColorTargetAbilities(t *testing.T) {
	for _, tc := range []struct {
		set, name, mana string
	}{
		{"3ED", "Northern Paladin", "{B}"},
		{"DRK", "Exorcist", "{B}"},
		{"LEG", "Spinal Villain", "{U}"},
		{"FEM", "Homarid Shaman", "{G}"},
		{"LEG", "Horror of Horrors", "{B}"},
	} {
		for _, state := range []string{"normal", "changed color", "source departure", "copy"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				p := g.PutOnBattlefield(mage.NewCreature("Subject", tc.mana, 2, 2), b.PlayerID())
				if tc.name == "Horror of Horrors" {
					g.PutOnBattlefield(mage.NewLand("Swamp", mage.WithSubTypes("Swamp")), a.PlayerID())
				}
				for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
					a.ManaPool().Add(color, 5)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				if state == "changed color" {
					g.ChangeObjectColor(p.ID(), core.White)
				}
				if state == "source departure" {
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				}
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
				}
				want := state != "changed color"
				if tc.name != "Homarid Shaman" && tc.name != "Horror of Horrors" {
					instructions, ok := mage.InspectDirectDestruction(mage.Sequence(obj.Effects...))
					if !ok || len(instructions) != 1 {
						t.Fatal("ability lost its native instruction")
					}
					plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
					if !ok {
						t.Fatal("ability lost its color predicate")
					}
					if got, known := plan.Select(g, obj); !known || (len(got) == 1) != want {
						t.Fatal("ability query differs")
					}
				}
				g.ResolveStack()
				switch tc.name {
				case "Homarid Shaman":
					if g.FindPermanent(p.ID()).Tapped != want {
						t.Fatal("tap ignored current color")
					}
				case "Horror of Horrors":
					if g.HasRegenerationShield(p.ID()) != want {
						t.Fatal("regeneration ignored current color")
					}
				default:
					if (g.FindPermanent(p.ID()) == nil) != want {
						t.Fatal("destruction ignored current color")
					}
				}
			})
		}
	}
}
