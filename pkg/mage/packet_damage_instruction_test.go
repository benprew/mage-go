package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPacketDamageInstructionDeclaration(t *testing.T) {
	target := DeclareDamageTarget("recipient")
	packets := []DamagePacket{DamagePacketTo(target, Fixed(4)), DamagePacketTo(You, Fixed(2))}
	effect := (PacketDamageInstruction{Source: This, Packets: packets}).Effect()
	packets[0] = DamagePacket{}
	groups, ok := InspectPacketDamageInstructions(Sequence(effect, Sequence(effect)))
	if !ok || len(groups) != 2 || len(groups[0].Packets) != 2 || groups[0].Packets[0].recipient != target {
		t.Fatal("lost complete packet group")
	}
	groups[0].Packets[0] = DamagePacket{}
	again, _ := InspectPacketDamageInstructions(effect)
	if again[0].Packets[0].recipient != target || groups[1].Packets[0].recipient != target {
		t.Fatal("packet results alias")
	}
	var absent *DamageTargetBinding
	for _, bad := range []PacketDamageInstruction{{}, {Source: This}, {Packets: packets}, {Source: This, Packets: []DamagePacket{DamagePacketTo(absent, Fixed(1))}}, {Source: This, Packets: []DamagePacket{DamagePacketTo(You, nil)}}} {
		if bad.Effect() != nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	var absentEffect *damagePacketsEffect
	for _, bad := range []Effect{nil, absentEffect, DealDamagePackets(nil, DamagePacketTo(You, Fixed(1))), DealDamagePackets(This), DealDamagePackets(This, DamagePacket{}), Sequence(effect, destructionInspectionTrap{})} {
		if _, ok := InspectPacketDamageInstructions(bad); ok {
			t.Fatal("incomplete tree accepted")
		}
	}
}

func TestPacketDamageInstructionSpell(t *testing.T) {
	for _, state := range []string{"only", "prefix", "suffix", "illegal target", "all illegal", "missing binding", "wrong binding", "opaque recipient", "opaque amount", "zero", "active player", "player target", "player illegal", "player missing", "opaque child"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := damageQueryFixture(t)
			other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
			second := DeclarePermanentTarget("second", IsLand)
			player := DeclarePlayerTarget("player")
			g.stack.Pop()
			obj.Targets = append(obj.Targets, other.ID())
			obj.TargetSpecs = append(obj.TargetSpecs, second.targetSpec())
			g.PushStack(obj)
			recipient := DamageRecipientRef(You)
			amount := ResolutionValue(Fixed(2))
			want, known := false, true
			switch state {
			case "active player":
				recipient = ActivePlayer
			case "player target", "player illegal", "player missing":
				recipient = player
				if state != "player missing" {
					obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
					id := obj.Controller
					if state == "player illegal" {
						id = uuid.New()
					}
					obj.Targets = append(obj.Targets, id)
				} else {
					known = false
				}
			case "opaque recipient":
				recipient = packetInspectionRecipientTrap{}
				known = false
			case "opaque amount":
				amount = damageInspectionValueTrap{}
				known = false
			case "zero":
				amount = Fixed(0)
			case "missing binding":
				obj.TargetSpecs = obj.TargetSpecs[1:]
				obj.Targets = obj.Targets[1:]
				known = false
			case "wrong binding":
				recipient = DeclarePlayerTarget("player")
				known = false
			}
			damage := DealDamagePackets(This, DamagePacketTo(target, Fixed(4)), DamagePacketTo(recipient, amount))
			obj.Effects = []Effect{damage}
			switch state {
			case "prefix", "zero":
				obj.Effects = []Effect{Sequence(damage, Destroy(second))}
				known = false
			case "suffix":
				obj.Effects = []Effect{Sequence(Destroy(second), damage)}
				want = true
			case "illegal target":
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = []Effect{Sequence(damage, Destroy(second))}
				known = false
			case "all illegal":
				p.GrantBaseAttr(core.Shroud)
				other.GrantBaseAttr(core.Shroud)
			case "opaque child":
				obj.Effects = []Effect{Sequence(Destroy(second), damage, destructionInspectionTrap{})}
				known = false
			}
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					if got, gotKnown := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{other.ID(), other.incarnationID}}); got != want || gotKnown != known {
						t.Fatalf("proof=%v/%v want=%v/%v", got, gotKnown, want, known)
					}
				}
			}
			if p.Damage != 0 || g.GetPlayer(obj.Controller).Life() != 20 {
				t.Fatal("query applied damage")
			}
		})
	}
}

type packetInspectionRecipientTrap struct{}

func (packetInspectionRecipientTrap) resolveRecipient(*EffectContext) (uuid.UUID, bool) {
	panic("recipient callback")
}
func (packetInspectionRecipientTrap) damageRecipient() {}

func TestPacketDamageInstructionRecipients(t *testing.T) {
	g, _, obj, target := damageQueryFixture(t)
	player := DeclarePlayerTarget("player")
	obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
	obj.Targets = append(obj.Targets, obj.Controller)
	for _, ref := range []DamageRecipientRef{You, ActivePlayer, player, target} {
		predicate, ok := InspectDamageRecipient(ref)
		if !ok {
			t.Fatal("native recipient rejected")
		}
		selected, known := predicate.Select(g, obj)
		if !known || selected.Empty() {
			t.Fatal("native recipient has no selection")
		}
	}
	var absent *PlayerTargetBinding
	for _, ref := range []DamageRecipientRef{nil, absent, This, DeclareOpponentChosenDamageTarget("chooser"), packetInspectionRecipientTrap{}} {
		if _, ok := InspectDamageRecipient(ref); ok {
			t.Fatalf("unsupported recipient accepted: %T", ref)
		}
	}
	if _, known := (DamageRecipientPredicate{}).Select(g, obj); known {
		t.Fatal("zero predicate is known")
	}
	predicate, _ := InspectDamageRecipient(You)
	if _, known := predicate.Select(nil, obj); known {
		t.Fatal("nil game accepted")
	}
	if _, known := predicate.Select(g, nil); known {
		t.Fatal("nil stack object accepted")
	}
	opponent := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	obj.TargetSpecs = append(obj.TargetSpecs, opponent.targetSpec())
	obj.Targets = append(obj.Targets, obj.Controller)
	predicate, _ = InspectDamageRecipient(opponent)
	if selected, known := predicate.Select(g, obj); !known || !selected.Empty() {
		t.Fatal("opponent target accepted controller")
	}
	predicate, _ = InspectDamageRecipient(DeclarePlayerTarget("player"))
	if _, known := predicate.Select(g, obj); known {
		t.Fatal("same name supplied foreign declaration")
	}
	obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
	obj.Targets = append(obj.Targets, obj.Controller)
	predicate, _ = InspectDamageRecipient(player)
	if _, known := predicate.Select(g, obj); known {
		t.Fatal("duplicate declaration accepted")
	}
}
