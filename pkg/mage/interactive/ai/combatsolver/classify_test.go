package combatsolver

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestClassifyCombat_Pump(t *testing.T) {
	// Giant Growth: target creature gets +3/+3 until end of turn.
	card := mage.NewInstant("Giant Growth", "{G}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.Boost(mage.Fixed(3), mage.Fixed(3))),
	)
	if got := ClassifyCombat(card); got != RolePump {
		t.Errorf("Giant Growth = %v, want Pump", got)
	}
}

func TestClassifyCombat_Damage(t *testing.T) {
	// Lightning Bolt: deal 3 damage to any target.
	card := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	if got := ClassifyCombat(card); got != RoleDamage {
		t.Errorf("Lightning Bolt = %v, want Damage", got)
	}
}

func TestClassifyCombat_Sorcery(t *testing.T) {
	// A creature is a sorcery-speed permanent — should not be classified as a trick.
	card := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	if got := ClassifyCombat(card); got != RoleNone {
		t.Errorf("Creature = %v, want None", got)
	}
}

func TestClassifyCombat_Nil(t *testing.T) {
	if got := ClassifyCombat(nil); got != RoleNone {
		t.Errorf("nil card = %v, want None", got)
	}
}
