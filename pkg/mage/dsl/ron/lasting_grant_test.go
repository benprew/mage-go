package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLastingGrantDelusion(t *testing.T) {
	for _, change := range []string{"normal", "zero-power", "control", "return", "phase", "cycle", "remove-before", "remove-after", "copy", "clone", "pending-return", "pending-copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			wall := g.PutOnBattlefield(repositoryCard(t, "LEB", "Wall of Wood"), b.PlayerID())
			power := 2
			if change == "zero-power" {
				power = 0
			}
			card := mage.NewCreature("Attacker", "", power, 5)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: wall.ID(), AttackerID: p.ID()}})
			g.GetCombat().Reset()
			spell := repositoryCard(t, "LEG", "Glyph of Delusion")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID(), p.ID()}, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID(), p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if change == "remove-before" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			}
			if change == "copy" && g.CopySpellOnStack(spell.ID(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			want := power
			if change == "copy" {
				want *= 2
			}
			if got := g.FindPermanent(p.ID()).Counters[core.Glyph]; got != want {
				t.Fatalf("counters %d want %d", got, want)
			}
			clone := g.Clone()
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "return":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				p = g.PutOnBattlefield(card, a.PlayerID())
				g.MutablePermanent(p.ID()).AddCounter(core.Glyph, 2)
			case "phase":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "remove-after":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			case "clone":
				clone.MutablePermanent(p.ID()).Counters[core.Glyph] = 0
			}
			if change == "phase" {
				queueRONUpkeep(g, a)
				if g.StackSize() != 0 {
					t.Fatal("phased trigger")
				}
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			g.TapPermanent(g.FindPermanent(p.ID()))
			active := 0
			player := a
			if change == "control" {
				active = 1
				player = b
			}
			g.SetActivePlayerIndex(active)
			g.RunStepWithPriority(core.Untap)
			blocked := change != "zero-power" && change != "return" && change != "remove-after"
			if g.FindPermanent(p.ID()).Tapped != blocked {
				t.Fatal("untap decision")
			}
			otherPlayer := b
			if change == "control" {
				otherPlayer = a
			}
			queueRONUpkeep(g, otherPlayer)
			if g.StackSize() != 0 {
				t.Fatal("trigger used the wrong controller")
			}
			queueRONUpkeep(g, player)
			triggers := 1
			if change == "copy" {
				triggers = 2
			}
			if change == "return" || change == "remove-after" {
				triggers = 0
			}
			if g.StackSize() != triggers {
				t.Fatalf("triggers %d want %d", g.StackSize(), triggers)
			}
			before := g.FindPermanent(p.ID()).Counters[core.Glyph]
			if change == "pending-return" {
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				p = g.PutOnBattlefield(card, a.PlayerID())
				g.MutablePermanent(p.ID()).AddCounter(core.Glyph, 4)
				before = 4
				triggers = 0
			}
			if change == "pending-copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("trigger copy")
				}
				triggers++
			}
			g.ResolveStack()
			expected := max(0, before-triggers)
			if got := g.FindPermanent(p.ID()).Counters[core.Glyph]; got != expected {
				t.Fatalf("after upkeep %d want %d", got, expected)
			}
			cloneWant := want
			if change == "clone" {
				cloneWant = 0
			}
			if clone.FindPermanent(p.ID()).Counters[core.Glyph] != cloneWant {
				t.Fatal("clone changed")
			}
			queueRONUpkeep(clone, clone.GetPlayer(a.PlayerID()))
			cloneTriggers := 1
			if change == "copy" {
				cloneTriggers = 2
			}
			if clone.StackSize() != cloneTriggers {
				t.Fatal("clone lost granted trigger")
			}
			clone.ResolveStack()
			if clone.FindPermanent(p.ID()).Counters[core.Glyph] != max(0, cloneWant-cloneTriggers) || g.FindPermanent(p.ID()).Counters[core.Glyph] != expected {
				t.Fatal("clone resolution isolation")
			}
			if change == "normal" {
				queueRONUpkeep(g, player)
				g.ResolveStack()
				g.TapPermanent(g.FindPermanent(p.ID()))
				g.RunStepWithPriority(core.Untap)
				if g.FindPermanent(p.ID()).Tapped {
					t.Fatal("zero counters still prevented untap")
				}
				g.MutablePermanent(p.ID()).AddCounter(core.Glyph, 1)
				g.TapPermanent(g.FindPermanent(p.ID()))
				g.RunStepWithPriority(core.Untap)
				if !g.FindPermanent(p.ID()).Tapped {
					t.Fatal("new counter failed to restore restriction")
				}
			}
			g.TapPermanent(g.FindPermanent(p.ID()))
			if !g.UntapPermanent(g.FindPermanent(p.ID())) {
				t.Fatal("other untap prevented")
			}
		})
	}
}

func TestRONLastingGrantInvalid(t *testing.T) {
	good := `CreatePermanentLayerEffect(Ref_TargetPermanent,[AddAbility([If(PermanentPassesFilter(ThisPermanent,HasACounterOfType(GlyphCounter)),[PermanentRuleEffect(ThisPermanent,[DoesntUntapDuringControllersUntap])])]),AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([RemoveCounters([ACounterOfTypeFromPermanent(GlyphCounter,ThisPermanent)])]))])])`
	for _, bad := range []string{
		strings.Replace(good, "ThisPermanent,HasACounter", "Ref_TargetPermanent,HasACounter", 1),
		strings.Replace(good, "GlyphCounter,ThisPermanent", "GlyphCounter,Ref_TargetPermanent", 1),
		strings.Replace(good, "DoesntUntapDuringControllersUntap", "DoesntUntapDuringControllersUntap()", 1),
		strings.Replace(good, "SinglePlayer(You)", "SinglePlayer(Opponent)", 1),
		strings.Replace(good, "[DoesntUntapDuringControllersUntap]", "[]", 1),
		strings.Replace(good, "HasACounterOfType(GlyphCounter)", "HasACounterOfType(UnknownCounter)", 1),
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([` + bad + `])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", bad)
		}
	}
}
