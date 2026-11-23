package mage

import (
	"strings"
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func hasViolation(violations []InvariantViolation, category, substr string) bool {
	for _, v := range violations {
		if v.Category == category && strings.Contains(v.Message, substr) {
			return true
		}
	}
	return false
}

func invariantTestGame() (*Game, *BasePlayer, *BasePlayer) {
	a := NewBasePlayer("Alice")
	b := NewBasePlayer("Bob")
	for range 5 {
		cA := NewLand("Forest")
		cA.SetOwner(a.PlayerID())
		a.AddToLibrary(cA)

		cB := NewLand("Mountain")
		cB.SetOwner(b.PlayerID())
		b.AddToLibrary(cB)
	}
	g := NewGame(a, b)
	return g, a, b
}

func TestCheckInvariants_ValidGamePasses(t *testing.T) {
	g, a, b := invariantTestGame()
	initial := g.SnapshotInitialCards()

	a.DrawCard()
	b.DrawCard()

	tok := NewCreature("Saproling", "", 1, 1)
	tok.SetOwner(a.PlayerID())
	tokPerm := g.PutTokenOnBattlefield(tok, a.PlayerID())
	if tokPerm == nil || !tokPerm.IsToken {
		t.Fatal("expected token permanent on battlefield")
	}

	g.SetStep(PrecombatMain)
	g.ApplyContinuousEffects()

	violations := g.CheckInvariants(InvariantOptions{
		PostSBA:                    true,
		InitialCards:               initial,
		CheckContinuousIdempotency: true,
	})
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got: %v", violations)
	}
}

func TestCheckInvariants_ZoneHygieneAndConservation(t *testing.T) {
	t.Run("duplicate card across zones", func(t *testing.T) {
		g, a, _ := invariantTestGame()
		initial := g.SnapshotInitialCards()
		c := a.Library()[0]
		a.AddToHand(c)

		violations := g.CheckInvariants(InvariantOptions{InitialCards: initial})
		if !hasViolation(violations, "conservation", "appears in multiple zones") {
			t.Fatalf("expected duplicate violation, got: %v", violations)
		}
	})

	t.Run("missing card from initial snapshot", func(t *testing.T) {
		g, a, _ := invariantTestGame()
		initial := g.SnapshotInitialCards()
		a.SetLibrary(a.Library()[1:])

		violations := g.CheckInvariants(InvariantOptions{InitialCards: initial})
		if !hasViolation(violations, "conservation", "missing from all zones") {
			t.Fatalf("expected missing card violation, got: %v", violations)
		}
	})

	t.Run("leaked token card in graveyard", func(t *testing.T) {
		g, a, _ := invariantTestGame()
		initial := g.SnapshotInitialCards()
		leaked := NewCreature("Goblin Token", "", 1, 1)
		leaked.SetOwner(a.PlayerID())
		a.AddToGraveyard(leaked)

		violations := g.CheckInvariants(InvariantOptions{InitialCards: initial})
		if !hasViolation(violations, "conservation", "not in initial card set") {
			t.Fatalf("expected leaked token violation, got: %v", violations)
		}
	})

	t.Run("wrong card owner in hand", func(t *testing.T) {
		g, a, b := invariantTestGame()
		c := b.Library()[0]
		b.SetLibrary(b.Library()[1:])
		a.hand = append(a.hand, c)

		violations := g.CheckInvariants(InvariantOptions{})
		if !hasViolation(violations, "zone", "belongs to") {
			t.Fatalf("expected wrong owner violation, got: %v", violations)
		}
	})

	t.Run("instant on battlefield face up vs face down", func(t *testing.T) {
		g, a, _ := invariantTestGame()
		inst := NewInstant("Bolt", "{R}", NewSpellAbility())
		inst.SetOwner(a.PlayerID())
		perm := NewPermanent(inst, a.PlayerID())
		g.AddToBattlefield(perm)

		violations := g.CheckInvariants(InvariantOptions{})
		if !hasViolation(violations, "zone", "non-permanent card type") {
			t.Fatalf("expected non-permanent battlefield violation, got: %v", violations)
		}

		perm.FaceDown = true
		violations = g.CheckInvariants(InvariantOptions{})
		if hasViolation(violations, "zone", "non-permanent card type") {
			t.Fatalf("face-down permanent should be allowed, got: %v", violations)
		}
	})

	t.Run("stale enteringPermanent", func(t *testing.T) {
		g, a, _ := invariantTestGame()
		c := NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(a.PlayerID())
		g.zones.enteringPermanent = NewPermanent(c, a.PlayerID())

		violations := g.CheckInvariants(InvariantOptions{})
		if !hasViolation(violations, "zone", "enteringPermanent") {
			t.Fatalf("expected enteringPermanent violation, got: %v", violations)
		}
	})
}

func TestCheckInvariants_Attachments(t *testing.T) {
	t.Run("stale attachment reference on host", func(t *testing.T) {
		g, a, _ := invariantTestGame()
		hostCard := NewCreature("Bear", "{1}{G}", 2, 2)
		hostCard.SetOwner(a.PlayerID())
		host := NewPermanent(hostCard, a.PlayerID())
		equipCard := NewArtifact("Sword", "{2}", WithSubTypes("Equipment"))
		equipCard.SetOwner(a.PlayerID())
		equip := NewPermanent(equipCard, a.PlayerID())
		g.AddToBattlefield(host, equip)
		host.Attachments = append(host.Attachments, equip.ID())

		violations := g.CheckInvariants(InvariantOptions{})
		if !hasViolation(violations, "attachment", "lists attachment") {
			t.Fatalf("expected stale attachment violation, got: %v", violations)
		}
	})

	t.Run("asymmetric AttachedTo without host entry", func(t *testing.T) {
		g, a, _ := invariantTestGame()
		hostCard := NewCreature("Bear", "{1}{G}", 2, 2)
		hostCard.SetOwner(a.PlayerID())
		host := NewPermanent(hostCard, a.PlayerID())
		equipCard := NewArtifact("Sword", "{2}", WithSubTypes("Equipment"))
		equipCard.SetOwner(a.PlayerID())
		equip := NewPermanent(equipCard, a.PlayerID())
		g.AddToBattlefield(host, equip)
		equip.AttachedTo = host.ID()

		violations := g.CheckInvariants(InvariantOptions{})
		if !hasViolation(violations, "attachment", "not listed in host") {
			t.Fatalf("expected asymmetric attachment violation, got: %v", violations)
		}
	})

	t.Run("unattached aura flagged only when PostSBA is true", func(t *testing.T) {
		g, a, _ := invariantTestGame()
		auraCard := NewEnchantment("Holy Strength", "{W}", WithSubTypes("Aura"))
		auraCard.SetOwner(a.PlayerID())
		aura := NewPermanent(auraCard, a.PlayerID())
		g.AddToBattlefield(aura)

		if v := g.CheckInvariants(InvariantOptions{PostSBA: false}); len(v) != 0 {
			t.Fatalf("expected no violations when PostSBA=false, got: %v", v)
		}
		v := g.CheckInvariants(InvariantOptions{PostSBA: true})
		if !hasViolation(v, "attachment", "unattached Aura") {
			t.Fatalf("expected unattached Aura violation, got: %v", v)
		}
	})
}

func TestCheckInvariants_PostSBA(t *testing.T) {
	t.Run("zero toughness and lethal damage", func(t *testing.T) {
		g, a, _ := invariantTestGame()
		zeroCard := NewCreature("Zero", "{0}", 0, 0)
		zeroCard.SetOwner(a.PlayerID())
		deadCard := NewCreature("Dead", "{1}{G}", 2, 2)
		deadCard.SetOwner(a.PlayerID())
		indCard := NewCreature("Indy", "{2}{W}", 2, 2, WithKeyword(Indestructible))
		indCard.SetOwner(a.PlayerID())

		zero := NewPermanent(zeroCard, a.PlayerID())
		dead := NewPermanent(deadCard, a.PlayerID())
		dead.Damage = 2
		indy := NewPermanent(indCard, a.PlayerID())
		indy.Damage = 5
		g.AddToBattlefield(zero, dead, indy)

		if v := g.CheckInvariants(InvariantOptions{PostSBA: false}); len(v) != 0 {
			t.Fatalf("expected no violations when PostSBA=false, got: %v", v)
		}

		v := g.CheckInvariants(InvariantOptions{PostSBA: true})
		if !hasViolation(v, "sba", "Zero") {
			t.Fatalf("expected zero toughness violation, got: %v", v)
		}
		if !hasViolation(v, "sba", "Dead") {
			t.Fatalf("expected lethal damage violation, got: %v", v)
		}
		if hasViolation(v, "sba", "Indy") {
			t.Fatalf("indestructible creature should not trigger SBA violation: %v", v)
		}
	})

	t.Run("counter annihilation legend and world rules", func(t *testing.T) {
		g, a, _ := invariantTestGame()
		c1 := NewCreature("Isamaru", "{W}", 2, 2, WithSuperTypes(SuperLegendary))
		c1.SetOwner(a.PlayerID())
		c2 := NewCreature("Isamaru", "{W}", 2, 2, WithSuperTypes(SuperLegendary))
		c2.SetOwner(a.PlayerID())
		p1 := NewPermanent(c1, a.PlayerID())
		p2 := NewPermanent(c2, a.PlayerID())
		p1.AddCounter(P1P1, 1)
		p1.AddCounter(M1M1, 1)

		w1 := NewEnchantment("World A", "{1}", WithSuperTypes(SuperWorld))
		w1.SetOwner(a.PlayerID())
		w2 := NewEnchantment("World B", "{1}", WithSuperTypes(SuperWorld))
		w2.SetOwner(a.PlayerID())
		g.AddToBattlefield(p1, p2, NewPermanent(w1, a.PlayerID()), NewPermanent(w2, a.PlayerID()))

		v := g.CheckInvariants(InvariantOptions{PostSBA: true})
		if !hasViolation(v, "sba", "-1/-1") {
			t.Fatalf("expected counter annihilation violation, got: %v", v)
		}
		if !hasViolation(v, "sba", "legendary") {
			t.Fatalf("expected legend rule violation, got: %v", v)
		}
		if !hasViolation(v, "sba", "World") {
			t.Fatalf("expected world rule violation, got: %v", v)
		}
	})

	t.Run("alive player with 10 poison or pending trigger", func(t *testing.T) {
		g, a, _ := invariantTestGame()
		a.AddPoisonCounters(10)
		g.triggers.pending = append(g.triggers.pending, &pendingTrigger{controller: a.PlayerID()})

		v := g.CheckInvariants(InvariantOptions{PostSBA: true})
		if !hasViolation(v, "sba", "poison") {
			t.Fatalf("expected poison SBA violation, got: %v", v)
		}
		if !hasViolation(v, "sba", "pending trigger") {
			t.Fatalf("expected pending trigger violation, got: %v", v)
		}
	})
}

func TestCheckInvariants_CombatAndStack(t *testing.T) {
	t.Run("combat active outside combat phase", func(t *testing.T) {
		g, a, b := invariantTestGame()
		g.SetStep(PrecombatMain)
		g.combat.AddAttacker(a.Library()[0].ID(), b.PlayerID())

		v := g.CheckInvariants(InvariantOptions{})
		if !hasViolation(v, "combat", "outside combat") {
			t.Fatalf("expected combat outside phase violation, got: %v", v)
		}
	})

	t.Run("stack object missing card and negative X", func(t *testing.T) {
		g, a, _ := invariantTestGame()
		g.stack.Push(&StackObject{
			ID:         uuid.New(),
			Controller: a.PlayerID(),
			IsAbility:  false,
			Card:       nil,
			XValue:     -1,
		})

		v := g.CheckInvariants(InvariantOptions{})
		if !hasViolation(v, "stack", "nil Card") {
			t.Fatalf("expected nil Card violation, got: %v", v)
		}
		if !hasViolation(v, "stack", "negative XValue") {
			t.Fatalf("expected negative XValue violation, got: %v", v)
		}
	})
}

func TestCheckInvariants_ContinuousIdempotency(t *testing.T) {
	g, a, _ := invariantTestGame()
	bearCard := NewCreature("Bear", "{1}{G}", 2, 2)
	bearCard.SetOwner(a.PlayerID())
	bear := NewPermanent(bearCard, a.PlayerID())
	g.AddToBattlefield(bear)

	// Add an intentionally non-idempotent effect that mutates permanent counters every Apply.
	g.AddContinuousEffect(FuncContinuousEffect(LayerPT, Indefinite, func(g *Game, _ uuid.UUID) error {
		if p := g.MutablePermanent(bear.ID()); p != nil {
			p.AddCounter(P1P1, 1)
		}
		return nil
	}, func(_ *Game, _ uuid.UUID) bool { return true }), bear.ID())
	g.ApplyContinuousEffects()

	v := g.CheckInvariants(InvariantOptions{CheckContinuousIdempotency: true})
	if !hasViolation(v, "layers", "not idempotent") {
		t.Fatalf("expected layer idempotency violation, got: %v", v)
	}
}

func addInvariantPermanent(g *Game, owner *BasePlayer, card *Card) *Permanent {
	card.SetOwner(owner.PlayerID())
	return g.PutOnBattlefield(card, owner.PlayerID())
}

func TestCheckInvariants_LegalCombatTransitions(t *testing.T) {
	g, a, b := invariantTestGame()
	g.SetStep(DeclareAttackers)
	walker := addInvariantPermanent(g, b, NewPlaneswalker("Walker", "", 3))
	attacker := g.PutTokenOnBattlefieldAttacking(NewCreature("Attacker", "", 2, 2), a.PlayerID(), walker.ID())
	options := InvariantOptions{PostSBA: true}
	if got := g.CheckInvariants(options); len(got) != 0 {
		t.Errorf("attack against a planeswalker: %v", got)
	}

	g.DestroyPermanent(walker)
	group := g.combat.GroupFor(attacker.ID())
	if group == nil || group.DefenderID != uuid.Nil() || !g.combat.IsAttacking(attacker.ID()) {
		t.Fatalf("attacker must remain in combat without a defender: %+v", group)
	}
	g.AddRegenerationShield(attacker.ID())
	g.DestroyPermanent(attacker)
	if got := g.CheckInvariants(options); len(got) != 0 {
		t.Errorf("after attacker regenerates: %v", got)
	}
}

func TestRegeneratedBlockerLeavesAttackerBlocked(t *testing.T) {
	g, a, b := invariantTestGame()
	attacker := addInvariantPermanent(g, a, NewCreature("Attacker", "", 2, 2))
	blocker := addInvariantPermanent(g, b, NewCreature("Blocker", "", 2, 2))
	g.SetStep(DeclareBlockers)
	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	g.combat.AddBlocker(blocker.ID(), attacker.ID())
	g.AddRegenerationShield(blocker.ID())
	g.DestroyPermanent(blocker)
	group := g.combat.GroupFor(attacker.ID())
	if !group.Blocked || len(group.BlockerIDs) != 0 {
		t.Errorf("after blocker regenerates: %+v", group)
	}
	g.RunStepWithPriority(CombatDamage)
	if b.Life() != 20 {
		t.Errorf("defender life = %d, want 20", b.Life())
	}
}

func TestCheckInvariants_PhasedAuraWithDepartedHost(t *testing.T) {
	g, a, _ := invariantTestGame()
	host := addInvariantPermanent(g, a, NewCreature("Host", "", 2, 2))
	aura := addInvariantPermanent(g, a, NewAura("Aura", "{W}"))
	g.Attach(aura.ID(), host.ID())
	phased := g.PhaseOut(aura.ID())
	g.DestroyPermanent(host)
	if got := g.CheckInvariants(InvariantOptions{PostSBA: true}); len(got) != 0 {
		t.Errorf("while the Aura is phased out: %v", got)
	}
	g.PhaseIn(phased)
	g.CheckStateBasedActions()
	if g.FindPermanent(aura.ID()) != nil {
		t.Error("Aura stayed on the battlefield after phasing in without a host")
	}
}

func TestCheckInvariants_IllegalAttachments(t *testing.T) {
	g, a, _ := invariantTestGame()
	host := addInvariantPermanent(g, a, NewCreature("Host", "", 2, 2))
	equipment := addInvariantPermanent(g, a, NewEquipment("Equipment", "{1}"))
	aura := addInvariantPermanent(g, a, NewAura("Aura", "{W}", WithKeyword(Indestructible)))
	artifact := addInvariantPermanent(g, a, NewArtifact("Other artifact", "{1}"))
	for _, attachment := range []*Permanent{equipment, aura, artifact} {
		g.Attach(attachment.ID(), host.ID())
	}
	host.RuntimeAbilities = append(host.RuntimeAbilities, ProtectionFromCardType(TypeArtifact))
	host.GrantBaseAttr(AttrCantBeEnchanted)
	options := InvariantOptions{PostSBA: true}
	for _, name := range []string{"Equipment", "Aura", "Other artifact"} {
		if got := g.CheckInvariants(options); !hasViolation(got, "attachment", name) {
			t.Errorf("missing attachment violation for %s: %v", name, got)
		}
	}
	if t.Failed() {
		return
	}
	g.CheckStateBasedActions()
	if g.FindPermanent(aura.ID()) != nil || equipment.IsAttached() || artifact.IsAttached() || len(host.Attachments) != 0 {
		t.Fatalf("SBAs did not clear the illegal attachments: %v", g.CheckInvariants(options))
	}
}

func TestCheckInvariants_FaceDownSupertypes(t *testing.T) {
	g, a, _ := invariantTestGame()
	for _, supertype := range []SuperType{SuperLegendary, SuperWorld} {
		addInvariantPermanent(g, a, NewCreature(supertype.String(), "", 2, 2, WithSuperTypes(supertype)))
		hidden := addInvariantPermanent(g, a, NewCreature(supertype.String(), "", 2, 2, WithSuperTypes(supertype)))
		hidden.FaceDown = true
		hidden.BasePTOverride = &[2]int{2, 2}
	}
	if got := g.CheckInvariants(InvariantOptions{PostSBA: true}); len(got) != 0 {
		t.Errorf("face-down cards do not have their printed supertypes: %v", got)
	}
	g.CheckStateBasedActions()
	if got := len(g.AllBattlefield()); got != 4 {
		t.Errorf("battlefield size = %d, want 4", got)
	}
}

func TestWorldRule_SimultaneousWorldPermanents(t *testing.T) {
	g, a, b := invariantTestGame()
	makeWorld := func(name string, owner *BasePlayer) *Permanent {
		card := NewEnchantment(name, "", WithSuperTypes(SuperWorld))
		card.SetOwner(owner.PlayerID())
		return NewPermanent(card, owner.PlayerID())
	}
	g.AddToBattlefield(makeWorld("Older world", a))
	g.AddToBattlefield(makeWorld("New world A", a), makeWorld("New world B", b))
	g.CheckStateBasedActions()
	if got := len(g.AllBattlefield()); got != 0 {
		t.Errorf("World permanents remaining after a tie = %d, want 0", got)
	}
}

func TestCheckInvariants_SettledAnte(t *testing.T) {
	_, a, b := invariantTestGame()
	g, err := NewGameWithAnte(a, b, a.Library()[:1], b.Library()[:1])
	if err != nil {
		t.Fatal(err)
	}
	initial := g.SnapshotInitialCards()
	b.SetLost()
	if _, err := g.AnteResult(); err != nil {
		t.Fatal(err)
	}
	if got := g.CheckInvariants(InvariantOptions{InitialCards: initial}); len(got) != 0 {
		t.Errorf("after settling ante: %v", got)
	}
}

func TestCheckInvariants_NilStack(t *testing.T) {
	g, _, _ := invariantTestGame()
	g.stack = nil
	defer func() {
		if p := recover(); p != nil {
			t.Errorf("checker panicked: %v", p)
		}
	}()
	if got := g.CheckInvariants(InvariantOptions{CheckContinuousIdempotency: true}); !hasViolation(got, "stack", "stack is nil") {
		t.Errorf("expected missing stack violation: %v", got)
	}
}

func TestCheckInvariants_PriorityRequiresLiveGame(t *testing.T) {
	g, a, _ := invariantTestGame()
	g.SetStep(PrecombatMain)
	a.SetLost()
	if got := g.CheckInvariants(InvariantOptions{AtPriority: true}); !hasViolation(got, "priority", "game ended") {
		t.Errorf("expected priority violation: %v", got)
	}
	if got := g.CheckInvariants(InvariantOptions{}); len(got) != 0 {
		t.Errorf("terminal state: %v", got)
	}
}

func TestCheckInvariants_ContinuousAbilityGrowth(t *testing.T) {
	g, a, _ := invariantTestGame()
	perm := addInvariantPermanent(g, a, NewCreature("Host", "", 2, 2))
	g.AddContinuousEffect(FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, id uuid.UUID) error {
		host := g.MutablePermanent(id)
		host.RuntimeAbilities = append(host.RuntimeAbilities, ProtectionFromColor(Blue))
		return nil
	}, func(_ *Game, _ uuid.UUID) bool { return true }), perm.ID())
	if got := g.CheckInvariants(InvariantOptions{CheckContinuousIdempotency: true}); !hasViolation(got, "layers", "not idempotent") {
		t.Errorf("expected runtime ability growth violation: %v", got)
	}
}

func TestCheckInvariants_StackDivisions(t *testing.T) {
	g, a, b := invariantTestGame()
	g.SetStep(PrecombatMain)
	card := NewSorcery("Divided damage", "", NewTargetedSpell(TargetPlayer(), DealDividedDamage(Fixed(3))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), true)
	if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != a.PlayerID() {
		t.Fatalf("expected a copy targeting Alice: %+v", copied)
	}
	if copied.DamageDistribution[a.PlayerID()] != 3 {
		t.Errorf("damage assignment after changing target: %v", copied.DamageDistribution)
	}
	if got := g.CheckInvariants(InvariantOptions{}); len(got) != 0 {
		t.Errorf("valid stack: %v", got)
	}
	copied.DamageDistribution = map[uuid.UUID]int{b.PlayerID(): 3}
	if got := g.CheckInvariants(InvariantOptions{}); !hasViolation(got, "stack", "damage distribution") {
		t.Errorf("expected stale target violation: %v", got)
	}
	copied.DamageDistribution = map[uuid.UUID]int{a.PlayerID(): 2}
	if got := g.CheckInvariants(InvariantOptions{}); !hasViolation(got, "stack", "damage distribution") {
		t.Errorf("expected wrong total violation: %v", got)
	}
}

func TestCheckInvariants_CounterDivisionSurvivesTargetLeaving(t *testing.T) {
	g, a, b := invariantTestGame()
	g.SetStep(PrecombatMain)
	newTarget := addInvariantPermanent(g, a, NewCreature("New target", "", 2, 2))
	target := addInvariantPermanent(g, b, NewCreature("Original target", "", 2, 2))
	spell := NewInstant("Divided counters", "{X}", NewTargetedSpell(TargetCreature(), RandomCounterDistribution(P1P1, XValue())))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 2); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(spell.ID(), a.PlayerID(), true)
	if copied == nil || copied.CounterDistribution[newTarget.ID()] != 2 {
		t.Fatalf("expected the copy to assign two counters to the new target: %+v", copied)
	}
	g.DestroyPermanent(target)
	if got := g.CheckInvariants(InvariantOptions{}); len(got) != 0 {
		t.Errorf("original target leaving must preserve the announced shares: %v", got)
	}
	copied.CounterDistribution[newTarget.ID()] = 1
	if got := g.CheckInvariants(InvariantOptions{}); !hasViolation(got, "stack", "counter distribution") {
		t.Errorf("expected incorrect total violation: %v", got)
	}
}

func TestDividedDamageRequiresPositiveShareForEachTarget(t *testing.T) {
	g, a, b := invariantTestGame()
	g.SetStep(PrecombatMain)
	creature := addInvariantPermanent(g, b, NewCreature("Bear", "", 2, 2))
	spell := NewSorcery("Divided damage", "{1}", NewTargetedSpell(TargetUpToNCreaturesOrPlayers(3), DealDividedDamage(Fixed(2))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(Colorless, 1)
	targets := []uuid.UUID{a.PlayerID(), b.PlayerID(), creature.ID()}
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), targets, 0); err == nil || len(a.Hand()) != 1 || !g.stack.IsEmpty() {
		t.Fatalf("casting with too many targets must fail before payment, got %v", err)
	}
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), targets[:2], 0); err != nil {
		t.Fatal(err)
	}
	if got := g.stack.Peek().DamageDistribution; got[a.PlayerID()] != 1 || got[b.PlayerID()] != 1 {
		t.Errorf("distribution = %v, want one damage per target", got)
	}
}

func TestDeathtouchSBAUsesDamageEvent(t *testing.T) {
	g, a, b := invariantTestGame()
	source := addInvariantPermanent(g, a, NewCreature("Snake", "", 1, 1, WithKeyword(Deathtouch)))
	victim := addInvariantPermanent(g, b, NewCreature("Host", "", 2, 4))
	g.DealDamageToPermanent(victim, 1, source.ID())
	if victim.Damage != 1 {
		t.Errorf("marked damage = %d, want 1", victim.Damage)
	}
	if got := g.CheckInvariants(InvariantOptions{PostSBA: true}); !hasViolation(got, "sba", "deathtouch") {
		t.Errorf("expected unprocessed deathtouch violation: %v", got)
	}
	g.AddContinuousEffect(FuncContinuousEffect(LayerPT, EndOfTurn, func(g *Game, id uuid.UUID) error {
		if perm := g.MutablePermanent(id); perm != nil {
			perm.BoostPT(0, 2)
		}
		return nil
	}, func(_ *Game, _ uuid.UUID) bool { return true }), victim.ID())
	g.CheckStateBasedActions()
	if g.FindPermanent(victim.ID()) != nil {
		t.Error("creature survived a deathtouch damage event")
	}
}
