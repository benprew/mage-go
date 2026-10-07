package mage

import (
	"math/bits"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func requirementAttacker(g *Game, player Player, name string, filter PermanentFilter, options ...CardOption) *Permanent {
	condition := StateConditionFunc("live requirement", []StateDependency{StateBattlefield}, func(StateReader, StateContext) bool { return true })
	options = append(options, WithAbility(Static(While(condition, AllCreaturesMustBlock(This, filter)))))
	return g.PutOnBattlefield(NewCreature(name, "", 2, 8, options...), player.PlayerID())
}

func TestBlockingRequirementsBothPaths(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(map[bool]string{false: "search", true: "native"}[native], func(t *testing.T) {
			a := NewBasePlayer("A")
			b := &fixedBlockPlayer{BasePlayer: NewBasePlayer("B")}
			g := NewGame(a, b)
			atk := requirementAttacker(g, a, "Required", IsCreature)
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 8), a.PlayerID())
			one := g.PutOnBattlefield(NewCreature("One", "", 1, 8, WithAbility(BlocksTrigger(GainLifeFor(You, Fixed(1)), false))), b.PlayerID())
			two := g.PutOnBattlefield(NewCreature("Two", "", 1, 8), b.PlayerID())
			tapped := g.PutOnBattlefield(NewCreature("Tapped", "", 1, 8), b.PlayerID())
			tapped.Tapped = true
			own := g.PutOnBattlefield(NewCreature("Wrong controller", "", 1, 8), a.PlayerID())
			g.combat.AddAttackerIncarnation(atk.ID(), atk.incarnationID, b.PlayerID())
			g.combat.AddAttackerIncarnation(other.ID(), other.incarnationID, b.PlayerID())
			proposed := []BlockAssignment{{one.ID(), other.ID()}, {own.ID(), atk.ID()}, {two.ID(), atk.ID()}, {two.ID(), atk.ID()}}
			if native {
				b.assignments = proposed
				g.doDeclareBlockers()
				g.PutTriggersOnStack()
				g.ResolveStack()
			} else {
				g.ExecuteBlockers(proposed)
			}
			if len(g.combat.GroupFor(atk.ID()).BlockerIDs) != 2 || g.combat.IsBlocking(tapped.ID()) || g.combat.IsBlocking(own.ID()) || len(g.combat.GroupFor(other.ID()).BlockerIDs) != 0 {
				t.Fatal("requirements or legality")
			}
			if b.Life() != 21 {
				t.Fatalf("final pair trigger life=%d", b.Life())
			}
		})
	}
}

func TestBlockingRequirementsMaximumAndMinimum(t *testing.T) {
	for _, scenario := range []string{"competing", "weighted", "additional", "any", "minimum", "impossible", "evasion", "landwalk", "one-required"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first := requirementAttacker(g, a, "First", IsCreature)
			filter := IsCreature
			if scenario == "minimum" || scenario == "impossible" {
				filter = Not(AnyPermanent)
			}
			second := requirementAttacker(g, a, "Second", filter)
			opts := []CardOption{}
			if scenario == "additional" {
				opts = append(opts, WithKeyword(core.CanBlockAdditional))
			}
			if scenario == "any" {
				opts = append(opts, WithKeyword(core.CanBlockAny))
			}
			blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8, opts...), b.PlayerID())
			if scenario == "weighted" {
				g.effects.AddAllCreaturesMustBlock(first.ID(), IsCreature)
			}
			if scenario == "minimum" || scenario == "impossible" {
				if scenario == "minimum" {
					g.PutOnBattlefield(NewCreature("Support", "", 1, 8), b.PlayerID())
				}
			}
			if scenario == "minimum" || scenario == "impossible" {
				g.effects.AddMinBlockers(first.ID(), 2)
			}
			if scenario == "evasion" {
				first.GrantBaseAttr(core.Flying)
			}
			if scenario == "landwalk" {
				first.GrantBaseAttr(core.Forestwalk)
				g.PutOnBattlefield(NewLand("Forest", WithSubTypes("Forest")), b.PlayerID())
			}
			if scenario == "one-required" {
				first.RevokeBaseAttr(core.AttrMustBeBlocked)
				g.effects.allCreaturesMustBlockRules = nil
				first.GrantBaseAttr(core.AttrMustBeBlockedIfAble)
				second.GrantBaseAttr(core.AttrMustBeBlockedIfAble)
			}
			g.combat.AddAttackerIncarnation(first.ID(), first.incarnationID, b.PlayerID())
			g.combat.AddAttackerIncarnation(second.ID(), second.incarnationID, b.PlayerID())
			g.ExecuteBlockers([]BlockAssignment{{blocker.ID(), second.ID()}})
			n1, n2 := len(g.combat.GroupFor(first.ID()).BlockerIDs), len(g.combat.GroupFor(second.ID()).BlockerIDs)
			switch scenario {
			case "weighted":
				if n1 != 1 || n2 != 0 {
					t.Fatal("did not maximize independent requirements")
				}
			case "additional", "any":
				if n1 != 1 || n2 != 1 {
					t.Fatal("multi-block capacity")
				}
			case "minimum":
				if n1 != 2 || n2 != 0 {
					t.Fatalf("minimum: %d/%d", n1, n2)
				}
			default:
				if n1 != 0 || n2 != 1 {
					t.Fatalf("legal preferred maximum: %d/%d", n1, n2)
				}
			}
		})
	}
}

func TestBlockingRequirementDefinitionAndIdentity(t *testing.T) {
	invalid := NewCreature("Invalid", "", 1, 1, WithAbility(Static(While(StateConditionFunc("true", nil, func(StateReader, StateContext) bool { return true }), AllCreaturesMustBlock(nil, IsCreature)))))
	if ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("nil subject accepted")
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := requirementAttacker(g, a, "First", HasSubType("Wall"))
	q := requirementAttacker(g, a, "Second", IsCreature)
	wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 8, WithSubTypes("Wall")), b.PlayerID())
	plain := g.PutOnBattlefield(NewCreature("Plain", "", 1, 8), b.PlayerID())
	g.combat.AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
	g.combat.AddAttackerIncarnation(q.ID(), q.incarnationID, b.PlayerID())
	clone := g.Clone()
	g.effects.AddAllCreaturesMustBlock(p.ID(), IsCreature)
	clone.ExecuteBlockers([]BlockAssignment{{wall.ID(), p.ID()}, {plain.ID(), q.ID()}})
	if len(clone.combat.GroupFor(p.ID()).BlockerIDs) != 1 || len(clone.combat.GroupFor(q.ID()).BlockerIDs) != 1 || g.combat.IsBlocking(wall.ID()) {
		t.Fatal("clone or filter isolation")
	}
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(p.Card, a.PlayerID())
	if returned.incarnationID == p.incarnationID {
		t.Fatal("returned incarnation")
	}
	g.ExecuteBlockers(nil)
	if group := g.combat.GroupFor(p.ID()); group != nil && len(group.BlockerIDs) != 0 {
		t.Fatal("returned attacker restored old combat")
	}
}

func TestBlockingRequirementsBand(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := requirementAttacker(g, a, "First", IsCreature, WithKeyword(core.Banding))
	q := requirementAttacker(g, a, "Second", IsCreature)
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
	g.combat.AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
	g.combat.AddAttackerIncarnation(q.ID(), q.incarnationID, b.PlayerID())
	g.combat.Bands[p.ID()] = []uuid.UUID{p.ID(), q.ID()}
	g.combat.Bands[q.ID()] = []uuid.UUID{p.ID(), q.ID()}
	g.ExecuteBlockers(nil)
	if len(g.combat.GroupFor(p.ID()).BlockerIDs) != 1 || len(g.combat.GroupFor(q.ID()).BlockerIDs) != 1 || !g.combat.IsBlocking(blocker.ID()) {
		t.Fatal("one band block must satisfy both requirements")
	}
}

func TestBlockingRequirementsAgainstExhaustiveAssignments(t *testing.T) {
	for seed := range 96 {
		plan := &blockingPlan{groups: make([]*CombatGroup, 3), counts: make([]int, 3), directCounts: make([]int, 3), minimum: []int{1, 1 + seed%2, 1 + seed%3}, single: []int{seed % 2, seed % 3, 1}}
		for rowIndex := range 4 {
			row := blockingRow{capacity: 1 + (seed+rowIndex)%2}
			for group := range 3 {
				if (seed+rowIndex*3+group*5)%7 == 0 {
					continue
				}
				row.choices = append(row.choices, blockingChoice{assignment: BlockAssignment{uuid.New(), uuid.New()}, groups: []int{group}, direct: group, weight: (seed + rowIndex + group) % 3})
			}
			if len(row.choices) > 0 {
				row.preferred = []int{(seed + rowIndex) % len(row.choices)}
			}
			plan.rows = append(plan.rows, row)
		}
		plan.suffixWeight = make([]int, 5)
		for i := 3; i >= 0; i-- {
			weights := []int{}
			for _, choice := range plan.rows[i].choices {
				weights = append(weights, choice.weight)
			}
			slices.SortFunc(weights, func(a, b int) int { return b - a })
			for _, weight := range weights[:min(len(weights), plan.rows[i].capacity)] {
				plan.suffixWeight[i] += weight
			}
			plan.suffixWeight[i] += plan.suffixWeight[i+1]
		}
		oracle := -1
		counts := make([]int, 3)
		var enumerate func(int, int)
		enumerate = func(rowIndex, weight int) {
			if rowIndex == 4 {
				for group, count := range counts {
					if count > 0 && count < plan.minimum[group] {
						return
					}
					if count > 0 && plan.single[group] > 0 {
						weight += plan.single[group]
					}
				}
				oracle = max(oracle, weight)
				return
			}
			row := plan.rows[rowIndex]
			for mask := 0; mask < 1<<len(row.choices); mask++ {
				if bits.OnesCount(uint(mask)) > row.capacity {
					continue
				}
				gained := 0
				for i, choice := range row.choices {
					if mask&(1<<i) != 0 {
						counts[choice.groups[0]]++
						gained += choice.weight
					}
				}
				enumerate(rowIndex+1, weight+gained)
				for i, choice := range row.choices {
					if mask&(1<<i) != 0 {
						counts[choice.groups[0]]--
					}
				}
			}
		}
		enumerate(0, 0)
		chosen := plan.choose()
		score := 0
		for _, assignment := range chosen {
			for _, row := range plan.rows {
				for _, choice := range row.choices {
					if assignment == choice.assignment {
						counts[choice.groups[0]]++
						score += choice.weight
					}
				}
			}
		}
		for group, count := range counts {
			if count > 0 && count < plan.minimum[group] {
				t.Fatalf("seed %d: illegal minimum", seed)
			}
			if count > 0 && plan.single[group] > 0 {
				score += plan.single[group]
			}
		}
		if score != oracle {
			t.Fatalf("seed %d: got %d, exhaustive maximum %d", seed, score, oracle)
		}
		for _, count := range plan.counts {
			if count != 0 {
				t.Fatal("query changed counts")
			}
		}
	}
}

func TestBlockingRequirementsLargeMultiBlockAndSingleCoverage(t *testing.T) {
	for _, single := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		for i := range 32 {
			var p *Permanent
			if single {
				p = g.PutOnBattlefield(NewCreature("Required", "", 1, 8, WithKeyword(core.AttrMustBeBlockedIfAble)), a.PlayerID())
			} else {
				p = requirementAttacker(g, a, "Required", IsCreature)
			}
			g.combat.AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
			if single || i == 0 {
				options := []CardOption{}
				if !single {
					options = append(options, WithKeyword(core.CanBlockAny))
				}
				g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8, options...), b.PlayerID())
			}
		}
		result := g.CompleteBlockAssignments(b.PlayerID(), nil)
		if len(result) != 32 {
			t.Fatalf("single=%v: assignments=%d", single, len(result))
		}
		if g.combat.IsBlocking(result[0].BlockerID) {
			t.Fatal("pure query changed combat")
		}
	}
}

func TestBlockingRequirementsBandRestrictions(t *testing.T) {
	for _, evasion := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		p := requirementAttacker(g, a, "First", IsCreature, WithKeyword(core.Banding), WithKeyword(core.Menace))
		q := requirementAttacker(g, a, "Second", IsCreature)
		if evasion {
			q.GrantBaseAttr(core.Flying)
		}
		blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
		g.combat.AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
		g.combat.AddAttackerIncarnation(q.ID(), q.incarnationID, b.PlayerID())
		g.combat.Bands[p.ID()] = []uuid.UUID{p.ID(), q.ID()}
		g.combat.Bands[q.ID()] = []uuid.UUID{p.ID(), q.ID()}
		result := g.CompleteBlockAssignments(b.PlayerID(), []BlockAssignment{{blocker.ID(), p.ID()}})
		if evasion {
			if len(result) != 0 {
				t.Fatal("no band member can be legally blocked")
			}
		} else if len(result) != 1 || result[0].AttackerID != q.ID() {
			t.Fatal("legal band member must permit an indirect block")
		}
	}
}

func TestBlockingRequirementsRepeatedAttributes(t *testing.T) {
	for _, attr := range []core.Attr{core.AttrMustBeBlocked, core.AttrMustBeBlockedIfAble} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		first := g.PutOnBattlefield(NewCreature("Two requirements", "", 1, 8, WithKeyword(attr), WithKeyword(attr)), a.PlayerID())
		second := g.PutOnBattlefield(NewCreature("One requirement", "", 1, 8, WithKeyword(attr)), a.PlayerID())
		blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
		g.combat.AddAttackerIncarnation(first.ID(), first.incarnationID, b.PlayerID())
		g.combat.AddAttackerIncarnation(second.ID(), second.incarnationID, b.PlayerID())
		result := g.CompleteBlockAssignments(b.PlayerID(), []BlockAssignment{{blocker.ID(), second.ID()}})
		if len(result) != 1 || result[0].AttackerID != first.ID() {
			t.Fatal("multiple requirements collapsed into one")
		}
	}
}

func TestBlockingRequirementCapturedDuration(t *testing.T) {
	for _, change := range []string{"live", "returned", "phased", "phase-cycle", "abilities", "cleanup", "clone"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Recipient", "", 2, 8)
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
			if err := ApplyEffect(g, Until(core.EndOfTurn, AllCreaturesMustBlock(This, IsCreature)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if change == "clone" {
				original := g
				g = g.Clone()
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				p.RevokeBaseAttr(core.AttrSummonSick)
				original.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
				if len(original.CompleteBlockAssignments(b.PlayerID(), nil)) != 1 {
					t.Fatal("clone expiry changed original")
				}
			}
			switch change {
			case "returned":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				p = g.PutOnBattlefield(card, a.PlayerID())
			case "phased":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "abilities":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.Indefinite))
			case "cleanup":
				g.RunStepWithPriority(core.Cleanup)
			}
			current := g.FindPermanent(p.ID())
			if current != nil {
				current.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
			}
			got := g.CompleteBlockAssignments(b.PlayerID(), nil)
			want := change == "live" || change == "phase-cycle" || change == "abilities"
			if (len(got) == 1) != want {
				t.Fatalf("change=%s required=%v, assignments=%v", change, want, got)
			}
		})
	}
	binding := DeclarePermanentTarget("undeclared", IsCreature)
	invalid := NewInstant("Invalid", "", WithAbility(Spell(Until(core.EndOfTurn, AllCreaturesMustBlock(binding, IsCreature)))))
	if ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("undeclared subject accepted")
	}
}

func TestBlockingRequirementsCreatureTypeLoss(t *testing.T) {
	for _, role := range []string{"attacker", "blocker"} {
		t.Run(role, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := requirementAttacker(g, a, "Required", IsCreature)
			blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
			g.combat.AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
			if role == "attacker" {
				p.RevokeBaseAttr(core.AttrIsCreature)
			} else {
				blocker.RevokeBaseAttr(core.AttrIsCreature)
			}
			result := g.CompleteBlockAssignments(b.PlayerID(), []BlockAssignment{{blocker.ID(), p.ID()}})
			if len(result) != 0 {
				t.Fatal("former creature used in declaration")
			}
		})
	}
}
