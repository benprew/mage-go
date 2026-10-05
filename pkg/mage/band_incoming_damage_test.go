package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type bandSourcePlayer struct {
	*BasePlayer
	assign      func(*Permanent, []*Permanent, int) map[uuid.UUID]int
	calls       []uuid.UUID
	numbers     []int
	numberCalls int
	onNumber    func()
	mode        int
}

func (p *bandSourcePlayer) ChooseMode(_ []string, _ string) int { return p.mode }

func (p *bandSourcePlayer) GetBandingDamageAssignment(_ *Game, source *Permanent, recipients []*Permanent, amount int) map[uuid.UUID]int {
	p.calls = append(p.calls, source.ID())
	if p.assign != nil {
		return p.assign(source, recipients, amount)
	}
	return map[uuid.UUID]int{recipients[0].ID(): amount}
}

func (p *bandSourcePlayer) ChooseNumber(minimum, maximum int, _ string) int {
	if p.onNumber != nil {
		p.onNumber()
	}
	i := p.numberCalls
	p.numberCalls++
	if i < len(p.numbers) {
		return p.numbers[i]
	}
	return maximum
}

type bandLegacyPlayer struct {
	*BasePlayer
	distribution map[uuid.UUID]int
	calls        int
}

func (p *bandLegacyPlayer) GetBandingDamageDistribution(_ []*Permanent) map[uuid.UUID]int {
	p.calls++
	return p.distribution
}

func bandIncomingGame(t *testing.T, a, b Player, keywords ...core.Attr) (*Game, *Permanent, *Permanent, *Permanent, *Permanent) {
	t.Helper()
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("Band", "", 0, 20, WithKeyword(core.Banding)), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Member", "", 0, 20), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("First blocker", "", 2, 20), b.PlayerID())
	options := make([]CardOption, len(keywords))
	for i, keyword := range keywords {
		options[i] = WithKeyword(keyword)
	}
	other := g.PutOnBattlefield(NewCreature("Second blocker", "", 3, 20, options...), b.PlayerID())
	first.RevokeBaseAttr(core.AttrSummonSick)
	second.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
	g.GetCombat().AddBand([]uuid.UUID{first.ID(), second.ID()})
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: first.ID()}, {BlockerID: other.ID(), AttackerID: first.ID()}})
	return g, first, second, blocker, other
}

func TestBandIncomingDamageSources(t *testing.T) {
	a := &bandSourcePlayer{BasePlayer: NewBasePlayer("A"), numbers: []int{1}}
	b := NewBasePlayer("B")
	g, first, _, blocker, other := bandIncomingGame(t, a, b, core.Lifelink)
	g.AddPreventionShield(first.ID(), 3)
	observed := 0
	g.damage.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
		observed++
		if first.Damage != 2 || b.Life() != 21 {
			t.Error("notification saw a partial batch or the wrong source")
		}
	})
	g.ExecuteCombatDamage()
	if first.Damage != 2 || b.Life() != 21 || a.numberCalls != 1 || observed != 2 {
		t.Fatalf("damage=%d life=%d allocation choices=%d notifications=%d", first.Damage, b.Life(), a.numberCalls, observed)
	}
	if len(a.calls) != 2 || a.calls[0] != blocker.ID() || a.calls[1] != other.ID() {
		t.Fatalf("source assignment calls=%v", a.calls)
	}
}

func TestBandIncomingDamageIndependentAssignments(t *testing.T) {
	a := &bandSourcePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g, first, second, blocker, other := bandIncomingGame(t, a, b, core.Lifelink)
	a.assign = func(source *Permanent, recipients []*Permanent, amount int) map[uuid.UUID]int {
		i := 0
		if source.ID() == other.ID() {
			i = 1
		}
		return map[uuid.UUID]int{recipients[i].ID(): amount}
	}
	g.AddPreventionShield(first.ID(), 2)
	g.ExecuteCombatDamage()
	if first.Damage != 0 || second.Damage != 3 || b.Life() != 23 {
		t.Fatalf("source-specific assignments: damage=%d/%d life=%d", first.Damage, second.Damage, b.Life())
	}
	if len(a.calls) != 2 || a.calls[0] != blocker.ID() || a.calls[1] != other.ID() {
		t.Fatal("a blocker was omitted or processed twice")
	}
}

func TestBandIncomingDamageInvalidAssignments(t *testing.T) {
	for _, invalid := range []string{"nil", "empty", "negative", "under", "over", "foreign", "overflow"} {
		t.Run(invalid, func(t *testing.T) {
			a := &bandSourcePlayer{BasePlayer: NewBasePlayer("A"), numbers: []int{-1, 99}}
			b := NewBasePlayer("B")
			g, first, second, _, _ := bandIncomingGame(t, a, b, core.Lifelink)
			a.assign = func(_ *Permanent, recipients []*Permanent, amount int) map[uuid.UUID]int {
				switch invalid {
				case "nil":
					return nil
				case "empty":
					return map[uuid.UUID]int{}
				case "negative":
					return map[uuid.UUID]int{recipients[0].ID(): -1, recipients[1].ID(): amount + 1}
				case "under":
					return map[uuid.UUID]int{recipients[0].ID(): amount - 1}
				case "over":
					return map[uuid.UUID]int{recipients[0].ID(): amount + 1}
				case "foreign":
					return map[uuid.UUID]int{uuid.New(): amount}
				default:
					return map[uuid.UUID]int{recipients[0].ID(): int(^uint(0) >> 1), recipients[1].ID(): int(^uint(0) >> 1)}
				}
			}
			g.ExecuteCombatDamage()
			if first.Damage != 5 || second.Damage != 0 || b.Life() != 23 || a.numberCalls != 2 {
				t.Fatalf("fallback: damage=%d/%d life=%d choices=%d", first.Damage, second.Damage, b.Life(), a.numberCalls)
			}
		})
	}
}

func TestBandIncomingDamageLegacyTotals(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		a := &bandLegacyPlayer{BasePlayer: NewBasePlayer("A")}
		b := NewBasePlayer("B")
		g, first, second, _, _ := bandIncomingGame(t, a, b, core.Lifelink)
		a.distribution = map[uuid.UUID]int{first.ID(): 1, second.ID(): 4}
		if invalid {
			a.distribution[second.ID()] = 400
		}
		g.ExecuteCombatDamage()
		wantFirst, wantSecond := 1, 4
		if invalid {
			wantFirst, wantSecond = 5, 0
		}
		if first.Damage != wantFirst || second.Damage != wantSecond || b.Life() != 23 || a.calls != 1 || a.distribution[first.ID()] != 1 {
			t.Fatal("legacy totals, validation, source abilities or map isolation")
		}
	}
}

func TestBandIncomingDamageSteps(t *testing.T) {
	for _, keyword := range []core.Attr{core.FirstStrike, core.DoubleStrike} {
		a := &bandSourcePlayer{BasePlayer: NewBasePlayer("A")}
		b := NewBasePlayer("B")
		g, first, _, blocker, other := bandIncomingGame(t, a, b, core.Lifelink, keyword)
		g.GetCombat().ResolveDamage(g, true)
		if first.Damage != 3 || b.Life() != 23 || len(a.calls) != 1 || a.calls[0] != other.ID() {
			t.Fatal("first-strike source or amount")
		}
		g.GetCombat().ResolveDamage(g, false)
		wantDamage, wantLife := 5, 23
		if keyword == core.DoubleStrike {
			wantDamage, wantLife = 8, 26
		}
		if first.Damage != wantDamage || b.Life() != wantLife || a.calls[1] != blocker.ID() {
			t.Fatal("normal-step source or amount")
		}
	}
}

func TestBandIncomingDamageIdentityAndClone(t *testing.T) {
	for _, state := range []string{"source-leaves", "source-returns", "source-phase", "source-phase-cycle", "member-leaves", "member-returns", "member-phase-cycle", "banding-removed"} {
		t.Run(state, func(t *testing.T) {
			a := &bandSourcePlayer{BasePlayer: NewBasePlayer("A")}
			b := &bandSourcePlayer{BasePlayer: NewBasePlayer("B")}
			g, first, second, _, other := bandIncomingGame(t, a, b, core.Lifelink)
			clone := g.Clone()
			switch state {
			case "source-leaves", "source-returns":
				g.RemoveFromBattlefield(g.FindPermanent(other.ID()))
				if state == "source-returns" {
					g.PutOnBattlefield(other.Card, b.PlayerID())
				}
			case "source-phase":
				g.PhaseOut(other.ID())
			case "source-phase-cycle":
				g.PhaseIn(g.PhaseOut(other.ID()))
			case "member-leaves", "member-returns":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
				if state == "member-returns" {
					g.PutOnBattlefield(first.Card, a.PlayerID())
				}
			case "member-phase-cycle":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "banding-removed":
				g.MutablePermanent(first.ID()).RevokeBaseAttr(core.Banding)
			}
			g.ExecuteCombatDamage()
			wantLife := 23
			if state == "source-leaves" || state == "source-returns" || state == "source-phase" || state == "source-phase-cycle" {
				wantLife = 20
			}
			if b.Life() != wantLife {
				t.Fatalf("life=%d want=%d", b.Life(), wantLife)
			}
			if state == "member-returns" || state == "member-phase-cycle" {
				if g.FindPermanent(first.ID()).Damage != 0 {
					t.Fatal("removed incarnation received damage")
				}
			}
			if state == "banding-removed" && (len(a.calls) != 0 || len(b.calls) != 2) {
				t.Fatal("lost banding did not restore the blocker controller's choice")
			}
			if clone.FindPermanent(first.ID()).Damage != 0 || clone.FindPermanent(second.ID()).Damage != 0 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("combat changed clone")
			}
			clone.ExecuteCombatDamage()
			if clone.GetPlayer(b.PlayerID()).Life() != 23 || clone.FindPermanent(first.ID()).Damage != 5 {
				t.Fatal("clone lost source packets")
			}
		})
	}
}

func TestBandIncomingDamageRetainedSource(t *testing.T) {
	a := &bandSourcePlayer{BasePlayer: NewBasePlayer("A"), numbers: []int{1}}
	b := NewBasePlayer("B")
	g, first, _, _, other := bandIncomingGame(t, a, b, core.Lifelink)
	saved := g.CaptureDamageSource(other.ID())
	g.AddPreventionShield(first.ID(), 3)
	a.onNumber = func() {
		g.RemoveFromBattlefield(g.FindPermanent(other.ID()))
		returned := g.PutOnBattlefield(other.Card, a.PlayerID())
		returned.RevokeBaseAttr(core.Lifelink)
	}
	g.ExecuteCombatDamage()
	if b.Life() != 21 || a.Life() != 20 || !g.HasDealtDamageToPermanentFromSource(saved, g.FindPermanent(first.ID())) || g.HasDealtDamageToPermanentFromSource(g.CaptureDamageSource(other.ID()), g.FindPermanent(first.ID())) {
		t.Fatal("replacement choice changed the saved source identity or abilities")
	}
}

func TestBandIncomingDamageReplacementOrder(t *testing.T) {
	for _, redirectFirst := range []bool{false, true} {
		a := &bandSourcePlayer{BasePlayer: NewBasePlayer("A"), numbers: []int{1}}
		b := NewBasePlayer("B")
		g, first, _, _, _ := bandIncomingGame(t, a, b, core.Lifelink)
		g.AddPreventionShield(first.ID(), 3)
		g.SetCreatureDamageRedirect(first.ID(), b.PlayerID())
		if redirectFirst {
			a.mode = 1
		}
		g.ExecuteCombatDamage()
		wantDamage, wantLife, wantChoices := 1, 20, 1
		if redirectFirst {
			wantDamage, wantLife, wantChoices = 0, 18, 0
		}
		if first.Damage != wantDamage || b.Life() != wantLife || a.numberCalls != wantChoices {
			t.Fatalf("redirect-first=%v: damage=%d life=%d choices=%d", redirectFirst, first.Damage, b.Life(), a.numberCalls)
		}
	}
}

func TestBandIncomingDamageUnpreventable(t *testing.T) {
	a := &bandSourcePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g, first, _, _, _ := bandIncomingGame(t, a, b, core.Lifelink)
	first.GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
	g.AddPreventionShield(first.ID(), 3)
	g.ExecuteCombatDamage()
	if first.Damage != 5 || b.Life() != 23 || a.numberCalls != 0 {
		t.Fatal("unpreventable damage consumed shield capacity")
	}
	g.FindPermanent(first.ID()).RevokeBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
	g.DealDamageToPermanent(g.FindPermanent(first.ID()), 4, uuid.New())
	if first.Damage != 6 {
		t.Fatal("unused capacity was lost")
	}
}

func TestBandIncomingDamageNonpositiveSources(t *testing.T) {
	for _, reduction := range []int{2, 3} {
		a := &bandSourcePlayer{BasePlayer: NewBasePlayer("A")}
		b := NewBasePlayer("B")
		g, first, _, blocker, other := bandIncomingGame(t, a, b, core.Lifelink)
		if err := ApplyEffect(g, Until(core.EndOfTurn, ModifyPT(This, Fixed(-reduction), Fixed(0))), blocker.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.ExecuteCombatDamage()
		if first.Damage != 3 || b.Life() != 23 || len(a.calls) != 1 || a.calls[0] != other.ID() {
			t.Fatal("nonpositive source changed another blocker's damage")
		}
	}
}

func TestBandIncomingDamageBlockRelationships(t *testing.T) {
	a := NewBasePlayer("A")
	b := NewBasePlayer("B")
	g, first, second, blocker, other := bandIncomingGame(t, a, b)
	g.RemoveFromCombat(other.ID())
	g.GetCombat().AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, first.ID())
	g.GetCombat().SnapshotBlockedAlone()
	if g.GetCombat().BlockedAlone != blocker.ID() {
		t.Fatal("one blocker was counted once for each band member")
	}
	for _, member := range []*Permanent{first, second} {
		group := g.GetCombat().GroupFor(member.ID())
		if !group.Blocked || len(group.BlockerIDs) != 1 || group.BlockerIDs[0] != blocker.ID() || group.BlockerIncarnations[0] != blocker.incarnationID {
			t.Fatal("band relation lost identity or contains a duplicate")
		}
	}
}
