package mage

import "testing"

func TestIsAuraOnLand(t *testing.T) {
	for _, tc := range []struct {
		name        string
		enchantment bool
		aura        bool
		attached    bool
		land        bool
		missingHost bool
		want        bool
	}{
		{"aura on land", true, true, true, true, false, true},
		{"non-aura enchantment on land", true, false, true, true, false, false},
		{"non-enchantment on land", false, true, true, true, false, false},
		{"unattached aura", true, true, false, true, false, false},
		{"aura on creature", true, true, true, false, false, false},
		{"departed host", true, true, true, true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			var host Card = NewCreature("Host", "", 1, 1)
			if tc.land {
				host = NewLand("Host")
			}
			host.SetOwner(b.PlayerID())
			hostPerm := g.PutOnBattlefield(host, b.PlayerID())
			var attachment Card = NewArtifact("Attachment", "")
			if tc.enchantment {
				attachment = NewEnchantment("Attachment", "")
			}
			if tc.aura {
				attachment.AddSubType("Aura")
			}
			attachment.SetOwner(a.PlayerID())
			perm := g.PutOnBattlefield(attachment, a.PlayerID())
			if tc.attached {
				perm.AttachedTo = hostPerm.ID()
			}
			if tc.missingHost {
				g.RemoveFromBattlefield(hostPerm)
			}
			if got := IsAuraOnLand.Match(perm, g); got != tc.want {
				t.Fatalf("IsAuraOnLand = %v, want %v", got, tc.want)
			}
		})
	}
}
