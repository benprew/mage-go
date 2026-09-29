package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// --- from composable_any_attack_batch_test.go ---
func TestAnyAttackBatchPatternBothPlayersAndCount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("declared")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesAttack().WithCount(count), GainLifeFor(You, FromNumber(count)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	first := NewCreature("First", "", 1, 1)
	second := NewCreature("Second", "", 1, 1)
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.FindPermanent(first.ID()).RevokeBaseAttr(AttrSummonSick)
	g.FindPermanent(second.ID()).RevokeBaseAttr(AttrSummonSick)
	g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatal("clone changed original or lost attack count")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatalf("life totals = %d, %d", a.Life(), b.Life())
	}
	g.FireEvent(GameEvent{Type: EvtAttackersDeclared, PlayerID: a.PlayerID(), Amount: 1})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("own attack stack size = %d, want 1", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatalf("life after own attack = %d, want 23", a.Life())
	}
	g.FireEvent(GameEvent{Type: EvtEntersAttacking, SourceID: first.ID(), PlayerID: b.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtAttackersDeclared, PlayerID: b.PlayerID(), Amount: 0})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("an empty declaration or entering attacking triggered")
	}
}

func TestAnyAttackBatchPatternLifetimeAndDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesAttack(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	phased := g.PhaseOut(watcher.ID())
	g.FireEvent(GameEvent{Type: EvtAttackersDeclared, PlayerID: b.PlayerID(), Amount: 1})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	g.FireEvent(GameEvent{Type: EvtAttackersDeclared, PlayerID: b.PlayerID(), Amount: 1})
	g.PutTriggersOnStack()
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("pending trigger followed returned source")
	}
	for _, pattern := range []EventPattern{
		OneOrMoreCreaturesAttack().WithCount(nil),
		OneOrMoreCreaturesAttack().WithCount(BindNumber("count")).WithCount(BindNumber("again")),
	} {
		bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("accepted invalid count declaration")
		}
	}
	spell := NewInstant("Invalid", "", Spell(GainLifeFor(You, Fixed(1))))
	spell.AddAbility(WhenEvent(OneOrMoreCreaturesAttack(), GainLifeFor(You, Fixed(1))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell registration = %v", err)
	}
}

// --- from composable_attack_batch_test.go ---
func TestAttackBatchPatternOnceAndController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(OneOrMoreCreaturesYouControlAttack(), GainLifeFor(You, Fixed(2)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtAttackersDeclared, PlayerID: b.PlayerID(), Amount: 2})
	g.FireEvent(GameEvent{Type: EvtAttackersDeclared, PlayerID: a.PlayerID(), Amount: 2})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatal("clone changed original life or lost trigger")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("life = %d, want 22", a.Life())
	}
}

func TestAttackBatchPatternPhasingAndRegistration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(OneOrMoreCreaturesYouControlAttack(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	phased := g.PhaseOut(watcher.ID())
	g.FireEvent(GameEvent{Type: EvtAttackersDeclared, PlayerID: a.PlayerID(), Amount: 1})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	g.FireEvent(GameEvent{Type: EvtAttackersDeclared, PlayerID: a.PlayerID(), Amount: 1})
	g.PutTriggersOnStack()
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("pending trigger lost event controller")
	}
	spell := NewInstant("Invalid", "", Spell(GainLifeFor(You, Fixed(1))))
	spell.AddAbility(WhenEvent(OneOrMoreCreaturesYouControlAttack(), GainLifeFor(You, Fixed(1))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell registration = %v", err)
	}
	for _, pattern := range []EventPattern{
		OneOrMoreCreaturesYouControlAttack().WithCount(nil),
		OneOrMoreCreaturesYouControlAttack().WithCount(BindNumber("count")).WithCount(BindNumber("again")),
	} {
		bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("accepted invalid count declaration")
		}
	}
}

func TestAttackBatchPatternSimulatedDeclarationCount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("attackers")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesYouControlAttack().WithCount(count), GainLifeFor(You, FromNumber(count)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	first := NewCreature("First", "", 1, 1)
	second := NewCreature("Second", "", 1, 1)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.FindPermanent(first.ID()).RevokeBaseAttr(AttrSummonSick)
	g.FindPermanent(second.ID()).RevokeBaseAttr(AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d, want 1", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("life = %d, want 22", a.Life())
	}
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("an existing attacker made a new group")
	}
	g.FireEvent(GameEvent{Type: EvtEntersAttacking, SourceID: first.ID(), PlayerID: a.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtAttackersDeclared, PlayerID: a.PlayerID(), Amount: 0})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("nondeclaration made an attack-group trigger")
	}
}

// --- from composable_attack_defender_test.go ---
func TestAttackDefendingPlayerCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	defender := BindPlayer("defender")
	card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		Attacks(This).WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(2)))))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: card.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("attack trigger count = %d", g.StackSize())
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.PutOnBattlefield(card, b.PlayerID())
	g.ResolveStack()
	if g.GetPlayer(b.PlayerID()).Life() != 22 || g.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("defending player changed after attack")
	}
	if clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("clone shared trigger state")
	}
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("clone lost event capture")
	}
}

func TestAttackDefendingPlayerDeclaration(t *testing.T) {
	defender := BindPlayer("defender")
	for _, pattern := range []EventPattern{Attacks(This).WithDefendingPlayer(nil), Attacks(This).WithDefendingPlayer(defender).WithDefendingPlayer(defender)} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid defender binding")
		}
	}
	card := NewCreature("Missing", "", 1, 1, WithAbility(WhenEvent(Attacks(This), GainLifeFor(FromPlayer(defender), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("missing binding validation = %v", err)
	}
}

func TestAttackDefendingPlayerRealDeclarationAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	defender := BindPlayer("defender")
	card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		Attacks(This).WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(1)))))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	g.FindPermanent(card.ID()).RevokeBaseAttr(AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{card.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("actual attack trigger count = %d", g.StackSize())
	}
	g.ResolveStack()
	if g.GetPlayer(b.PlayerID()).Life() != 21 {
		t.Fatal("actual attack did not capture defender")
	}
	phase := g.PhaseOut(card.ID())
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: card.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phase)
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: card.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("absent source triggered")
	}
}

// --- from composable_blocked_attackers_set_test.go ---
func TestBlockedAttackersOfCurrentCombatIncarnations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 2), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 2), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 2), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 2, 2), b.PlayerID())
	unrelated := g.PutOnBattlefield(NewCreature("Unrelated", "", 2, 2), b.PlayerID())
	for _, attacker := range []*Permanent{first, second, other} {
		g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
	}
	g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, first.ID())
	g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, second.ID())
	g.combat.AddBlockerIncarnation(unrelated.ID(), unrelated.incarnationID, other.ID())
	ctx := &EffectContext{Game: g, SourceID: blocker.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: blocker.incarnationID}
	count := CountPermanents(BlockedAttackersOf(This))
	if got := count.resolveValue(ctx); got != 2 {
		t.Fatalf("blocked attackers = %d, want 2", got)
	}
	clone := g.Clone()
	clone.RemoveFromCombat(first.ID())
	cloneCtx := &EffectContext{Game: clone, SourceID: blocker.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: blocker.incarnationID}
	if got := count.resolveValue(cloneCtx); got != 1 {
		t.Fatalf("clone count = %d, want 1", got)
	}
	if got := count.resolveValue(ctx); got != 2 {
		t.Fatalf("original count after clone edit = %d", got)
	}
	phase := g.Clone()
	phase.PhaseOut(second.ID())
	phaseCtx := &EffectContext{Game: phase, SourceID: blocker.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: blocker.incarnationID}
	if got := count.resolveValue(phaseCtx); got != 1 {
		t.Fatalf("phased attacker count = %d", got)
	}
	phase.PhaseOut(blocker.ID())
	if got := count.resolveValue(phaseCtx); got != 0 {
		t.Fatalf("phased blocker count = %d", got)
	}
	g.RemoveFromBattlefield(first)
	g.PutOnBattlefield(first.Card, a.PlayerID())
	if got := count.resolveValue(ctx); got != 1 {
		t.Fatalf("returned attacker count = %d", got)
	}
	g.RemoveFromBattlefield(blocker)
	g.PutOnBattlefield(blocker.Card, b.PlayerID())
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("returned blocker count = %d", got)
	}
}

func TestBlockedAttackersOfDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("blocker", IsCreature)
	valid := NewInstant("Valid", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(BlockedAttackersOf(target))), WithTargets(target))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if info := InspectDefinition(BlockedAttackersOf(target)); info.Role != "permanent-set" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
	for _, subject := range []PermanentRef{nil, target, From(BindPermanent("missing"))} {
		bad := NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(BlockedAttackersOf(subject))))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid blocked-attacker set accepted")
		}
	}
}

func TestBlockedAttackersOfTargetResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 2), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 2, 2), b.PlayerID())
	g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
	g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, attacker.ID())
	target := DeclarePermanentTarget("blocker", IsCreature)
	card := NewInstant("Count", "", Spell(GainLifeFor(You, CountPermanents(BlockedAttackersOf(target))), WithTargets(target)))
	ability := card.Abilities()[0].(*SpellAbility)
	push := func(game *Game) {
		game.pushStack(&StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{blocker.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	}
	push(g)
	clone := g.Clone()
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("clone life = %d, want 21", got)
	}
	if got := a.Life(); got != 20 {
		t.Fatalf("original life = %d, want 20", got)
	}
	g.RemoveFromBattlefield(blocker)
	g.PutOnBattlefield(blocker.Card, b.PlayerID())
	g.ResolveStack()
	if got := a.Life(); got != 20 {
		t.Fatalf("returned target life = %d, want 20", got)
	}
}

// --- from composable_blocked_by_test.go ---
func TestBlockedByPatternPairsAndBinding(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := BindPermanent("blocking Wall")
	watcher := NewCreature("Watcher", "", 4, 4, WithAbility(WhenEvent(
		BecomesBlockedBy(This).OfSubtype("Wall").WithBlocker(blocker),
		AddCountersTo(From(blocker), P1P1, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	wall1 := NewCreature("Wall 1", "", 1, 4, WithSubTypes("Wall"))
	wall2 := NewCreature("Wall 2", "", 1, 4, WithSubTypes("Wall"))
	other := NewCreature("Other", "", 1, 1)
	for _, card := range []Card{wall1, wall2, other} {
		g.PutOnBattlefield(card, b.PlayerID())
	}
	g.FireEvent(GameEvent{Type: EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: wall1.ID(), PlayerID: a.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: wall2.ID(), PlayerID: a.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: other.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("got %d triggers, want 2", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if g.FindPermanent(wall1.ID()).Counters[P1P1] != 0 {
		t.Fatal("clone changed original blocker")
	}
	g.ResolveStack()
	for _, game := range []*Game{g, clone} {
		if game.FindPermanent(wall1.ID()).Counters[P1P1] != 1 || game.FindPermanent(wall2.ID()).Counters[P1P1] != 1 || game.FindPermanent(other.ID()).Counters[P1P1] != 0 {
			t.Fatal("wrong per-blocker capture")
		}
	}
}

func TestBlockedByPatternAbsentBlocker(t *testing.T) {
	for _, change := range []string{"returned", "phased"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			blocker := BindPermanent("blocker")
			watcher := NewCreature("Watcher", "", 4, 4, WithAbility(WhenEvent(BecomesBlockedBy(This).WithBlocker(blocker), AddCountersTo(From(blocker), P1P1, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			wall := NewCreature("Wall", "", 1, 4, WithSubTypes("Wall"))
			g.PutOnBattlefield(wall, b.PlayerID())
			g.FireEvent(GameEvent{Type: EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: wall.ID(), PlayerID: a.PlayerID()})
			if change == "returned" {
				g.RemoveFromBattlefield(g.FindPermanent(wall.ID()))
				g.PutOnBattlefield(wall, b.PlayerID())
			} else {
				g.PhaseOut(wall.ID())
			}
			g.PutTriggersOnStack()
			g.ResolveStack()
			if change == "returned" && g.FindPermanent(wall.ID()).Counters[P1P1] != 0 {
				t.Fatal("old trigger changed returned blocker")
			}
		})
	}
}

func TestBlockedByPatternCombatAndSourceIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := BindPermanent("blocker")
	watcher := NewCreature("Watcher", "", 4, 4, WithAbility(WhenEvent(BecomesBlockedBy(This).OfSubtype("Wall").WithBlocker(blocker), AddCountersTo(From(blocker), P1P1, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	wall := NewCreature("Wall", "", 1, 4, WithSubTypes("Wall"))
	g.PutOnBattlefield(wall, b.PlayerID())
	g.combat.AddAttacker(watcher.ID(), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: wall.ID(), AttackerID: watcher.ID()}})
	if got := g.FindPermanent(wall.ID()).Counters[P1P1]; got != 1 {
		t.Fatalf("real combat did not capture blocker: %d", got)
	}
	g.FireEvent(GameEvent{Type: EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: wall.ID(), PlayerID: a.PlayerID()})
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.FindPermanent(wall.ID()).Counters[P1P1]; got != 2 {
		t.Fatalf("source departure lost event controller or blocker: %d", got)
	}
}

func TestBlockedByPatternStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := BindPermanent("blocker")
	watcher := NewCreature("Watcher", "", 4, 4, WithAbility(WhenEvent(BecomesBlockedBy(This).WithBlocker(blocker), AddCountersTo(From(blocker), P1P1, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	wall := NewCreature("Wall", "", 1, 4, WithSubTypes("Wall"))
	g.PutOnBattlefield(wall, b.PlayerID())
	g.FireEvent(GameEvent{Type: EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: wall.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("could not copy trigger")
	}
	g.ResolveStack()
	if got := g.FindPermanent(wall.ID()).Counters[P1P1]; got != 2 {
		t.Fatalf("copy lost blocker capture: %d", got)
	}
}

func TestBlockedByPatternValidation(t *testing.T) {
	for _, pattern := range []EventPattern{BecomesBlockedBy(This).OfSubtype(""), BecomesBlockedBy(This).OfSubtype("Island"), BecomesBlockedBy(This).WithBlocker(nil), BecomesBlockedBy(This).WithBlocker(BindPermanent(""))} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid blocked-by pattern accepted")
		}
	}
	blocker := BindPermanent("unbound")
	card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(BecomesBlockedBy(This), AddCountersTo(From(blocker), P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("unbound blocker accepted: %v", err)
	}
}

func TestBlockedByPatternWithoutSubtype(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := BindPermanent("non-Wall blocker")
	watcher := NewCreature("Watcher", "", 4, 4, WithAbility(WhenEvent(
		BecomesBlockedBy(This).WithoutSubtype("Wall").WithBlocker(blocker),
		AddCountersTo(From(blocker), P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	wall := NewCreature("Wall", "", 1, 4, WithSubTypes("Wall"))
	other := NewCreature("Other", "", 1, 4, WithSubTypes("Human"))
	g.PutOnBattlefield(wall, b.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	g.combat.AddAttacker(watcher.ID(), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: wall.ID(), AttackerID: watcher.ID()}, {BlockerID: other.ID(), AttackerID: watcher.ID()}})
	if got := g.FindPermanent(other.ID()).Counters[P1P1]; got != 1 {
		t.Fatalf("non-Wall blocker got %d counters, want 1", got)
	}
	if got := g.FindPermanent(wall.ID()).Counters[P1P1]; got != 0 {
		t.Fatalf("Wall blocker got %d counters, want 0", got)
	}
}

func TestBlockedByPatternWithoutSubtypeValidation(t *testing.T) {
	patterns := []EventPattern{
		BecomesBlockedBy(This).WithoutSubtype(""),
		BecomesBlockedBy(This).WithoutSubtype("Island"),
		BecomesBlockedBy(This).WithoutSubtype("Wall").WithoutSubtype("Orc"),
		BecomesBlockedBy(This).OfSubtype("Orc").WithoutSubtype("Wall"),
	}
	for _, pattern := range patterns {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid blocked-by exclusion accepted")
		}
	}
}

func TestBlockedByPatternWithoutSubtypeEventCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := BindPermanent("blocker")
	watcher := NewCreature("Watcher", "", 4, 4, WithAbility(WhenEvent(
		BecomesBlockedBy(This).WithoutSubtype("Wall").WithBlocker(blocker),
		AddCountersTo(From(blocker), P1P1, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	human := NewCreature("Human", "", 1, 4, WithSubTypes("Human"))
	g.PutOnBattlefield(human, b.PlayerID())
	g.FireEvent(GameEvent{Type: EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: human.ID(), PlayerID: a.PlayerID()})
	human.AddSubType("Wall")
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("got %d triggers, want 1", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("could not copy trigger")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if got := g.FindPermanent(human.ID()).Counters[P1P1]; got != 0 {
		t.Fatalf("clone changed original blocker: %d", got)
	}
	g.ResolveStack()
	for _, game := range []*Game{g, clone} {
		if got := game.FindPermanent(human.ID()).Counters[P1P1]; got != 2 {
			t.Fatalf("event-time match or copy lost: %d", got)
		}
	}
	g.PhaseOut(human.ID())
	g.FireEvent(GameEvent{Type: EvtCreatureBlockedBy, SourceID: watcher.ID(), TargetID: human.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 0 {
		t.Fatalf("phased-out blocker produced %d triggers", got)
	}
}

// --- from composable_blocked_defender_test.go ---
func TestBecomesBlockedCapturesDefendingPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	defender := BindPlayer("defender")
	attacker := NewCreature("Attacker", "", 2, 2, WithAbility(WhenEvent(
		BecomesBlocked(This).WithDefendingPlayer(defender), Discard(FromPlayer(defender), Fixed(1)))))
	attacker.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(attacker.Name(), attacker); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(attacker, a.PlayerID())
	blocker := NewCreature("Blocker", "", 1, 3)
	g.PutOnBattlefield(blocker, b.PlayerID())
	b.AddToHand(NewArtifact("Card", ""))
	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	g.combat.AddBlockerIncarnation(blocker.ID(), g.FindPermanent(blocker.ID()).incarnationID, attacker.ID())
	g.fireCreatureBecomesBlocked(attacker.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(attacker.ID()))
	g.PutOnBattlefield(attacker, b.PlayerID())
	g.ResolveStack()
	if len(b.Hand()) != 0 || len(a.Hand()) != 0 {
		t.Fatal("defending player's card was not discarded")
	}
	if len(clone.GetPlayer(b.PlayerID()).Hand()) != 1 {
		t.Fatal("clone shared trigger result")
	}
	clone.ResolveStack()
	if len(clone.GetPlayer(b.PlayerID()).Hand()) != 0 {
		t.Fatal("clone lost defender capture")
	}
}

func TestBecomesBlockedDefenderDeclaration(t *testing.T) {
	defender := BindPlayer("defender")
	for _, pattern := range []EventPattern{
		BecomesBlocked(This).WithDefendingPlayer(nil),
		BecomesBlocked(This).WithDefendingPlayer(defender).WithDefendingPlayer(defender),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid defender capture")
		}
	}
	card := NewCreature("Missing", "", 1, 1, WithAbility(WhenEvent(BecomesBlocked(This), GainLifeFor(FromPlayer(defender), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("validation = %v", err)
	}
}

func TestBecomesBlockedDefenderRealCombatEvent(t *testing.T) {
	for _, entering := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		defender := BindPlayer("defender")
		attacker := NewCreature("Attacker", "", 2, 2, WithAbility(WhenEvent(
			BecomesBlocked(This).WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(1)))))
		g.PutOnBattlefield(attacker, a.PlayerID())
		g.combat.AddAttacker(attacker.ID(), b.PlayerID())
		if entering {
			g.PutOnBattlefieldBlocking(NewCreature("Entering blocker", "", 1, 1), b.PlayerID(), attacker.ID())
			g.PutTriggersOnStack()
			g.ResolveStack()
		} else {
			blocker := NewCreature("Blocker", "", 1, 1)
			g.PutOnBattlefield(blocker, b.PlayerID())
			g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
		}
		if b.Life() != 21 || a.Life() != 20 {
			t.Fatalf("entering=%t: defender life = %d, attacker life = %d", entering, b.Life(), a.Life())
		}
	}
}

func TestBecomesBlockedDefenderPermanentController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	defender := BindPlayer("defender")
	attacker := NewCreature("Attacker", "", 2, 2, WithAbility(WhenEvent(
		BecomesBlocked(This).WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(1)))))
	g.PutOnBattlefield(attacker, a.PlayerID())
	walker := g.PutOnBattlefield(NewPlaneswalker("Walker", "", 3), b.PlayerID())
	g.combat.AddAttacker(attacker.ID(), walker.ID())
	g.PutOnBattlefieldBlocking(NewCreature("Blocker", "", 1, 1), b.PlayerID(), attacker.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if b.Life() != 21 || a.Life() != 20 {
		t.Fatalf("defending player life = %d, attacker player life = %d", b.Life(), a.Life())
	}
}

func TestBecomesBlockedDefenderStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	defender := BindPlayer("defender")
	attacker := NewCreature("Attacker", "", 2, 2, WithAbility(WhenEvent(
		BecomesBlocked(This).WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(1)))))
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	g.fireCreatureBecomesBlocked(attacker.ID())
	g.PutTriggersOnStack()
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	g.ResolveStack()
	if b.Life() != 22 || a.Life() != 20 {
		t.Fatalf("copied defender life = %d, attacker life = %d", b.Life(), a.Life())
	}
}

// --- from composable_blocker_group_test.go ---
func TestBlocksOrBecomesBlockedByGroup(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 3, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedBy(This).OfSubtype("Orc"), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	human := NewCreature("Human", "", 1, 2, WithSubTypes("Human"))
	orc1 := NewCreature("Orc 1", "", 1, 2, WithSubTypes("Orc"))
	orc2 := NewCreature("Orc 2", "", 1, 2, WithSubTypes("Orc"))
	for _, card := range []Card{human, orc1, orc2} {
		g.PutOnBattlefield(card, b.PlayerID())
	}
	g.combat.AddAttacker(watcher.ID(), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: human.ID(), AttackerID: watcher.ID()}, {BlockerID: orc1.ID(), AttackerID: watcher.ID()}, {BlockerID: orc2.ID(), AttackerID: watcher.ID()}})
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("two matching blockers produced life %d, want 21", got)
	}
	clone := g.Clone()
	g.FireEvent(GameEvent{Type: EvtBlockerGroup, SourceID: watcher.ID(), PlayerID: a.PlayerID(), RelatedIDs: []uuid.UUID{human.ID()}})
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("unmatched group produced life %d", got)
	}
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("clone life changed to %d", got)
	}
}

func TestBlocksOrBecomesBlockedByGroupValidation(t *testing.T) {
	for _, subtype := range []string{"", "Island"} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
			BlocksOrBecomesBlockedBy(This).OfSubtype(subtype), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "subtype") {
			t.Fatalf("subtype %q accepted: %v", subtype, err)
		}
	}
}

func TestBlocksOrBecomesBlockedByGroupBlockingAndTiming(t *testing.T) {
	a := NewBasePlayer("A")
	b := &fixedBlockPlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	first := NewCreature("First", "", 2, 2, WithSubTypes("Orc"))
	second := NewCreature("Second", "", 2, 2, WithSubTypes("Orc"))
	watcher := NewCreature("Watcher", "", 2, 5, WithKeyword(CanBlockAny), WithAbility(WhenEvent(
		BlocksOrBecomesBlockedBy(This).OfSubtype("Orc"), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.combat.AddAttacker(first.ID(), b.PlayerID())
	g.combat.AddAttacker(second.ID(), b.PlayerID())
	b.assignments = []BlockAssignment{{BlockerID: watcher.ID(), AttackerID: first.ID()}, {BlockerID: watcher.ID(), AttackerID: second.ID()}}
	g.doDeclareBlockers()
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.GetPlayer(b.PlayerID()).Life(); got != 21 {
		t.Fatalf("two attacked Orcs produced life %d, want 21", got)
	}
	g.FireEvent(GameEvent{Type: EvtBlockerGroup, SourceID: watcher.ID(), PlayerID: b.PlayerID(), Flag: true, RelatedIDs: []uuid.UUID{first.ID()}})
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.GetPlayer(b.PlayerID()).Life(); got != 22 {
		t.Fatalf("old trigger controller changed after source returned: %d", got)
	}
}

func TestBlocksOrBecomesBlockedByGroupEventTimeSubtype(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 3, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedBy(This).OfSubtype("Orc"), GainLifeFor(You, Fixed(1)))))
	orc := NewCreature("Orc", "", 1, 1, WithSubTypes("Orc"))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(orc, b.PlayerID())
	g.FireEvent(GameEvent{Type: EvtBlockerGroup, SourceID: watcher.ID(), PlayerID: a.PlayerID(), RelatedIDs: []uuid.UUID{orc.ID()}})
	g.PhaseOut(orc.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("later absence changed event-time match: %d", got)
	}
	g.FireEvent(GameEvent{Type: EvtBlockerGroup, SourceID: watcher.ID(), PlayerID: a.PlayerID(), RelatedIDs: []uuid.UUID{orc.ID()}})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("phased-out creature matched group: %d", got)
	}
}

func TestBlocksOrBecomesBlockedByWithoutSubtype(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 5, WithKeyword(CanBlockAny), WithAbility(WhenEvent(
		BlocksOrBecomesBlockedBy(This).WithoutSubtype("Wall"), GainLifeFor(You, Fixed(1)))))
	wall := NewCreature("Wall", "", 1, 2, WithSubTypes("Wall"))
	human := NewCreature("Human", "", 1, 2, WithSubTypes("Human"))
	orc := NewCreature("Orc", "", 1, 2, WithSubTypes("Orc"))
	g.PutOnBattlefield(watcher, a.PlayerID())
	for _, card := range []Card{wall, human, orc} {
		g.PutOnBattlefield(card, b.PlayerID())
	}
	g.combat.AddAttacker(watcher.ID(), b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: wall.ID(), AttackerID: watcher.ID()}, {BlockerID: human.ID(), AttackerID: watcher.ID()}, {BlockerID: orc.ID(), AttackerID: watcher.ID()}})
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("mixed group produced life %d, want 21", got)
	}
	clone := g.Clone()
	g.FireEvent(GameEvent{Type: EvtBlockerGroup, SourceID: watcher.ID(), RelatedIDs: []uuid.UUID{wall.ID()}})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("Wall-only group produced life %d", got)
	}
	g.PhaseOut(human.ID())
	g.FireEvent(GameEvent{Type: EvtBlockerGroup, SourceID: watcher.ID(), RelatedIDs: []uuid.UUID{wall.ID(), human.ID()}})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("phased-out creature matched group: %d", got)
	}
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("clone life changed to %d", got)
	}
	clone.FireEvent(GameEvent{Type: EvtBlockerGroup, SourceID: watcher.ID(), RelatedIDs: []uuid.UUID{orc.ID()}})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 22 {
		t.Fatalf("clone did not match its own group: %d", got)
	}
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("clone trigger changed original life to %d", got)
	}
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatalf("valid pattern rejected: %v", err)
	}
}

func TestBlocksOrBecomesBlockedByWithoutSubtypeBlockingAndEventTime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 5, WithKeyword(CanBlockAny), WithAbility(WhenEvent(
		BlocksOrBecomesBlockedBy(This).WithoutSubtype("Wall"), GainLifeFor(You, Fixed(1)))))
	human := NewCreature("Human", "", 1, 2, WithSubTypes("Human"))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(human, b.PlayerID())
	g.FireEvent(GameEvent{Type: EvtBlockerGroup, SourceID: watcher.ID(), Flag: true, RelatedIDs: []uuid.UUID{human.ID()}})
	human.AddSubType("Wall")
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("later subtype change altered pending trigger: %d", got)
	}
	g.FireEvent(GameEvent{Type: EvtBlockerGroup, SourceID: watcher.ID(), Flag: true, RelatedIDs: []uuid.UUID{human.ID()}})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("Wall matched blocking group: %d", got)
	}
}

func TestBlocksOrBecomesBlockedByWithoutSubtypeValidation(t *testing.T) {
	patterns := []blockerGroupPattern{
		BlocksOrBecomesBlockedBy(This).WithoutSubtype(""),
		BlocksOrBecomesBlockedBy(This).WithoutSubtype("Island"),
		BlocksOrBecomesBlockedBy(This).WithoutSubtype("Wall").WithoutSubtype("Orc"),
		BlocksOrBecomesBlockedBy(This).OfSubtype("Orc").WithoutSubtype("Wall"),
		BlocksOrBecomesBlockedBy(This).WithoutSubtype("Wall").OfSubtype("Orc"),
	}
	for _, pattern := range patterns {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("invalid pattern accepted: %+v", pattern)
		}
	}
}

// --- from composable_blockers_set_test.go ---
func TestBlockersOfCurrentCombatIncarnations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 3, 3), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 3, 3), a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 2), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 2), b.PlayerID())
	unrelated := g.PutOnBattlefield(NewCreature("Unrelated", "", 2, 2), b.PlayerID())
	g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
	g.combat.AddAttackerIncarnation(other.ID(), other.incarnationID, b.PlayerID())
	g.combat.AddBlockerIncarnation(first.ID(), first.incarnationID, attacker.ID())
	g.combat.AddBlockerIncarnation(second.ID(), second.incarnationID, attacker.ID())
	g.combat.AddBlockerIncarnation(unrelated.ID(), unrelated.incarnationID, other.ID())
	ctx := &EffectContext{Game: g, SourceID: attacker.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: attacker.incarnationID}
	count := CountPermanents(BlockersOf(This))
	if got := count.resolveValue(ctx); got != 2 {
		t.Fatalf("blocker count = %d, want 2", got)
	}
	clone := g.Clone()
	clone.RemoveFromCombat(first.ID())
	if got := count.resolveValue(&EffectContext{Game: clone, SourceID: attacker.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: attacker.incarnationID}); got != 1 {
		t.Fatalf("clone blocker count = %d, want 1", got)
	}
	clone.RemoveFromCombat(second.ID())
	if got := count.resolveValue(&EffectContext{Game: clone, SourceID: attacker.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: attacker.incarnationID}); got != 0 {
		t.Fatalf("blocked attacker with no blockers has %d blockers", got)
	}
	if got := count.resolveValue(ctx); got != 2 {
		t.Fatalf("original blocker count = %d, want 2", got)
	}
	phaseClone := g.Clone()
	phaseClone.PhaseOut(first.ID())
	if got := count.resolveValue(&EffectContext{Game: phaseClone, SourceID: attacker.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: attacker.incarnationID}); got != 1 {
		t.Fatalf("phased blocker count = %d, want 1", got)
	}
	phaseClone.PhaseOut(attacker.ID())
	if got := count.resolveValue(&EffectContext{Game: phaseClone, SourceID: attacker.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: attacker.incarnationID}); got != 0 {
		t.Fatalf("phased attacker count = %d, want 0", got)
	}
	g.RemoveFromBattlefield(second)
	g.PutOnBattlefield(second.Card, b.PlayerID())
	if got := count.resolveValue(ctx); got != 1 {
		t.Fatalf("returned blocker count = %d, want 1", got)
	}
	g.RemoveFromBattlefield(attacker)
	g.PutOnBattlefield(attacker.Card, a.PlayerID())
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("returned attacker count = %d, want 0", got)
	}
}

func TestBlockersOfDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("attacker", IsCreature)
	good := NewInstant("Good", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(BlockersOf(target))), WithTargets(target))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	if info := InspectDefinition(BlockersOf(target)); info.Role != "permanent-set" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
	for _, subject := range []PermanentRef{nil, target, From(BindPermanent("missing"))} {
		bad := NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(BlockersOf(subject))))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatalf("invalid blocker set accepted: %+v", InspectDefinition(BlockersOf(subject)))
		}
	}
}

// --- from composable_blocking_test.go ---
func TestBlockingProhibitionSurvivesAbilityRemoval(t *testing.T) {
	for _, removalFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "removal after", true: "removal before"}[removalFirst], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("attacker", IsCreature)
			source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(), Until(EndOfTurn, CantBeBlocked(target)), WithTargets(target)))), a.PlayerID())
			attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 2), a.PlayerID())
			blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 2, 2), b.PlayerID())
			remove := func() { g.AddContinuousEffect(RemoveAllAbilities(attacker.ID(), EndOfTurn)) }
			if removalFirst {
				remove()
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveTopOfStack()
			if !removalFirst {
				remove()
			}
			g.ApplyContinuousEffects()
			if CanBlock(blocker, attacker, g) {
				t.Fatal("ability removal erased the blocking prohibition")
			}
			g.RemoveFromBattlefield(source)
			g.ApplyContinuousEffects()
			if CanBlock(blocker, attacker, g) {
				t.Fatal("source departure erased the blocking prohibition")
			}
			clone := g.Clone()
			clone.effects.RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			if !CanBlock(clone.FindPermanent(blocker.ID()), clone.FindPermanent(attacker.ID()), clone) || CanBlock(blocker, attacker, g) {
				t.Fatal("expiry did not stay in the clone")
			}
		})
	}
}

func TestTypedBlockingProhibitionLifetime(t *testing.T) {
	for _, change := range []string{"none", "target returns", "phased", "phase cycle", "control", "power", "ability removal"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Attacker", "", 2, 2)
			attacker := g.PutOnBattlefield(card, a.PlayerID())
			blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 2, 2), b.PlayerID())
			body := Until(EndOfTurn, CantBeBlocked(This))
			if err := ApplyEffect(g, body, attacker.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "target returns":
				g.RemoveFromBattlefield(attacker)
				attacker = g.PutOnBattlefield(card, a.PlayerID())
			case "phased":
				attacker.PhasedOut = true
			case "phase cycle":
				attacker.PhasedOut = true
				g.ApplyContinuousEffects()
				attacker.PhasedOut = false
			case "control":
				if err := ApplyEffect(g, GainControl().Until(EndOfTurn), b.PlayerID(), b.PlayerID(), []uuid.UUID{attacker.ID()}); err != nil {
					t.Fatal(err)
				}
			case "power":
				g.AddContinuousEffect(TemporaryBoost(attacker.ID(), 4, 4))
			case "ability removal":
				g.AddContinuousEffect(RemoveAllAbilities(attacker.ID(), EndOfTurn))
			}
			g.ApplyContinuousEffects()
			wantBlock := change == "target returns" || change == "phased"
			if got := CanBlock(blocker, attacker, g); got != wantBlock {
				t.Fatalf("CanBlock = %v, want %v", got, wantBlock)
			}
			late := g.PutOnBattlefield(NewCreature("Late blocker", "", 2, 2), b.PlayerID())
			g.ApplyContinuousEffects()
			if CanBlock(late, attacker, g) != wantBlock {
				t.Fatal("restriction missed a later blocker")
			}
			clone := g.Clone()
			clone.effects.RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			if !CanBlock(clone.FindPermanentIncludingPhased(blocker.ID()), clone.FindPermanentIncludingPhased(attacker.ID()), clone) {
				t.Fatal("restriction did not expire")
			}
		})
	}
}

func TestBlockingProhibitionPreservesPrintedAbilityRemoval(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 2, WithKeyword(UnblockableKW)), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 2, 2), b.PlayerID())
	g.AddContinuousEffect(RemoveAllAbilities(attacker.ID(), EndOfTurn))
	g.ApplyContinuousEffects()
	if !CanBlock(blocker, attacker, g) {
		t.Fatal("printed ability survived ability removal")
	}
}

// --- from composable_blocks_creature_test.go ---
func TestBlocksCreaturePerPairAndCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	opponent := BindPermanent("attacker")
	blocker := NewCreature("Blocker", "", 2, 4, WithKeyword(CanBlockAny), WithAbility(WhenEvent(
		BlocksCreature(This).WithoutSubtype("Wall").WithOpponent(opponent),
		AddCountersTo(From(opponent), P1P1, Fixed(1)))))
	blocker.SetOwner(b.PlayerID())
	if err := ValidateCardDefinition(blocker.Name(), blocker); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(blocker, b.PlayerID())
	first := NewCreature("First", "", 2, 2)
	second := NewCreature("Second", "", 2, 2)
	wall := NewCreature("Wall", "", 2, 2, WithSubTypes("Wall"))
	for _, card := range []Card{first, second, wall} {
		g.PutOnBattlefield(card, a.PlayerID())
		g.combat.AddAttacker(card.ID(), b.PlayerID())
	}
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: first.ID()}, {BlockerID: blocker.ID(), AttackerID: second.ID()}, {BlockerID: blocker.ID(), AttackerID: wall.ID()}})
	if got := g.FindPermanent(first.ID()).Counters[P1P1]; got != 1 {
		t.Fatalf("first attacker counters = %d", got)
	}
	if got := g.FindPermanent(second.ID()).Counters[P1P1]; got != 1 {
		t.Fatalf("second attacker counters = %d", got)
	}
	if got := g.FindPermanent(wall.ID()).Counters[P1P1]; got != 0 {
		t.Fatalf("Wall counters = %d", got)
	}
}

func TestBlocksCreatureDeclarationValidation(t *testing.T) {
	opponent := BindPermanent("opponent")
	for _, pattern := range []EventPattern{
		BlocksCreature(This).WithoutSubtype(""),
		BlocksCreature(This).WithoutSubtype("Island"),
		BlocksCreature(This).OfSubtype("Wall").WithoutSubtype("Orc"),
		BlocksCreature(This).WithOpponent(nil),
		BlocksCreature(This).WithOpponent(opponent).WithOpponent(opponent),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid event pattern accepted")
		}
	}
	card := NewCreature("Missing", "", 1, 1, WithAbility(WhenEvent(BlocksCreature(This), AddCountersTo(From(opponent), P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("unbound opponent accepted")
	}
}

func TestBlocksCreatureOpponentIncarnationAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	opponent := BindPermanent("opponent")
	blocker := NewCreature("Blocker", "", 2, 4, WithAbility(WhenEvent(
		BlocksCreature(This).WithOpponent(opponent),
		AddCountersTo(From(opponent), P1P1, Fixed(1)))))
	blocker.SetOwner(b.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtDeclaredBlocker, SourceID: blocker.ID(), TargetID: attacker.ID(), PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if g.FindPermanent(attacker.ID()).Counters[P1P1] != 0 || clone.FindPermanent(attacker.ID()).Counters[P1P1] != 1 {
		t.Fatal("clone changed the original pending trigger")
	}
	g.RemoveFromBattlefield(g.FindPermanent(attacker.ID()))
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.ResolveStack()
	if g.FindPermanent(attacker.ID()).Counters[P1P1] != 0 {
		t.Fatal("pending trigger followed a returned attacker")
	}
	g.FireEvent(GameEvent{Type: EvtDeclaredBlocker, SourceID: blocker.ID(), TargetID: attacker.ID(), PlayerID: b.PlayerID()})
	g.PhaseOut(attacker.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanentIncludingPhased(attacker.ID()).Counters[P1P1] != 0 {
		t.Fatal("pending trigger affected a phased-out attacker")
	}
}

func TestBlocksCreatureSourcePhasingAtDetection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := NewCreature("Blocker", "", 2, 4, WithAbility(WhenEvent(
		BlocksCreature(This), GainLifeFor(You, Fixed(1)))))
	blocker.SetOwner(b.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.PhaseOut(blocker.ID())
	g.FireEvent(GameEvent{Type: EvtDeclaredBlocker, SourceID: blocker.ID(), TargetID: attacker.ID(), PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("phased-out source observed the block event")
	}
}

// --- from composable_combat_damage_batch_test.go ---
func TestCombatDamageBatchRecipientAndController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := BindPlayer("damaged player")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithPlayer(recipient),
		GainLifeFor(FromPlayer(recipient), Fixed(2)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtCombatDamageDealt, PlayerID: b.PlayerID(), TargetID: a.PlayerID(), Amount: 3})
	g.FireEvent(GameEvent{Type: EvtCombatDamageDealt, PlayerID: a.PlayerID(), TargetID: b.PlayerID(), Amount: 3})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 1 {
		t.Fatalf("triggers = %d, want 1", got)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 24 {
		t.Fatal("clone lost captured recipient or changed original")
	}
	g.ResolveStack()
	if b.Life() != 24 {
		t.Fatalf("recipient life = %d, want 24", b.Life())
	}
}

func TestCombatDamageBatchRealStepAndPrevention(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 2), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 2), a.PlayerID())
	g.combat.AddAttacker(first.ID(), b.PlayerID())
	g.combat.AddAttacker(second.ID(), b.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 2)
	g.ExecuteCombatDamage()
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("two sources gave %d triggers, want 1", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 18 {
		t.Fatalf("post-combat life = %d, %d", a.Life(), b.Life())
	}
	g.AddPreventionShield(b.PlayerID(), 4)
	g.ExecuteCombatDamage()
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("fully prevented combat damage triggered")
	}
}

func TestCombatDamageBatchSourceLifetimeAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	phase := g.PhaseOut(watcher.ID())
	g.FireEvent(GameEvent{Type: EvtCombatDamageDealt, PlayerID: a.PlayerID(), TargetID: b.PlayerID(), Amount: 2})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phase)
	g.FireEvent(GameEvent{Type: EvtCombatDamageDealt, PlayerID: a.PlayerID(), TargetID: b.PlayerID(), Amount: 2})
	g.PutTriggersOnStack()
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("pending trigger changed controller")
	}
	for _, pattern := range []EventPattern{
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithPlayer(nil),
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithPlayer(BindPlayer("")),
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithPlayer(BindPlayer("first")).WithPlayer(BindPlayer("second")),
	} {
		bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid event declaration accepted")
		}
	}
	missing := BindPlayer("missing")
	bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer(), GainLifeFor(FromPlayer(missing), Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer: %v", err)
	}
}

// --- from composable_combat_damage_event_test.go ---
func TestSourceCombatDamageToPlayerEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("damaged"), BindNumber("amount")
	source := NewCreature("Source", "", 2, 3, WithAbility(WhenEvent(
		DealsCombatDamageToPlayer(This, player).WithAmount(amount),
		GainLifeFor(FromPlayer(player), FromNumber(amount)))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: b.PlayerID(), Amount: 3})
	phased := g.PhaseOut(source.ID())
	g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: b.PlayerID(), Amount: 2, Flag: true})
	g.PhaseIn(phased)
	g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: b.PlayerID(), Amount: 2, Flag: true})
	g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: source.ID(), Amount: 1, Flag: true})
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	for _, obj := range g.StackObjects() {
		if g.CopyStackObjectDirect(obj, b.PlayerID(), false) == nil {
			t.Fatal("trigger copy failed")
		}
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 24 {
			t.Fatalf("damaged player life = %d", got)
		}
	}
}

func TestSourceCombatDamageToPlayerRealCombatAndPrevention(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p, n := BindPlayer("damaged"), BindNumber("amount")
	source := g.PutOnBattlefield(NewCreature("Source", "", 3, 3, WithAbility(WhenEvent(
		DealsCombatDamageToPlayer(This, p).WithAmount(n),
		GainLifeFor(FromPlayer(p), FromNumber(n))))), a.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 1)
	g.combat.AddAttacker(source.ID(), b.PlayerID())
	g.ExecuteCombatDamage()
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers after partial prevention = %d", g.StackSize())
	}
	g.ResolveStack()
	if got := b.Life(); got != 20 {
		t.Fatalf("damage and trigger result = %d", got)
	}
	g.AddPreventionShield(b.PlayerID(), 3)
	g.ExecuteCombatDamage()
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatalf("triggers after full prevention = %d", g.StackSize())
	}
}

func TestSourceCombatDamageToPlayerDeclaration(t *testing.T) {
	p, n := BindPlayer("player"), BindNumber("amount")
	for _, tc := range []struct {
		name    string
		pattern EventPattern
		want    string
	}{
		{"valid", DealsCombatDamageToPlayer(This, p).WithAmount(n), ""},
		{"nil player", DealsCombatDamageToPlayer(This, nil), "non-nil"},
		{"nil amount", DealsCombatDamageToPlayer(This, p).WithAmount(nil), "non-nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewCreature("Test", "", 1, 1, WithAbility(WhenEvent(tc.pattern, GainLifeFor(You, Fixed(1)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	if got := InspectDefinition(DealsCombatDamageToPlayer(This, p)); got.Role != "event-pattern" {
		t.Fatalf("inspection = %#v", got)
	}
}

// --- from composable_combat_damage_filter_test.go ---
func TestCombatDamageBatchWithoutSubtypeUsesDamageTimeSources(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithoutSubtype("Human"),
		GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	human := g.PutOnBattlefield(NewCreature("Human", "", 2, 2, WithSubTypes("Human")), a.PlayerID())
	wolf := g.PutOnBattlefield(NewCreature("Wolf", "", 2, 2, WithSubTypes("Wolf")), a.PlayerID())
	g.damage.resolvingCombatDamage = true
	g.DealDamageToPlayer(b, 2, human.ID())
	g.DealDamageToPlayer(b, 2, wolf.ID())
	g.damage.resolvingCombatDamage = false
	g.RemoveFromBattlefield(wolf)
	g.PutOnBattlefield(wolf.Card, a.PlayerID())
	g.damage.FlushCombatDamageAggregator(g)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 21 {
		t.Fatal("clone changed the original trigger")
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("life = %d, want 21", a.Life())
	}
}

func TestCombatDamageBatchWithoutSubtypeRequiresPositiveQualifyingDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithoutSubtype("Human"),
		GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	human := g.PutOnBattlefield(NewCreature("Human", "", 2, 2, WithSubTypes("Human")), a.PlayerID())
	wolf := g.PutOnBattlefield(NewCreature("Wolf", "", 2, 2, WithSubTypes("Wolf")), a.PlayerID())
	g.damage.resolvingCombatDamage = true
	g.AddPreventionShield(b.PlayerID(), 2)
	g.DealDamageToPlayer(b, 2, wolf.ID())
	g.DealDamageToPlayer(b, 2, human.ID())
	g.damage.resolvingCombatDamage = false
	g.damage.FlushCombatDamageAggregator(g)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatalf("triggers = %d, want 0", g.StackSize())
	}
	phase := g.PhaseOut(wolf.ID())
	g.damage.resolvingCombatDamage = true
	g.DealDamageToPlayer(b, 1, human.ID())
	g.damage.resolvingCombatDamage = false
	g.damage.FlushCombatDamageAggregator(g)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source or Human damage matched")
	}
	g.PhaseIn(phase)
}

func TestCombatDamageBatchWithoutSubtypeRegistration(t *testing.T) {
	for _, pattern := range []EventPattern{
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithoutSubtype(""),
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithoutSubtype("Island"),
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithoutSubtype("Human").WithoutSubtype("Elf"),
	} {
		card := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid subtype filter accepted")
		}
	}
}

func TestCombatDamageBatchSourceSnapshotsCloneIndependently(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithoutSubtype("Human"),
		GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	wolf := g.PutOnBattlefield(NewCreature("Wolf", "", 2, 2, WithSubTypes("Wolf")), a.PlayerID())
	g.damage.resolvingCombatDamage = true
	g.DealDamageToPlayer(b, 2, wolf.ID())
	g.damage.resolvingCombatDamage = false
	clone := g.Clone()
	clone.damage.combatDamageSourceStatesThisStep[a.PlayerID()][b.PlayerID()][0].card.AddSubType("Human")
	clone.damage.FlushCombatDamageAggregator(clone)
	clone.PutTriggersOnStack()
	if clone.StackSize() != 0 {
		t.Fatal("clone source snapshot did not change independently")
	}
	g.damage.FlushCombatDamageAggregator(g)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("clone changed original source snapshot")
	}
}

// --- from composable_combat_opponents_test.go ---
func TestCombatOpponentsSelectCurrentIncarnations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 2, 4)
	first := NewCreature("First", "", 1, 2)
	second := NewCreature("Second", "", 1, 2)
	for _, card := range []Card{source, first, second} {
		g.PutOnBattlefield(card, b.PlayerID())
	}
	g.combat.AddAttackerIncarnation(source.ID(), g.FindPermanent(source.ID()).incarnationID, b.PlayerID())
	g.combat.AddBlockerIncarnation(first.ID(), g.FindPermanent(first.ID()).incarnationID, source.ID())
	g.combat.AddBlockerIncarnation(second.ID(), g.FindPermanent(second.ID()).incarnationID, source.ID())
	set := BlockingOrBlockedBy(This)
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceIncarnation: g.FindPermanent(source.ID()).incarnationID, sourceCaptured: true}
	if got := len(set.resolvePermanents(ctx)); got != 2 {
		t.Fatalf("blockers = %d, want 2", got)
	}
	g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
	g.PutOnBattlefield(second, b.PlayerID())
	if got := len(set.resolvePermanents(ctx)); got != 1 {
		t.Fatalf("blockers after return = %d, want 1", got)
	}
	g.PhaseOut(first.ID())
	if got := len(set.resolvePermanents(ctx)); got != 0 {
		t.Fatalf("blockers after phasing = %d, want 0", got)
	}
	g.PhaseIn([]uuid.UUID{first.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	if got := len(set.resolvePermanents(ctx)); got != 0 {
		t.Fatalf("old source found %d opponents", got)
	}
}

func TestCombatOpponentsBlockingAndTriggerResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Watcher", "", 2, 4, WithKeyword(CanBlockAny), WithAbility(WhenEvent(
		BlocksOrBecomesBlockedBy(This), GainLifeFor(You, CountPermanents(BlockingOrBlockedBy(This))))))
	first := NewCreature("First", "", 2, 2)
	second := NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(source, b.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.combat.AddAttackerIncarnation(first.ID(), g.FindPermanent(first.ID()).incarnationID, b.PlayerID())
	g.combat.AddAttackerIncarnation(second.ID(), g.FindPermanent(second.ID()).incarnationID, b.PlayerID())
	g.combat.AddBlockerIncarnation(source.ID(), g.FindPermanent(source.ID()).incarnationID, first.ID())
	g.combat.AddBlockerIncarnation(source.ID(), g.FindPermanent(source.ID()).incarnationID, second.ID())
	g.FireEvent(GameEvent{Type: EvtBlockerGroup, SourceID: source.ID(), RelatedIDs: []uuid.UUID{first.ID(), second.ID()}, Flag: true})
	g.PutTriggersOnStack()
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 22 {
			t.Fatalf("life after one grouped trigger = %d, want 22", got)
		}
	}
}

func TestCombatOpponentsDeclaration(t *testing.T) {
	good := NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedBy(This), GainLifeFor(You, CountPermanents(BlockingOrBlockedBy(This))))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	bad := NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedBy(This), GainLifeFor(You, CountPermanents(BlockingOrBlockedBy(nil))))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("bad declaration: %v", err)
	}
}

// --- from composable_combat_pair_test.go ---
func TestCombatPairPatternBothDirectionsPerOpponent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	opponent := BindPermanent("opponent")
	watcher := NewCreature("Watcher", "", 2, 4, WithKeyword(CanBlockAny), WithAbility(WhenEvent(
		BlocksOrBecomesBlockedByCreature(This).WithoutSubtype("Wall").WithOpponent(opponent),
		AddCountersTo(From(opponent), P1P1, Fixed(1)))))
	watcher.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, b.PlayerID())
	first := NewCreature("First", "", 2, 2)
	second := NewCreature("Second", "", 2, 2)
	wall := NewCreature("Wall", "", 2, 2, WithSubTypes("Wall"))
	for _, card := range []Card{first, second, wall} {
		g.PutOnBattlefield(card, a.PlayerID())
		g.combat.AddAttacker(card.ID(), b.PlayerID())
	}
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: watcher.ID(), AttackerID: first.ID()}, {BlockerID: watcher.ID(), AttackerID: second.ID()}, {BlockerID: watcher.ID(), AttackerID: wall.ID()}})
	if got := g.FindPermanent(first.ID()).Counters[P1P1]; got != 1 {
		t.Fatalf("first attacker counters = %d", got)
	}
	if got := g.FindPermanent(second.ID()).Counters[P1P1]; got != 1 {
		t.Fatalf("second attacker counters = %d", got)
	}
	if got := g.FindPermanent(wall.ID()).Counters[P1P1]; got != 0 {
		t.Fatalf("Wall counters = %d", got)
	}

	g2 := NewGame(NewBasePlayer("C"), NewBasePlayer("D"))
	c, d := g2.players[0], g2.players[1]
	opponent2 := BindPermanent("blocker")
	attacker := NewCreature("Attacker", "", 2, 4, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedByCreature(This).WithoutSubtype("Wall").WithOpponent(opponent2),
		AddCountersTo(From(opponent2), P1P1, Fixed(1)))))
	g2.PutOnBattlefield(attacker, c.PlayerID())
	g2.combat.AddAttacker(attacker.ID(), d.PlayerID())
	blocker1 := NewCreature("Blocker 1", "", 2, 2)
	blocker2 := NewCreature("Blocker 2", "", 2, 2)
	wall2 := NewCreature("Wall", "", 2, 2, WithSubTypes("Wall"))
	for _, card := range []Card{blocker1, blocker2, wall2} {
		g2.PutOnBattlefield(card, d.PlayerID())
	}
	g2.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker1.ID(), AttackerID: attacker.ID()}, {BlockerID: blocker2.ID(), AttackerID: attacker.ID()}, {BlockerID: wall2.ID(), AttackerID: attacker.ID()}})
	for _, card := range []Card{blocker1, blocker2} {
		if got := g2.FindPermanent(card.ID()).Counters[P1P1]; got != 1 {
			t.Fatalf("%s counters = %d", card.Name(), got)
		}
	}
	if got := g2.FindPermanent(wall2.ID()).Counters[P1P1]; got != 0 {
		t.Fatalf("Wall blocker counters = %d", got)
	}
}

func TestCombatPairPatternDeclarationValidation(t *testing.T) {
	opponent := BindPermanent("opponent")
	for _, pattern := range []EventPattern{
		BlocksOrBecomesBlockedByCreature(This).OfSubtype(""),
		BlocksOrBecomesBlockedByCreature(This).WithoutSubtype("Island"),
		BlocksOrBecomesBlockedByCreature(This).OfSubtype("Wall").WithoutSubtype("Orc"),
		BlocksOrBecomesBlockedByCreature(This).WithOpponent(nil),
		BlocksOrBecomesBlockedByCreature(This).WithOpponent(opponent).WithOpponent(opponent),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid combat-pair pattern accepted")
		}
	}
	card := NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(BlocksOrBecomesBlockedByCreature(This), AddCountersTo(From(opponent), P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("unbound opponent accepted")
	}
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(BlocksOrBecomesBlockedByCreature(This).WithOpponent(opponent), AddCountersTo(From(opponent), P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
}

func TestCombatPairPatternCaptureIdentityAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	opponent := BindPermanent("opponent")
	watcher := NewCreature("Watcher", "", 2, 4, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedByCreature(This).WithOpponent(opponent),
		AddCountersTo(From(opponent), P1P1, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	blocker := NewCreature("Blocker", "", 2, 2)
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.FireEvent(GameEvent{Type: EvtCombatPair, SourceID: watcher.ID(), TargetID: blocker.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if g.FindPermanent(blocker.ID()).Counters[P1P1] != 0 || clone.FindPermanent(blocker.ID()).Counters[P1P1] != 1 {
		t.Fatal("clone changed original pending capture")
	}
	g.RemoveFromBattlefield(g.FindPermanent(blocker.ID()))
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.ResolveStack()
	if g.FindPermanent(blocker.ID()).Counters[P1P1] != 0 {
		t.Fatal("capture followed a returned opponent")
	}
	g.FireEvent(GameEvent{Type: EvtCombatPair, SourceID: watcher.ID(), TargetID: blocker.ID(), PlayerID: a.PlayerID()})
	g.PhaseOut(blocker.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanentIncludingPhased(blocker.ID()).Counters[P1P1] != 0 {
		t.Fatal("capture affected a phased-out opponent")
	}
}

func TestCombatPairPatternEntryAndAbsentSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 4, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedByCreature(This), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.combat.AddAttacker(watcher.ID(), b.PlayerID())
	entrant := NewCreature("Entrant", "", 2, 2)
	g.PutOnBattlefieldBlocking(entrant, b.PlayerID(), watcher.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("entry pair life = %d, %d", a.Life(), b.Life())
	}
	g.PhaseOut(watcher.ID())
	g.FireEvent(GameEvent{Type: EvtCombatPair, SourceID: watcher.ID(), TargetID: entrant.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("phased-out source observed a pair")
	}
}

func TestCombatPairPatternEventTimeFilterAndDepartedSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	opponent := BindPermanent("opponent")
	watcher := NewCreature("Watcher", "", 2, 4, WithAbility(WhenEvent(
		BlocksOrBecomesBlockedByCreature(This).WithoutSubtype("Wall").WithOpponent(opponent),
		AddCountersTo(From(opponent), P1P1, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	blocker := NewCreature("Blocker", "", 2, 2)
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.FireEvent(GameEvent{Type: EvtCombatPair, SourceID: watcher.ID(), TargetID: blocker.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	blocker.AddSubType("Wall")
	g.ResolveStack()
	if got := g.FindPermanent(blocker.ID()).Counters[P1P1]; got != 1 {
		t.Fatalf("queued trigger lost event-time match: %d", got)
	}
}

// --- from composable_combat_prevention_test.go ---
func TestCombatPreventionIncarnation(t *testing.T) {
	for _, change := range []string{"none", "return", "phase cycle", "control", "expire"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Creature", "", 3, 3)
			g.PutOnBattlefield(card, a.PlayerID())
			g.PreventCombatDamageToAndBy(card.ID())
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(card.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "expire":
				g.effects.ClearReplacementsEndOfTurn()
			}
			clone := g.Clone()
			clone.damage.SetResolvingCombatDamage(true)
			clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 2, card.ID())
			clone.DealDamageToPermanent(clone.FindPermanent(card.ID()), 1, uuid.Nil)
			prevented := change == "none" || change == "phase cycle" || change == "control"
			wantLife, wantDamage := 18, 1
			if prevented {
				wantLife, wantDamage = 20, 0
			}
			if clone.GetPlayer(b.PlayerID()).Life() != wantLife || clone.FindPermanent(card.ID()).Damage != wantDamage {
				t.Fatal("prevention did not preserve lifetime")
			}
			if b.Life() != 20 || g.FindPermanent(card.ID()).Damage != 0 {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestTypedCombatPreventionSubject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 3, 3)
	other := NewCreature("Other", "", 3, 3)
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	body := PreventCombatDamageToAndBy(This)
	if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), []uuid.UUID{other.ID()}); err != nil {
		t.Fatal(err)
	}
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayer(b, 2, source.ID())
	g.DealDamageToPermanent(g.FindPermanent(source.ID()), 2, other.ID())
	g.DealDamageToPlayer(a, 2, other.ID())
	if b.Life() != 20 || g.FindPermanent(source.ID()).Damage != 0 || a.Life() != 18 {
		t.Fatal("wrong prevention subject")
	}
}

func TestTypedControlledTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	own := NewCreature("Own", "", 2, 2)
	other := NewCreature("Other", "", 2, 2)
	g.PutOnBattlefield(own, a.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	target := DeclarePermanentTarget("creature", IsCreature, TargetYouControl())
	ability := Spell(Sequence(UntapPermanent(target), PreventCombatDamageToAndBy(target)), WithTargets(target))
	card := NewInstant("Spell", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	possible := ability.Targets()[0].Possible(a.PlayerID(), card, g)
	if len(possible) != 1 || possible[0] != own.ID() {
		t.Fatal("controlled target included opponent")
	}
	g.TapPermanent(g.FindPermanent(own.ID()))
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{own.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: own.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.ResolveStack()
	if !g.FindPermanent(own.ID()).Tapped || len(g.effects.replacements) != 0 {
		t.Fatal("target stayed legal after control change")
	}
}

// --- from composable_combat_source_prevention_test.go ---
func TestCombatSourcePreventionScopeAndLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocked := g.PutOnBattlefield(NewCreature("Blocked", "", 3, 3), a.PlayerID())
	unrelated := g.PutOnBattlefield(NewCreature("Unrelated", "", 3, 3), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 3, 3), b.PlayerID())
	g.combat.AddAttackerIncarnation(blocked.ID(), blocked.incarnationID, b.PlayerID())
	g.combat.AddAttackerIncarnation(unrelated.ID(), unrelated.incarnationID, b.PlayerID())
	g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, blocked.ID())
	ctx := &EffectContext{Game: g, SourceID: blocker.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: blocker.incarnationID}
	if err := PreventCombatDamageBy(This).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := PreventCombatDamageByEach(BlockedAttackersOf(This)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.damage.SetResolvingCombatDamage(true)
	clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 2, blocked.ID())
	clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 2, blocker.ID())
	clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 2, unrelated.ID())
	clone.DealDamageToPermanent(clone.FindPermanent(blocker.ID()), 1, blocked.ID())
	clone.DealDamageToPermanent(clone.FindPermanent(blocker.ID()), 1, unrelated.ID())
	clone.damage.SetResolvingCombatDamage(false)
	clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 1, blocked.ID())
	if got := clone.GetPlayer(b.PlayerID()).Life(); got != 17 {
		t.Fatalf("defender life = %d, want 17", got)
	}
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 20 {
		t.Fatalf("attacker life = %d, want 20", got)
	}
	if got := clone.FindPermanent(blocker.ID()).Damage; got != 1 {
		t.Fatalf("blocker damage = %d, want 1", got)
	}
	if b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.effects.RemoveEndOfCombat()
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayer(b, 1, blocked.ID())
	if b.Life() != 19 {
		t.Fatal("prevention survived combat")
	}
}

func TestCombatSourcePreventionIncarnation(t *testing.T) {
	for _, change := range []string{"return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("Source", "", 2, 2), a.PlayerID())
			if err := PreventCombatDamageBy(This).Apply(&EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, a.PlayerID())
			case "phase":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			g.damage.SetResolvingCombatDamage(true)
			g.DealDamageToPlayer(b, 1, p.ID())
			want := 20
			if change == "return" {
				want = 19
			}
			if b.Life() != want {
				t.Fatalf("life = %d, want %d", b.Life(), want)
			}
		})
	}
}

func TestCombatSourcePreventionDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	valid := NewInstant("Valid", "", Spell(Sequence(PreventCombatDamageBy(target), PreventCombatDamageByEach(BlockedAttackersOf(target))), WithTargets(target)))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{PreventCombatDamageBy(nil), PreventCombatDamageByEach(nil), PreventCombatDamageBy(target), PreventCombatDamageByEach(BlockedAttackersOf(target))} {
		bad := NewInstant("Bad", "", Spell(effect))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid prevention accepted")
		}
	}
	if info := InspectDefinition(PreventCombatDamageBy(target)); info.Role != "install-combat-source-prevention" {
		t.Fatalf("inspection = %+v", info)
	}
}

func TestCombatSourcePreventionCapturedDeparture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 2), a.PlayerID())
	source := g.CaptureDamageSource(p.ID())
	if err := PreventCombatDamageBy(This).Apply(&EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(p.Card, a.PlayerID())
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayerFromSource(b, 2, source)
	if b.Life() != 20 {
		t.Fatal("captured source damage was not prevented")
	}
	g.DealDamageToPlayer(b, 1, p.ID())
	if b.Life() != 19 {
		t.Fatal("new incarnation inherited prevention")
	}
}

func TestCombatSourcePreventionTargetReturn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Target", "", 2, 2), a.PlayerID())
	target := DeclarePermanentTarget("creature", IsCreature)
	card := NewInstant("Prevention", "", Spell(PreventCombatDamageBy(target), WithTargets(target)))
	ability := card.Abilities()[0].(*SpellAbility)
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(p.Card, a.PlayerID())
	g.ResolveStack()
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayer(b, 1, p.ID())
	if b.Life() != 19 {
		t.Fatal("returned target received prevention")
	}
}

// --- from composable_combat_state_branch_test.go ---
func TestCombatStateBranchCurrentIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := NewCreature("Attacker", "", 2, 2)
	blocker := NewCreature("Blocker", "", 2, 2)
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.combat.AddAttackerIncarnation(attacker.ID(), g.FindPermanent(attacker.ID()).incarnationID, b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
	attack := IfPermanentAttacking(This, GainLifeFor(You, Fixed(2)))
	block := IfPermanentBlocking(This, GainLifeFor(You, Fixed(3)))
	if err := ApplyEffect(g, attack, attacker.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, block, blocker.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || b.Life() != 23 {
		t.Fatalf("combat branches: %d, %d", a.Life(), b.Life())
	}
	clone := g.Clone()
	phased := g.PhaseOut(attacker.ID())
	if err := ApplyEffect(g, attack, attacker.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(clone, attack, attacker.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 24 {
		t.Fatal("phasing or clone changed combat membership")
	}
	g.PhaseIn(phased)
	g.RemoveFromBattlefield(g.FindPermanent(attacker.ID()))
	g.PutOnBattlefield(attacker, a.PlayerID())
	if err := ApplyEffect(g, attack, attacker.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 {
		t.Fatal("returned object inherited combat membership")
	}
}

func TestCombatStateBranchTargetScope(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	valid := NewInstant("Valid", "", Spell(Sequence(
		IfPermanentAttacking(target, GainLifeFor(You, Fixed(1))),
		IfPermanentBlocking(target, GainLifeFor(You, Fixed(1))),
	), WithTargets(target)))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		IfPermanentAttacking(nil, GainLifeFor(You, Fixed(1))),
		IfPermanentBlocking(target, GainLifeFor(You, Fixed(1))),
		IfPermanentAttacking(target, nil),
	} {
		bad := NewInstant("Invalid", "", Spell(effect))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid combat branch accepted")
		}
	}

	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Target", "", 2, 2)
	g.PutOnBattlefield(c, b.PlayerID())
	g.combat.AddAttackerIncarnation(c.ID(), g.FindPermanent(c.ID()).incarnationID, a.PlayerID())
	ability := valid.Abilities()[0].(*SpellAbility)
	obj := &StackObject{ID: uuid.New(), SourceID: valid.ID(), Card: valid, Controller: a.PlayerID(), Targets: []uuid.UUID{c.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("target combat branch life = %d", a.Life())
	}
	obj = &StackObject{ID: uuid.New(), SourceID: valid.ID(), Card: valid, Controller: a.PlayerID(), Targets: []uuid.UUID{c.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 21 {
		t.Fatal("cloned target branch changed the original frame")
	}
	g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
	g.PutOnBattlefield(c, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("stale target matched a returned object")
	}
}

// --- from composable_defending_control_attack_test.go ---
func TestDefendingPlayerControlAttackRestriction(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetActivePlayerIndex(0)
	card := NewCreature("Attacker", "", 2, 2, WithAbility(Static(CanAttackOnlyIfDefendingPlayerControls(HasSubType("Island")))))
	attacker := g.PutOnBattlefield(card, a.PlayerID())
	land := NewLand("Island", WithSubTypes("Island"))
	land.SetOwner(a.PlayerID())
	island := g.PutOnBattlefield(land, a.PlayerID())
	check := func(want bool) {
		t.Helper()
		g.effects.Apply(g)
		if got := g.FindPermanent(attacker.ID()).HasAttr(AttrCanAttack); got != want {
			t.Fatalf("can attack = %v, want %v", got, want)
		}
	}
	check(false)
	g.RemoveFromBattlefield(island)
	island = g.PutOnBattlefield(land, b.PlayerID())
	check(true)
	g.PhaseOut(island.ID())
	check(false)
	g.PhaseIn([]uuid.UUID{island.ID()})
	check(true)
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(island.ID()))
	clone.effects.Apply(clone)
	if clone.FindPermanent(attacker.ID()).HasAttr(AttrCanAttack) {
		t.Fatal("clone kept permission after defender lost Island")
	}
	check(true)
	g.RemoveFromBattlefield(g.FindPermanent(island.ID()))
	check(false)
	g.PutOnBattlefield(land, b.PlayerID())
	check(true)
}

func TestDefendingPlayerControlAttackRestrictionDeclaration(t *testing.T) {
	rule := CanAttackOnlyIfDefendingPlayerControls(HasSubType("Island"))
	card := NewCreature("Attacker", "", 2, 2, WithAbility(Static(rule)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if meta := InspectDefinition(rule); meta.Role == "" || meta.Text == "" {
		t.Fatalf("inspection = %+v", meta)
	}
}

// --- from composable_each_attacker_test.go ---
func TestEachControlledAttackerCapturesIndependentEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := BindPermanent("attacker")
	defender := BindPlayer("defender")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		EachCreatureYouControlAttacks().WithAttacker(attacker).WithDefendingPlayer(defender),
		Sequence(AddCountersTo(From(attacker), P1P1, Fixed(1)), GainLifeFor(FromPlayer(defender), Fixed(1))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(watcher.Abilities()[0])
	if meta.Role != "event-trigger" || len(meta.Children) == 0 || len(meta.Children[0].Children) != 2 {
		t.Fatalf("attack capture inspection = %+v", meta)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	first := NewCreature("First", "", 2, 2)
	second := NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.FindPermanent(first.ID()).RevokeBaseAttr(AttrSummonSick)
	g.FindPermanent(second.ID()).RevokeBaseAttr(AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("got %d per-attacker triggers", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 || g.FindPermanent(first.ID()).Counters[P1P1] != 0 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if b.Life() != 22 || g.FindPermanent(first.ID()).Counters[P1P1] != 1 || g.FindPermanent(second.ID()).Counters[P1P1] != 1 {
		t.Fatal("per-attacker event values were mixed")
	}
}

func TestEachControlledAttackerIdentityAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := BindPermanent("attacker")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		EachCreatureYouControlAttacks().WithAttacker(attacker), AddCountersTo(From(attacker), P1P1, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	creature := NewCreature("Creature", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	event := GameEvent{Type: EvtDeclaredAttacker, SourceID: creature.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()}
	g.FireEvent(event)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("missing attack trigger")
	}
	g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
	g.PutOnBattlefield(creature, a.PlayerID())
	g.ResolveStack()
	if g.FindPermanent(creature.ID()).Counters[P1P1] != 0 {
		t.Fatal("captured attacker followed a returned card")
	}
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: creature.ID(), PlayerID: b.PlayerID(), TargetID: a.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtEntersAttacking, SourceID: creature.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("wrong controller or nondeclaration triggered")
	}
	phase := g.PhaseOut(creature.ID())
	g.FireEvent(event)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased attacker triggered")
	}
	g.PhaseIn(phase)
}

func TestEachControlledAttackerDeclaration(t *testing.T) {
	attacker := BindPermanent("attacker")
	defender := BindPlayer("defender")
	for _, pattern := range []EventPattern{
		EachCreatureYouControlAttacks().WithAttacker(nil),
		EachCreatureYouControlAttacks().WithDefendingPlayer(nil),
		EachCreatureYouControlAttacks().WithAttacker(attacker).WithAttacker(attacker),
		EachCreatureYouControlAttacks().WithDefendingPlayer(defender).WithDefendingPlayer(defender),
	} {
		card := NewEnchantment("Invalid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid event capture")
		}
	}
	card := NewEnchantment("Missing", "", WithAbility(WhenEvent(EachCreatureYouControlAttacks(), AddCountersTo(From(attacker), P1P1, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("scope check = %v", err)
	}
}

func TestEachControlledAttackerSourceLifecycleAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	defender := BindPlayer("defender")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		EachCreatureYouControlAttacks().WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(attacker, a.PlayerID())
	event := GameEvent{Type: EvtDeclaredAttacker, SourceID: attacker.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()}
	phase := g.PhaseOut(watcher.ID())
	g.FireEvent(event)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased watcher triggered")
	}
	g.PhaseIn(phase)
	g.FireEvent(event)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("present watcher did not trigger")
	}
	original := g.StackPeek()
	if copied := g.CopyStackObjectDirect(original, b.PlayerID(), false); copied == nil {
		t.Fatal("trigger copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("clone lost or shared event capture")
	}
	g.ResolveStack()
	if b.Life() != 22 {
		t.Fatal("pending source return or copy changed defender")
	}
}

// --- from composable_permanent_attacking_test.go ---
func TestPermanentAttackingStateAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(card, a.PlayerID())
	old := g.FindPermanent(card.ID()).incarnationID
	ctx := StateContext{SourceID: card.ID(), SourceIncarnation: old}
	condition := PermanentIsAttacking(This)
	if condition.checkState(g, ctx) {
		t.Fatal("matched before combat")
	}
	g.combat.AddAttackerIncarnation(card.ID(), old, b.PlayerID())
	if !condition.checkState(g, ctx) {
		t.Fatal("current attacker did not match")
	}
	clone := g.Clone()
	clone.RemoveFromCombat(card.ID())
	if condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("clone shared combat state")
	}
	g.FindPermanent(card.ID()).PhasedOut = true
	if condition.checkState(g, ctx) {
		t.Fatal("phased attacker matched")
	}
	g.FindPermanentIncludingPhased(card.ID()).PhasedOut = false
	if !condition.checkState(g, ctx) {
		t.Fatal("phased-in attacker lost identity")
	}
	phase := g.PhaseOut(card.ID())
	g.PhaseIn(phase)
	if condition.checkState(g, ctx) {
		t.Fatal("phasing kept attacker in combat")
	}
	g.combat.AddAttackerIncarnation(card.ID(), old, b.PlayerID())
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	if condition.checkState(g, ctx) {
		t.Fatal("absent attacker matched")
	}
	g.PutOnBattlefield(card, a.PlayerID())
	if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: card.ID()}) || condition.checkState(g, StateContext{SourceID: uuid.Nil}) {
		t.Fatal("returned or missing attacker matched")
	}
}

func TestPermanentAttackingAttachedAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	aura := NewAura("Watcher", "", WithAbility(Static(While(PermanentIsAttacking(AttachedPermanent), GrantKeywordModification(AttachedPermanent, Shroud)))))
	host := NewCreature("Host", "", 2, 2)
	g.PutOnBattlefield(host, a.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	ctx := StateContext{SourceID: aura.ID(), SourceIncarnation: g.FindPermanent(aura.ID()).incarnationID}
	condition := PermanentIsAttacking(AttachedPermanent)
	g.combat.AddAttackerIncarnation(host.ID(), g.FindPermanent(host.ID()).incarnationID, b.PlayerID())
	if condition.checkState(g, ctx) {
		t.Fatal("unattached source matched")
	}
	g.Attach(aura.ID(), host.ID())
	if !condition.checkState(g, ctx) {
		t.Fatal("attacking attachment did not match")
	}
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).HasKeyword(Shroud) {
		t.Fatal("static rule did not see attacking host")
	}
	g.RemoveFromCombat(host.ID())
	g.ApplyContinuousEffects()
	if condition.checkState(g, ctx) || g.FindPermanent(host.ID()).HasKeyword(Shroud) {
		t.Fatal("removed attacker still matched")
	}
	g.combat.AddAttackerIncarnation(host.ID(), g.FindPermanent(host.ID()).incarnationID, b.PlayerID())
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.PutOnBattlefield(host, a.PlayerID())
	if condition.checkState(g, ctx) {
		t.Fatal("returned host inherited attack or attachment identity")
	}
	if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(condition); got.Role != "state-condition" || got.Text != "attached permanent is attacking" {
		t.Fatalf("inspection = %+v", got)
	}
	target := DeclarePermanentTarget("target", IsCreature)
	bad := NewAura("Bad", "", WithAbility(Static(While(PermanentIsAttacking(target), GrantKeywordModification(AttachedPermanent, Shroud)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "state") {
		t.Fatalf("invalid state reference accepted: %v", err)
	}
	spell := NewInstant("Bad spell", "", WithEffect(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil {
		t.Fatal("nonbattlefield source accepted")
	}
}

func TestPermanentAttackingEntryAndNormalDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entered := NewCreature("Entered attacker", "", 2, 2)
	g.PutOnBattlefieldAttacking(entered, a.PlayerID(), b.PlayerID())
	entry := StateContext{SourceID: entered.ID(), SourceIncarnation: g.FindPermanent(entered.ID()).incarnationID}
	if !PermanentIsAttacking(This).checkState(g, entry) {
		t.Fatal("creature entering attacking did not match current state")
	}
	declared := NewCreature("Declared attacker", "", 2, 2)
	g.PutOnBattlefield(declared, a.PlayerID())
	g.FindPermanent(declared.ID()).RevokeBaseAttr(AttrSummonSick)
	declaration := StateContext{SourceID: declared.ID(), SourceIncarnation: g.FindPermanent(declared.ID()).incarnationID}
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{declared.ID()})
	if !PermanentIsAttacking(This).checkState(g, declaration) {
		t.Fatal("declared attacker did not match current state")
	}
}

// --- from composable_permanent_blocking_test.go ---
func TestPermanentBlockingStateIdentityAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocker := NewCreature("Blocker", "", 2, 2)
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(blocker, a.PlayerID())
	g.PutOnBattlefield(attacker, b.PlayerID())
	old := g.FindPermanent(blocker.ID()).incarnationID
	ctx := StateContext{SourceID: blocker.ID(), SourceIncarnation: old}
	condition := PermanentIsBlocking(This)
	if condition.checkState(g, ctx) {
		t.Fatal("matched before block")
	}
	g.combat.AddAttackerIncarnation(attacker.ID(), g.FindPermanent(attacker.ID()).incarnationID, a.PlayerID())
	g.combat.AddBlockerIncarnation(blocker.ID(), old, attacker.ID())
	if !condition.checkState(g, ctx) || !condition.checkState(g.Clone(), ctx) {
		t.Fatal("current blocker did not match")
	}
	clone := g.Clone()
	clone.RemoveFromCombat(blocker.ID())
	if condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("clone shared combat state")
	}
	g.FindPermanent(blocker.ID()).PhasedOut = true
	if condition.checkState(g, ctx) {
		t.Fatal("phased blocker matched")
	}
	g.FindPermanentIncludingPhased(blocker.ID()).PhasedOut = false
	phase := g.PhaseOut(blocker.ID())
	g.PhaseIn(phase)
	if condition.checkState(g, ctx) {
		t.Fatal("phased blocker stayed in combat")
	}
	g.combat.AddBlockerIncarnation(blocker.ID(), old, attacker.ID())
	g.RemoveFromBattlefield(g.FindPermanent(blocker.ID()))
	if condition.checkState(g, ctx) {
		t.Fatal("absent blocker matched")
	}
	g.PutOnBattlefield(blocker, a.PlayerID())
	if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: blocker.ID()}) || condition.checkState(g, StateContext{SourceID: uuid.Nil}) {
		t.Fatal("returned or missing blocker matched")
	}
}

func TestPermanentBlockingAttachedAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	aura := NewAura("Watcher", "", WithAbility(Static(While(PermanentIsBlocking(AttachedPermanent), GrantKeywordModification(AttachedPermanent, Shroud)))))
	host := NewCreature("Host", "", 2, 2)
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(host, a.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	g.PutOnBattlefield(attacker, b.PlayerID())
	ctx := StateContext{SourceID: aura.ID(), SourceIncarnation: g.FindPermanent(aura.ID()).incarnationID}
	condition := PermanentIsBlocking(AttachedPermanent)
	g.combat.AddAttackerIncarnation(attacker.ID(), g.FindPermanent(attacker.ID()).incarnationID, a.PlayerID())
	g.combat.AddBlockerIncarnation(host.ID(), g.FindPermanent(host.ID()).incarnationID, attacker.ID())
	if condition.checkState(g, ctx) {
		t.Fatal("unattached source matched")
	}
	g.Attach(aura.ID(), host.ID())
	if !condition.checkState(g, ctx) {
		t.Fatal("blocking host did not match")
	}
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).HasKeyword(Shroud) {
		t.Fatal("static rule did not see blocking host")
	}
	g.RemoveFromCombat(host.ID())
	g.ApplyContinuousEffects()
	if condition.checkState(g, ctx) || g.FindPermanent(host.ID()).HasKeyword(Shroud) {
		t.Fatal("removed host still matched")
	}
	g.combat.AddBlockerIncarnation(host.ID(), g.FindPermanent(host.ID()).incarnationID, attacker.ID())
	g.FindPermanent(aura.ID()).PhasedOut = true
	if condition.checkState(g, ctx) {
		t.Fatal("phased attachment matched")
	}
	g.FindPermanentIncludingPhased(aura.ID()).PhasedOut = false
	g.FindPermanent(host.ID()).PhasedOut = true
	if condition.checkState(g, ctx) {
		t.Fatal("phased host matched")
	}
	g.FindPermanentIncludingPhased(host.ID()).PhasedOut = false
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.PutOnBattlefield(host, a.PlayerID())
	if condition.checkState(g, ctx) {
		t.Fatal("returned host inherited attachment or block")
	}
	if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(condition); got.Role != "state-condition" || got.Text != "attached permanent is blocking" {
		t.Fatalf("inspection = %+v", got)
	}
	target := DeclarePermanentTarget("target", IsCreature)
	bad := NewAura("Bad", "", WithAbility(Static(While(PermanentIsBlocking(target), GrantKeywordModification(AttachedPermanent, Shroud)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "state") {
		t.Fatalf("invalid state reference accepted: %v", err)
	}
	spell := NewInstant("Bad spell", "", WithEffect(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil {
		t.Fatal("nonbattlefield source accepted")
	}
}

func TestPermanentBlockingNormalDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := NewCreature("Attacker", "", 2, 2)
	blocker := NewCreature("Blocker", "", 2, 2)
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.FindPermanent(attacker.ID()).RevokeBaseAttr(AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
	ctx := StateContext{SourceID: blocker.ID(), SourceIncarnation: g.FindPermanent(blocker.ID()).incarnationID}
	g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
	if !PermanentIsBlocking(This).checkState(g, ctx) {
		t.Fatal("declared blocker did not match")
	}
	entered := NewCreature("Entering blocker", "", 1, 1)
	g.PutOnBattlefieldBlocking(entered, b.PlayerID(), attacker.ID())
	entry := StateContext{SourceID: entered.ID(), SourceIncarnation: g.FindPermanent(entered.ID()).incarnationID}
	if !PermanentIsBlocking(This).checkState(g, entry) {
		t.Fatal("creature entering blocking did not match current state")
	}
}

// --- from composable_source_attacked_or_blocked_test.go ---
func TestSourceAttackedOrBlockedThisTurnIdentity(t *testing.T) {
	for _, eventType := range []EventType{EvtDeclaredAttacker, EvtDeclaredBlocker} {
		t.Run(eventType.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("History", "", 2, 2)
			g.PutOnBattlefield(card, a.PlayerID())
			p := g.FindPermanent(card.ID())
			ctx := StateContext{SourceID: card.ID(), ControllerID: a.PlayerID(), SourceIncarnation: p.incarnationID}
			condition := SourceAttackedOrBlockedThisTurnCondition()
			if condition.checkState(g, ctx) {
				t.Fatal("history exists before declaration")
			}
			g.FireEvent(GameEvent{Type: eventType, SourceID: card.ID(), PlayerID: a.PlayerID(), Flag: true})
			if !condition.checkState(g, ctx) {
				t.Fatal("declaration was not recorded")
			}
			clone := g.Clone()
			if !condition.checkState(clone, ctx) {
				t.Fatal("clone lost history")
			}
			g.FindPermanent(card.ID()).PhasedOut = true
			if condition.checkState(g, ctx) {
				t.Fatal("phased source matched")
			}
			g.FindPermanentIncludingPhased(card.ID()).PhasedOut = false
			g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
			g.PutOnBattlefield(card, a.PlayerID())
			if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: card.ID(), SourceIncarnation: g.FindPermanent(card.ID()).incarnationID}) {
				t.Fatal("new incarnation inherited history")
			}
			if !condition.checkState(clone, ctx) {
				t.Fatal("clone changed with original")
			}
			clone.trackers.Turn.ResetForNewTurn(a.PlayerID())
			if condition.checkState(clone, ctx) {
				t.Fatal("history survived turn reset")
			}
		})
	}
}

func TestSourceAttackedOrBlockedThisTurnDeclaration(t *testing.T) {
	condition := SourceAttackedOrBlockedThisTurnCondition()
	card := NewCreature("Watcher", "", 2, 2, WithAbility(At(EachEndStep, IfState(condition, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(condition)
	if meta.Role != "state-condition" || meta.Text != "source attacked or blocked this turn" {
		t.Fatalf("metadata = %+v", meta)
	}
	spell := NewInstant("Invalid", "", WithEffect(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell validation = %v", err)
	}
}

func TestSourceAttackedOrBlockedThisTurnTrigger(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Watcher", "", 2, 2, WithAbility(At(EachEndStep,
		IfState(SourceAttackedOrBlockedThisTurnCondition(), GainLifeFor(You, Fixed(1))))))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtDeclaredBlocker, SourceID: card.ID(), PlayerID: a.PlayerID(), Flag: true})
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("trigger did not use clone history")
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.PutOnBattlefield(card, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("returned source used the old history")
	}
}

// --- from composable_source_attacked_test.go ---
func TestSourceAttackedThisTurnState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(card, a.PlayerID())
	p := g.FindPermanent(card.ID())
	ctx := StateContext{SourceID: card.ID(), ControllerID: a.PlayerID(), SourceIncarnation: p.incarnationID}
	condition := SourceAttackedThisTurnCondition()
	if condition.checkState(g, ctx) {
		t.Fatal("source matched before attacking")
	}
	g.trackers.Turn.RecordAttacked(card.ID())
	g.trackers.Turn.RecordAttackedIncarnation(card.ID(), p.incarnationID)
	if !condition.checkState(g, ctx) || !condition.checkState(g.Clone(), ctx) {
		t.Fatal("declared attack was not observed in game and clone")
	}
	g.FindPermanent(card.ID()).PhasedOut = true
	if condition.checkState(g, ctx) {
		t.Fatal("phased-out source matched")
	}
	g.FindPermanentIncludingPhased(card.ID()).PhasedOut = false
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	if condition.checkState(g, ctx) {
		t.Fatal("absent source matched")
	}
	g.PutOnBattlefield(card, a.PlayerID())
	if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: card.ID()}) {
		t.Fatal("returned source inherited attack history")
	}
	g.trackers.Turn.ResetForNewTurn(a.PlayerID())
	if condition.checkState(g, StateContext{SourceID: card.ID(), SourceIncarnation: g.FindPermanent(card.ID()).incarnationID}) {
		t.Fatal("attack history survived turn reset")
	}
}

func TestSourceAttackedThisTurnDeclaration(t *testing.T) {
	condition := SourceAttackedThisTurnCondition()
	card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(Attacks(This), IfState(condition, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(condition)
	if meta.Role != "state-condition" || meta.Text != "source attacked this turn" {
		t.Fatalf("condition metadata = %+v", meta)
	}
	spell := NewInstant("Invalid", "", WithEffect(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell validation = %v", err)
	}
}

func TestSourceAttackedThisTurnTriggerRecheck(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(Attacks(This),
		IfState(SourceAttackedThisTurnCondition(), GainLifeFor(You, Fixed(1))))))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	incarnation := g.FindPermanent(card.ID()).incarnationID
	g.trackers.Turn.RecordAttacked(card.ID())
	g.trackers.Turn.RecordAttackedIncarnation(card.ID(), incarnation)
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: card.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("clone did not keep its own attack history")
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.PutOnBattlefield(card, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("returned source used the old attack history")
	}
}

// --- from composable_source_blocking_test.go ---
func TestSourceBlockingConditionState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Blocker", "", 2, 2)
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(card, a.PlayerID())
	g.PutOnBattlefield(attacker, b.PlayerID())
	incarnation := g.FindPermanent(card.ID()).incarnationID
	ctx := StateContext{SourceID: card.ID(), ControllerID: a.PlayerID(), SourceIncarnation: incarnation}
	condition := SourceBlockingCondition()
	if condition.checkState(g, ctx) {
		t.Fatal("source matched before blocking")
	}
	g.combat.AddAttacker(attacker.ID(), a.PlayerID())
	g.combat.AddBlockerIncarnation(card.ID(), incarnation, attacker.ID())
	if !condition.checkState(g, ctx) || !condition.checkState(g.Clone(), ctx) {
		t.Fatal("blocking source was not observed in game and clone")
	}
	clone := g.Clone()
	clone.combat.RemoveFromCombat(card.ID())
	if condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("clone combat state was shared")
	}
	g.FindPermanent(card.ID()).PhasedOut = true
	if condition.checkState(g, ctx) {
		t.Fatal("phased-out source matched")
	}
	g.FindPermanentIncludingPhased(card.ID()).PhasedOut = false
	if !condition.checkState(g, ctx) {
		t.Fatal("phased-in blocker lost identity")
	}
	phased := g.PhaseOut(card.ID())
	g.PhaseIn(phased)
	if condition.checkState(g, ctx) {
		t.Fatal("phasing kept the source in combat")
	}
	g.combat.AddBlockerIncarnation(card.ID(), incarnation, attacker.ID())
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	if condition.checkState(g, ctx) {
		t.Fatal("absent source matched")
	}
	g.PutOnBattlefield(card, a.PlayerID())
	if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: card.ID()}) {
		t.Fatal("returned source inherited blocking state")
	}
}

func TestSourceBlockingConditionDeclaration(t *testing.T) {
	condition := SourceBlockingCondition()
	card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(Blocks(This),
		IfState(condition, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(condition)
	if meta.Role != "state-condition" || meta.Text != "source is blocking" {
		t.Fatalf("condition metadata = %+v", meta)
	}
	spell := NewInstant("Invalid", "", WithEffect(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell validation = %v", err)
	}
}

func TestSourceBlockingConditionResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(Blocks(This),
		IfState(SourceBlockingCondition(), GainLifeFor(You, Fixed(1))))))
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(card, a.PlayerID())
	g.PutOnBattlefield(attacker, b.PlayerID())
	g.combat.AddAttacker(attacker.ID(), a.PlayerID())
	g.combat.AddBlockerIncarnation(card.ID(), g.FindPermanent(card.ID()).incarnationID, attacker.ID())
	g.FireEvent(GameEvent{Type: EvtDeclaredBlocker, SourceID: card.ID(), TargetID: attacker.ID(), Flag: true})
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("clone failed to resolve its blocking trigger")
	}
	g.combat.RemoveFromCombat(card.ID())
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("source resolved after it stopped blocking")
	}
}

func TestSourceBlockingConditionStaticAndInterveningIf(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := SourceBlockingCondition()
	card := NewCreature("Guardian", "", 2, 2,
		WithAbility(Static(While(condition, ModifyPT(This, Fixed(0), Fixed(2))))),
		WithAbility(WhenEvent(Blocks(This), GainLifeFor(You, Fixed(1)), WithInterveningIf(condition))))
	attacker := NewCreature("Attacker", "", 2, 2)
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	g.PutOnBattlefield(attacker, b.PlayerID())
	g.ApplyContinuousEffects()
	if got := g.FindPermanent(card.ID()).CurrentToughness(g); got != 2 {
		t.Fatalf("before block toughness = %d", got)
	}
	g.combat.AddAttacker(attacker.ID(), a.PlayerID())
	g.combat.AddBlockerIncarnation(card.ID(), g.FindPermanent(card.ID()).incarnationID, attacker.ID())
	g.ApplyContinuousEffects()
	if got := g.FindPermanent(card.ID()).CurrentToughness(g); got != 4 {
		t.Fatalf("blocking toughness = %d", got)
	}
	g.FireEvent(GameEvent{Type: EvtDeclaredBlocker, SourceID: card.ID(), TargetID: attacker.ID(), Flag: true})
	g.PutTriggersOnStack()
	g.RemoveFromCombat(card.ID())
	g.ApplyContinuousEffects()
	if got := g.FindPermanent(card.ID()).CurrentToughness(g); got != 2 {
		t.Fatalf("after combat removal toughness = %d", got)
	}
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("intervening condition did not recheck blocking state")
	}
}

func TestCombatDeparturePreservesAttackerBlockedStatus(t *testing.T) {
	for _, departure := range []string{"leave", "phase"} {
		t.Run(departure, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			attacker := NewCreature("Attacker", "", 2, 2)
			blocker := NewCreature("Blocker", "", 2, 2)
			g.PutOnBattlefield(attacker, a.PlayerID())
			g.PutOnBattlefield(blocker, b.PlayerID())
			g.combat.AddAttacker(attacker.ID(), b.PlayerID())
			g.combat.AddBlockerIncarnation(blocker.ID(), g.FindPermanent(blocker.ID()).incarnationID, attacker.ID())
			oldIncarnation := g.FindPermanent(blocker.ID()).incarnationID
			if departure == "leave" {
				g.RemoveFromBattlefield(g.FindPermanent(blocker.ID()))
				g.PutOnBattlefield(blocker, b.PlayerID())
			} else {
				phased := g.PhaseOut(blocker.ID())
				g.PhaseIn(phased)
			}
			if !g.combat.GroupFor(attacker.ID()).Blocked {
				t.Fatal("departed blocker remained in combat or attacker became unblocked")
			}
			if departure == "leave" {
				if !g.IsBlockingIncarnation(blocker.ID(), oldIncarnation) ||
					g.IsBlockingIncarnation(blocker.ID(), g.FindPermanent(blocker.ID()).incarnationID) {
					t.Fatal("return changed the old combat relationship")
				}
			} else if g.IsBlockingInCombat(blocker.ID()) {
				t.Fatal("phased blocker stayed in combat")
			}
		})
	}
}

// --- from composable_source_combat_event_test.go ---
func TestSourceCombatEvents(t *testing.T) {
	for _, tc := range []struct {
		name  string
		event EventType
		flag  bool
		make  func(sourceRef) EventPattern
	}{
		{"attacks", EvtDeclaredAttacker, false, func(ref sourceRef) EventPattern { return Attacks(ref) }},
		{"blocks", EvtDeclaredBlocker, true, Blocks},
		{"becomes blocked", EvtCreatureBecomesBlocked, false, func(ref sourceRef) EventPattern { return BecomesBlocked(ref) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(tc.make(This),
				Sequence(Until(EndOfTurn, ModifyPT(This, Fixed(2), Fixed(2))), GainLifeFor(You, Fixed(1))))))
			source.SetOwner(a.PlayerID())
			other := NewCreature("Other", "", 1, 1)
			other.SetOwner(b.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(other, b.PlayerID())
			g.FireEvent(GameEvent{Type: tc.event, SourceID: other.ID(), PlayerID: b.PlayerID(), Flag: tc.flag})
			g.FireEvent(GameEvent{Type: EvtEntersAttacking, SourceID: source.ID(), PlayerID: a.PlayerID()})
			g.FireEvent(GameEvent{Type: EvtEntersBlocking, SourceID: source.ID(), PlayerID: a.PlayerID()})
			g.FireEvent(GameEvent{Type: tc.event, SourceID: source.ID(), PlayerID: a.PlayerID(), Flag: tc.flag})
			if tc.event == EvtDeclaredBlocker {
				g.FireEvent(GameEvent{Type: tc.event, SourceID: source.ID(), PlayerID: a.PlayerID(), Flag: false})
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
				t.Fatal("wrong trigger count or unexpected target")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || g.FindPermanent(source.ID()).CurrentPower(g) != 1 {
				t.Fatal("clone changed original game")
			}
			g.ResolveStack()
			for _, game := range []*Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != 21 || game.FindPermanent(source.ID()).CurrentPower(game) != 3 {
					t.Fatal("source combat trigger did not resolve")
				}
			}
		})
	}
}

func TestSourceCombatEventIncarnationAndPhasing(t *testing.T) {
	for _, change := range []string{"leaves", "returns", "phased", "phases in"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(BecomesBlocked(This),
				Sequence(Until(EndOfTurn, ModifyPT(This, Fixed(2), Fixed(2))), GainLifeFor(You, Fixed(1))))))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.FireEvent(GameEvent{Type: EvtCreatureBecomesBlocked, SourceID: source.ID()})
			switch change {
			case "leaves", "returns":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "returns" {
					g.PutOnBattlefield(source, b.PlayerID())
				}
			case "phased", "phases in":
				phased := g.PhaseOut(source.ID())
				if change == "phases in" {
					g.PhaseIn(phased)
				}
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("trigger lost after source changed")
			}
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 20 {
				t.Fatal("trigger controller changed")
			}
			if change == "returns" && g.FindPermanent(source.ID()).CurrentPower(g) != 1 {
				t.Fatal("new incarnation received old bonus")
			}
		})
	}
}

func TestSourceCombatEventValidation(t *testing.T) {
	for _, pattern := range []EventPattern{Attacks(This), Blocks(This), BecomesBlocked(This)} {
		card := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		meta := InspectDefinition(card.Abilities()[0])
		if meta.Role != "event-trigger" || len(meta.Children) == 0 || meta.Children[0].Role != "event-pattern" {
			t.Fatal("combat event missing from inspection")
		}
	}
	spell := NewInstant("Invalid", "", WithEffect(GainLifeFor(You, Fixed(1))))
	spell.AddAbility(WhenEvent(BecomesBlocked(This), GainLifeFor(You, Fixed(1))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell event validation = %v", err)
	}
}

func TestSourceBecomesBlockedAbsentOrPhasedBeforeEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(BecomesBlocked(This), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(source, a.PlayerID())
	phase := g.PhaseOut(source.ID())
	g.FireEvent(GameEvent{Type: EvtCreatureBecomesBlocked, SourceID: source.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phase)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.FireEvent(GameEvent{Type: EvtCreatureBecomesBlocked, SourceID: source.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("absent source triggered")
	}
}

func TestSourceCombatEventCopyAndTarget(t *testing.T) {
	a, b := newRecordingPlayer("A"), newRecordingPlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("player")
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(BecomesBlocked(This),
		GainLifeFor(target, Fixed(2)), WithTargets(target))))
	source.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	g.FireEvent(GameEvent{Type: EvtCreatureBecomesBlocked, SourceID: source.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	original := g.StackPeek()
	if original == nil || len(original.Targets) != 1 || original.Targets[0] != a.PlayerID() {
		t.Fatal("trigger target was not announced")
	}
	copied := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != a.PlayerID() {
		t.Fatal("copy lost target")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || g.StackSize() != 2 {
		t.Fatal("clone changed original stack")
	}
	g.ResolveStack()
	if a.Life() != 24 || b.Life() != 20 {
		t.Fatal("copied combat trigger changed target")
	}
}

// --- from composable_source_combat_history_test.go ---
func TestSourceAttackedOrBlockedThisCombatHistory(t *testing.T) {
	for _, eventType := range []EventType{EvtDeclaredAttacker, EvtDeclaredBlocker} {
		t.Run(eventType.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("History", "", 2, 2)
			g.PutOnBattlefield(card, a.PlayerID())
			p := g.FindPermanent(card.ID())
			ctx := StateContext{SourceID: card.ID(), ControllerID: a.PlayerID(), SourceIncarnation: p.incarnationID}
			condition := SourceAttackedOrBlockedThisCombatCondition()
			g.FireEvent(GameEvent{Type: EvtBeginCombat, PlayerID: a.PlayerID()})
			if condition.checkState(g, ctx) {
				t.Fatal("history exists before declaration")
			}
			g.FireEvent(GameEvent{Type: eventType, SourceID: card.ID(), PlayerID: a.PlayerID(), Flag: true})
			if !condition.checkState(g, ctx) {
				t.Fatal("declaration was not recorded")
			}
			clone := g.Clone()
			g.FindPermanent(card.ID()).PhasedOut = true
			if condition.checkState(g, ctx) {
				t.Fatal("phased source matched")
			}
			g.FindPermanentIncludingPhased(card.ID()).PhasedOut = false
			g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
			g.PutOnBattlefield(card, a.PlayerID())
			if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: card.ID()}) {
				t.Fatal("returned source inherited combat history")
			}
			if !condition.checkState(clone, ctx) {
				t.Fatal("clone lost independent history")
			}
			clone.RunStepWithPriority(EndCombat)
			if condition.checkState(clone, ctx) {
				t.Fatal("history survived end of combat")
			}
			clone.FireEvent(GameEvent{Type: eventType, SourceID: card.ID(), PlayerID: a.PlayerID(), Flag: true})
			clone.FireEvent(GameEvent{Type: EvtBeginCombat, PlayerID: a.PlayerID()})
			if condition.checkState(clone, ctx) {
				t.Fatal("history survived next combat")
			}
			if !SourceAttackedOrBlockedThisTurnCondition().checkState(clone, ctx) {
				t.Fatal("combat reset cleared turn history")
			}
		})
	}
}

func TestSourceAttackedOrBlockedThisCombatDeclaration(t *testing.T) {
	condition := SourceAttackedOrBlockedThisCombatCondition()
	card := NewCreature("Watcher", "", 2, 2, WithAbility(At(EachEndOfCombat, IfState(condition, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if meta := InspectDefinition(condition); meta.Role != "state-condition" || meta.Text != "source attacked or blocked this combat" {
		t.Fatalf("metadata = %+v", meta)
	}
	spell := NewInstant("Invalid", "", Spell(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell validation = %v", err)
	}
}

func TestSourceAttackedOrBlockedThisCombatEndTrigger(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Watcher", "", 2, 2, WithAbility(At(EachEndOfCombat,
		IfState(SourceAttackedOrBlockedThisCombatCondition(), GainLifeFor(You, Fixed(1))))))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtBeginCombat, PlayerID: a.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: card.ID(), PlayerID: a.PlayerID()})
	g.RunStepWithPriority(EndCombat)
	if a.Life() != 21 {
		t.Fatalf("end of combat life = %d", a.Life())
	}
	ctx := StateContext{SourceID: card.ID(), SourceIncarnation: g.FindPermanent(card.ID()).incarnationID}
	if SourceAttackedOrBlockedThisCombatCondition().checkState(g, ctx) {
		t.Fatal("combat history remained after end of combat")
	}
	if !SourceAttackedOrBlockedThisTurnCondition().checkState(g, ctx) {
		t.Fatal("turn history was cleared at end of combat")
	}
}

// --- from composable_unblocked_attack_test.go ---
func TestUnblockedAttackAfterLegalBlockers(t *testing.T) {
	for _, simulated := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		defender := BindPlayer("defender")
		watcher := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(AttacksUnblocked(This).WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(1)))))
		watcher.SetOwner(a.PlayerID())
		other := NewCreature("Other", "", 2, 2)
		other.SetOwner(a.PlayerID())
		blocker := NewCreature("Blocker", "", 2, 2)
		blocker.SetOwner(b.PlayerID())
		if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(watcher, a.PlayerID())
		g.PutOnBattlefield(other, a.PlayerID())
		g.PutOnBattlefield(blocker, b.PlayerID())
		g.combat.AddAttacker(watcher.ID(), b.PlayerID())
		g.combat.AddAttacker(other.ID(), b.PlayerID())
		if simulated {
			g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: other.ID()}})
		} else {
			g.combat.AddBlocker(blocker.ID(), other.ID())
			g.doDeclareBlockers()
		}
		if simulated {
			if g.GetPlayer(b.PlayerID()).Life() != 21 {
				t.Fatal("simulated declaration missed unblocked attacker")
			}
		} else {
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatalf("triggers = %d", g.StackSize())
			}
			clone := g.Clone()
			if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("could not copy trigger")
			}
			g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
			g.PutOnBattlefield(watcher, b.PlayerID())
			g.ResolveStack()
			clone.ResolveStack()
			if g.GetPlayer(b.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 21 {
				t.Fatal("defender capture or clone failed")
			}
		}
	}
}

func TestUnblockedAttackRejectsEarlierBlockAndAbsentSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(AttacksUnblocked(This), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	blocker := NewCreature("Blocker", "", 2, 2)
	blocker.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(blocker, b.PlayerID())
	g.combat.AddAttacker(watcher.ID(), b.PlayerID())
	g.combat.AddBlocker(blocker.ID(), watcher.ID())
	g.combat.Groups[0].BlockerIDs = nil
	g.doDeclareBlockers()
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("blocked attacker triggered after blocker departed")
	}
	g.combat.Reset()
	g.combat.AddAttacker(watcher.ID(), b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.doDeclareBlockers()
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased attacker triggered")
	}
}

func TestUnblockedAttackDeclarationValidation(t *testing.T) {
	defender := BindPlayer("defender")
	for _, pattern := range []EventPattern{AttacksUnblocked(This).WithDefendingPlayer(nil), AttacksUnblocked(This).WithDefendingPlayer(defender).WithDefendingPlayer(defender)} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid binding")
		}
	}
	card := NewCreature("Missing", "", 1, 1, WithAbility(WhenEvent(AttacksUnblocked(This), GainLifeFor(FromPlayer(defender), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("missing binding error = %v", err)
	}
}
