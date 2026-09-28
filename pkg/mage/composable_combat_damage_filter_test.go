package mage

import "testing"

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
