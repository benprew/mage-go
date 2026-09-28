package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastTargetsCreatureUsesCurrentTargetAndExactSpell(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByController().OfInstantOrSorcery().TargetsCreature(),
		GainLifeFor(You, Fixed(1))))), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
	other := g.PutOnBattlefield(NewArtifact("Other", ""), b.PlayerID())
	card := NewInstant("Spell", "")
	card.SetOwner(a.PlayerID())
	spell := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{other.ID(), creature.ID()}}
	g.pushStack(spell)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: spell.ID, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want spell and trigger", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if a.Life() != 20 {
		t.Fatal("clone changed original player")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.RemoveFromBattlefield(creature)
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if a.Life() != 22 {
		t.Fatalf("pending triggers lost event result: life=%d", a.Life())
	}
	g.PhaseOut(watcher.ID())
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: spell.ID, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("phased watcher observed cast")
	}
}

func TestSpellCastTargetsCreatureDeclaration(t *testing.T) {
	pattern := SpellCast().OfInstantOrSorcery().TargetsCreature()
	if got := InspectDefinition(pattern); !strings.Contains(got.Text, "targets a creature") {
		t.Fatalf("inspection=%+v", got)
	}
	valid := NewArtifact("Valid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	invalid := NewArtifact("Invalid", "", WithAbility(WhenEvent(pattern.TargetsCreature(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("repeated target filter accepted")
	}
	for _, bad := range []EventPattern{pattern.OfType(TypeInstant), pattern.OfInstantOrSorcery()} {
		card := NewArtifact("Bad type", "", WithAbility(WhenEvent(bad, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("repeated type filter accepted")
		}
	}
	for _, bad := range []EventPattern{pattern.ByController().ByOpponent(), pattern.ByOpponent().ByOpponent()} {
		card := NewArtifact("Bad caster", "", WithAbility(WhenEvent(bad, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("conflicting caster filter accepted")
		}
	}
}

func TestSpellCastTargetsCreatureCurrentState(t *testing.T) {
	for _, tc := range []struct {
		name        string
		spellType   CardType
		targetType  CardType
		remove      bool
		phase       bool
		returnSame  bool
		zeroTargets bool
		otherCaster bool
		want        bool
	}{
		{name: "instant creature", spellType: TypeInstant, targetType: TypeCreature, want: true},
		{name: "sorcery creature", spellType: TypeSorcery, targetType: TypeCreature, want: true},
		{name: "artifact spell", spellType: TypeArtifact, targetType: TypeCreature},
		{name: "artifact target", spellType: TypeInstant, targetType: TypeArtifact},
		{name: "no chosen target", spellType: TypeInstant, targetType: TypeCreature, zeroTargets: true},
		{name: "departed target", spellType: TypeInstant, targetType: TypeCreature, remove: true},
		{name: "phased target", spellType: TypeInstant, targetType: TypeCreature, phase: true},
		{name: "returned target", spellType: TypeInstant, targetType: TypeCreature, returnSame: true},
		{name: "other caster", spellType: TypeInstant, targetType: TypeCreature, otherCaster: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(WhenEvent(
				SpellCast().ByController().OfInstantOrSorcery().TargetsCreature(), GainLifeFor(You, Fixed(1))))), a.PlayerID())
			var target *Permanent
			if tc.targetType == TypeCreature {
				target = g.PutOnBattlefield(NewCreature("Target", "", 2, 2), b.PlayerID())
			} else {
				target = g.PutOnBattlefield(NewArtifact("Target", ""), b.PlayerID())
			}
			var card Card
			switch tc.spellType {
			case TypeInstant:
				card = NewInstant("Spell", "")
			case TypeSorcery:
				card = NewSorcery("Spell", "")
			default:
				card = NewArtifact("Spell", "")
			}
			card.SetOwner(a.PlayerID())
			var targets []uuid.UUID
			if !tc.zeroTargets {
				targets = []uuid.UUID{target.ID()}
			}
			caster := a.PlayerID()
			if tc.otherCaster {
				caster = b.PlayerID()
			}
			spell := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: caster, Targets: targets}
			g.pushStack(spell)
			if tc.remove || tc.returnSame {
				g.RemoveFromBattlefield(target)
			}
			if tc.returnSame {
				g.PutOnBattlefield(target.Card, b.PlayerID())
			}
			if tc.phase {
				g.PhaseOut(target.ID())
			}
			g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: spell.ID, PlayerID: caster})
			g.PutTriggersOnStack()
			if got := g.StackSize() == 2; got != tc.want {
				t.Fatalf("triggered=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestSpellCastTargetsCreatureRealCastAndFailedPayment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByController().OfInstantOrSorcery().TargetsCreature(),
		GainLifeFor(You, Fixed(1))))), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
	target := DeclarePermanentTarget("creature", IsCreature)
	unpaid := NewInstant("Unpaid", "{U}", WithAbility(Spell(GainLifeFor(You, Fixed(1)), WithTargets(target))))
	unpaid.SetOwner(a.PlayerID())
	a.AddToHand(unpaid)
	if err := g.CastSpellByName(a.PlayerID(), unpaid.Name(), []uuid.UUID{creature.ID()}); err == nil {
		t.Fatal("unpaid spell was cast")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("failed cast produced a trigger")
	}
	free := NewSorcery("Free", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)), WithTargets(target))))
	free.SetOwner(a.PlayerID())
	a.AddToHand(free)
	g.turns.SetStep(PrecombatMain)
	if err := g.CastSpellByName(a.PlayerID(), free.Name(), []uuid.UUID{creature.ID()}); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("successful cast stack=%d, want two", g.StackSize())
	}
}

func TestSpellCastTargetsCreatureUsesCastStackObject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().TargetsCreature(), GainLifeFor(You, Fixed(1))))), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
	artifact := g.PutOnBattlefield(NewArtifact("Artifact", ""), b.PlayerID())
	card := NewInstant("Shared", "")
	card.SetOwner(a.PlayerID())
	old := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{creature.ID()}}
	current := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{artifact.ID()}}
	g.pushStack(old)
	g.pushStack(current)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: current.ID, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("cast event matched another stack object with the same source card")
	}
}
