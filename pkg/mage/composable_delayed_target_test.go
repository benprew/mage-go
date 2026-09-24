package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDelayedNewTargetsAtPlacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePermanentTarget("future creature", IsCreature)
	player := BindPlayer("future player")
	body := AtNext(TurnStepBegins(EachEndStep, player), Sequence(Destroy(target), GainLifeFor(FromPlayer(player), Fixed(2))), WithTargets(target))
	card := NewSorcery("Schedule", "", Spell(body))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, body, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	if g.StackSize() != 0 {
		t.Fatal("selected targets before priority")
	}
	creature := NewCreature("Late entry", "", 2, 2)
	creature.SetOwner(b.PlayerID())
	g.PutOnBattlefield(creature, b.PlayerID())
	clone := g.Clone()
	clone.PutTriggersOnStack()
	if clone.StackSize() != 1 || len(clone.StackPeek().TargetSpecs) != 1 || clone.StackPeek().Targets[0] != creature.ID() {
		t.Fatal("missing delayed target")
	}
	clone.ResolveStack()
	if clone.FindPermanent(creature.ID()) != nil || clone.GetPlayer(b.PlayerID()).Life() != 22 || g.FindPermanent(creature.ID()) == nil || b.Life() != 20 {
		t.Fatal("target, event capture, or clone isolation failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || len(g.DelayedTriggers()) != 0 {
		t.Fatal("unavailable required target did not consume the one-shot trigger")
	}
	g.PutOnBattlefield(creature, b.PlayerID())
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.ResolveStack()
	if g.FindPermanent(creature.ID()) == nil {
		t.Fatal("targetless trigger retried")
	}
}

func TestDelayedNewTargetValidation(t *testing.T) {
	target := DeclarePlayerTarget("future target")
	for _, tt := range []struct {
		body ResolvingEffect
		want string
	}{
		{AtNext(EachEndStep, GainLifeFor(target, Fixed(1)), WithTargets(target)), ""},
		{AtNext(EachEndStep, GainLifeFor(target, Fixed(1))), "declaration"},
		{Sequence(AtNext(EachEndStep, GainLifeFor(target, Fixed(1)), WithTargets(target)), GainLifeFor(target, Fixed(1))), "declaration"},
		{AtNext(EachEndStep, GainLifeFor(target, Fixed(1)), WithTargets(target, target)), "once"},
	} {
		c := NewSorcery("Schedule", "", Spell(tt.body))
		err := ValidateCardDefinition(c.Name(), c)
		if tt.want == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("validation=%v, want %s", err, tt.want)
		}
	}
}

func TestDelayedNewTargetLegality(t *testing.T) {
	for _, change := range []string{"return", "phase out", "phase in", "shroud", "partial", "optional"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("creature", IsCreature)
			recipient := DeclarePlayerTarget("player", TargetOpponentOnly())
			effect := AtNext(EachEndStep, Sequence(Destroy(target), GainLifeFor(You, Fixed(2))), WithTargets(target))
			if change == "partial" {
				effect = AtNext(EachEndStep, Sequence(Destroy(target), GainLifeFor(recipient, Fixed(3))), WithTargets(target, recipient))
			}
			if change == "optional" {
				targets := DeclarePermanentTargets("optional", 0, 1, IsCreature)
				effect = AtNext(EachEndStep, GainLifeFor(You, Fixed(2)), WithTargets(targets))
			}
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			card := NewCreature("Creature", "", 2, 2)
			card.SetOwner(b.PlayerID())
			if change != "optional" {
				g.PutOnBattlefield(card, b.PlayerID())
			}
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("missing trigger")
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
			case "phase out", "partial":
				g.PhaseOut(card.ID())
			case "phase in":
				ids := g.PhaseOut(card.ID())
				g.PhaseIn(ids)
			case "shroud":
				g.MutablePermanent(card.ID()).GrantBaseAttr(Shroud)
			}
			g.ResolveStack()
			wantA, wantB := 20, 20
			if change == "phase in" || change == "optional" {
				wantA = 22
			}
			if change == "partial" {
				wantB = 23
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life=%d/%d", a.Life(), b.Life())
			}
			if change == "phase in" && g.FindPermanent(card.ID()) != nil {
				t.Fatal("phasing invalidated target")
			}
			if change == "return" && g.FindPermanent(card.ID()) == nil {
				t.Fatal("target followed returned card")
			}
		})
	}
}

func TestDelayedNewTargetCopyAndSavedFrame(t *testing.T) {
	a, b := newRecordingPlayer("A"), newRecordingPlayer("B")
	g := NewGame(a, b)
	earlier := DeclarePlayerTarget("earlier target")
	future := DeclarePlayerTarget("future target")
	amount := BindNumber("amount")
	effect := Sequence(SnapshotNumber(Fixed(3), amount), AtNext(EachEndStep, Sequence(
		GainLifeFor(earlier, Fixed(1)), GainLifeFor(future, FromNumber(amount)), GainLifeFor(You, Fixed(2)),
	), WithTargets(future)))
	card := NewSorcery("Schedule", "", Spell(effect, WithTargets(earlier)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.resolution.targetSpecs = []Target{earlier.targetDeclaration}
	if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.resolution.targetSpecs = nil
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	g.PutTriggersOnStack()
	original := g.StackPeek()
	b.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
	copied := g.CopyStackObjectDirect(original, b.PlayerID(), true)
	if copied == nil || copied.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() {
		t.Fatal("copy targets not independent")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("stack clone leaked")
	}
	g.ResolveStack()
	for _, game := range []*Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).Life() != 27 || game.GetPlayer(b.PlayerID()).Life() != 25 {
			t.Fatal("copy lost saved target, number, or controller")
		}
	}
}

func TestDelayedNewTargetNestedCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("future opponent", TargetOpponentOnly())
	effect := AtNext(EachUpkeep, AtNext(EachEndStep, GainLifeFor(target, Fixed(4))), WithTargets(target))
	source := NewCreature("Source", "", 1, 1, WithAbility(Activate(Costs(), effect)))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.PhaseOut(source.ID())
	registered := g.Clone()
	for _, game := range []*Game{g, registered} {
		game.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: b.PlayerID()})
		game.ResolveStack()
		if len(game.DelayedTriggers()) != 1 || game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatal("nested delay resolved early")
		}
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
		game.ResolveStack()
		if game.GetPlayer(b.PlayerID()).Life() != 24 || game.GetPlayer(a.PlayerID()).Life() != 20 {
			t.Fatal("lost delayed controller or nested target")
		}
	}
}
