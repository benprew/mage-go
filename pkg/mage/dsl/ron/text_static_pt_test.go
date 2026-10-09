package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func textStaticID(t *testing.T, p *mage.Permanent) uuid.UUID {
	t.Helper()
	for _, ability := range p.RuntimeAbilities {
		if _, ok := ability.(*mage.StaticAbilityHolder); ok {
			return ability.AbilityID()
		}
	}
	t.Fatal("missing static declaration")
	return uuid.Nil
}

func staticPTCheck(t *testing.T, g *mage.Game, p *mage.Permanent, power, toughness int) {
	t.Helper()
	current := g.FindPermanent(p.ID())
	if current == nil || current.CurrentPower(g) != power || current.CurrentToughness(g) != toughness {
		t.Fatalf("P/T for %s: want %d/%d; got %v", p.Name(), power, toughness, current)
	}
}

func TestRONTextStaticPTColors(t *testing.T) {
	for _, tc := range []struct {
		set, name        string
		color            core.Color
		power, toughness int
	}{
		{"LEA", "Bad Moon", core.Black, 1, 1}, {"LEA", "Crusade", core.White, 1, 1},
		{"DRK", "Sunken City", core.Blue, 1, 1},
		{"LEG", "Jacques le Vert", core.Green, 0, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			old := g.PutOnBattlefield(textColorCard(tc.color), a.PlayerID())
			red := g.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
			other := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
			id := textStaticID(t, source)
			change, _ := mage.ReplaceColorWord(tc.color, core.Red)
			if tc.color == core.Red {
				change, _ = mage.ReplaceColorWord(core.Red, core.Blue)
			}
			if err := g.RevisePrintedStaticPTAbility(source, id, change); err != nil {
				t.Fatal(err)
			}
			if textStaticID(t, g.FindPermanent(source.ID())) != id {
				t.Fatal("active identity changed")
			}
			if tc.color != core.Red {
				staticPTCheck(t, g, old, 1, 1)
				staticPTCheck(t, g, red, 1+tc.power, 1+tc.toughness)
				wantP, wantT := 1+tc.power, 1+tc.toughness
				if tc.name == "Jacques le Vert" {
					wantP, wantT = 1, 1
				}
				staticPTCheck(t, g, other, wantP, wantT)
			} else {
				staticPTCheck(t, g, red, 1, 1)
			}
			if source.Card.ManaCost().String() != repositoryCard(t, tc.set, tc.name).ManaCost().String() {
				t.Fatal("printed cost changed")
			}
		})
	}
}

func TestRONTextStaticPTLifetime(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Bad Moon")
	source := g.PutOnBattlefield(card, a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bad Moon"), b.PlayerID())
	black := g.PutOnBattlefield(textColorCard(core.Black), b.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
	id := textStaticID(t, source)
	change, _ := mage.ReplaceColorWord(core.Black, core.Red)
	if err := g.RevisePrintedStaticPTAbility(source, textStaticID(t, second), change); err == nil {
		t.Fatal("other instance ID accepted")
	}
	if err := g.RevisePrintedStaticPTAbility(source, id, change); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, black, 2, 2)
	staticPTCheck(t, g, red, 2, 2)
	departed := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	g.BouncePermanentToHand(departed)
	if lki := g.LKI(departed.ID()); lki == nil || lki.Power != 2 || lki.Toughness != 2 {
		t.Fatal("departure lost the revised P/T")
	}
	clone := g.Clone()
	next, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := clone.RevisePrintedStaticPTAbility(clone.FindPermanent(source.ID()), id, next); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, clone, red, 1, 1)
	staticPTCheck(t, g, red, 2, 2)
	if clone.LKI(departed.ID()).Power != 2 || g.LKI(departed.ID()).Power != 2 {
		t.Fatal("revision changed departure information")
	}
	phased := g.PhaseOut(source.ID())
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, red, 1, 1)
	if err := g.RevisePrintedStaticPTAbility(source, id, next); err == nil {
		t.Fatal("phased source accepted")
	}
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, red, 2, 2)
	remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
	removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(remover.ID())
	g.AddContinuousEffect(removal)
	staticPTCheck(t, g, red, 1, 1)
	if err := g.RevisePrintedStaticPTAbility(source, id, next); err == nil {
		t.Fatal("removed declaration accepted")
	}
	g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
	staticPTCheck(t, g, red, 2, 2)
	if err := g.RevisePrintedStaticPTAbility(g.FindPermanent(source.ID()), id, next); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, red, 1, 1)
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	staticPTCheck(t, g, green, 2, 2)
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	staticPTCheck(t, g, green, 1, 1)
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.RevisePrintedStaticPTAbility(source, id, change); err == nil {
		t.Fatal("returned source inherited old visit")
	}
	staticPTCheck(t, g, black, 3, 3)
	staticPTCheck(t, g, red, 1, 1)
	if textStaticID(t, returned) != id {
		t.Fatal("printed declaration changed")
	}
}

func TestRONTextStaticPTIndependentDeclarations(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(And([IsColor(Red),IsCardtype(Creature)]),[AdjustPT(1,1)]),EachPermanentLayerEffect(And([IsColor(Red),IsCardtype(Creature)]),[AdjustPT(2,2)])`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	first, second := source.RuntimeAbilities[0].AbilityID(), source.RuntimeAbilities[1].AbilityID()
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedStaticPTAbility(source, first, change); err != nil {
		t.Fatal(err)
	}
	change, _ = mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.RevisePrintedStaticPTAbility(source, second, change); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, red, 1, 1)
	staticPTCheck(t, g, green, 2, 2)
	staticPTCheck(t, g, blue, 3, 3)
}

func TestRONTextStaticPTControlAndShapes(t *testing.T) {
	for _, tc := range []struct{ set, name string }{
		{"LEA", "Castle"}, {"LEA", "Orcish Oriflamme"}, {"FEM", "Thrull Champion"},
		{"LEG", "Arcades Sabboth"}, {"LEG", "Kobold Taskmaster"}, {"ATQ", "Mightstone"}, {"ATQ", "Weakstone"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			subject := func(player mage.Player) *mage.Permanent {
				return g.PutOnBattlefield(mage.NewCreature("Subject", "", 1, 1, mage.WithSubTypes("Thrull", "Kobold")), player.PlayerID())
			}
			own, other := subject(a), subject(b)
			attacking := tc.name == "Orcish Oriflamme" || tc.name == "Mightstone" || tc.name == "Weakstone"
			if attacking {
				g.GetCombat().AddAttacker(own.ID(), b.PlayerID())
				g.GetCombat().AddAttacker(other.ID(), a.PlayerID())
			}
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := textStaticID(t, p)
			change, _ := mage.ReplaceColorWord(core.Blue, core.Red)
			if err := g.RevisePrintedStaticPTAbility(p, id, change); err != nil {
				t.Fatal(err)
			}
			if textStaticID(t, g.FindPermanent(p.ID())) != id {
				t.Fatal("identity changed")
			}
			power, toughness := 1, 1
			switch tc.name {
			case "Castle", "Arcades Sabboth":
				toughness = 3
			case "Orcish Oriflamme", "Mightstone", "Kobold Taskmaster":
				power = 2
			case "Weakstone":
				power = 0
			case "Thrull Champion":
				power, toughness = 2, 2
			}
			staticPTCheck(t, g, own, power, toughness)
			if tc.name == "Mightstone" || tc.name == "Weakstone" || tc.name == "Thrull Champion" {
				staticPTCheck(t, g, other, power, toughness)
			} else {
				staticPTCheck(t, g, other, 1, 1)
			}
			if tc.name == "Thrull Champion" {
				staticPTCheck(t, g, p, 3, 3)
			}
			if tc.name == "Kobold Taskmaster" {
				staticPTCheck(t, g, p, 1, 2)
			}
			if tc.name == "Castle" || tc.name == "Arcades Sabboth" {
				g.TapPermanent(g.FindPermanent(own.ID()))
				g.ApplyContinuousEffects()
				staticPTCheck(t, g, own, 1, 1)
			}
			if attacking {
				g.GetCombat().RemoveFromCombat(own.ID())
				g.ApplyContinuousEffects()
				staticPTCheck(t, g, own, 1, 1)
			}
		})
	}
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Jacques le Vert"), a.PlayerID())
	redA := g.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
	redB := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	change, _ := mage.ReplaceColorWord(core.Green, core.Red)
	if err := g.RevisePrintedStaticPTAbility(source, textStaticID(t, source), change); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, redA, 1, 3)
	staticPTCheck(t, g, redB, 1, 1)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	staticPTCheck(t, g, redA, 1, 1)
	staticPTCheck(t, g, redB, 1, 3)
	for _, rule := range []string{`EachPermanentLayerEffect(IsCardtype(Creature),[AdjustPT(1)])`, `EachPermanentLayerEffect(IsCardtype(Creature),[AdjustPT(ValueX,1)])`, `EachPermanentLayerEffect(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer)),[AdjustPT(1,1)])`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),ManaCost:[],Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
}

func TestRONTextStaticPTBoundaries(t *testing.T) {
	change, _ := mage.ReplaceColorWord(core.Green, core.Red)
	for _, name := range []string{"Holy Strength", "Goblin King"} {
		g, a, _ := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, "LEA", name), a.PlayerID())
		if err := g.RevisePrintedStaticPTAbility(source, textStaticID(t, source), change); err == nil {
			t.Fatal("unmarked shape accepted", name)
		}
	}
	for _, tc := range []struct{ set, name string }{{"LEG", "Rohgahh of Kher Keep"}, {"LEA", "Magical Hack"}, {"LEA", "Sleight of Mind"}} {
		if _, err := Compile(staticBaseRecord(t, tc.set, tc.name)); err == nil {
			t.Fatal("dependent record accepted", tc.name)
		}
	}
}
