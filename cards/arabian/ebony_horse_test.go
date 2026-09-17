package arabian

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEbonyHorseKeepsAttackerInCombat(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	card, err := mage.CreateCard("Ebony Horse")
	if err != nil {
		t.Fatal(err)
	}
	horse := g.PutOnBattlefield(card, a.PlayerID())
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 3, 3), a.PlayerID())
	blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 2, 2), b.PlayerID())
	g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
	g.GetCombat().AddBlocker(blocker.ID(), attacker.ID())
	g.TapPermanent(attacker)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), horse.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.IsAttackingInCombat(attacker.ID()) || !g.IsBlockingInCombat(blocker.ID()) {
		t.Fatal("Ebony Horse removed a creature from combat")
	}
	if g.FindPermanent(attacker.ID()).Tapped || !g.FindPermanent(horse.ID()).Tapped {
		t.Fatal("untap or tap cost failed")
	}
	g.ExecuteCombatDamage()
	if g.FindPermanent(attacker.ID()).Damage != 0 || g.FindPermanent(blocker.ID()).Damage != 0 {
		t.Fatal("combat damage was not prevented in both directions")
	}
	g.DealDamageToPermanent(g.FindPermanent(attacker.ID()), 1, blocker.ID())
	if g.FindPermanent(attacker.ID()).Damage != 1 {
		t.Fatal("noncombat damage was prevented")
	}
}

func TestEbonyHorseRestrictionsAndStun(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	card, err := mage.CreateCard("Ebony Horse")
	if err != nil {
		t.Fatal(err)
	}
	horse := g.PutOnBattlefield(card, a.PlayerID())
	own := g.PutOnBattlefield(mage.NewCreature("Own", "", 3, 3), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 3, 3), b.PlayerID())
	g.GetCombat().AddAttacker(other.ID(), a.PlayerID())
	a.ManaPool().Add(core.Colorless, 2)
	for _, id := range []uuid.UUID{own.ID(), other.ID()} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), horse.ID(), 0, []uuid.UUID{id}); err == nil {
			t.Fatal("accepted a nonattacker or opposing creature")
		}
	}
	if g.FindPermanent(horse.ID()).Tapped || a.ManaPool().TotalMana() != 2 {
		t.Fatal("invalid target spent costs")
	}
	g.GetCombat().AddAttacker(own.ID(), b.PlayerID())
	g.TapPermanent(own)
	g.MutablePermanent(own.ID()).AddCounter(core.Stun, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), horse.ID(), 0, []uuid.UUID{own.ID()}); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(horse.ID()))
	g.ResolveStack()
	if !g.FindPermanent(own.ID()).Tapped || g.FindPermanent(own.ID()).Counters[core.Stun] != 0 {
		t.Fatal("stun did not replace untap")
	}
	g.ExecuteCombatDamage()
	if b.Life() != 20 || a.Life() != 17 {
		t.Fatal("prevention depended on untap success or prevented unrelated combat damage")
	}
}
