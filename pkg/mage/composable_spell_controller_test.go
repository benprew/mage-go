package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestControllerOfSpellPaymentUsesStackController(t *testing.T) {
	a := NewBasePlayer("A")
	b := &acceptingMayPlayer{NewBasePlayer("B")}
	g := NewGame(a, b)
	victim := stackTestSpell("Victim", a.PlayerID())
	target := &StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
	g.pushStack(target)
	spell := DeclareSpellTarget("spell")
	body := UnlessPaid(ControllerOfSpell(spell), Costs(LifePayCost(3)), Counter(spell))
	ability := Spell(body, WithTargets(spell))
	card := NewInstant("Test", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Test", card); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{target.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.ResolveTopOfStack()
	if got := b.Life(); got != 17 || a.Life() != 20 || g.stack.FindByID(target.ID) == nil {
		t.Fatalf("payment used wrong player: A=%d B=%d targetPresent=%t", a.Life(), got, g.stack.FindByID(target.ID) != nil)
	}
}

func TestControllerOfSpellAbsenceCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	firstCard := stackTestSpell("First", a.PlayerID())
	secondCard := stackTestSpell("Second", b.PlayerID())
	first := &StackObject{ID: uuid.New(), SourceID: firstCard.ID(), Card: firstCard, Controller: a.PlayerID()}
	second := &StackObject{ID: uuid.New(), SourceID: secondCard.ID(), Card: secondCard, Controller: b.PlayerID()}
	g.pushStack(first)
	g.pushStack(second)
	spell := DeclareSpellTarget("spell")
	ref := ControllerOfSpell(spell)
	ctx := &EffectContext{Game: g, targetBindings: map[*targetDeclaration][]capturedPermanent{spell.targetDeclaration: {{id: first.ID}}}}
	if got, ok := ref.resolvePlayer(ctx); !ok || got != a.PlayerID() {
		t.Fatalf("first controller = %v, %t", got, ok)
	}
	clone := g.Clone()
	cloneCtx := *ctx
	cloneCtx.Game = clone
	cloneCtx.targetBindings = map[*targetDeclaration][]capturedPermanent{spell.targetDeclaration: {{id: second.ID}}}
	if got, ok := ref.resolvePlayer(&cloneCtx); !ok || got != b.PlayerID() {
		t.Fatalf("clone controller = %v, %t", got, ok)
	}
	if got, ok := ref.resolvePlayer(ctx); !ok || got != a.PlayerID() {
		t.Fatalf("original controller changed = %v, %t", got, ok)
	}
	g.stack.RemoveByID(first.ID)
	recast := &StackObject{ID: uuid.New(), SourceID: firstCard.ID(), Card: firstCard, Controller: b.PlayerID()}
	g.pushStack(recast)
	if _, ok := ref.resolvePlayer(ctx); ok {
		t.Fatal("old stack reference resolved after same card was cast again")
	}
}

func TestControllerOfSpellCopiedRetargetedFrame(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	firstCard := stackTestSpell("First", a.PlayerID())
	secondCard := stackTestSpell("Second", b.PlayerID())
	first := &StackObject{ID: uuid.New(), SourceID: firstCard.ID(), Card: firstCard, Controller: a.PlayerID()}
	second := &StackObject{ID: uuid.New(), SourceID: secondCard.ID(), Card: secondCard, Controller: b.PlayerID()}
	g.pushStack(first)
	g.pushStack(second)
	spell := DeclareSpellTarget("spell")
	ability := Spell(GainLifeFor(ControllerOfSpell(spell), Fixed(1)), WithTargets(spell))
	card := NewInstant("Reward controller", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	copyObj := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	copyObj.Targets[0] = second.ID
	copyObj.TargetZones[second.ID] = g.zoneOfTarget(second.ID)
	g.ResolveTopOfStack()
	if a.Life() != 20 || b.Life() != 21 || obj.Targets[0] != first.ID {
		t.Fatal("copied stack frame did not use its retargeted spell controller")
	}
	g.ResolveTopOfStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatal("original stack frame lost its spell controller")
	}
}

func TestControllerOfSpellTargetLegalityAtResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewArtifact("Victim", "")
	victim.SetOwner(b.PlayerID())
	target := &StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
	g.pushStack(target)
	allowed := true
	filter := NewCardFilter("artifact", func(card Card) bool { return allowed && card.HasType(core.TypeArtifact) })
	spell := DeclareSpellTarget("spell", filter)
	ability := Spell(GainLifeFor(ControllerOfSpell(spell), Fixed(2)), WithTargets(spell))
	card := NewInstant("Test", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{target.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	allowed = false
	g.ResolveTopOfStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("illegal spell target supplied a controller at resolution")
	}
}

func TestControllerOfSpellDeclarationValidation(t *testing.T) {
	spell := DeclareSpellTarget("spell")
	for _, tc := range []struct {
		name    string
		ref     PlayerRef
		targets []TargetBinding
		want    string
	}{
		{"nil", ControllerOfSpell(nil), nil, ".spell"},
		{"undeclared", ControllerOfSpell(spell), nil, ".spell"},
		{"valid", ControllerOfSpell(spell), []TargetBinding{spell}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewInstant("Test", "", WithAbility(Spell(GainLifeFor(tc.ref, Fixed(1)), WithTargets(tc.targets...))))
			err := ValidateCardDefinition("Test", card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation error = %v, want %q", err, tc.want)
			}
		})
	}
}
