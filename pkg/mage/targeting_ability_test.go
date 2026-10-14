package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetingAbilityView(t *testing.T) {
	g, _, _ := newSpellCopyGame()
	wall := DeclarePermanentTarget("wall", AnyPermanent, TargetHasSubtype("Wall"))
	player := DeclarePlayerTarget("player")
	first := Activate(Costs(GenericCost(0)), TapSubject(wall), WithTargets(wall))
	second := Activate(Costs(GenericCost(0)), GainLifeFor(player, Fixed(1)), WithTargets(player))
	card := NewArtifact("Two abilities", "", WithAbility(first), WithAbility(second))
	declared := first.Targets()
	view := g.AbilityTargetSource(card, first, declared)
	declared[0] = second.Targets()[0]
	id, targets, known := TargetSourceAbility(view)
	if !known || id != first.AbilityID() || len(targets) != 1 || !slices.Equal(TargetRequiredSubtypes(targets[0]), []string{"Wall"}) {
		t.Fatalf("first declaration: %s %v %v", id, targets, known)
	}
	targets[0] = nil
	_, again, _ := TargetSourceAbility(view)
	if again[0] == nil {
		t.Fatal("query changed saved declarations")
	}
	otherID, otherTargets, ok := TargetSourceAbility(g.AbilityTargetSource(card, second, second.Targets()))
	if !ok || otherID == id || len(otherTargets) != 1 || otherTargets[0] != second.Targets()[0] {
		t.Fatal("source card conflated abilities")
	}
	empty := Activate(Costs(GenericCost(0)), GainLifeFor(You, Fixed(1)))
	emptyID, emptyTargets, ok := TargetSourceAbility(g.AbilityTargetSource(card, empty, nil))
	if !ok || emptyID != empty.AbilityID() || len(emptyTargets) != 0 {
		t.Fatal("known empty declaration lost")
	}
	for _, raw := range []Card{nil, card, SpellTargetSource(card), g.AbilityTargetSource(card, nil, nil), g.AbilityTargetSource(card, (*ActionDefinition)(nil), nil), g.AbilityTargetSource(card, &BaseAbility{abilityType: core.AbilityActivated}, nil), g.AbilityTargetSource(nil, first, first.Targets()), g.AbilityTargetSource(card, ProtectionFromColor(core.Blue), nil), g.AbilityTargetSource(card, NewAction(ActionSpell), nil)} {
		if id, targets, known := TargetSourceAbility(raw); known || id != uuid.Nil || len(targets) != 0 {
			t.Fatal("invented active ability")
		}
	}
}

func TestTargetingAbilityActivation(t *testing.T) {
	for _, change := range []string{"present", "phase", "return", "removed", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			wall := DeclarePermanentTarget("wall", AnyPermanent, TargetHasSubtype("Wall"))
			ability := Activate(Costs(GenericCost(1)), TapSubject(wall), WithTargets(wall))
			card := NewArtifact("Source", "", WithAbility(ability))
			source := g.PutOnBattlefield(card, a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Wall", "", 0, 5, WithSubTypes("Wall")), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err == nil || g.StackSize() != 0 {
				t.Fatal("unpaid activation")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			assertTargetingAbility(t, obj.TargetSource, ability.AbilityID(), 1)
			switch change {
			case "phase":
				g.PhaseOut(source.ID())
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
			case "removed":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			assertTargetingAbility(t, g.resolutionTargetSource(obj), ability.AbilityID(), 1)
			clone := g.Clone()
			copied := g.CopyStackObjectDirect(obj, a.PlayerID(), true)
			assertTargetingAbility(t, g.resolutionTargetSource(copied), ability.AbilityID(), 1)
			assertTargetingAbility(t, clone.resolutionTargetSource(clone.StackPeek()), ability.AbilityID(), 1)
			g.ResolveStack()
			if !g.FindPermanent(victim.ID()).Tapped || clone.FindPermanent(victim.ID()).Tapped {
				t.Fatal("resolution or clone isolation")
			}
		})
	}
}

func assertTargetingAbility(t *testing.T, source Card, want uuid.UUID, count int) {
	t.Helper()
	id, declared, known := TargetSourceAbility(source)
	if !known || id != want || len(declared) != count {
		t.Fatalf("ability = %s, declarations = %d, known = %v; want %s/%d", id, len(declared), known, want, count)
	}
}

func TestTargetingAbilityCurrentCharacteristics(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclarePermanentTarget("target", IsCreature)
	ability := Activate(Costs(GenericCost(1)), TapSubject(target), WithTargets(target))
	source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(ability)), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Protected", "", 1, 5, WithAbility(ProtectionFromColor(core.Blue))), b.PlayerID())
	g.ChangeObjectColor(source.ID(), core.Blue)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
		t.Fatal("ability view lost current source colors or rollback")
	}
}

func TestTargetingAbilityQueries(t *testing.T) {
	g, a, b := newSpellCopyGame()
	probe := &targetingAbilityProbe{PermanentTarget: TargetPermanent(IsCreature).(*PermanentTarget), t: t}
	ability := NewActivatedAbility(FuncEffect("tap", EffectProperties{}, func(g *Game, _, _ uuid.UUID, targets []uuid.UUID) error {
		g.TapPermanent(g.FindPermanent(targets[0]))
		return nil
	}), ManaPayment("{0}"), WithTarget(probe))
	probe.expected = ability.AbilityID()
	source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(ability)), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 5), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	announcementChecks := probe.checks
	if announcementChecks == 0 {
		t.Fatal("announcement made no query")
	}
	g.ResolveStack()
	if probe.checks <= announcementChecks || !g.FindPermanent(victim.ID()).Tapped {
		t.Fatal("resolution made no identity query")
	}
}

type targetingAbilityProbe struct {
	*PermanentTarget
	t        *testing.T
	expected uuid.UUID
	checks   int
}

func (p *targetingAbilityProbe) Possible(controller uuid.UUID, card Card, g *Game) []uuid.UUID {
	p.t.Helper()
	id, declared, known := TargetSourceAbility(card)
	if !known || id != p.expected || len(declared) != 1 || declared[0] != p {
		p.t.Fatal("target query lacks active ability declaration")
	}
	p.checks++
	return p.PermanentTarget.Possible(controller, card, g)
}

func TestTargetingAbilityPendingAndDelayed(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		g, a, b := newSpellCopyGame()
		target := DeclarePermanentTarget("wall", IsCreature, TargetHasSubtype("Wall"))
		trigger := BeginningOfUpkeepTrigger(TapSubject(target), false).AddTarget(target.targetSpec())
		source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(trigger)), a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Wall", "", 0, 5, WithSubTypes("Wall")), b.PlayerID())
		want := trigger.AbilityID()
		if delayed {
			dt := &DelayedTrigger{EventType: core.EvtUpkeep, SourceID: source.ID(), Controller: a.PlayerID(), targets: []Target{target.targetSpec()}, Effects: []Effect{TapSubject(target)}}
			g.triggers.RegisterDelayed(dt)
			want = dt.targetingAbilityID
			source.RuntimeAbilities = nil
		}
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		if len(g.triggers.Pending()) != 1 {
			t.Fatal("missing pending trigger")
		}
		clone := g.Clone()
		g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		g.PutOnBattlefield(source.Card, b.PlayerID())
		for _, game := range []*Game{g, clone} {
			game.PutTriggersOnStack()
			assertTargetingAbility(t, game.resolutionTargetSource(game.StackPeek()), want, 1)
			copied := game.CopyStackObjectDirect(game.StackPeek(), a.PlayerID(), false)
			assertTargetingAbility(t, game.resolutionTargetSource(copied), want, 1)
			game.ResolveStack()
			if !game.FindPermanent(victim.ID()).Tapped {
				t.Fatal("pending body failed")
			}
		}
	}
}

func TestTargetingAbilityGraveyard(t *testing.T) {
	g, a, b := newSpellCopyGame()
	g.SetActivePlayerIndex(0)
	g.SetStep(core.PrecombatMain)
	target := DeclarePermanentTarget("wall", IsCreature, TargetHasSubtype("Wall"))
	card := NewCreature("Source", "", 1, 1, WithGraveyardActivatedAbility(TapSubject(target), ExileSelfFromGraveyardCost(), WithTarget(target.targetSpec())))
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	if _, _, ok := g.FindGraveyardActivatableCard(a.PlayerID(), card.Name()); !ok {
		t.Fatal("missing graveyard ability")
	}
	wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 5, WithSubTypes("Wall")), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 5), b.PlayerID())
	if err := g.ActivateGraveyardAbility(a.PlayerID(), card.ID(), 0, []uuid.UUID{other.ID()}); err == nil || len(a.Graveyard()) != 1 || g.StackSize() != 0 {
		t.Fatal("illegal target paid graveyard cost")
	}
	if err := g.ActivateGraveyardAbility(a.PlayerID(), card.ID(), 0, []uuid.UUID{wall.ID()}); err != nil {
		t.Fatal(err)
	}
	ability := card.Abilities()[0]
	assertTargetingAbility(t, g.resolutionTargetSource(g.StackPeek()), ability.AbilityID(), 1)
	wall.GrantBaseAttr(core.Shroud)
	g.ResolveStack()
	if wall.Tapped {
		t.Fatal("graveyard ability failed resolution legality")
	}
}
