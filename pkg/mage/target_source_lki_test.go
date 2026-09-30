package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetLegalityUsesDepartedSourceColors(t *testing.T) {
	for _, returned := range []bool{false, true} {
		for _, cloneGame := range []bool{false, true} {
			g, a, b := newSpellCopyGame()
			target := DeclareDamageTarget("recipient")
			card := NewCreature("Green source", "{G}", 2, 2, WithAbility(Activate(Costs(SacrificeSourceCost()),
				Sequence(DealDamageFrom(This, target, Fixed(2)), GainLifeFor(You, Fixed(1))), WithTargets(target))))
			source := g.PutOnBattlefield(card, a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Protected", "", 2, 9), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			victim.RuntimeAbilities = append(victim.RuntimeAbilities, ProtectionFromColor(Green))
			if returned {
				g.PutOnBattlefield(card, b.PlayerID()).FaceDown = true
			}
			original := g
			if cloneGame {
				g = g.Clone()
			}
			g.ResolveStack()
			if g.GetPlayer(a.PlayerID()).Life() != 20 || g.FindPermanent(victim.ID()).Damage != 0 {
				t.Fatalf("returned=%v clone=%v: illegal target resolved", returned, cloneGame)
			}
			if cloneGame && original.StackSize() != 1 {
				t.Fatal("clone changed original stack")
			}
		}
	}
}

func TestTargetLegalitySourceLKIColorsAndCopies(t *testing.T) {
	for _, color := range []Color{Green, Blue} {
		for _, live := range []bool{false, true} {
			g, a, b := newSpellCopyGame()
			targets := DeclareDamageTargets("recipients", 1, 2)
			card := NewCreature("Source", "{G}", 2, 2, WithAbility(Activate(Costs(ManaPayment("{0}")),
				Sequence(DealDamageEach(This, targets, Fixed(2)), GainLifeFor(You, Fixed(1))), WithTargets(targets))))
			source := g.PutOnBattlefield(card, a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID(), b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			g.ChangeObjectColor(source.ID(), color)
			if !live {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.zones.ClearLKI()
				g.PutOnBattlefield(card, b.PlayerID()).FaceDown = true
			}
			g.FindPermanent(victim.ID()).RuntimeAbilities = append(g.FindPermanent(victim.ID()).RuntimeAbilities, ProtectionFromColor(Blue))
			clone := g.Clone()
			copied := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
			clone.ResolveStackObject(copied)
			wantDamage := 2
			if color == Blue {
				wantDamage = 0
			}
			if clone.FindPermanent(victim.ID()).Damage != wantDamage || clone.GetPlayer(b.PlayerID()).Life() != 18 || clone.GetPlayer(a.PlayerID()).Life() != 21 {
				t.Fatalf("color=%v live=%v: copy lost source or independent target", color, live)
			}
			if b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 0 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()).Damage != wantDamage || b.Life() != 18 || a.Life() != 21 {
				t.Fatal("original source state changed")
			}
		}
	}
}

func TestTargetLegalitySourceLKITypesAndIdentity(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclareDamageTarget("recipient")
	card := NewArtifact("Artifact source", "", WithAbility(Activate(Costs(SacrificeSourceCost()),
		Sequence(DealDamageFrom(This, target, Fixed(2)), GainLifeFor(You, Fixed(1))), WithTargets(target))))
	source := g.PutOnBattlefield(card, a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, b.PlayerID()).FaceDown = true
	victim.RuntimeAbilities = append(victim.RuntimeAbilities, ProtectionFromCardType(TypeArtifact))
	g.ResolveStack()
	if a.Life() != 20 || victim.Damage != 0 {
		t.Fatal("departed artifact source was not protected")
	}
}

func TestTargetLegalitySourceLKIRetarget(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclareDamageTarget("recipient")
	card := NewCreature("Source", "{G}", 2, 2, WithAbility(Activate(Costs(SacrificeSourceCost()),
		DealDamageFrom(This, target, Fixed(2)), WithTargets(target))))
	source := g.PutOnBattlefield(card, a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Protected", "", 2, 9, WithAbility(ProtectionFromColor(Green))), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, b.PlayerID()).FaceDown = true
	a.chooseQueue = [][]uuid.UUID{{victim.ID()}}
	original := g.StackPeek()
	copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copied.Targets[0] != b.PlayerID() || original.Targets[0] != b.PlayerID() {
		t.Fatal("copy accepted a protected target")
	}
	g.ResolveStackObject(copied)
	if b.Life() != 18 || g.FindPermanent(victim.ID()).Damage != 0 {
		t.Fatal("copy lost legal target")
	}
}

func TestTargetLegalitySourceLKIUnrelatedAndAbsent(t *testing.T) {
	g, a, b := newSpellCopyGame()
	card := NewArtifact("Unrelated", "")
	obj := &StackObject{SourceID: uuid.New(), TargetSource: card, sourceIncarnation: uuid.New()}
	if g.resolutionTargetSource(obj) != card {
		t.Fatal("unrelated source was replaced")
	}
	if g.resolutionTargetSource(&StackObject{}) != nil {
		t.Fatal("absent source was created")
	}
	spell := NewInstant("Spell", "{G}")
	if g.resolutionTargetSource(&StackObject{SourceID: spell.ID(), Card: spell}).ID() != spell.ID() {
		t.Fatal("spell source changed")
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	obj = &StackObject{SourceID: source.ID(), TargetSource: card, sourceIncarnation: source.incarnationID}
	g.PhaseOut(source.ID())
	if g.resolutionTargetSource(obj).ID() != card.ID() || b.Life() != 20 {
		t.Fatal("phasing changed source identity")
	}
}

func TestTargetLegalityPhasedSourceColors(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclareDamageTarget("recipient")
	card := NewCreature("Source", "{G}", 2, 2, WithAbility(Activate(Costs(ManaPayment("{0}")),
		Sequence(DealDamageFrom(This, target, Fixed(2)), GainLifeFor(You, Fixed(1))), WithTargets(target))))
	source := g.PutOnBattlefield(card, a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ChangeObjectColor(source.ID(), Blue)
	g.PhaseOut(source.ID())
	g.FindPermanent(victim.ID()).RuntimeAbilities = append(g.FindPermanent(victim.ID()).RuntimeAbilities, ProtectionFromColor(Blue))
	g.ResolveStack()
	if a.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 0 {
		t.Fatal("phased source lost protection quality")
	}
}
