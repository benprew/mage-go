package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTapStateChangeRealCards(t *testing.T) {
	for _, name := range []string{"Riptide", "Reset", "Twiddle"} {
		for _, changes := range []bool{false, true} {
			t.Run(name+map[bool]string{false: "/unchanged", true: "/changes"}[changes], func(t *testing.T) {
				g, a, b := ronGame()
				g.SetStep(core.Upkeep)
				g.SetStep(core.Draw)
				subject := mage.NewLand("Land")
				if name == "Riptide" {
					subject = mage.NewCreature("Blue creature", "{U}", 2, 2)
				}
				p := g.PutOnBattlefield(subject, b.PlayerID())
				p.Tapped = (name == "Riptide") != changes
				p.AddCounter(core.Stun, 2)
				watched, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
				visits := watched.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
				set := "LEB"
				if name == "Riptide" {
					set = "DRK"
				}
				if name == "Reset" {
					set = "LEG"
				}
				var obj *mage.StackObject
				switch name {
				case "Reset":
					obj = castRONSpell(t, g, b, repositoryCard(t, set, name))
				case "Twiddle":
					obj = castRONSpell(t, g, a, repositoryCard(t, set, name), p.ID())
				default:
					obj = castRONSpell(t, g, a, repositoryCard(t, set, name))
				}
				instructions, ok := mage.InspectTapInstructions(mage.Sequence(obj.Effects...))
				if !ok || len(instructions) != 1 {
					t.Fatal("lowering lost instruction")
				}
				wantChange := changes || name == "Twiddle"
				if change, known := g.InspectTapStateChange(instructions[0].Operation, visits); change != wantChange || !known {
					t.Fatal("lowering lost operation")
				}
				clone := g.Clone()
				clone.ResolveStack()
				live := clone.FindPermanent(p.ID())
				if live == nil {
					t.Fatal("tap card destroyed a permanent")
				}
				switch name {
				case "Riptide":
					if !live.Tapped || live.Counters[core.Stun] != 2 {
						t.Fatal("Riptide gameplay differs")
					}
				case "Reset":
					if live.Tapped != p.Tapped || live.Counters[core.Stun] != 2-map[bool]int{false: 0, true: 1}[changes] {
						t.Fatal("Reset stun gameplay differs")
					}
				}
				obj.Effects = append(obj.Effects, (mage.DirectDestructionInstruction{Subjects: mage.MatchingPermanents(mage.AnyPermanent)}).Effect())
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, visits); certain != !wantChange || known != !wantChange {
						t.Fatalf("prefix proof=%v/%v change=%v", certain, known, wantChange)
					}
				}
				if p.Counters[core.Stun] != 2 {
					t.Fatal("query removed a counter")
				}
			})
		}
	}
}

func TestRONTapStateChangeSequence(t *testing.T) {
	for _, action := range []string{"TapPermanent", "UntapPermanent"} {
		for _, state := range []string{"unchanged", "changes", "phase", "cycle", "return", "shroud", "clone", "copy"} {
			t.Run(action+"/"+state, func(t *testing.T) {
				factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Land)),TargetPermanent(IsCardtype(Land))],ActionList([`+action+`(Ref_TargetPermanent1),DestroyPermanent(Ref_TargetPermanent2)])))`)
				g, a, b := ronGame()
				one := g.PutOnBattlefield(mage.NewLand("Prefix"), b.PlayerID())
				two := g.PutOnBattlefield(mage.NewLand("Watched"), b.PlayerID())
				one.Tapped = action == "TapPermanent"
				one.AddCounter(core.Stun, 2)
				query, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.IsLand))
				visits := query.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
				obj := castRONSpell(t, g, a, factory(), one.ID(), two.ID())
				switch state {
				case "changes":
					one.Tapped = !one.Tapped
				case "phase":
					g.PhaseOut(one.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(one.ID()))
				case "return":
					g.BouncePermanentToHand(one)
					g.PutOnBattlefield(one.Card, b.PlayerID())
				case "shroud":
					one.GrantBaseAttr(core.Shroud)
				case "copy":
					if g.CopyStackObjectDirect(obj, b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				for _, game := range []*mage.Game{g, g.Clone()} {
					current := game.StackPeek()
					if certain, known := game.InspectSpellDestruction(current, visits); certain != (state != "changes") || known != (state != "changes") {
						t.Fatalf("proof=%v/%v", certain, known)
					}
				}
				g.ResolveStack()
				if g.FindPermanent(two.ID()) != nil {
					t.Fatal("destruction failed")
				}
				if state != "changes" && state != "return" && g.FindPermanentIncludingPhased(one.ID()).Counters[core.Stun] != 2 {
					t.Fatal("no-change prefix removed stun")
				}
			})
		}
	}
}
