package mage

import "testing"

func TestIsEnchantedMatchesPermanentWithAttachedAura(t *testing.T) {
	g, a, _ := randomTestGame()
	host := addRandomTargetCreature(g, a, "Host")
	auraCard := NewAura("Aura", "{W}")
	auraCard.SetOwner(a.PlayerID())
	aura := g.PutOnBattlefield(auraCard, a.PlayerID())
	aura.AttachedTo = host.ID()
	host.Attachments = append(host.Attachments, aura.ID())

	if !IsEnchanted.Match(host, g) {
		t.Fatal("permanent with an attached Aura did not match IsEnchanted")
	}
}

func TestIsEnchantedIgnoresNonAuraAttachments(t *testing.T) {
	g, a, _ := randomTestGame()
	host := addRandomTargetCreature(g, a, "Host")
	equipmentCard := NewArtifact("Equipment", "{1}", WithSubTypes("Equipment"))
	equipmentCard.SetOwner(a.PlayerID())
	equipment := g.PutOnBattlefield(equipmentCard, a.PlayerID())
	equipment.AttachedTo = host.ID()
	host.Attachments = append(host.Attachments, equipment.ID())

	if IsEnchanted.Match(host, g) {
		t.Fatal("permanent with only Equipment attached matched IsEnchanted")
	}
}
