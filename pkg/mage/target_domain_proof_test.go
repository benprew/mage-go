package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetDomainProof(t *testing.T) {
	wall := SubtypeDomain("Wall")
	elf := SubtypeDomain("Elf")
	for _, tc := range []struct {
		name   string
		domain PermanentTargetDomain
		want   bool
	}{
		{"wall", wall, true},
		{"unknown", UnknownPermanentDomain(), false},
		{"union", AnyPermanentDomain(wall, elf), false},
		{"shared", AnyPermanentDomain(AllPermanentDomains(wall, elf), AllPermanentDomains(wall, UnknownPermanentDomain())), true},
		{"negative", NonSubtypeDomain("Wall"), false},
		{"eliminate branch", AllPermanentDomains(AnyPermanentDomain(wall, elf), NonSubtypeDomain("Elf")), true},
		{"impossible", AllPermanentDomains(wall, NonSubtypeDomain("Wall")), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binding := DeclarePermanentTarget("subject", AnyPermanent, TargetWithDomain(tc.domain))
			card := NewInstant("Test", "", Spell(TapSubject(binding), WithTargets(binding)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			for _, target := range []Target{binding, binding.spec, TargetRandom(binding.spec), TargetOpponentChoice(binding.spec), &countBoundTarget{inner: binding.spec, count: 0}} {
				if got := TargetRequiresSubtype(target, "Wall"); got != tc.want {
					t.Fatalf("%T proof = %v, want %v", target, got, tc.want)
				}
			}
		})
	}
	for _, target := range []Target{nil, TargetPlayer(), TargetSpellOnStack(), TargetDamageAnyTarget(), TargetPermanent(HasSubType("Wall"))} {
		if TargetRequiresSubtype(target, "Wall") {
			t.Fatalf("invented proof for %T", target)
		}
	}
	if !TargetRequiresSubtype(DeclarePermanentTarget("legacy", AnyPermanent, TargetHasSubtype("Wall")), "Wall") {
		t.Fatal("lost existing constraint")
	}
}

func TestTargetDomainProofEnforcement(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 5, WithSubTypes("Wall")), a.PlayerID())
	elf := g.PutOnBattlefield(NewCreature("Elf", "", 1, 5, WithSubTypes("Elf")), a.PlayerID())
	domain := AllPermanentDomains(AnyPermanentDomain(SubtypeDomain("Wall"), SubtypeDomain("Elf")), NonSubtypeDomain("Elf"))
	binding := DeclarePermanentTarget("host", AnyPermanent, TargetWithDomain(domain))
	aura := NewAura("Aura", "", WithCastTarget(binding))
	if !auraHostIsLegal(aura, wall, g) || auraHostIsLegal(aura, elf, g) {
		t.Fatal("attachment ignored domain")
	}
	binding.spec.(*PermanentTarget).Filters = nil
	if len(binding.Possible(a.PlayerID(), SpellTargetSource(aura), g)) != 1 {
		t.Fatal("ordinary filters replaced enforced domain")
	}
	g.AddContinuousEffect(SetSubtypes(wall.ID(), SubtypeCreature, core.Indefinite, "Elf"))
	if auraHostIsLegal(aura, wall, g) || !TargetRequiresSubtype(binding, "Wall") {
		t.Fatal("live subtype or declaration changed")
	}
	if sameSpellTextTargetSpec(binding.spec, DeclarePermanentTarget("host", AnyPermanent, TargetWithDomain(SubtypeDomain("Elf"))).spec) {
		t.Fatal("text continuation ignored domain")
	}
	binding.spec.(*PermanentTarget).Filters = []PermanentFilter{AnyPermanent}
	if !sameSpellTextTargetSpec(binding.spec, DeclarePermanentTarget("host", AnyPermanent, TargetWithDomain(domain)).spec) {
		t.Fatal("same declaration rejected")
	}
}

func TestTargetDomainProofInvalid(t *testing.T) {
	for _, domain := range []PermanentTargetDomain{{}, SubtypeDomain(""), NonSubtypeDomain(" Wall"), AllPermanentDomains(), AnyPermanentDomain(), AllPermanentDomains(SubtypeDomain("Wall"), PermanentTargetDomain{})} {
		binding := DeclarePermanentTarget("bad", AnyPermanent, TargetWithDomain(domain))
		card := NewInstant("Bad", "", Spell(TapSubject(binding), WithTargets(binding)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid domain")
		}
		binding.spec = &countBoundTarget{inner: binding.spec, count: 1}
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("wrapper hid invalid domain")
		}
	}
}

func TestTargetDomainProofInputIsolation(t *testing.T) {
	children := []PermanentTargetDomain{SubtypeDomain("Wall"), SubtypeDomain("Wall")}
	domain := AnyPermanentDomain(children...)
	children[0] = UnknownPermanentDomain()
	binding := DeclarePermanentTargets("walls", 0, 2, AnyPermanent, TargetWithDomain(domain))
	if !TargetRequiresSubtype(binding, "Wall") {
		t.Fatal("constructor retained caller storage")
	}
	if TargetRequiresSubtype(DeclarePermanentTarget("unknown", AnyPermanent, TargetWithDomain(AnyPermanentDomain(SubtypeDomain("Wall"), UnknownPermanentDomain()))), "Wall") {
		t.Fatal("unknown union branch supplied proof")
	}
}
