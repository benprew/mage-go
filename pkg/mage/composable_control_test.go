package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestChangeControllerLifetimeAndIdentity(t *testing.T) {
	for _, change := range []string{"none", "source leaves", "target phases", "target returns", "cannot change"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, target := controllerTestGame(t)
			sourceCard := NewCreature("Source", "", 1, 1)
			sourceCard.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(sourceCard, b.PlayerID())
			if change == "cannot change" {
				target.GrantBaseAttr(AttrCantChangeControl)
			}
			ref := DeclarePermanentTarget("target", IsCreature)
			ability := Spell(Until(EndOfTurn, ChangeController(ref, You)), WithTargets(ref))
			card := NewInstant("Control", "", WithAbility(ability))
			card.SetOwner(b.PlayerID())
			if err := ValidateCardDefinition("Control", card); err != nil {
				t.Fatal(err)
			}
			obj := &StackObject{ID: uuid.New(), SourceID: source.ID(), Card: card, Controller: b.PlayerID(), Targets: []uuid.UUID{target.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
			g.pushStack(obj)
			g.ResolveStackObject(obj)
			if change == "source leaves" {
				g.RemoveFromBattlefield(source)
			}
			if change == "target phases" {
				g.PhaseIn(g.PhaseOut(target.ID()))
			}
			if change == "target returns" {
				g.RemoveFromBattlefield(target)
				target = g.PutOnBattlefield(target.Card, a.PlayerID())
			}
			g.ApplyContinuousEffects()
			want := b.PlayerID()
			if change == "target returns" || change == "cannot change" {
				want = a.PlayerID()
			}
			if got := target.ControllerID(); got != want {
				t.Fatalf("controller = %s, want %s", got, want)
			}
			g.effects.RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if got := target.ControllerID(); got != a.PlayerID() {
				t.Fatalf("controller after expiry = %s, want %s", got, a.PlayerID())
			}
		})
	}
}

func TestChangeControllerTimestampAndClone(t *testing.T) {
	g, a, b, target := controllerTestGame(t)
	g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: target.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	effect := Until(EndOfTurn, ChangeController(This, You))
	if err := ApplyEffect(g, effect, target.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := target.ControllerID(); got != a.PlayerID() {
		t.Fatalf("later controller = %s", got)
	}
	clone := g.Clone()
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if got := clone.FindPermanent(target.ID()).ControllerID(); got != b.PlayerID() {
		t.Fatalf("clone controller after expiry = %s", got)
	}
	if got := target.ControllerID(); got != a.PlayerID() {
		t.Fatalf("clone changed original controller to %s", got)
	}
	if effect.Properties().Outcome != OutcomeDetriment {
		t.Fatal("control change lost detrimental metadata")
	}
}

func TestChangeControllerCopiedSpellUsesCopyController(t *testing.T) {
	g, a, b, target := controllerTestGame(t)
	ref := DeclarePermanentTarget("target", IsCreature)
	ability := Spell(Until(EndOfTurn, ChangeController(ref, You)), WithTargets(ref))
	card := NewInstant("Copy Control", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{target.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	copyObj := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	g.ResolveStackObject(copyObj)
	if got := target.ControllerID(); got != b.PlayerID() {
		t.Fatalf("copy controller = %s, want %s", got, b.PlayerID())
	}
	g.ResolveStackObject(obj)
	if got := target.ControllerID(); got != a.PlayerID() {
		t.Fatalf("original controller = %s, want %s", got, a.PlayerID())
	}
}

func TestChangeControllerValidation(t *testing.T) {
	target := DeclarePermanentTarget("target", IsCreature)
	player := DeclarePlayerTarget("player")
	for _, tc := range []struct {
		name     string
		body     ResolvingEffect
		declared []TargetBinding
		want     string
	}{
		{"nil subject", Until(EndOfTurn, ChangeController(nil, You)), nil, ".subject"},
		{"nil controller", Until(EndOfTurn, ChangeController(target, nil)), []TargetBinding{target}, ".controller"},
		{"undeclared controller", Until(EndOfTurn, ChangeController(target, player)), []TargetBinding{target}, ".controller"},
		{"wrong duration", Until(EndOfCombat, ChangeController(target, You)), []TargetBinding{target}, "requires Until(EndOfTurn)"},
		{"source duration", UntilSourceLeaves(This, ChangeController(target, You)), []TargetBinding{target}, "requires Until(EndOfTurn)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewInstant("Invalid Control", "", WithAbility(Spell(tc.body, WithTargets(tc.declared...))))
			if err := ValidateCardDefinition("Invalid Control", card); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validation error = %v, want %q", err, tc.want)
			}
		})
	}
	static := NewCreature("Invalid Static", "", 1, 1, WithAbility(Static(While(Controls(IsCreature), ChangeController(This, You)))))
	if err := ValidateCardDefinition("Invalid Static", static); err == nil || !strings.Contains(err.Error(), "requires Until(EndOfTurn)") {
		t.Fatalf("static validation error = %v", err)
	}
}
