package mage

import (
	"math"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type bandOutgoingPlayer struct {
	*BasePlayer
	assign  func(*Permanent, []*Permanent, int) map[uuid.UUID]int
	calls   []uuid.UUID
	amounts []int
}

func (p *bandOutgoingPlayer) GetBlockerOrder(_ *Game, _ *Permanent, _ []*Permanent, _ int) []uuid.UUID {
	return nil
}
func (p *bandOutgoingPlayer) GetCombatDamageAssignment(_ *Game, source *Permanent, recipients []*Permanent, amount int) map[uuid.UUID]int {
	p.calls = append(p.calls, source.ID())
	p.amounts = append(p.amounts, amount)
	if p.assign != nil {
		return p.assign(source, recipients, amount)
	}
	return map[uuid.UUID]int{recipients[0].ID(): amount}
}

func bandOutgoingGame(a, b Player, keywords ...core.Attr) (*Game, *Permanent, *Permanent, *Permanent, *Permanent) {
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("Band source", "", 2, 20, WithKeyword(core.Banding)), a.PlayerID())
	options := make([]CardOption, 0, 1+len(keywords))
	options = append(options, WithKeyword(core.Lifelink))
	for _, keyword := range keywords {
		options = append(options, WithKeyword(keyword))
	}
	second := g.PutOnBattlefield(NewCreature("Other source", "", 3, 20, options...), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("First recipient", "", 0, 20), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Second recipient", "", 0, 20), b.PlayerID())
	first.RevokeBaseAttr(core.AttrSummonSick)
	second.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
	g.GetCombat().AddBand([]uuid.UUID{first.ID(), second.ID()})
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: first.ID()}, {BlockerID: other.ID(), AttackerID: first.ID()}})
	return g, first, second, blocker, other
}

func TestBandOutgoingDamageSources(t *testing.T) {
	a := &bandOutgoingPlayer{BasePlayer: NewBasePlayer("A")}
	b := &bandSourcePlayer{BasePlayer: NewBasePlayer("B"), numbers: []int{1}}
	g, first, second, blocker, _ := bandOutgoingGame(a, b)
	firstSource, secondSource := g.CaptureDamageSource(first.ID()), g.CaptureDamageSource(second.ID())
	g.AddPreventionShield(blocker.ID(), 3)
	observed := 0
	g.damage.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
		observed++
		if blocker.Damage != 2 || a.Life() != 21 {
			t.Error("notification saw partial damage or incorrect lifelink")
		}
	})
	g.ExecuteCombatDamage()
	if blocker.Damage != 2 || a.Life() != 21 || b.numberCalls != 1 || observed != 2 {
		t.Fatalf("damage=%d life=%d choices=%d notifications=%d", blocker.Damage, a.Life(), b.numberCalls, observed)
	}
	if len(a.calls) != 2 || a.calls[0] != first.ID() || a.calls[1] != second.ID() || a.amounts[0] != 2 || a.amounts[1] != 3 {
		t.Fatal("assignment combined attacking sources")
	}
	if !g.HasDealtDamageToPermanentFromSource(firstSource, blocker) || !g.HasDealtDamageToPermanentFromSource(secondSource, blocker) {
		t.Fatal("source histories were combined")
	}
}

func TestBandOutgoingDamageTrampleIsPerSource(t *testing.T) {
	for _, noBlockers := range []bool{false, true} {
		a := NewBasePlayer("A")
		b := NewBasePlayer("B")
		g, first, second, blocker, other := bandOutgoingGame(a, b, core.Trample)
		if noBlockers {
			g.RemoveFromCombat(blocker.ID())
			g.RemoveFromCombat(other.ID())
		} else {
			blocker.Card.SetBasePT(0, 2)
			other.Card.SetBasePT(0, 2)
		}
		g.ExecuteCombatDamage()
		want := 19
		if noBlockers {
			want = 17
		}
		if b.Life() != want || a.Life() != 23 {
			t.Fatalf("no blockers=%v: defender=%d lifelink=%d", noBlockers, b.Life(), a.Life())
		}
		if g.HasDealtDamageToPlayerFromSource(g.CaptureDamageSource(first.ID()), b.PlayerID()) {
			t.Fatal("nontrampling member dealt player damage")
		}
		if !g.HasDealtDamageToPlayerFromSource(g.CaptureDamageSource(second.ID()), b.PlayerID()) {
			t.Fatal("trampling source attribution lost")
		}
	}
}

func TestBandOutgoingDamageSharedLethal(t *testing.T) {
	for _, deathtouch := range []bool{false, true} {
		a := &bandOutgoingPlayer{BasePlayer: NewBasePlayer("A")}
		b := NewBasePlayer("B")
		g, first, second, blocker, other := bandOutgoingGame(a, b, core.Trample)
		g.RemoveFromCombat(other.ID())
		blocker.Card.SetBasePT(0, 2)
		if deathtouch {
			blocker.Card.SetBasePT(0, 20)
			first.GrantBaseAttr(core.Deathtouch)
		}
		a.assign = func(source *Permanent, _ []*Permanent, amount int) map[uuid.UUID]int {
			if source.ID() == second.ID() {
				return map[uuid.UUID]int{}
			}
			return map[uuid.UUID]int{blocker.ID(): amount}
		}
		g.ExecuteCombatDamage()
		if b.Life() != 17 || a.Life() != 23 {
			t.Fatalf("shared lethal: defender=%d life=%d", b.Life(), a.Life())
		}
	}
}

func TestBandOutgoingDamageInvalidAssignments(t *testing.T) {
	for _, invalid := range []string{"nil", "empty", "under", "over", "negative", "foreign", "overflow", "illegal-trample"} {
		t.Run(invalid, func(t *testing.T) {
			a := &bandOutgoingPlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g, _, _, blocker, other := bandOutgoingGame(a, b, core.Trample)
			a.assign = func(_ *Permanent, recipients []*Permanent, amount int) map[uuid.UUID]int {
				switch invalid {
				case "nil":
					return nil
				case "empty", "illegal-trample":
					return map[uuid.UUID]int{}
				case "under":
					return map[uuid.UUID]int{recipients[0].ID(): amount - 1}
				case "over":
					return map[uuid.UUID]int{recipients[0].ID(): amount + 1}
				case "negative":
					return map[uuid.UUID]int{recipients[0].ID(): -1, recipients[1].ID(): amount + 1}
				case "foreign":
					return map[uuid.UUID]int{uuid.New(): amount}
				default:
					return map[uuid.UUID]int{recipients[0].ID(): math.MaxInt, recipients[1].ID(): math.MaxInt}
				}
			}
			g.ExecuteCombatDamage()
			if blocker.Damage+other.Damage != 5 || b.Life() != 20 || a.Life() != 23 {
				t.Fatalf("invalid assignment changed total: %d/%d life=%d/%d", blocker.Damage, other.Damage, a.Life(), b.Life())
			}
		})
	}
}

func TestBandOutgoingDamageBlockingBandChooser(t *testing.T) {
	a := &bandOutgoingPlayer{BasePlayer: NewBasePlayer("A")}
	b := &bandSourcePlayer{BasePlayer: NewBasePlayer("B")}
	g, first, second, blocker, other := bandOutgoingGame(a, b)
	blocker.GrantBaseAttr(core.Banding)
	b.assign = func(source *Permanent, _ []*Permanent, amount int) map[uuid.UUID]int {
		if source.ID() == first.ID() {
			return map[uuid.UUID]int{other.ID(): amount}
		}
		return map[uuid.UUID]int{blocker.ID(): amount}
	}
	g.ExecuteCombatDamage()
	if len(a.calls) != 0 || len(b.calls) != 2 || b.calls[0] != first.ID() || b.calls[1] != second.ID() || blocker.Damage != 3 || other.Damage != 2 {
		t.Fatal("defending band did not assign each attacker independently")
	}
}

func TestBandOutgoingDamageSteps(t *testing.T) {
	for _, keyword := range []core.Attr{core.FirstStrike, core.DoubleStrike} {
		a := &bandOutgoingPlayer{BasePlayer: NewBasePlayer("A")}
		b := NewBasePlayer("B")
		g, first, second, blocker, _ := bandOutgoingGame(a, b, keyword)
		g.GetCombat().ResolveDamage(g, true)
		if blocker.Damage != 3 || a.Life() != 23 || len(a.calls) != 1 || a.calls[0] != second.ID() {
			t.Fatal("first strike sources mixed")
		}
		g.GetCombat().ResolveDamage(g, false)
		want := 5
		if keyword == core.DoubleStrike {
			want = 8
		}
		if blocker.Damage != want || a.calls[1] != first.ID() {
			t.Fatal("normal damage used the wrong step sources")
		}
	}
}

func TestBandOutgoingDamageIdentityAndClone(t *testing.T) {
	for _, scenario := range []string{"return", "phase", "remove", "control", "nonpositive"} {
		t.Run(scenario, func(t *testing.T) {
			a := &bandOutgoingPlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g, first, second, blocker, _ := bandOutgoingGame(a, b)
			clone := g.Clone()
			switch scenario {
			case "return":
				g.BouncePermanentToHand(second)
				a.RemoveFromHand(second.Card.ID())
				g.PutOnBattlefield(second.Card, a.PlayerID())
			case "phase":
				g.PhaseOut(second.ID())
				g.PhaseIn([]uuid.UUID{second.ID()})
			case "remove":
				g.RemoveFromCombat(second.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: second.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "nonpositive":
				if err := ApplyEffect(g, Until(core.EndOfTurn, ModifyPT(This, Fixed(-4), Fixed(0))), second.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ExecuteCombatDamage()
			if g.FindPermanent(blocker.ID()).Damage != 2 || a.Life() != 20 || len(a.calls) != 1 || a.calls[0] != first.ID() {
				t.Fatal("absent or nonpositive source changed another member's damage")
			}
			if clone.FindPermanent(blocker.ID()).Damage != 0 {
				t.Fatal("combat mutated clone")
			}
			clone.ExecuteCombatDamage()
			if clone.FindPermanent(blocker.ID()).Damage != 5 || g.FindPermanent(blocker.ID()).Damage != 2 {
				t.Fatal("clone lost independent source assignments")
			}
		})
	}
}

func TestBandOutgoingDamageRetainedSource(t *testing.T) {
	a := &bandOutgoingPlayer{BasePlayer: NewBasePlayer("A")}
	b := &bandSourcePlayer{BasePlayer: NewBasePlayer("B"), numbers: []int{1}}
	g, _, second, blocker, _ := bandOutgoingGame(a, b)
	saved := g.CaptureDamageSource(second.ID())
	g.AddPreventionShield(blocker.ID(), 3)
	b.onNumber = func() {
		g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
		returned := g.PutOnBattlefield(second.Card, b.PlayerID())
		returned.RevokeBaseAttr(core.Lifelink)
	}
	g.ExecuteCombatDamage()
	if a.Life() != 21 || b.Life() != 20 || !g.HasDealtDamageToPermanentFromSource(saved, blocker) || g.HasDealtDamageToPermanentFromSource(g.CaptureDamageSource(second.ID()), blocker) {
		t.Fatal("replacement changed saved source controller or incarnation")
	}
}

func TestBandOutgoingDamageIndependentAssignments(t *testing.T) {
	a := &bandOutgoingPlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g, first, second, blocker, other := bandOutgoingGame(a, b)
	shared := map[uuid.UUID]int{}
	a.assign = func(source *Permanent, _ []*Permanent, amount int) map[uuid.UUID]int {
		clear(shared)
		id := blocker.ID()
		if source.ID() == second.ID() {
			id = other.ID()
		}
		shared[id] = amount
		return shared
	}
	g.ExecuteCombatDamage()
	if blocker.Damage != 2 || other.Damage != 3 || a.Life() != 23 || shared[other.ID()] != 3 || len(shared) != 1 {
		t.Fatal("assignments share mutable state or source abilities")
	}
	if !g.HasDealtDamageToPermanentFromSource(g.CaptureDamageSource(first.ID()), blocker) || !g.HasDealtDamageToPermanentFromSource(g.CaptureDamageSource(second.ID()), other) {
		t.Fatal("independent histories lost")
	}
}

func TestBandOutgoingDamageReplacementOrder(t *testing.T) {
	for _, redirectFirst := range []bool{false, true} {
		a := &bandOutgoingPlayer{BasePlayer: NewBasePlayer("A")}
		b := &bandSourcePlayer{BasePlayer: NewBasePlayer("B"), numbers: []int{1}}
		g, _, _, blocker, _ := bandOutgoingGame(a, b)
		g.AddPreventionShield(blocker.ID(), 3)
		g.SetCreatureDamageRedirect(blocker.ID(), a.PlayerID())
		if redirectFirst {
			b.mode = 1
		}
		g.ExecuteCombatDamage()
		wantDamage, wantLife, wantChoices := 1, 20, 1
		if redirectFirst {
			wantDamage, wantLife, wantChoices = 0, 18, 0
		}
		if blocker.Damage != wantDamage || a.Life() != wantLife || b.numberCalls != wantChoices {
			t.Fatalf("redirect first=%v: life=%d choices=%d", redirectFirst, a.Life(), b.numberCalls)
		}
	}
}

func TestBandOutgoingDamageUnpreventable(t *testing.T) {
	a := &bandOutgoingPlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g, _, _, blocker, _ := bandOutgoingGame(a, b)
	blocker.GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
	g.AddPreventionShield(blocker.ID(), 3)
	g.ExecuteCombatDamage()
	if blocker.Damage != 5 || a.Life() != 23 {
		t.Fatal("unpreventable packets lost source or damage")
	}
	blocker.RevokeBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
	g.DealDamageToPermanent(blocker, 4, uuid.New())
	if blocker.Damage != 6 {
		t.Fatal("unpreventable packets spent shield capacity")
	}
}

func TestBandOutgoingDamageLegacyDefendingChoices(t *testing.T) {
	a := &bandOutgoingPlayer{BasePlayer: NewBasePlayer("A")}
	b := &bandLegacyPlayer{BasePlayer: NewBasePlayer("B")}
	g, _, _, blocker, other := bandOutgoingGame(a, b)
	blocker.GrantBaseAttr(core.Banding)
	b.distribution = map[uuid.UUID]int{other.ID(): 2}
	g.ExecuteCombatDamage()
	if other.Damage != 2 || blocker.Damage != 3 || b.calls != 2 || len(a.calls) != 0 || b.distribution[other.ID()] != 2 {
		t.Fatal("legacy choice lost valid source amount or changed caller map")
	}
}

func TestBandOutgoingDamageKeywordsStayWithSource(t *testing.T) {
	a := &bandOutgoingPlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g, first, second, blocker, other := bandOutgoingGame(a, b)
	first.GrantBaseAttr(core.Deathtouch)
	a.assign = func(source *Permanent, _ []*Permanent, amount int) map[uuid.UUID]int {
		id := other.ID()
		if source.ID() == second.ID() {
			id = blocker.ID()
		}
		return map[uuid.UUID]int{id: amount}
	}
	g.ExecuteCombatDamage()
	if g.FindPermanent(other.ID()) != nil || g.FindPermanent(blocker.ID()) == nil || blocker.Damage != 3 || a.Life() != 23 {
		t.Fatal("band members shared deathtouch or lifelink")
	}
}
