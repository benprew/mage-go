package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func suppressCombatAssignment(t *testing.T, g *Game, p *Permanent) {
	t.Helper()
	if err := ApplyEffect(g, Until(core.EndOfTurn, AssignsNoCombatDamage(This)), p.ID(), p.ControllerID(), nil); err != nil {
		t.Fatal(err)
	}
}

func TestCombatAssignmentRulePaths(t *testing.T) {
	for _, path := range []string{"unblocked", "blocked", "trample", "departed-blocker", "blocking-band", "attacking-band", "unblocked-band"} {
		for _, suppressBlocker := range []bool{false, true} {
			t.Run(path+map[bool]string{false: "/attacker", true: "/blocker"}[suppressBlocker], func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				g.SetTurn(1)
				attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 4, 20, WithKeyword(core.Lifelink)), a.PlayerID())
				other := g.PutOnBattlefield(NewCreature("Other", "", 2, 20, WithKeyword(core.Banding)), a.PlayerID())
				blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 3, 20), b.PlayerID())
				second := g.PutOnBattlefield(NewCreature("Second", "", 1, 20, WithKeyword(core.Banding)), b.PlayerID())
				if path == "trample" || path == "departed-blocker" {
					attacker.GrantBaseAttr(core.Trample)
				}
				for _, p := range []*Permanent{attacker, other} {
					p.RevokeBaseAttr(core.AttrSummonSick)
				}
				ids := []uuid.UUID{attacker.ID()}
				if path == "attacking-band" || path == "unblocked-band" {
					ids = append(ids, other.ID())
				}
				g.ExecuteAttackers(a.PlayerID(), ids)
				if len(ids) == 2 {
					g.GetCombat().AddBand(ids)
				}
				var blocks []BlockAssignment
				if path != "unblocked" && path != "unblocked-band" {
					blocks = append(blocks, BlockAssignment{BlockerID: blocker.ID(), AttackerID: attacker.ID()})
					if path == "blocking-band" {
						blocks = append(blocks, BlockAssignment{BlockerID: second.ID(), AttackerID: attacker.ID()})
					}
				}
				g.ExecuteBlockers(blocks)
				subject := attacker
				if suppressBlocker {
					subject = blocker
				}
				suppressCombatAssignment(t, g, subject)
				if path == "departed-blocker" {
					g.RemoveFromCombat(blocker.ID())
				}
				g.ExecuteCombatDamage()
				if !suppressBlocker {
					if blocker.Damage+second.Damage > 2 || a.Life() != 20 {
						t.Fatal("suppressed attacker assigned damage or gained life")
					}
					if path != "unblocked-band" && b.Life() != 20 {
						t.Fatal("suppressed attacker dealt player damage")
					}
					if path == "unblocked-band" && b.Life() != 18 {
						t.Fatal("other band member lost damage")
					}
				} else if attacker.Damage > 1 || other.Damage > 0 {
					t.Fatal("suppressed blocker assigned damage")
				}
				if !suppressBlocker && path == "blocked" && attacker.Damage != 3 {
					t.Fatal("incoming combat damage was suppressed")
				}
			})
		}
	}
}

func TestCombatAssignmentRuleIdentityAndCleanup(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	card := NewCreature("Subject", "", 4, 20)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	suppressCombatAssignment(t, g, p)
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PhaseIn(g.PhaseOut(p.ID()))
	p = g.FindPermanent(p.ID())
	g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
	clone := g.Clone()
	clone.GetCombat().ResolveDamage(clone, false)
	if clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("clone lost assignment rule")
	}
	clone.RunStepWithPriority(core.Cleanup)
	clone.GetCombat().Reset()
	clone.GetCombat().AddAttacker(p.ID(), b.PlayerID())
	clone.GetCombat().ResolveDamage(clone, false)
	if clone.GetPlayer(b.PlayerID()).Life() != 16 {
		t.Fatal("cleanup did not end rule")
	}
	g.GetCombat().ResolveDamage(g, false)
	if b.Life() != 20 {
		t.Fatal("phase cycle, ability removal or clone cleanup erased rule")
	}
	g.DealDamageToPlayer(b, 2, p.ID())
	if b.Life() != 18 {
		t.Fatal("noncombat damage was suppressed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	g.GetCombat().Reset()
	p = g.PutOnBattlefield(card, a.PlayerID())
	g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
	g.GetCombat().ResolveDamage(g, false)
	if b.Life() != 14 {
		t.Fatal("returned incarnation inherited rule")
	}
}

func TestCombatAssignmentRuleStrikeSteps(t *testing.T) {
	for _, keyword := range []core.Attr{core.FirstStrike, core.DoubleStrike} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.SetTurn(1)
		p := g.PutOnBattlefield(NewCreature("Subject", "", 4, 20, WithKeyword(keyword)), a.PlayerID())
		suppressCombatAssignment(t, g, p)
		g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
		g.GetCombat().ResolveDamage(g, true)
		g.GetCombat().ResolveDamage(g, false)
		if b.Life() != 20 {
			t.Fatal("strike step bypassed rule")
		}
		if !g.GetCombat().FirstStruck[p.ID()] {
			t.Fatal("suppression changed strike eligibility history")
		}
	}
}

func TestCombatAssignmentRuleDeclarations(t *testing.T) {
	for _, subject := range []PermanentRef{nil, DeclarePermanentTarget("missing", IsCreature)} {
		card := NewCreature("Invalid", "", 1, 1, WithAction(Activate(Costs(), Until(core.EndOfTurn, AssignsNoCombatDamage(subject)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestCombatAssignmentRuleNoDamageEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 4, 20, WithKeyword(core.Lifelink)), a.PlayerID())
	p.GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
	suppressCombatAssignment(t, g, p)
	g.AddPreventionShield(b.PlayerID(), 2)
	notifications := 0
	g.damage.SetOnDamageDealt(func(_, _ string, _ int, _ bool) { notifications++ })
	g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
	g.GetCombat().ResolveDamage(g, false)
	if b.Life() != 20 || a.Life() != 20 || notifications != 0 || g.HasDealtDamageToPlayerFromSource(g.CaptureDamageSource(p.ID()), b.PlayerID()) {
		t.Fatal("assignment suppression reached damage execution")
	}
	p.RevokeBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
	g.DealDamageToPlayer(b, 3, p.ID())
	if b.Life() != 19 || a.Life() != 21 || notifications != 1 {
		t.Fatal("suppression spent a shield or stopped noncombat damage")
	}
}

func TestCombatAssignmentRuleInstallation(t *testing.T) {
	for _, change := range []string{"source-left", "control", "phase", "absent-install", "phased-install"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetTurn(1)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			card := NewCreature("Subject", "", 4, 20)
			p := g.PutOnBattlefield(card, a.PlayerID())
			binding := BindPermanent("saved")
			target := DeclarePermanentTarget("subject", IsCreature)
			g.resolution.targetSpecs = []Target{target.targetDeclaration}
			changeBeforeInstall := FuncEffect("change before installation", EffectProperties{}, func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
				if change == "absent-install" {
					g.RemoveFromBattlefield(p)
				}
				if change == "phased-install" {
					g.PhaseOut(p.ID())
				}
				return nil
			})
			body := Sequence(Snapshot(target, binding), changeBeforeInstall, Until(core.EndOfTurn, AssignsNoCombatDamage(From(binding))))
			if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "source-left":
				g.RemoveFromBattlefield(source)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "absent-install":
				p = g.PutOnBattlefield(card, a.PlayerID())
			case "phased-install":
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
			g.GetCombat().ResolveDamage(g, false)
			want := 20
			if change == "absent-install" || change == "phased-install" {
				want = 16
			}
			if b.Life() != want {
				t.Fatalf("life=%d want=%d", b.Life(), want)
			}
		})
	}
}
