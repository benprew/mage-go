package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type preventionAllocationPlayer struct {
	*BasePlayer
	numbers     []int
	modes       []int
	numberCalls int
	modeCalls   int
	order       *[]string
}

func (p *preventionAllocationPlayer) ChooseNumber(minimum, maximum int, reason string) int {
	if p.order != nil {
		*p.order = append(*p.order, p.Name())
	}
	i := p.numberCalls
	p.numberCalls++
	if i < len(p.numbers) {
		return p.numbers[i]
	}
	return maximum
}
func (p *preventionAllocationPlayer) ChooseMode(options []string, reason string) int {
	i := p.modeCalls
	p.modeCalls++
	if i < len(p.modes) {
		return p.modes[i]
	}
	return 0
}

func TestPreventionAllocationSplit(t *testing.T) {
	for _, recipient := range []string{"player", "permanent"} {
		t.Run(recipient, func(t *testing.T) {
			a := NewBasePlayer("A")
			b := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("B"), numbers: []int{1}}
			g := NewGame(a, b)
			first := g.PutOnBattlefield(NewCreature("First", "", 1, 10, WithKeyword(core.Lifelink)), a.PlayerID())
			second := g.PutOnBattlefield(NewCreature("Second", "", 1, 10), a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 10), b.PlayerID())
			id := b.PlayerID()
			assignments := []DamageAssignment{PlayerDamageAssignment(b, 3, g.CaptureDamageSource(first.ID())), PlayerDamageAssignment(b, 3, g.CaptureDamageSource(second.ID()))}
			if recipient == "permanent" {
				id = victim.ID()
				assignments = []DamageAssignment{PermanentDamageAssignment(victim, 3, g.CaptureDamageSource(first.ID())), PermanentDamageAssignment(victim, 3, g.CaptureDamageSource(second.ID()))}
			}
			g.AddPreventionShield(id, 3)
			observed := 0
			g.damage.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
				observed++
				if a.Life() != 22 || (recipient == "player" && b.Life() != 17) || (recipient == "permanent" && victim.Damage != 3) {
					t.Error("notification saw partial or incorrect allocation")
				}
			})
			outcomes := g.dealSimultaneousDamageOutcomes(assignments...)
			if len(outcomes) != 2 || outcomes[0].amount != 2 || outcomes[1].amount != 1 || b.numberCalls != 1 || observed != 2 {
				t.Fatalf("outcomes=%+v choices=%d notifications=%d", outcomes, b.numberCalls, observed)
			}
			g.damage.SetOnDamageDealt(nil)
			g.DealDamageToPlayer(b, 1, uuid.New())
			if recipient == "player" && b.Life() != 16 {
				t.Fatal("shield was not consumed")
			}
		})
	}
}

func TestPreventionAllocationAPNAP(t *testing.T) {
	for active := range 2 {
		order := []string{}
		a := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("A"), numbers: []int{1}, order: &order}
		b := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("B"), numbers: []int{1}, order: &order}
		g := NewGame(a, b)
		g.SetActivePlayerIndex(active)
		first := g.PutOnBattlefield(NewCreature("First", "", 1, 10), a.PlayerID())
		second := g.PutOnBattlefield(NewCreature("Second", "", 1, 10), b.PlayerID())
		g.AddPreventionShield(a.PlayerID(), 2)
		g.AddPreventionShield(b.PlayerID(), 2)
		g.DealSimultaneousDamage(PlayerDamageAssignment(b, 2, g.CaptureDamageSource(first.ID())), PlayerDamageAssignment(a, 2, g.CaptureDamageSource(first.ID())), PlayerDamageAssignment(b, 2, g.CaptureDamageSource(second.ID())), PlayerDamageAssignment(a, 2, g.CaptureDamageSource(second.ID())))
		expected := []string{"A", "B"}
		if active == 1 {
			expected = []string{"B", "A"}
		}
		if len(order) != 2 || order[0] != expected[0] || order[1] != expected[1] {
			t.Fatalf("decision order=%v", order)
		}
		if a.Life() != 18 || b.Life() != 18 {
			t.Fatal("allocation totals")
		}
	}
}

func TestPreventionAllocationRechecks(t *testing.T) {
	for _, redirectFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "prevention", true: "redirection"}[redirectFirst], func(t *testing.T) {
			a := NewBasePlayer("A")
			mode := 0
			if redirectFirst {
				mode = 1
			}
			b := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("B"), modes: []int{mode}, numbers: []int{1}}
			g := NewGame(a, b)
			first := g.PutOnBattlefield(NewCreature("First", "", 1, 10), a.PlayerID())
			second := g.PutOnBattlefield(NewCreature("Second", "", 1, 10), a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 10), b.PlayerID())
			g.AddPreventionShield(victim.ID(), 3)
			g.SetCreatureDamageRedirect(victim.ID(), b.PlayerID())
			outcomes := g.dealSimultaneousDamageOutcomes(PermanentDamageAssignment(victim, 3, g.CaptureDamageSource(first.ID())), PermanentDamageAssignment(victim, 3, g.CaptureDamageSource(second.ID())))
			wantDamage, wantLife := 1, 18
			if redirectFirst {
				wantDamage, wantLife = 0, 17
			}
			if victim.Damage != wantDamage || b.Life() != wantLife {
				t.Fatalf("life=%d damage=%d outcomes=%+v", b.Life(), victim.Damage, outcomes)
			}
			if !redirectFirst && (len(outcomes) != 2 || outcomes[0].amount != 2 || outcomes[1].amount != 1 || b.numberCalls != 1) {
				t.Fatalf("prevention split=%+v calls=%d", outcomes, b.numberCalls)
			}
			if redirectFirst && (len(outcomes) != 1 || outcomes[0].amount != 3 || b.numberCalls != 0) {
				t.Fatalf("redirect recheck=%+v calls=%d", outcomes, b.numberCalls)
			}
		})
	}
}

func TestPreventionAllocationBoundsAndCapacity(t *testing.T) {
	for _, tc := range []struct {
		name                                         string
		answer, shield, wantFirst, wantSecond, calls int
	}{
		{"split", 1, 3, 2, 1, 1},
		{"all-first", 3, 3, 0, 3, 1},
		{"all-second", 0, 3, 3, 0, 1},
		{"negative-answer", -1, 3, 0, 3, 1},
		{"excess-answer", 9, 3, 0, 3, 1},
		{"enough", 0, 9, 0, 0, 0},
		{"empty", 0, 0, 3, 3, 0},
		{"negative", 0, -1, 3, 3, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := NewBasePlayer("A")
			b := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("B"), numbers: []int{tc.answer}}
			g := NewGame(a, b)
			first := g.PutOnBattlefield(NewCreature("First", "", 1, 10, WithKeyword(core.Lifelink)), a.PlayerID())
			second := g.PutOnBattlefield(NewCreature("Second", "", 1, 10), a.PlayerID())
			g.AddPreventionShield(b.PlayerID(), tc.shield)
			g.DealSimultaneousDamage(PlayerDamageAssignment(b, 0, g.CaptureDamageSource(first.ID())), PlayerDamageAssignment(b, 3, g.CaptureDamageSource(first.ID())), PlayerDamageAssignment(b, -2, g.CaptureDamageSource(first.ID())), PlayerDamageAssignment(b, 3, g.CaptureDamageSource(second.ID())), DamageAssignment{})
			if a.Life() != 20+tc.wantFirst || b.Life() != 20-tc.wantFirst-tc.wantSecond || b.numberCalls != tc.calls {
				t.Fatalf("life=%d/%d calls=%d", a.Life(), b.Life(), b.numberCalls)
			}
			if tc.name == "enough" {
				g.DealDamageToPlayer(b, 4, uuid.New())
				if b.Life() != 19 {
					t.Fatal("unused capacity was lost")
				}
			}
		})
	}
}

func TestPreventionAllocationIdentityAndClone(t *testing.T) {
	for _, state := range []string{"normal", "control", "phase-cycle", "phased", "return", "expired", "unpreventable"} {
		t.Run(state, func(t *testing.T) {
			a := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("A"), numbers: []int{1}}
			b := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("B"), numbers: []int{2}}
			g := NewGame(a, b)
			firstCard := NewCreature("First", "", 1, 10, WithKeyword(core.Lifelink))
			first := g.PutOnBattlefield(firstCard, a.PlayerID())
			source := g.CaptureDamageSource(first.ID())
			second := g.PutOnBattlefield(NewCreature("Second", "", 1, 10), a.PlayerID())
			victimCard := NewCreature("Victim", "", 1, 20)
			victim := g.PutOnBattlefield(victimCard, a.PlayerID())
			g.AddPreventionShield(victim.ID(), 3)
			g.RemoveFromBattlefield(first)
			g.PutOnBattlefield(firstCard, b.PlayerID())
			switch state {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: second.ID(), TargetID: victim.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "phased":
				g.PhaseOut(victim.ID())
			case "return":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, a.PlayerID())
			case "expired":
				g.RunStepWithPriority(core.Cleanup)
			case "unpreventable":
				victim.GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
			}
			clone := g.Clone()
			g.DealSimultaneousDamage(PermanentDamageAssignment(victim, 3, source), PermanentDamageAssignment(victim, 3, g.CaptureDamageSource(second.ID())))
			wantDamage, wantLife := 3, 22
			if state == "control" {
				wantLife = 21
			}
			if state == "return" || state == "expired" || state == "unpreventable" {
				wantDamage, wantLife = 6, 23
			}
			if state == "phased" {
				wantDamage, wantLife = 0, 20
			}
			if g.FindPermanentIncludingPhased(victim.ID()).Damage != wantDamage || a.Life() != wantLife || b.Life() != 20 {
				t.Fatalf("damage=%d life=%d/%d", g.FindPermanentIncludingPhased(victim.ID()).Damage, a.Life(), b.Life())
			}
			if state == "control" && (a.numberCalls != 0 || b.numberCalls != 1) {
				t.Fatal("owner chose for controlled recipient")
			}
			if clone.FindPermanentIncludingPhased(victim.ID()).Damage != 0 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("damage mutated clone")
			}
			clone.DealSimultaneousDamage(PermanentDamageAssignment(clone.FindPermanentIncludingPhased(victim.ID()), 3, source), PermanentDamageAssignment(clone.FindPermanentIncludingPhased(victim.ID()), 3, clone.CaptureDamageSource(second.ID())))
			if clone.FindPermanentIncludingPhased(victim.ID()).Damage != wantDamage {
				t.Fatal("clone lost shield or identity")
			}
			if state == "unpreventable" {
				g.FindPermanent(victim.ID()).RevokeBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
				g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 3, second.ID())
				if g.FindPermanent(victim.ID()).Damage != 6 {
					t.Fatal("unpreventable damage consumed shield")
				}
			}
		})
	}
}

func TestPreventionAllocationOneSource(t *testing.T) {
	a := NewBasePlayer("A")
	b := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 10), a.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 3)
	g.DealSimultaneousDamage(PlayerDamageAssignment(b, 2, g.CaptureDamageSource(source.ID())), PlayerDamageAssignment(b, 2, g.CaptureDamageSource(source.ID())))
	if b.Life() != 19 || b.numberCalls != 0 {
		t.Fatal("one source required an allocation choice")
	}
}

func TestPreventionAllocationRedirectIntoShield(t *testing.T) {
	a := NewBasePlayer("A")
	b := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("B"), numbers: []int{1}}
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 10, WithKeyword(core.Lifelink)), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 10), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 10), b.PlayerID())
	g.SetCreatureDamageRedirect(victim.ID(), b.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 3)
	g.DealSimultaneousDamage(PermanentDamageAssignment(victim, 3, g.CaptureDamageSource(first.ID())), PlayerDamageAssignment(b, 3, g.CaptureDamageSource(second.ID())))
	if b.Life() != 17 || a.Life() != 22 || victim.Damage != 0 || b.numberCalls != 1 {
		t.Fatalf("redirected allocation life=%d/%d damage=%d choices=%d", a.Life(), b.Life(), victim.Damage, b.numberCalls)
	}
}

func TestPreventionAllocationCombatStepBatch(t *testing.T) {
	for _, firstStrike := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "first-strike"}[firstStrike], func(t *testing.T) {
			a := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("A"), numbers: []int{1}}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 4, 20), a.PlayerID())
			first := g.PutOnBattlefield(NewCreature("First blocker", "", 2, 10), b.PlayerID())
			second := g.PutOnBattlefield(NewCreature("Second blocker", "", 3, 10), b.PlayerID())
			if firstStrike {
				attacker.GrantBaseAttr(core.FirstStrike)
				first.GrantBaseAttr(core.FirstStrike)
				second.GrantBaseAttr(core.FirstStrike)
			}
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
			g.ExecuteBlockers([]BlockAssignment{{BlockerID: first.ID(), AttackerID: attacker.ID()}, {BlockerID: second.ID(), AttackerID: attacker.ID()}})
			g.AddPreventionShield(attacker.ID(), 3)
			calls := 0
			g.damage.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
				calls++
				if attacker.Damage != 2 || first.Damage != 4 || second.Damage != 0 {
					t.Error("combat notification saw partial step")
				}
			})
			g.damage.SetResolvingCombatDamage(true)
			g.GetCombat().ResolveDamage(g, firstStrike)
			g.damage.SetResolvingCombatDamage(false)
			if attacker.Damage != 2 || calls != 3 || a.numberCalls != 1 {
				t.Fatalf("damage=%d callbacks=%d decisions=%d", attacker.Damage, calls, a.numberCalls)
			}
		})
	}
}

type allocationDoubleReplacement struct {
	replacementBase
	source uuid.UUID
}

func (r *allocationDoubleReplacement) Matches(action Action, _ GameReader) bool {
	return action.ActionSource() == r.source && damageActionAmount(action) > 0
}
func (r *allocationDoubleReplacement) Replace(action Action, _ *Game) Action {
	return damageActionWithAmount(action, damageActionAmount(action)*2)
}
func (r *allocationDoubleReplacement) IsActive(GameReader) bool { return true }
func (r *allocationDoubleReplacement) Clone() ReplacementEffect { cp := *r; return &cp }
func (r *allocationDoubleReplacement) ReplacementPriority() ReplacementPriority {
	return ReplacementPrioritySelf
}

func TestPreventionAllocationPriority(t *testing.T) {
	a := NewBasePlayer("A")
	b := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("B"), numbers: []int{1}}
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 10, WithKeyword(core.Lifelink)), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 10), a.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 3)
	g.effects.AddReplacement(&allocationDoubleReplacement{source: second.ID()})
	outcomes := g.dealSimultaneousDamageOutcomes(PlayerDamageAssignment(b, 3, g.CaptureDamageSource(first.ID())), PlayerDamageAssignment(b, 3, g.CaptureDamageSource(second.ID())))
	if len(outcomes) != 2 || outcomes[0].amount != 2 || outcomes[1].amount != 4 || b.Life() != 14 || a.Life() != 22 {
		t.Fatalf("priority outcomes=%+v life=%d/%d", outcomes, a.Life(), b.Life())
	}
}

func TestPreventionAllocationChooseLaterPacket(t *testing.T) {
	a := NewBasePlayer("A")
	b := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("B"), modes: []int{2}}
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 10, WithKeyword(core.Lifelink)), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 10), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 10), b.PlayerID())
	g.AddPreventionShield(victim.ID(), 3)
	g.SetCreatureDamageRedirect(victim.ID(), b.PlayerID())
	g.DealSimultaneousDamage(PermanentDamageAssignment(victim, 3, g.CaptureDamageSource(first.ID())), PermanentDamageAssignment(victim, 3, g.CaptureDamageSource(second.ID())))
	if a.Life() != 20 || b.Life() != 17 || victim.Damage != 0 || b.numberCalls != 0 {
		t.Fatalf("later packet choice life=%d/%d damage=%d", a.Life(), b.Life(), victim.Damage)
	}
}

func TestPreventionAllocationSourceIncarnations(t *testing.T) {
	a := NewBasePlayer("A")
	b := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("B"), numbers: []int{1}}
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 10, WithKeyword(core.Lifelink))
	first := g.PutOnBattlefield(card, a.PlayerID())
	old := g.CaptureDamageSource(first.ID())
	g.RemoveFromBattlefield(first)
	current := g.PutOnBattlefield(card, b.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 3)
	g.DealSimultaneousDamage(PlayerDamageAssignment(b, 3, old), PlayerDamageAssignment(b, 3, g.CaptureDamageSource(current.ID())))
	if a.Life() != 22 || b.Life() != 18 || b.numberCalls != 1 {
		t.Fatalf("source incarnations life=%d/%d decisions=%d", a.Life(), b.Life(), b.numberCalls)
	}
}

func TestPreventionAllocationLargeAmounts(t *testing.T) {
	a := NewBasePlayer("A")
	b := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("B"), numbers: []int{1}}
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 10), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 10), a.PlayerID())
	maximum := int(^uint(0) >> 1)
	actions := []Action{NewDamageToPlayerAction(first.ID(), b.PlayerID(), maximum, false).withDamageSource(g.CaptureDamageSource(first.ID())), NewDamageToPlayerAction(second.ID(), b.PlayerID(), maximum, false).withDamageSource(g.CaptureDamageSource(second.ID()))}
	g.AddPreventionShield(b.PlayerID(), 3)
	results := g.effects.applyDamageBatchReplacements(actions, g)
	if damageActionAmount(results[0]) != maximum-1 || damageActionAmount(results[1]) != maximum-2 || b.numberCalls != 1 {
		t.Fatal("allocation overflow changed shield amounts")
	}
}
