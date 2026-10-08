package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func otherBandsGrant(g *Game, player uuid.UUID, quality PermanentFilter) *Permanent {
	condition := StateConditionFunc("always", nil, func(StateReader, StateContext) bool { return true })
	return g.PutOnBattlefield(NewEnchantment("Grant", "", WithAbility(Static(While(condition, GrantEachBandsWithOther(ControlledPermanents(IsCreature), quality))))), player)
}

func TestBandsWithOtherFormation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := otherBandsGrant(g, a.PlayerID(), Named("Member"))
	ids := make([]uuid.UUID, 0, 3)
	for range 3 {
		p := g.PutOnBattlefield(NewCreature("Member", "", 1, 20), a.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		ids = append(ids, p.ID())
	}
	g.ExecuteAttackers(a.PlayerID(), ids)
	if !g.isValidBand(ids) {
		t.Fatal("matching creatures cannot form a band")
	}
	if g.isValidBand([]uuid.UUID{ids[0], ids[0]}) {
		t.Fatal("duplicate member")
	}
	group := g.combat.GroupFor(ids[1])
	group.DefenderID = uuid.New()
	if g.isValidBand(ids) {
		t.Fatal("different defenders")
	}
	group.DefenderID = b.PlayerID()
	if err := g.FormAttackingBand(ids); err != nil {
		t.Fatal(err)
	}
	if err := g.FormAttackingBand(ids[:2]); err == nil {
		t.Fatal("overlapping band")
	}
	clone := g.Clone()
	g.BouncePermanentToHand(source)
	if !g.combat.IsBandedWith(ids[0], ids[2]) {
		t.Fatal("ability loss erased formed band")
	}
	if !clone.hasBandingAssignment([]uuid.UUID{ids[0], ids[1]}) || g.hasBandingAssignment(ids) {
		t.Fatal("clone or live ability query")
	}
	phased := clone.PhaseOut(ids[0])
	clone.PhaseIn(phased)
	if clone.combat.IsInBand(ids[0]) {
		t.Fatal("phase cycle restored membership")
	}
}

func TestBandsWithOtherDamage(t *testing.T) {
	for _, path := range []string{"incoming", "outgoing", "multiple-attacks", "nonmatching", "lost", "first-strike", "trample"} {
		t.Run(path, func(t *testing.T) {
			a := &bandSourcePlayer{BasePlayer: NewBasePlayer("A")}
			b := &bandSourcePlayer{BasePlayer: NewBasePlayer("B")}
			g := NewGame(a, b)
			grantPlayer := a.PlayerID()
			if path == "outgoing" || path == "trample" {
				grantPlayer = b.PlayerID()
			}
			grant := otherBandsGrant(g, grantPlayer, Named("Member"))
			attackers, blockers := make([]*Permanent, 0, 2), make([]*Permanent, 0, 2)
			for i := range 2 {
				name := "Member"
				if path == "nonmatching" && i == 1 {
					name = "Other"
				}
				attackers = append(attackers, g.PutOnBattlefield(NewCreature(name, "", 2, 20), a.PlayerID()))
				blockers = append(blockers, g.PutOnBattlefield(NewCreature("Member", "", 3, 20), b.PlayerID()))
			}
			if path == "first-strike" {
				g.MutablePermanent(blockers[0].ID()).GrantBaseAttr(core.FirstStrike)
			}
			if path == "trample" {
				g.MutablePermanent(attackers[0].ID()).GrantBaseAttr(core.Trample)
			}
			for _, p := range attackers {
				p.RevokeBaseAttr(core.AttrSummonSick)
			}
			ids := []uuid.UUID{attackers[0].ID(), attackers[1].ID()}
			g.ExecuteAttackers(a.PlayerID(), ids)
			if path != "multiple-attacks" {
				g.combat.AddBand(ids)
			} else {
				blockers[0].GrantBaseAttr(core.CanBlockAny)
			}
			assignments := []BlockAssignment{{BlockerID: blockers[0].ID(), AttackerID: ids[0]}}
			if path == "multiple-attacks" {
				assignments = append(assignments, BlockAssignment{BlockerID: blockers[0].ID(), AttackerID: ids[1]})
			} else {
				assignments = append(assignments, BlockAssignment{BlockerID: blockers[1].ID(), AttackerID: ids[0]})
			}
			g.ExecuteBlockers(assignments)
			if path == "lost" {
				g.BouncePermanentToHand(grant)
			}
			if path == "first-strike" {
				g.combat.ResolveDamage(g, true)
				if len(a.calls) != 1 {
					t.Fatal("first strike assignment")
				}
				g.combat.ResolveDamage(g, false)
			} else {
				g.ExecuteCombatDamage()
			}
			switch path {
			case "outgoing", "trample":
				if len(b.calls) != 4 {
					t.Fatalf("defender assignment calls=%d, want 4", len(b.calls))
				}
			case "nonmatching", "lost":
				if len(a.calls) != 0 {
					t.Fatal("inactive quality supplied assignment permission")
				}
			default:
				want := 2
				if path == "multiple-attacks" {
					want = 1
				}
				if len(a.calls) != want {
					t.Fatalf("attacker assignment calls=%d, want %d", len(a.calls), want)
				}
			}
			if path == "multiple-attacks" && attackers[0].Damage+attackers[1].Damage != 3 {
				t.Fatal("blocker damage counted more than once")
			}
			if path == "trample" && b.Life() != 20 {
				t.Fatal("nonlethal trample escaped")
			}
		})
	}
}

func TestBandsWithOtherFormationWindow(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	otherBandsGrant(g, a.PlayerID(), Named("Member"))
	ids := make([]uuid.UUID, 0, 2)
	for range 2 {
		p := g.PutOnBattlefield(NewCreature("Member", "", 1, 20), a.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		ids = append(ids, p.ID())
	}
	g.ExecuteAttackers(a.PlayerID(), ids)
	g.ExecuteBlockers(nil)
	if err := g.FormAttackingBand(ids); err == nil {
		t.Fatal("formed a band after blocker declaration")
	}
}

func TestBandsWithOtherAssignmentRecipients(t *testing.T) {
	for _, matching := range []bool{true, false} {
		a := &bandSourcePlayer{BasePlayer: NewBasePlayer("A")}
		b := &bandSourcePlayer{BasePlayer: NewBasePlayer("B")}
		g := NewGame(a, b)
		otherBandsGrant(g, a.PlayerID(), Named("Member"))
		attackers := make([]*Permanent, 0, 3)
		for i := range 3 {
			name := "Member"
			if i == 2 || (!matching && i == 1) {
				name = "Other"
			}
			p := g.PutOnBattlefield(NewCreature(name, "", 0, 20), a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			attackers = append(attackers, p)
		}
		blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 4, 20, WithKeyword(core.CanBlockAny), WithKeyword(core.Lifelink)), b.PlayerID())
		ids := []uuid.UUID{attackers[0].ID(), attackers[1].ID(), attackers[2].ID()}
		g.ExecuteAttackers(a.PlayerID(), ids)
		pairs := make([]BlockAssignment, 0, len(ids))
		for _, id := range ids {
			pairs = append(pairs, BlockAssignment{blocker.ID(), id})
		}
		g.ExecuteBlockers(pairs)
		a.assign = func(_ *Permanent, recipients []*Permanent, amount int) map[uuid.UUID]int {
			return map[uuid.UUID]int{recipients[2].ID(): amount}
		}
		b.assign = func(_ *Permanent, recipients []*Permanent, amount int) map[uuid.UUID]int {
			return map[uuid.UUID]int{recipients[0].ID(): amount}
		}
		g.AddPreventionShield(attackers[2].ID(), 1)
		g.ExecuteCombatDamage()
		if matching {
			if attackers[2].Damage != 3 || len(a.calls) != 1 || b.Life() != 23 {
				t.Fatal("quality restricted recipients or lost actual damage")
			}
		} else if attackers[0].Damage != 4 || len(a.calls) != 0 || b.Life() != 24 {
			t.Fatal("one qualifying attacker changed chooser")
		}
	}
}
