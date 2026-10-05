package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestDamagePacketsSimultaneousAndMerged(t *testing.T) {
	for _, overlap := range []bool{false, true} {
		g, a, b := newSpellCopyGame()
		target := DeclareDamageTarget("victim")
		packets := []DamagePacket{DamagePacketTo(target, Fixed(2)), DamagePacketTo(You, Fixed(3))}
		spell := NewInstant("Packets", "", Spell(DealDamagePackets(This, packets...), WithTargets(target)))
		packets[0] = DamagePacketTo(You, Fixed(99))
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		recipient := b.PlayerID()
		if overlap {
			recipient = a.PlayerID()
		}
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{recipient}, 0); err != nil {
			t.Fatal(err)
		}
		callbacks := 0
		g.SetOnDamageDealt(func(_, _ string, amount int, combat bool) {
			callbacks++
			if combat || a.Life() != 17 && !overlap || overlap && (a.Life() != 15 || amount != 5) || !overlap && b.Life() != 18 {
				t.Error("partial batch or duplicate recipient event")
			}
		})
		g.ResolveStack()
		want := 2
		if overlap {
			want = 1
		}
		if callbacks != want {
			t.Fatalf("callbacks=%d want=%d", callbacks, want)
		}
	}
}

func TestDamagePacketsDefinitionValidation(t *testing.T) {
	target := DeclareDamageTarget("target")
	for _, tc := range []struct {
		name    string
		effect  ResolvingEffect
		invalid bool
	}{
		{"valid", DealDamagePackets(This, DamagePacketTo(You, Fixed(2))), false},
		{"empty", DealDamagePackets(This), true},
		{"source", DealDamagePackets(nil, DamagePacketTo(You, Fixed(2))), true},
		{"recipient", DealDamagePackets(This, DamagePacketTo(nil, Fixed(2))), true},
		{"amount", DealDamagePackets(This, DamagePacketTo(You, nil)), true},
		{"zero packet", DealDamagePackets(This, DamagePacket{}), true},
		{"undeclared", DealDamagePackets(This, DamagePacketTo(target, Fixed(2))), true},
		{"unbound value", DealDamagePackets(This, DamagePacketTo(You, FromNumber(BindNumber("missing")))), true},
	} {
		card := NewInstant(tc.name, "", Spell(tc.effect))
		if err := ValidateCardDefinition(card.Name(), card); (err != nil) != tc.invalid {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}

type packetControllerLife struct{}

func (packetControllerLife) Text() string { return "controller life" }
func (packetControllerLife) resolveValue(ctx *EffectContext) int {
	return ctx.Game.GetPlayer(ctx.Controller).Life()
}
func (packetControllerLife) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value"}
}

func TestDamagePacketsValuesBeforeDamage(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclarePlayerTarget("recipient")
	spell := NewInstant("Read before damage", "", Spell(DealDamagePackets(This,
		DamagePacketTo(You, Fixed(2)), DamagePacketTo(target, packetControllerLife{}),
		DamagePacketTo(You, Fixed(0)), DamagePacketTo(target, Fixed(-4))), WithTargets(target)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	callbacks := 0
	g.SetOnDamageDealt(func(string, string, int, bool) { callbacks++ })
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 0 || callbacks != 2 {
		t.Fatalf("values evaluated after damage or nonpositive packet applied: %d/%d/%d", a.Life(), b.Life(), callbacks)
	}
}
