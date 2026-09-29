package mage

import (
	"errors"
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// --- from composable_card_type_branch_test.go ---
func TestSavedCardTypeBranch(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	land := NewLand("Forest")
	artifact := NewArtifact("Relic", "")
	a.AddToLibrary(land)
	b := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	check := Sequence(
		DrawFor(You, Fixed(1), DrawCardInto(b)),
		IfCardHasType(FromCard(b), TypeLand, GainLifeFor(You, Fixed(2))),
		IfCardLacksType(FromCard(b), TypeLand, GainLifeFor(You, Fixed(3))),
	)
	if err := check.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := a.Life(); got != 22 {
		t.Fatalf("land life = %d", got)
	}
	a.AddToLibrary(artifact)
	if err := check.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := a.Life(); got != 25 {
		t.Fatalf("nonland life = %d", got)
	}
	if err := check.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := a.Life(); got != 25 {
		t.Fatalf("absent card matched negative condition: %d", got)
	}
}

func TestSavedCardTypeBranchVisitAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewLand("Forest")
	a.AddToLibrary(card)
	b := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(You, Fixed(1), DrawCardInto(b)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if _, ok := a.RemoveFromHand(card.ID()); !ok {
		t.Fatal("missing card")
	}
	a.AddToHand(card)
	branch := IfCardHasType(FromCard(b), TypeLand, GainLifeFor(You, Fixed(2)))
	if err := branch.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := a.Life(); got != 20 {
		t.Fatalf("old visit matched: %d", got)
	}
	if err := branch.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 22 {
		t.Fatalf("clone life = %d", got)
	}
}

func TestSavedCardTypeBranchDeclaration(t *testing.T) {
	b := BindCard("drawn")
	inner := BindCard("inner")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid branch",
			Card: NewInstant("Valid", "", Spell(Sequence(DrawFor(You, Fixed(1), DrawCardInto(b)), IfCardLacksType(FromCard(b), TypeLand, DiscardCard(FromCard(b)))))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(IfCardLacksType(FromCard(b), TypeLand, DiscardCard(FromCard(b)))); got.Role != "resolution-branch" {
					t.Fatalf("role = %s", got.Role)
				}
			},
		},
		{
			Name:          "nil card source",
			Card:          NewInstant("Invalid", "", Spell(IfCardHasType(nil, TypeLand, GainLifeFor(You, Fixed(1))))),
			WantErrSubstr: "card",
		},
		{
			Name:          "unbound producer",
			Card:          NewInstant("Invalid", "", Spell(IfCardLacksType(FromCard(b), TypeLand, GainLifeFor(You, Fixed(1))))),
			WantErrSubstr: "producer",
		},
		{
			Name:          "invalid card type",
			Card:          NewInstant("Invalid", "", Spell(IfCardHasType(FromCard(b), CardType(-1), GainLifeFor(You, Fixed(1))))),
			WantErrSubstr: "type",
		},
		{
			Name:          "nil then effect",
			Card:          NewInstant("Invalid", "", Spell(IfCardHasType(FromCard(b), TypeLand, nil))),
			WantErrSubstr: "then",
		},
		{
			Name:          "unbound inner producer",
			Card:          NewInstant("Invalid", "", Spell(Sequence(DrawFor(You, Fixed(1), DrawCardInto(b)), IfCardHasType(FromCard(b), TypeLand, DrawFor(You, Fixed(1), DrawCardInto(inner))), DiscardCard(FromCard(inner))))),
			WantErrSubstr: "producer",
		},
	})
}

func TestCardTypeBranchGraveyardTargetVisit(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	land := NewLand("Forest")
	a.AddToGraveyard(land)
	target := DeclareGraveyardCardTarget("card")
	ability := Spell(IfCardHasType(target, TypeLand, GainLifeFor(You, Fixed(2))), WithTargets(target))
	spell := NewInstant("Check", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{land.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.ResolveTopOfStack()
	if got := a.Life(); got != 22 {
		t.Fatalf("legal target life = %d", got)
	}
	obj = &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{land.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	moved, ok := g.MoveFromGraveyard(a.PlayerID(), land.ID(), ZoneHand)
	if !ok {
		t.Fatal("move from graveyard failed")
	}
	a.AddToHand(moved)
	returned, ok := a.RemoveFromHand(land.ID())
	if !ok {
		t.Fatal("remove from hand failed")
	}
	a.AddToGraveyard(returned)
	g.ResolveTopOfStack()
	if got := a.Life(); got != 22 {
		t.Fatalf("returned target life = %d", got)
	}
}

// --- from composable_cast_condition_test.go ---
func TestControlledCreatureSubtypeAtCastSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name        string
		before      bool
		noncreature bool
		phaseBefore bool
		after       bool
		want        int
	}{
		{name: "controlled at cast, gone at resolution", before: true, want: 23},
		{name: "absent at cast, present at resolution", after: true, want: 20},
		{name: "phased at cast", before: true, phaseBefore: true, want: 20},
		{name: "noncreature subtype at cast", before: true, noncreature: true, want: 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			spell := NewInstant("Test roar", "", Spell(IfCast(ControlledCreatureSubtypeAtCast("Dragon"), GainLifeFor(You, Fixed(3)))))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			var dragon *BaseCard
			if tc.before {
				if tc.noncreature {
					dragon = NewEnchantment("Dragon enchantment", "", WithSubTypes("Dragon"))
				} else {
					dragon = NewCreature("Dragon", "", 2, 2, WithSubTypes("Dragon"))
				}
				g.PutOnBattlefield(dragon, a.PlayerID())
				if tc.phaseBefore {
					g.PhaseOut(dragon.ID())
				}
			}
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
				t.Fatal(err)
			}
			if tc.before && !tc.phaseBefore {
				g.RemoveFromBattlefield(g.FindPermanent(dragon.ID()))
			}
			if tc.after {
				g.PutOnBattlefield(NewCreature("Late dragon", "", 2, 2, WithSubTypes("Dragon")), a.PlayerID())
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if a.Life() != tc.want || clone.GetPlayer(a.PlayerID()).Life() != tc.want {
				t.Fatalf("life=%d clone=%d want=%d", a.Life(), clone.GetPlayer(a.PlayerID()).Life(), tc.want)
			}
		})
	}
}

func TestControlledCreatureSubtypeAtCastDeclaration(t *testing.T) {
	condition := ControlledCreatureSubtypeAtCast("Dragon")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid card",
			Card: NewInstant("Valid", "", Spell(IfCast(condition, DealDamageFrom(This, You, Fixed(1))))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(condition); got.Role != "cast-condition" || !strings.Contains(got.Text, "Dragon") {
					t.Fatalf("inspection=%+v", got)
				}
			},
		},
		{
			Name:    "empty subtype",
			Card:    NewInstant("Invalid", "", Spell(IfCast(ControlledCreatureSubtypeAtCast(""), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "non-creature subtype Aura",
			Card:    NewInstant("Invalid", "", Spell(IfCast(ControlledCreatureSubtypeAtCast("Aura"), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:          "activation context error",
			Card:          NewArtifact("Invalid context", "", WithAbility(Activate(Costs(), IfCast(condition, GainLifeFor(You, Fixed(1)))))),
			WantErrSubstr: "spell ability",
		},
	})
}

func TestControlledCreatureSubtypeAtCastCopyAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	dragon := NewCreature("Dragon", "", 2, 2, WithSubTypes("Dragon"))
	g.PutOnBattlefield(dragon, a.PlayerID())
	spell := NewInstant("Snapshot", "", Spell(IfCast(ControlledCreatureSubtypeAtCast("Dragon"), GainLifeFor(You, Fixed(2)))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(dragon.ID())
	original := g.StackObjects()[0]
	spellCopy := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	clone := g.Clone()
	spellCopy.CastContext.ControllerCreatureSubtypesAtCast["Dragon"] = false
	if !original.CastContext.ControllerCreatureSubtypesAtCast["Dragon"] {
		t.Fatal("copy changed original snapshot")
	}
	if !clone.StackObjects()[0].CastContext.ControllerCreatureSubtypesAtCast["Dragon"] {
		t.Fatal("copy changed clone snapshot")
	}
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone resolution changed original")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatalf("life A=%d B=%d", a.Life(), b.Life())
	}
}

// --- from composable_control_since_turn_test.go ---
func TestControlledSinceTurnStartState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Launcher", "")
	card.SetOwner(a.PlayerID())
	g.SetTurn(2)
	p := g.PutOnBattlefield(card, a.PlayerID())
	ctx := StateContext{SourceID: p.ID(), SourceIncarnation: p.incarnationID, ControllerID: a.PlayerID()}
	condition := ControlledSinceTurnStart(This)
	if condition.checkState(g, ctx) {
		t.Fatal("new permanent passed continuous control")
	}
	g.SetTurn(3)
	if !condition.checkState(g, ctx) || !condition.checkState(g.Clone(), ctx) {
		t.Fatal("continuous control was not retained")
	}
	if condition.checkState(g, StateContext{SourceID: p.ID(), SourceIncarnation: p.incarnationID, ControllerID: b.PlayerID()}) {
		t.Fatal("another controller passed continuous control")
	}
	p.PhasedOut = true
	if condition.checkState(g, ctx) {
		t.Fatal("phased source passed continuous control")
	}
	p.PhasedOut = false
	if err := ApplyEffect(g, GainControl(), b.PlayerID(), b.PlayerID(), []uuid.UUID{p.ID()}); err != nil {
		t.Fatal(err)
	}
	if condition.checkState(g, ctx) {
		t.Fatal("former controller passed after control change")
	}
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(card, a.PlayerID())
	if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: card.ID(), ControllerID: a.PlayerID()}) {
		t.Fatal("returned permanent inherited continuous control")
	}
}

func TestControlledSinceTurnStartDefinition(t *testing.T) {
	condition := ControlledSinceTurnStart(This)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid artifact",
			Card: NewArtifact("Launcher", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), ActivationIf(condition)))),
			Inspect: func(t *testing.T) {
				if meta := InspectDefinition(condition); meta.Role != "state-condition" || meta.Text == "" {
					t.Fatalf("metadata = %+v", meta)
				}
			},
		},
		{
			Name:    "nil source",
			Card:    NewArtifact("Invalid", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), ActivationIf(ControlledSinceTurnStart(nil))))),
			WantErr: true,
		},
		{
			Name:    "unbound permanent source",
			Card:    NewArtifact("Invalid", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), ActivationIf(ControlledSinceTurnStart(From(BindPermanent("missing"))))))),
			WantErr: true,
		},
		{
			Name:          "spell validation error",
			Card:          NewInstant("Invalid spell", "", Spell(IfState(condition, GainLifeFor(You, Fixed(1))))),
			WantErrSubstr: "battlefield",
		},
	})
}

func TestControlledSinceTurnStartActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(2)
	card := NewArtifact("Launcher", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), ActivationIf(ControlledSinceTurnStart(This)))))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err == nil || g.StackSize() != 0 {
		t.Fatal("new permanent allowed activation")
	}
	g.SetTurn(3)
	clone := g.Clone()
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("activation or clone isolation failed")
	}
	if err := ApplyEffect(g, GainControl(), b.PlayerID(), b.PlayerID(), []uuid.UUID{card.ID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), card.ID(), 0, nil); err == nil || g.StackSize() != 0 {
		t.Fatal("new controller allowed activation")
	}
}

func TestControlledSinceTurnStartAttachedPermanent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(2)
	host := NewCreature("Host", "", 2, 2)
	aura := NewAura("Watcher", "")
	g.PutOnBattlefield(host, a.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	ctx := StateContext{SourceID: aura.ID(), SourceIncarnation: g.FindPermanent(aura.ID()).incarnationID, ControllerID: a.PlayerID()}
	condition := ControlledSinceTurnStart(AttachedPermanent)
	if condition.checkState(g, ctx) {
		t.Fatal("new host passed continuous control")
	}
	g.SetTurn(3)
	if !condition.checkState(g, ctx) {
		t.Fatal("controlled host did not match")
	}
	if err := ApplyEffect(g, GainControl(), b.PlayerID(), b.PlayerID(), []uuid.UUID{host.ID()}); err != nil {
		t.Fatal(err)
	}
	if condition.checkState(g, ctx) {
		t.Fatal("opponent-controlled host matched")
	}
	valid := NewAura("Valid", "", WithAbility(Static(While(condition, GrantKeywordModification(This, Shroud)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
}

// --- from composable_delayed_condition_test.go ---
func TestDelayedInterveningIfTiming(t *testing.T) {
	for _, keep := range []bool{false, true} {
		t.Run(map[bool]string{false: "false at resolution", true: "true at resolution"}[keep], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			effect := AtNext(YourEndStep, GainLifeFor(You, Fixed(3)), WithInterveningIf(Controls(IsArtifact)))
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 || len(g.DelayedTriggers()) != 1 {
				t.Fatal("false condition consumed the delay")
			}
			artifact := NewArtifact("Artifact", "")
			g.PutOnBattlefield(artifact, a.PlayerID())
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 || len(g.DelayedTriggers()) != 1 {
				t.Fatal("ignored step filter")
			}
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
			if g.StackSize() != 0 || len(g.DelayedTriggers()) != 0 {
				t.Fatal("incorrect pending lifecycle")
			}
			g.RemoveFromBattlefield(g.FindPermanent(artifact.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("condition was checked again at placement")
			}
			if keep {
				g.PutOnBattlefield(artifact, a.PlayerID())
			}
			g.ResolveStack()
			want := 20
			if keep {
				want = 23
			}
			if a.Life() != want {
				t.Fatalf("life=%d, want %d", a.Life(), want)
			}
			if !keep {
				g.PutOnBattlefield(artifact, a.PlayerID())
			}
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
			g.ResolveStack()
			if a.Life() != want {
				t.Fatal("consumed trigger retried")
			}
		})
	}
}

func TestDelayedInterveningIfCopyAndClone(t *testing.T) {
	a, b := newRecordingPlayer("A"), newRecordingPlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("future player")
	amount := BindNumber("saved amount")
	target := DeclarePlayerTarget("future target")
	effect := Sequence(SnapshotNumber(Fixed(3), amount), AtNext(TurnStepBegins(EachEndStep, player),
		Sequence(GainLifeFor(target, FromNumber(amount)), GainLifeFor(FromPlayer(player), Fixed(2))),
		WithTargets(target), WithInterveningIf(Controls(IsArtifact))))
	source := NewCreature("Source", "", 1, 1, WithAbility(Activate(Costs(), effect)))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	artifact := NewArtifact("Artifact", "")
	g.PutOnBattlefield(artifact, a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.PhaseOut(source.ID())
	registered := g.Clone()
	registered.RemoveFromBattlefield(registered.FindPermanent(artifact.ID()))
	registered.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	registered.PutTriggersOnStack()
	if registered.StackSize() != 0 || len(registered.DelayedTriggers()) != 1 {
		t.Fatal("registered clone ignored its condition")
	}
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	pending := g.Clone()
	pending.RemoveFromBattlefield(pending.FindPermanent(artifact.ID()))
	pending.ResolveStack()
	if pending.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("pending clone ignored resolution condition")
	}
	a.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("missing delayed trigger")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("missing copy")
	}
	stackClone := g.Clone()
	stackClone.PutOnBattlefield(NewArtifact("Copy controller artifact", ""), b.PlayerID())
	stackClone.ResolveStack()
	if stackClone.GetPlayer(b.PlayerID()).Life() != 30 {
		t.Fatal("copy lost condition, target, or saved bindings")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 25 {
		t.Fatal("copy used original controller or clone leaked")
	}
}

func TestDelayedInterveningIfInvalidOptions(t *testing.T) {
	for _, options := range [][]DelayedStepOption{
		{WithInterveningIf(nil)},
		{WithInterveningIf(Controls(IsArtifact)), WithInterveningIf(Controls(IsCreature))},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid condition accepted")
				}
			}()
			AtNext(EachEndStep, GainLifeFor(You, Fixed(1)), options...)
		}()
	}
}

func TestDelayedInterveningIfValidation(t *testing.T) {
	effect := AtNext(EachEndStep, GainLifeFor(You, Fixed(1)), WithInterveningIf(AllConditions()))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "empty all conditions",
			Card:          NewSorcery("Invalid delayed condition", "", Spell(effect)),
			WantErrSubstr: "condition",
		},
	})
}

// --- from composable_delayed_leave_test.go ---
func TestWhenLeavesThisTurnDestinationsAndIncarnation(t *testing.T) {
	for _, destination := range []string{"graveyard", "hand", "exile"} {
		t.Run(destination, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := g.PutOnBattlefield(NewArtifact("Watcher", ""), a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
			bound := BindPermanent("victim")
			target := DeclarePermanentTarget("target", IsCreature)
			g.resolution.targetSpecs = []Target{target.targetDeclaration}
			effect := Sequence(Snapshot(target, bound), WhenLeavesThisTurn(This, DestroyWithoutRegeneration(From(bound))))
			if err := ApplyEffect(g, effect, watcher.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			phased := g.PhaseOut(watcher.ID())
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("phasing queued the delayed trigger")
			}
			g.PhaseIn(phased)
			clone := g.Clone()
			switch destination {
			case "graveyard":
				g.PutPermanentIntoGraveyard(g.FindPermanent(watcher.ID()))
			case "hand":
				g.BouncePermanentToHand(g.FindPermanent(watcher.ID()))
			case "exile":
				g.ExilePermanent(g.FindPermanent(watcher.ID()))
			}
			g.PutOnBattlefield(watcher.Card, b.PlayerID())
			g.PutTriggersOnStack()
			g.ResolveStack()
			if g.FindPermanent(victim.ID()) != nil {
				t.Fatal("delayed body did not act on the saved creature")
			}
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("turn-scoped subscription was removed before cleanup")
			}
			g.PutPermanentIntoGraveyard(g.FindPermanent(watcher.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("subscription followed the returned artifact")
			}
			clone.PutPermanentIntoGraveyard(clone.FindPermanent(watcher.ID()))
			clone.PutTriggersOnStack()
			if clone.StackSize() != 1 || clone.FindPermanent(victim.ID()) == nil {
				t.Fatal("clone did not retain an independent pending trigger")
			}
			clone.ResolveStack()
			if clone.FindPermanent(victim.ID()) != nil {
				t.Fatal("cloned delayed body lost its binding")
			}
		})
	}
}

func TestWhenLeavesThisTurnValidationAndExpiry(t *testing.T) {
	saved := BindPermanent("saved")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid card",
			Card: NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(Snapshot(This, saved), WhenLeavesThisTurn(From(saved), GainLifeFor(You, Fixed(1))))))),
		},
		{
			Name:          "unbound saved permanent",
			Card:          NewInstant("Invalid", "", Spell(WhenLeavesThisTurn(From(saved), GainLifeFor(You, Fixed(1))))),
			WantErrSubstr: "producer",
		},
		{
			Name:    "nil subject",
			Card:    NewInstant("Invalid", "", Spell(WhenLeavesThisTurn(nil, GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "nil effect",
			Card:    NewInstant("Invalid", "", Spell(WhenLeavesThisTurn(This, nil))),
			WantErr: true,
		},
	})
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := g.PutOnBattlefield(NewArtifact("Watcher", ""), a.PlayerID())
	if err := ApplyEffect(g, WhenLeavesThisTurn(This, GainLifeFor(You, Fixed(1))), watcher.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.triggers.ClearEndOfTurn()
	g.ExilePermanent(g.FindPermanent(watcher.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("expired trigger observed a later departure")
	}
}

func TestWhenLeavesThisTurnSimultaneousAndAbsent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	watcher := g.PutOnBattlefield(NewCreature("Watcher", "", 2, 2), b.PlayerID())
	target := DeclarePermanentTarget("watcher", IsCreature)
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	if err := ApplyEffect(g, WhenLeavesThisTurn(target, GainLifeFor(You, Fixed(2))), source.ID(), a.PlayerID(), []uuid.UUID{watcher.ID()}); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanents([]*Permanent{source, watcher})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("simultaneous source and watched departure lost the delayed trigger")
	}
	returned := g.PutOnBattlefield(watcher.Card, b.PlayerID())
	g.PhaseOut(returned.ID())
	if err := ApplyEffect(g, WhenLeavesThisTurn(target, GainLifeFor(You, Fixed(2))), source.ID(), a.PlayerID(), []uuid.UUID{returned.ID()}); err != nil {
		t.Fatal(err)
	}
	if len(g.DelayedTriggers()) != 1 {
		t.Fatal("absent target created a subscription")
	}
}

func TestWhenLeavesThisTurnDestinationReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := g.PutOnBattlefield(NewArtifact("Watcher", ""), a.PlayerID())
	if err := ApplyEffect(g, WhenLeavesThisTurn(This, GainLifeFor(You, Fixed(2))), watcher.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.AddExileIfWouldGoToGraveyardThisTurn(watcher.ID(), watcher.ID())
	g.PutPermanentIntoGraveyard(g.FindPermanent(watcher.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 || g.FindPermanent(watcher.ID()) != nil {
		t.Fatal("replaced departure did not trigger")
	}
}

// --- from composable_delayed_mode_test.go ---
func TestDelayedModesCaptureCopyAndClone(t *testing.T) {
	a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A")}
	b := newRecordingPlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("recipient")
	missing := DeclarePermanentTarget("missing creature", IsCreature)
	player := BindPlayer("future player")
	amount := BindNumber("saved amount")
	effect := Sequence(SnapshotNumber(Fixed(3), amount), AtNextModes(TurnStepBegins(EachEndStep, player),
		ModeOf("Unavailable", Destroy(missing), WithTargets(missing)),
		ModeOf("Life", Sequence(GainLifeFor(target, FromNumber(amount)), GainLifeFor(FromPlayer(player), Fixed(2))), WithTargets(target))))
	card := NewSorcery("Schedule", "", Spell(effect))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.menus) != 0 {
		t.Fatal("mode chosen at installation")
	}
	registered := g.Clone()
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	if len(a.menus) != 0 || g.StackSize() != 0 || len(g.DelayedTriggers()) != 0 {
		t.Fatal("incorrect detection lifecycle")
	}
	pending := g.Clone()
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	g.PutTriggersOnStack()
	original := g.StackPeek()
	if original == nil || original.ModeChoice != 1 || len(a.menus) != 1 || len(a.menus[0]) != 1 || a.menus[0][0] != "Life" {
		t.Fatal("incorrect legal mode menu")
	}
	b.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
	cp := g.CopyStackObjectDirect(original, b.PlayerID(), true)
	if cp == nil || cp.ModeChoice != 1 || cp.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() {
		t.Fatal("copy lost mode or target")
	}
	cloned := g.Clone()
	cloned.ResolveStack()
	if cloned.GetPlayer(a.PlayerID()).Life() != 23 || cloned.GetPlayer(b.PlayerID()).Life() != 27 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("stack clone lost captures or leaked")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 27 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
	for _, c := range []*Game{registered, pending} {
		if len(c.DelayedTriggers()) > 0 {
			c.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
		}
		c.ResolveStack()
		if c.GetPlayer(a.PlayerID()).Life()+c.GetPlayer(b.PlayerID()).Life() != 45 {
			t.Fatal("clone lost delayed modes or captures")
		}
	}
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 27 {
		t.Fatal("one-shot trigger repeated")
	}
}

func TestDelayedModesNoLegalMode(t *testing.T) {
	a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	target := DeclarePermanentTarget("creature", IsCreature)
	other := DeclarePermanentTarget("artifact", IsArtifact)
	effect := AtNextModes(EachEndStep, ModeOf("Creature", Destroy(target), WithTargets(target)), ModeOf("Artifact", Destroy(other), WithTargets(other)))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || len(g.DelayedTriggers()) != 0 || len(a.menus) != 0 {
		t.Fatal("empty menu did not consume trigger")
	}
}

func TestDelayedModesValidation(t *testing.T) {
	target := DeclarePlayerTarget("first mode target")
	effect := AtNextModes(EachEndStep, ModeOf("First", GainLifeFor(target, Fixed(1)), WithTargets(target)), ModeOf("Second", GainLifeFor(target, Fixed(2))))
	err := ValidateCardDefinition("Scope", NewSorcery("Scope", "", Spell(effect)))
	if err == nil || !strings.Contains(err.Error(), "modes[1]") {
		t.Fatalf("scope diagnostic=%v", err)
	}
	for _, build := range []func(){
		func() { AtNextModes(EachEndStep) },
		func() {
			AtNextModes(nil, ModeOf("One", GainLifeFor(You, Fixed(1))), ModeOf("Two", GainLifeFor(You, Fixed(2))))
		},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid modes accepted")
				}
			}()
			build()
		}()
	}
}

func TestDelayedModesConditionAndLegality(t *testing.T) {
	for _, scenario := range []string{"resolve", "condition false", "target returned", "invalid choice"} {
		t.Run(scenario, func(t *testing.T) {
			a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("creature", IsCreature)
			effect := AtNextModes(EachEndStep,
				ModeOf("Destroy", Sequence(Destroy(target), GainLifeFor(You, Fixed(3))), WithTargets(target)),
				ModeOf("Other", GainLifeFor(You, Fixed(1))),
				WithInterveningIf(Controls(IsArtifact)))
			source := NewEnchantment("Source", "", WithAbility(Activate(Costs(), effect)))
			creature := NewCreature("Creature", "", 1, 1)
			creature.SetOwner(b.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(creature, b.PlayerID())
			if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.PutOnBattlefield(source, b.PlayerID())
			g.PhaseOut(source.ID())
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if len(g.DelayedTriggers()) != 1 || g.StackSize() != 0 {
				t.Fatal("false condition consumed delay")
			}
			artifact := NewArtifact("Artifact", "")
			g.PutOnBattlefield(artifact, a.PlayerID())
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
			if scenario == "condition false" {
				g.RemoveFromBattlefield(g.FindPermanent(artifact.ID()))
			}
			if scenario == "invalid choice" {
				a.choice = 99
			}
			a.chooseQueue = [][]uuid.UUID{{creature.ID()}}
			g.PutTriggersOnStack()
			if scenario == "invalid choice" {
				if g.StackSize() != 0 || len(g.DelayedTriggers()) != 0 {
					t.Fatal("invalid choice retained trigger")
				}
				return
			}
			if g.StackSize() != 1 {
				t.Fatal("missing pending trigger")
			}
			if scenario == "target returned" {
				g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
				g.PutOnBattlefield(creature, b.PlayerID())
			}
			g.ResolveStack()
			want := 20
			if scenario == "resolve" {
				want = 23
				if g.FindPermanent(creature.ID()) != nil {
					t.Fatal("target survived")
				}
			}
			if a.Life() != want || b.Life() != 20 {
				t.Fatal("condition, source controller, or target legality lost")
			}
		})
	}
}

func TestDelayedModesOptionalTarget(t *testing.T) {
	a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	target := DeclarePermanentTargets("optional", 0, 1, IsCreature)
	effect := AtNextModes(EachEndStep, ModeOf("Optional", Sequence(DestroyEach(target), GainLifeFor(You, Fixed(2))), WithTargets(target)), ModeOf("Other", GainLifeFor(You, Fixed(1))))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if a.Life() != 22 || len(a.menus) != 1 || len(a.menus[0]) != 2 {
		t.Fatal("optional target mode was unavailable")
	}
}

// --- from composable_delayed_player_test.go ---
func TestDelayedStepPlayerCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("future player")
	prior := BindPlayer("earlier player")
	effect := Sequence(SnapshotPlayer(You, prior), SnapshotPlayer(You, p),
		AtNext(TurnStepBegins(EachUpkeep, p), Sequence(
			GainLifeFor(FromPlayer(p), Fixed(3)), GainLifeFor(FromPlayer(prior), Fixed(2)), GainLifeFor(You, Fixed(1)))),
		SnapshotPlayer(You, p))
	source := NewCreature("Source", "", 1, 1, WithAbility(Activate(Costs(), effect)))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	perm := g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(perm)
	g.PutOnBattlefield(source, b.PlayerID())
	registeredClone := g.Clone()
	for _, game := range []*Game{g, registeredClone} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
		if len(game.DelayedTriggers()) != 1 {
			t.Fatal("wrong event consumed trigger")
		}
		event := GameEvent{Type: EvtUpkeep, PlayerID: b.PlayerID(), TargetID: a.PlayerID()}
		game.FireEvent(event)
		event.PlayerID = a.PlayerID()
		if game.StackSize() != 0 || len(game.triggers.Pending()) != 1 {
			t.Fatal("did not wait for priority")
		}
		pendingClone := game.Clone()
		for _, pending := range []*Game{game, pendingClone} {
			pending.PutTriggersOnStack()
			if pending.CopyStackObjectDirect(pending.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			stackClone := pending.Clone()
			stackClone.ResolveStack()
			if pending.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone leaked")
			}
			pending.ResolveStack()
			for _, resolved := range []*Game{pending, stackClone} {
				if resolved.GetPlayer(a.PlayerID()).Life() != 25 || resolved.GetPlayer(b.PlayerID()).Life() != 27 {
					t.Fatalf("life = %d/%d", resolved.GetPlayer(a.PlayerID()).Life(), resolved.GetPlayer(b.PlayerID()).Life())
				}
				resolved.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
				if len(resolved.triggers.Pending()) != 0 {
					t.Fatal("trigger repeated")
				}
				if _, ok := FromPlayer(p).resolvePlayer(&EffectContext{Game: resolved}); ok {
					t.Fatal("binding leaked")
				}
			}
		}
	}
}

func TestDelayedStepPlayerValidation(t *testing.T) {
	p := BindPlayer("future player")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid upkeep delayed step",
			Card: NewSorcery("Schedule", "", Spell(AtNext(TurnStepBegins(EachUpkeep, p), GainLifeFor(FromPlayer(p), Fixed(1))))),
		},
		{
			Name:          "unbound player producer",
			Card:          NewSorcery("Schedule", "", Spell(Sequence(AtNext(TurnStepBegins(EachUpkeep, p), GainLifeFor(FromPlayer(p), Fixed(1))), GainLifeFor(FromPlayer(p), Fixed(1))))),
			WantErrSubstr: "producer",
		},
		{
			Name:          "nil player binding",
			Card:          NewSorcery("Schedule", "", Spell(AtNext(TurnStepBegins(EachUpkeep, nil), GainLifeFor(You, Fixed(1))))),
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty name player binding",
			Card:          NewSorcery("Schedule", "", Spell(AtNext(TurnStepBegins(EachUpkeep, BindPlayer("")), GainLifeFor(You, Fixed(1))))),
			WantErrSubstr: "name",
		},
	})
}

func TestDelayedStepPlayerFiltersAndNestedCapture(t *testing.T) {
	for _, tt := range []struct {
		name  string
		step  TurnStepPattern
		event EventType
		flag  bool
		own   bool
	}{
		{"each upkeep", EachUpkeep, EvtUpkeep, false, false},
		{"your upkeep", YourUpkeep, EvtUpkeep, false, true},
		{"each end", EachEndStep, EvtEndStep, false, false},
		{"your end", YourEndStep, EvtEndStep, false, true},
		{"combat", EachEndOfCombat, EvtEndOfCombat, false, false},
		{"draw", EachDrawStep, EvtDrawStep, false, false},
		{"first main", YourFirstMainPhase, EvtMainPhase, true, true},
		{"postcombat main", YourPostcombatMainPhase, EvtMainPhase, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			player := BindPlayer("future player")
			body := AtNext(TurnStepBegins(tt.step, player), AtNext(EachEndStep, GainLifeFor(FromPlayer(player), Fixed(2))))
			source := NewCreature("Source", "", 1, 1, WithAbility(Activate(Costs(), body)))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.PhaseOut(source.ID())
			if tt.event == EvtMainPhase {
				g.FireEvent(GameEvent{Type: tt.event, PlayerID: a.PlayerID(), Flag: !tt.flag})
				if len(g.triggers.Pending()) != 0 {
					t.Fatal("wrong phase triggered")
				}
			}
			g.FireEvent(GameEvent{Type: tt.event, PlayerID: b.PlayerID(), Flag: tt.flag})
			if tt.own {
				if len(g.triggers.Pending()) != 0 {
					t.Fatal("opponent step triggered")
				}
				g.FireEvent(GameEvent{Type: tt.event, PlayerID: a.PlayerID(), Flag: tt.flag})
			}
			g.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 || len(g.DelayedTriggers()) != 1 {
				t.Fatal("nested trigger resolved early")
			}
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
			g.ResolveStack()
			wantA, wantB := 20, 22
			if tt.own {
				wantA, wantB = 22, 20
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life = %d/%d", a.Life(), b.Life())
			}
		})
	}
}

// --- from composable_delayed_target_test.go ---
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
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid delayed target",
			Card: NewSorcery("Schedule", "", Spell(AtNext(EachEndStep, GainLifeFor(target, Fixed(1)), WithTargets(target)))),
		},
		{
			Name:          "undeclared target in delayed effect",
			Card:          NewSorcery("Schedule", "", Spell(AtNext(EachEndStep, GainLifeFor(target, Fixed(1))))),
			WantErrSubstr: "declaration",
		},
		{
			Name:          "undeclared target in outer sequence",
			Card:          NewSorcery("Schedule", "", Spell(Sequence(AtNext(EachEndStep, GainLifeFor(target, Fixed(1)), WithTargets(target)), GainLifeFor(target, Fixed(1))))),
			WantErrSubstr: "declaration",
		},
		{
			Name:          "duplicate target declaration",
			Card:          NewSorcery("Schedule", "", Spell(AtNext(EachEndStep, GainLifeFor(target, Fixed(1)), WithTargets(target, target)))),
			WantErrSubstr: "once",
		},
	})
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

// --- from composable_delayed_test.go ---
func TestDelayedTriggersWaitForPriorityAndAPNAP(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for _, controller := range []Player{b, a} {
		g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtEndStep, Controller: controller.PlayerID(), Effects: []Effect{GainLifeFor(You, Fixed(1))}})
	}
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	if g.GetStack().Size() != 0 || len(g.triggers.Pending()) != 2 {
		t.Fatal("delayed triggers bypassed pending placement")
	}
	clone := g.Clone()
	clone.PutTriggersOnStack()
	if clone.GetStack().Peek().Controller != b.PlayerID() {
		t.Fatal("non-active player's delayed trigger is not on top")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("pending delayed resolution leaked across clone")
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	if len(g.triggers.Pending()) != 0 {
		t.Fatal("one-shot delayed trigger fired again")
	}
}

func TestComposableDelayedCaptureAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source, host := NewCreature("Source", "", 1, 1), NewCreature("Host", "", 2, 2)
	source.SetOwner(a.PlayerID())
	host.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(host, a.PlayerID())
	g.MutablePermanent(source.ID()).AttachedTo = host.ID()
	saved := BindPermanent("host")
	effect := Sequence(Snapshot(AttachedPermanent, saved),
		AtNext(EachEndStep, Sequence(Destroy(From(saved)), GainLifeFor(You, Fixed(1)))),
		Snapshot(This, saved),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	clone := g.Clone()
	clone.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.FindPermanent(host.ID()) != nil || g.FindPermanent(host.ID()) == nil || clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("delayed binding, controller, or clone isolation failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.PutOnBattlefield(host, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(host.ID()) == nil || a.Life() != 21 {
		t.Fatal("delayed reference followed a returned host or stopped the sequence")
	}
}

func TestComposableDelayedStepAndCreationTime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	if err := ApplyEffect(g, AtNext(YourUpkeep, GainLifeFor(You, Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.GetStack().Size() != 0 {
		t.Fatal("delayed trigger saw an earlier event")
	}
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: b.PlayerID()})
	if len(g.triggers.Pending()) != 0 || len(g.DelayedTriggers()) != 1 {
		t.Fatal("opponent upkeep consumed your-upkeep trigger")
	}
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 || len(g.DelayedTriggers()) != 0 {
		t.Fatal("delayed trigger lost its controller or fired more than once")
	}
}

func TestComposableDelayedBindingScope(t *testing.T) {
	saved := BindPermanent("saved")
	cases := []struct {
		effect ResolvingEffect
		valid  bool
	}{
		{Sequence(Snapshot(This, saved), AtNext(EachEndStep, Destroy(From(saved)))), true},
		{AtNext(EachEndStep, Sequence(Snapshot(This, saved), Destroy(From(saved)))), true},
		{AtNext(EachEndStep, Destroy(From(saved))), false},
		{Sequence(AtNext(EachEndStep, Snapshot(This, saved)), Destroy(From(saved))), false},
		{Sequence(AtNext(EachEndStep, Destroy(From(saved))), Snapshot(This, saved)), false},
		{AtNext(nil, Destroy(This)), false},
		{AtNext(EachEndStep, nil), false},
	}
	for i, tt := range cases {
		card := NewCreature("Delayed scope", "", 1, 1, WithAbility(Activate(Costs(), tt.effect)))
		if err := ValidateCardDefinition("Delayed scope", card); (err == nil) != tt.valid {
			t.Fatalf("case %d: validation = %v", i, err)
		}
	}
}

func TestComposableDelayedTargetFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePermanentTarget("creature", IsCreature)
	effect := AtNext(EachEndStep, Sequence(Destroy(target), GainLifeFor(You, ValueAtResolution(XValue()))))
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	for i, card := range []Card{first, second} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
		g.resolution.SetX(i + 1)
		g.resolution.targetSpecs = []Target{target.targetDeclaration}
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), []uuid.UUID{card.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	g.resolution.SetX(0)
	g.resolution.targetSpecs = nil
	phased := g.PhaseOut(first.ID())
	g.PhaseIn(phased)
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.ResolveStack()
	if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil || a.Life() != 23 {
		t.Fatal("repeated schedules shared targets or lost captured X")
	}
}

func TestComposableDelayedSourceAlreadyReturned(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 2, 2)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: card.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(card, a.PlayerID())
	if err := AtNext(EachEndStep, Destroy(This)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.DelayedTriggers()[0].sourceIncarnation != ctx.sourceIncarnation {
		t.Fatal("registration replaced the captured source incarnation")
	}
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil {
		t.Fatal("delayed source action followed a returned card")
	}
}

func TestComposableDelayedDirectContextCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 2, 2)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: card.ID(), Controller: a.PlayerID()}
	if err := AtNext(EachEndStep, Destroy(This)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil {
		t.Fatal("direct-context delayed action followed a returned source")
	}
}

func TestComposableDelayedRestoresFrameAfterError(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPermanent("saved")
	original := capturedPermanent{id: uuid.New(), incarnation: uuid.New()}
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{saved: original}
	g.resolution.SetX(4)
	effect := AtNext(EachEndStep, Sequence(Snapshot(This, saved),
		FuncEffect("fail", EffectProperties{}, func(*Game, uuid.UUID, uuid.UUID, []uuid.UUID) error { return ErrPlayerNotFound }),
	))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.resolution.SetX(9)
	body := g.DelayedTriggers()[0].Effects[0]
	if err := body.Apply(ctx); !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("error = %v", err)
	}
	if g.resolution.X() != 9 || g.resolution.permanentBindings[saved] != original {
		t.Fatal("failed delayed body leaked resolution state")
	}
	if body.(*capturedDelayedBody).permanents[saved] != original {
		t.Fatal("failed delayed body changed its immutable capture")
	}
}

// --- from composable_granted_condition_test.go ---
func TestStaticGrantedCondition(t *testing.T) {
	for _, scenario := range []string{"allowed", "missing source counters", "missing controlled creature"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(
				MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}")), GainLifeFor(You, Fixed(2)),
				ActivationIf(SourceHasCounters(Charge, 1)), ActivationIf(Controls(IsCreature)),
			))))
			g.PutOnBattlefield(donor, a.PlayerID())
			recipient := NewArtifact("Recipient", "")
			g.PutOnBattlefield(recipient, b.PlayerID())
			other := NewArtifact("Other recipient", "")
			g.PutOnBattlefield(other, b.PlayerID())
			if scenario != "missing source counters" {
				g.MutablePermanent(recipient.ID()).AddCounter(Charge, 1)
			} else {
				g.MutablePermanent(donor.ID()).AddCounter(Charge, 1)
			}
			creature := NewCreature("Creature", "", 1, 1)
			controller := b.PlayerID()
			if scenario == "missing controlled creature" {
				controller = a.PlayerID()
			}
			g.PutOnBattlefield(creature, controller)
			g.ApplyContinuousEffects()
			b.ManaPool().Add(Colorless, 1)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), other.ID(), 0, nil); err == nil {
				t.Fatal("recipient without counters activated")
			}
			err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil)
			if scenario != "allowed" {
				if err == nil || b.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
					t.Fatal("rejected activation changed payment or stack", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if b.ManaPool().TotalMana() != 0 || g.StackSize() != 1 || b.Life() != 20 {
				t.Fatal("incorrect payment or resolution timing")
			}
			clone := g.Clone()
			clone.RemoveFromBattlefield(clone.FindPermanent(creature.ID()))
			clone.ApplyContinuousEffects()
			clone.GetPlayer(b.PlayerID()).ManaPool().Add(Colorless, 1)
			if err := clone.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err == nil {
				t.Fatal("clone used stale condition state")
			}
			if clone.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 1 || clone.StackSize() != 1 {
				t.Fatal("failed clone activation changed payment or stack")
			}
			clone.RemoveFromBattlefield(clone.FindPermanent(donor.ID()))
			clone.ApplyContinuousEffects()
			clone.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("resolution rechecked condition or used donor controller")
			}
			if b.Life() != 20 || g.StackSize() != 1 || g.FindPermanent(creature.ID()) == nil {
				t.Fatal("clone changed original state")
			}
			b.ManaPool().Add(Colorless, 1)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err != nil {
				t.Fatal("original lost its valid condition", err)
			}
		})
	}
}

func TestStaticGrantedConditionValidation(t *testing.T) {
	for _, condition := range []StateCondition{nil, AllConditions(), SourceHasCounters(Charge, -1)} {
		err := validateFactory("Invalid grant condition", func() Card {
			return NewEnchantment("Invalid grant condition", "", WithAbility(Static(GrantEachActivatedAbility(
				MatchingPermanents(IsArtifact), Costs(), Regenerate(This), ActivationIf(condition),
			))))
		})
		if err == nil {
			t.Fatal("invalid condition accepted")
		}
		if condition != nil && !strings.Contains(err.Error(), "restrictions[0].condition") {
			t.Fatalf("missing condition path: %v", err)
		}
	}
}

func TestStaticGrantedManaCondition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor := NewEnchantment("Mana donor", "", WithAbility(Static(GrantEachActivatedAbility(
		MatchingPermanents(IsLand), Costs(TapCost(This)), AddManaAmount(Blue, Fixed(1)),
		ActivationIf(SourceHasCounters(Charge, 1)),
	))))
	g.PutOnBattlefield(donor, a.PlayerID())
	recipient := NewLand("Recipient")
	g.PutOnBattlefield(recipient, b.PlayerID())
	g.MutablePermanent(donor.ID()).AddCounter(Charge, 1)
	g.ApplyContinuousEffects()
	if g.CanAfford(b.PlayerID(), ManaCost{Blue: 1}, nil) {
		t.Fatal("planner ignored recipient condition")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err == nil || g.FindPermanent(recipient.ID()).Tapped {
		t.Fatal("rejected mana activation paid tap cost", err)
	}
	clone := g.Clone()
	clone.MutablePermanent(recipient.ID()).AddCounter(Charge, 1)
	clone.ApplyContinuousEffects()
	if !clone.CanAfford(b.PlayerID(), ManaCost{Blue: 1}, nil) {
		t.Fatal("planner did not read current recipient condition")
	}
	if err := clone.AutoTapForCost(b.PlayerID(), ManaCost{Blue: 1}); err != nil {
		t.Fatal(err)
	}
	if clone.StackSize() != 0 || clone.GetPlayer(b.PlayerID()).ManaPool().Count(Blue) != 1 || !clone.FindPermanent(recipient.ID()).Tapped {
		t.Fatal("granted mana did not resolve immediately")
	}
	if g.FindPermanent(recipient.ID()).Tapped || b.ManaPool().TotalMana() != 0 || g.CanAfford(b.PlayerID(), ManaCost{Blue: 1}, nil) {
		t.Fatal("clone changed original condition or payment")
	}
}

// --- from composable_hand_size_condition_test.go ---
func TestControllerHandSizeIsActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Hand gate", WithAbility(Activate(Costs(TapCost(This)), DrawFor(You, Fixed(1)), ActivationIf(ControllerHandSizeIs(7)))))
	card.SetOwner(b.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	for i := range 8 {
		a.AddToHand(NewArtifact("Hand card", ""))
		if i == 5 || i == 6 {
			want := i == 6
			if got := card.Abilities()[0].(*ActionDefinition).CanActivate(a.PlayerID(), g); got != want {
				t.Fatalf("hand size %d: can activate = %v, want %v", i+1, got, want)
			}
		}
	}
	if card.Abilities()[0].(*ActionDefinition).CanActivate(a.PlayerID(), g) {
		t.Fatal("eight cards allowed activation")
	}
	clone := g.Clone()
	if _, ok := clone.GetPlayer(a.PlayerID()).RemoveFromHand(a.Hand()[0].ID()); !ok {
		t.Fatal("clone hand removal failed")
	}
	if !clone.FindPermanent(card.ID()).Card.Abilities()[0].(*ActionDefinition).CanActivate(a.PlayerID(), clone) {
		t.Fatal("clone did not read its own hand")
	}
	if card.Abilities()[0].(*ActionDefinition).CanActivate(a.PlayerID(), g) {
		t.Fatal("clone changed original hand")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err == nil || g.FindPermanent(card.ID()).Tapped || g.StackSize() != 0 {
		t.Fatal("failed activation changed payment state")
	}
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil || !clone.FindPermanent(card.ID()).Tapped || clone.StackSize() != 1 {
		t.Fatal("seven-card activation did not pay and enter the stack", err)
	}
}

func TestControllerHandSizeIsStateAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("Card", ""))
	condition := ControllerHandSizeIs(1)
	if !condition.checkState(g, StateContext{ControllerID: a.PlayerID()}) {
		t.Fatal("absent source changed controller hand check")
	}
	if condition.checkState(g, StateContext{ControllerID: b.PlayerID()}) || condition.checkState(g, StateContext{ControllerID: uuid.Nil}) {
		t.Fatal("condition used the wrong or missing controller")
	}
	if got := InspectDefinition(condition); got.Role != "state-condition" || got.Text != "controller hand size is 1" {
		t.Fatalf("inspection = %+v", got)
	}
	invalid := NewLand("Invalid", WithAbility(Activate(Costs(), DrawFor(You, Fixed(1)), ActivationIf(ControllerHandSizeIs(-1)))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("negative hand size passed registration")
	}
}

// --- from composable_life_gain_history_condition_test.go ---
func TestControllerGainedLifeThisTurnActualOutcomeAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := ControllerGainedLifeThisTurn()
	if condition.checkState(g, StateContext{ControllerID: a.PlayerID()}) {
		t.Fatal("matched before life gain")
	}
	source := NewInstant("Gain", "")
	g.effects.AddReplacement(&lifeOutcomeReplacement{consume: true})
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if condition.checkState(g, StateContext{ControllerID: a.PlayerID()}) {
		t.Fatal("matched a replaced life gain")
	}
	g.effects.replacements = nil
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(0)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if condition.checkState(g, StateContext{ControllerID: a.PlayerID()}) {
		t.Fatal("matched zero life gain")
	}
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !condition.checkState(g, StateContext{SourceID: uuid.New(), ControllerID: a.PlayerID()}) {
		t.Fatal("absent source hid controller history")
	}
	if condition.checkState(g, StateContext{ControllerID: b.PlayerID()}) || condition.checkState(g, StateContext{ControllerID: uuid.Nil}) {
		t.Fatal("condition matched another or missing controller")
	}
	clone := g.Clone()
	clone.trackers.Turn.ResetForNewTurn(b.PlayerID())
	if condition.checkState(clone, StateContext{ControllerID: a.PlayerID()}) || !condition.checkState(g, StateContext{ControllerID: a.PlayerID()}) {
		t.Fatal("clone shared life-gain history")
	}
	g.trackers.Turn.ResetForNewTurn(b.PlayerID())
	if condition.checkState(g, StateContext{ControllerID: a.PlayerID()}) {
		t.Fatal("history survived turn reset")
	}
}

func TestControllerGainedLifeThisTurnResolutionAndStatic(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := ControllerGainedLifeThisTurn()
	effect := IfState(condition, GainLifeFor(You, Fixed(1)))
	spell := NewInstant("Conditional", "", Spell(effect))
	static := NewCreature("Conditional creature", "", 1, 1, WithAbility(Static(While(condition, GrantKeywordModification(This, Flying)))))
	g.PutOnBattlefield(static, a.PlayerID())
	for _, card := range []Card{spell, static} {
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	if err := ApplyEffect(g, effect, spell.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || g.FindPermanent(static.ID()).HasKeyword(Flying) {
		t.Fatal("condition was active before gain")
	}
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), spell.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	if err := ApplyEffect(g, effect, spell.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 20 || !g.FindPermanent(static.ID()).HasKeyword(Flying) {
		t.Fatal("condition used the wrong controller or did not update static rule")
	}
	if err := ApplyEffect(g, effect, spell.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 23 {
		t.Fatal("resolution branch did not use controller history")
	}
	g.trackers.Turn.ResetForNewTurn(b.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(static.ID()).HasKeyword(Flying) {
		t.Fatal("static condition survived turn reset")
	}
	meta := InspectDefinition(condition)
	if meta.Role != "state-condition" || meta.Text != "controller gained life this turn" {
		t.Fatalf("inspection = %+v", meta)
	}
}

// --- from composable_permanent_subtype_branch_test.go ---
func TestPermanentSubtypeBranchEntryAndDeparture(t *testing.T) {
	for _, tc := range []struct {
		name, subtype            string
		depart, returned, phased bool
		want                     int
	}{
		{"aura", "Aura", false, false, false, 22},
		{"saga", "Saga", false, false, false, 20},
		{"departed aura", "Aura", true, false, false, 22},
		{"returned aura", "Aura", true, true, false, 22},
		{"phased aura", "Aura", false, false, true, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			entered := BindPermanent("entered")
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				EnchantmentYouControlEnters().WithPermanent(entered),
				IfPermanentHasEnchantmentSubtype(From(entered), "Aura", GainLifeFor(You, Fixed(2))))))
			watcher.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			entry := NewEnchantment("Entry", "")
			entry.AddSubType(tc.subtype)
			entry.SetOwner(b.PlayerID())
			g.PutOnBattlefield(entry, a.PlayerID())
			g.PutTriggersOnStack()
			if tc.depart {
				g.RemoveFromBattlefield(g.FindPermanent(entry.ID()))
			}
			if tc.returned {
				g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
				g.PutOnBattlefield(entry, a.PlayerID())
			}
			if tc.phased {
				g.PhaseOut(entry.ID())
			}
			g.ResolveStack()
			if got := a.Life(); got != tc.want {
				t.Fatalf("life = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPermanentSubtypeBranchDepartureKeepsEffectiveSubtype(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entered := BindPermanent("entered")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().WithPermanent(entered),
		IfPermanentHasEnchantmentSubtype(From(entered), "Aura", GainLifeFor(You, Fixed(2))))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	entry := NewEnchantment("Entry", "", WithAbility(Static(While(
		Controls(AnyPermanent), SetSubtypeFamily(This, SubtypeEnchantment, "Aura")))))
	entry.AddSubType("Saga")
	entry.SetOwner(b.PlayerID())
	g.PutOnBattlefield(entry, a.PlayerID())
	g.PutTriggersOnStack()
	g.RemoveFromBattlefield(g.FindPermanent(entry.ID()))
	g.ResolveStack()
	if got := a.Life(); got != 22 {
		t.Fatalf("life after effective Aura departure = %d", got)
	}
}

func TestPermanentSubtypeBranchDeclaration(t *testing.T) {
	bound := BindPermanent("entered")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid card",
			Card: NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
				EnchantmentYouControlEnters().WithPermanent(bound),
				IfPermanentHasEnchantmentSubtype(From(bound), "Aura", GainLifeFor(You, Fixed(1)))))),
		},
		{
			Name:          "nil permanent",
			Card:          NewInstant("Invalid", "", Spell(IfPermanentHasEnchantmentSubtype(nil, "Aura", GainLifeFor(You, Fixed(1))))),
			WantErrSubstr: "permanent",
		},
		{
			Name:          "unbound producer",
			Card:          NewInstant("Invalid", "", Spell(IfPermanentHasEnchantmentSubtype(From(bound), "Aura", GainLifeFor(You, Fixed(1))))),
			WantErrSubstr: "producer",
		},
		{
			Name:          "empty subtype",
			Card:          NewInstant("Invalid", "", Spell(IfPermanentHasEnchantmentSubtype(This, "", GainLifeFor(You, Fixed(1))))),
			WantErrSubstr: "subtype",
		},
		{
			Name:          "invalid enchantment subtype Goblin",
			Card:          NewInstant("Invalid", "", Spell(IfPermanentHasEnchantmentSubtype(This, "Goblin", GainLifeFor(You, Fixed(1))))),
			WantErrSubstr: "subtype",
		},
		{
			Name:          "nil then effect",
			Card:          NewInstant("Invalid", "", Spell(IfPermanentHasEnchantmentSubtype(This, "Aura", nil))),
			WantErrSubstr: "then",
		},
	})
}

func TestPermanentSubtypeBranchCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entered := BindPermanent("entered")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().WithPermanent(entered),
		IfPermanentHasEnchantmentSubtype(From(entered), "Aura", GainLifeFor(You, Fixed(2))))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	entry := NewEnchantment("Entry", "")
	entry.AddSubType("Aura")
	entry.SetOwner(a.PlayerID())
	g.PutOnBattlefield(entry, a.PlayerID())
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if got := a.Life(); got != 20 {
		t.Fatalf("clone changed original life: %d", got)
	}
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 22 {
		t.Fatalf("clone controller life = %d", got)
	}
	if got := clone.GetPlayer(b.PlayerID()).Life(); got != 22 {
		t.Fatalf("clone copy controller life = %d", got)
	}
	g.ResolveStack()
	if got := a.Life(); got != 22 {
		t.Fatalf("original controller life = %d", got)
	}
	if got := b.Life(); got != 22 {
		t.Fatalf("original copy controller life = %d", got)
	}
}

func TestPermanentSubtypeBranchTargetDoesNotUseDepartureLKI(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePermanentTarget("enchantment", IsEnchantment)
	ability := Spell(IfPermanentHasEnchantmentSubtype(target, "Aura", GainLifeFor(You, Fixed(2))), WithTargets(target))
	spell := NewInstant("Check", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	entry := NewEnchantment("Entry", "")
	entry.AddSubType("Aura")
	entry.SetOwner(b.PlayerID())
	g.PutOnBattlefield(entry, b.PlayerID())
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{entry.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.RemoveFromBattlefield(g.FindPermanent(entry.ID()))
	g.PutOnBattlefield(entry, b.PlayerID())
	g.ResolveStack()
	if got := a.Life(); got != 20 {
		t.Fatalf("illegal target gained life: %d", got)
	}
}

// --- from composable_player_controls_test.go ---
func TestPlayerControlsStateUsesCurrentController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	island := NewLand("Island", WithSubTypes("Island"))
	island.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(island, b.PlayerID())
	ctx := StateContext{ControllerID: a.PlayerID()}
	condition := PlayerControls(ActivePlayer, HasSubType("Island"))
	g.SetActivePlayerIndex(1)
	if !condition.checkState(g, ctx) {
		t.Fatal("active controller's Island did not match")
	}
	if PlayerControls(You, HasSubType("Island")).checkState(g, ctx) {
		t.Fatal("owner matched in place of controller")
	}
	g.PhaseOut(p.ID())
	if condition.checkState(g, ctx) {
		t.Fatal("phased Island matched")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
	if condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("clone changed the original battlefield")
	}
	g.RemoveFromBattlefield(p)
	if condition.checkState(g, ctx) {
		t.Fatal("departed Island matched")
	}
	g.PutOnBattlefield(island, a.PlayerID())
	if condition.checkState(g, ctx) || !PlayerControls(You, HasSubType("Island")).checkState(g, ctx) {
		t.Fatal("returned Island kept its former controller")
	}
}

func TestPlayerControlsDeclaration(t *testing.T) {
	condition := PlayerControls(ActivePlayer, IsLand)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid spell",
			Card: NewInstant("Watch", "", WithAbility(Spell(IfState(condition, GainLifeFor(You, Fixed(1)))))),
			Inspect: func(t *testing.T) {
				if meta := InspectDefinition(condition); meta.Role != "state-condition" || meta.Text == "" {
					t.Fatalf("inspection = %+v", meta)
				}
			},
		},
		{
			Name: "valid static",
			Card: NewArtifact("Static", "", WithAbility(Static(While(PlayerControls(You, IsLand), ModifyPT(This, Fixed(1), Fixed(0)))))),
		},
		{
			Name:    "nil player",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(IfState(PlayerControls(nil, IsLand), GainLifeFor(You, Fixed(1)))))),
			WantErr: true,
		},
		{
			Name:    "target player in state condition",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(IfState(PlayerControls(DeclarePlayerTarget("player"), IsLand), GainLifeFor(You, Fixed(1)))))),
			WantErr: true,
		},
	})
}

func TestPlayerControlsResolutionReadsCurrentState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Land")
	g.PutOnBattlefield(land, b.PlayerID())
	effect := IfState(PlayerControls(ActivePlayer, IsLand), GainLifeFor(You, Fixed(2)))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.SetActivePlayerIndex(0)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 {
		t.Fatal("inactive player's land satisfied active-player condition")
	}
	g.SetActivePlayerIndex(1)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatalf("life totals = %d, %d", a.Life(), b.Life())
	}
}

// --- from composable_random_discard_test.go ---
func TestTypedRandomDiscardContextAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	cards := []Card{NewArtifact("One", ""), NewArtifact("Two", ""), NewArtifact("Three", "")}
	for _, c := range cards {
		a.AddToHand(c)
	}
	source := NewCreature("Amount", "{2}", 2, 2)
	g.PutOnBattlefield(source, b.PlayerID())
	g.SetRandomResults([]int{2, 0, 7})
	clone := g.Clone()
	body := DiscardAtRandom(You, ValueAtResolution(TargetPermanentManaValue()))
	if err := ApplyEffect(clone, body, source.ID(), a.PlayerID(), []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	p := clone.GetPlayer(a.PlayerID())
	if len(p.Hand()) != 1 || p.Hand()[0].ID() != cards[1].ID() || len(p.Graveyard()) != 2 || p.Graveyard()[0].ID() != cards[2].ID() {
		t.Fatal("wrong random selection or dynamic amount")
	}
	if len(a.Hand()) != 3 || g.RandIntn(10) != 2 || clone.RandIntn(10) != 7 {
		t.Fatal("clone changed original state or random queue")
	}
}

func TestTypedRandomDiscardReplacementAndAbsence(t *testing.T) {
	a := NewBasePlayer("A")
	b := &discardChoicePlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	for range 2 {
		b.AddToHand(NewArtifact("Card", ""))
	}
	chosen := b.Hand()[1].ID()
	events := 0
	watcher := NewTriggered(EvtDiscard, false, DrawFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		events++
		if evt.PlayerID != b.PlayerID() || evt.SourceID != chosen {
			t.Error("wrong discard event")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Library", "", WithStaticAbility(DiscardToLibraryReplacement())), b.PlayerID())
	g.SetActivePlayerIndex(1)
	g.SetRandomResults([]int{1, 7})
	calls := 0
	for _, body := range []ResolvingEffect{DiscardAtRandom(ActivePlayer, Fixed(0)), DiscardAtRandom(ActivePlayer, Fixed(-1)), DiscardAtRandom(ControllerOf(This), countedLifeAmount{&calls})} {
		if err := ApplyEffect(g, body, uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 0 {
		t.Fatal("absent subject evaluated amount")
	}
	if err := ApplyEffect(g, DiscardAtRandom(ActivePlayer, MinValue(Fixed(1), countedLifeAmount{&calls})), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(b.requests) != 0 || len(b.Library()) != 1 || b.Library()[0].ID() != chosen || events != 1 || g.RandIntn(10) != 7 {
		t.Fatal("choice, amount, replacement, event, or random queue changed")
	}
}

func TestTypedRandomDiscardShortHandAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("Only", ""))
	g.SetRandomResults([]int{0, 7})
	body := Sequence(DiscardAtRandom(You, Fixed(5)), GainLifeFor(You, Fixed(1)))
	for range 2 {
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(a.Graveyard()) != 1 || a.Life() != 22 || g.RandIntn(10) != 7 {
		t.Fatal("short hand changed continuation or random consumption")
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil player",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(DiscardAtRandom(nil, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "nil amount",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(DiscardAtRandom(You, nil)))),
			WantErr: true,
		},
		{
			Name:    "unbound power source",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(DiscardAtRandom(You, PowerOf(From(BindPermanent("missing"))))))),
			WantErr: true,
		},
	})
	info := InspectDefinition(DiscardAtRandom(You, Fixed(2)))
	if len(info.Children) != 2 || info.Properties.Outcome != OutcomeDetriment {
		t.Fatal("missing metadata")
	}
}

func TestTypedRandomDiscardCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 3 {
		a.AddToHand(NewArtifact("A card", ""))
		b.AddToHand(NewArtifact("B card", ""))
	}
	source := NewCreature("Amount", "", 2, 3)
	g.PutOnBattlefield(source, a.PlayerID())
	target := DeclarePermanentTarget("amount", IsCreature)
	ability := Spell(DiscardAtRandom(You, ValueAtResolution(XValue())), WithTargets(target))
	card := NewSorcery("Discard", "", WithAbility(ability))
	original := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), XValue: 2, Targets: []uuid.UUID{source.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveStackObject(copied)
	if len(a.Hand()) != 3 || len(b.Hand()) != 1 {
		t.Fatal("copy lost controller or amount binding")
	}
	g.ResolveStackObject(original)
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("original lost its amount")
	}
}

// --- from composable_random_test.go ---
func TestComposableCoinFlipBranches(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetCoinFlipResults([]bool{true, false})
	effect := FlipCoin(You, OnWin(GainLifeFor(You, Fixed(2))), OnLoss(LoseLifeFor(You, Fixed(3))))
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		for _, want := range []int{22, 19} {
			if err := ApplyEffect(game, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if got := game.GetPlayer(a.PlayerID()).Life(); got != want {
				t.Fatalf("life = %d, want %d", got, want)
			}
		}
	}
	metadata := InspectDefinition(effect)
	if len(metadata.Children) != 3 || len(metadata.Children[1].Children) != 1 || len(metadata.Children[2].Children) != 1 {
		t.Fatalf("missing branch metadata: %+v", metadata)
	}
}

func TestComposableCoinFlipValidation(t *testing.T) {
	value := BindPermanent("value")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "empty",
			Card:          NewSorcery("Coin", "", WithAbility(Spell(FlipCoin(You)))),
			WantErrSubstr: "branch",
		},
		{
			Name:          "duplicate",
			Card:          NewSorcery("Coin", "", WithAbility(Spell(FlipCoin(You, OnWin(GainLifeFor(You, Fixed(1))), OnWin(GainLifeFor(You, Fixed(2))))))),
			WantErrSubstr: "duplicate",
		},
		{
			Name:          "nil effect",
			Card:          NewSorcery("Coin", "", WithAbility(Spell(FlipCoin(You, OnLoss(nil))))),
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "nil player",
			Card:          NewSorcery("Coin", "", WithAbility(Spell(FlipCoin(nil, OnLoss(GainLifeFor(You, Fixed(1))))))),
			WantErrSubstr: "player",
		},
		{
			Name:    "coin branch exported conditional assignment",
			Card:    NewCreature("Scope", "", 1, 1, WithAbility(Activate(Costs(), Sequence(FlipCoin(You, OnWin(Snapshot(This, value))), Sacrifice(From(value)))))),
			WantErr: true,
		},
	})
}

// --- from composable_revealed_cast_condition_test.go ---
func TestRevealedSubtypeAtCastPaymentSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name   string
		accept bool
		card   bool
		want   int
	}{
		{"paid Dragon reveal", true, true, 23},
		{"declined reveal", false, true, 20},
		{"no matching card", true, false, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: tc.accept}
			g := NewGame(a, NewBasePlayer("B"))
			paid, revealed := BindNumber("paid"), BindCard("revealed")
			spell := NewInstant("Reveal test", "", WithAbility(Spell(
				IfCast(RevealedSubtypeAtCast("Dragon"), GainLifeFor(You, Fixed(3))))),
				WithAdditionalCost(OptionalRevealFromHandCost(HasSubTypeCardFilter("Dragon"), "Reveal Dragon", "Reveal?", paid, revealed)))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if tc.card {
				a.AddToHand(NewCreature("Dragon", "", 2, 2, WithSubTypes("Dragon")))
			} else {
				a.AddToHand(NewCreature("Bear", "", 2, 2))
			}
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.Life() != tc.want {
				t.Fatalf("life = %d, want %d", a.Life(), tc.want)
			}
		})
	}
}

func TestRevealedSubtypeAtCastImmutableAcrossCopies(t *testing.T) {
	a, b := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}, NewBasePlayer("B")
	g := NewGame(a, b)
	paid, revealed := BindNumber("paid"), BindCard("revealed")
	spell := NewInstant("Reveal test", "", WithAbility(Spell(
		IfCast(RevealedSubtypeAtCast("Dragon"), GainLifeFor(You, Fixed(3))))),
		WithAdditionalCost(OptionalRevealFromHandCost(HasSubTypeCardFilter("Dragon"), "Reveal Dragon", "Reveal?", paid, revealed)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	dragon := NewCreature("Dragon", "", 2, 2, WithSubTypes("Dragon"))
	a.AddToHand(dragon)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	original := g.StackObjects()[0]
	spellCopy := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	clone := g.Clone()
	if _, ok := a.RemoveFromHand(dragon.ID()); !ok {
		t.Fatal("revealed card did not leave hand")
	}
	a.AddToHand(dragon)
	dragon.AddSubType("Elf")
	spellCopy.CastContext.RevealedAtCast[0].AddSubType("Goblin")
	if original.CastContext.RevealedSubtypeAtCast("Elf") || original.CastContext.RevealedSubtypeAtCast("Goblin") || clone.StackObjects()[0].CastContext.RevealedSubtypeAtCast("Goblin") {
		t.Fatal("cast snapshot changed after a card or copy changed")
	}
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone resolution changed live players")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 23 {
		t.Fatalf("life A=%d B=%d", a.Life(), b.Life())
	}
}

func TestRevealedSubtypeAtCastDeclaration(t *testing.T) {
	condition := RevealedSubtypeAtCast("Dragon")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid card",
			Card: NewInstant("Valid", "", Spell(IfCast(RevealedSubtypeAtCast("Aura"), GainLifeFor(You, Fixed(1))))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(condition); got.Role != "cast-condition" || !strings.Contains(got.Text, "Dragon") {
					t.Fatalf("inspection = %+v", got)
				}
			},
		},
		{
			Name:    "empty subtype",
			Card:    NewInstant("Invalid", "", Spell(IfCast(RevealedSubtypeAtCast(""), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "activation with cast condition",
			Card:    NewArtifact("Invalid", "", WithAbility(Activate(Costs(), IfCast(condition, GainLifeFor(You, Fixed(1)))))),
			WantErr: true,
		},
	})
}

// --- from composable_starting_life_condition_test.go ---
func TestControllerLifeAtLeastStartingLifeStateAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	a.SetStartingLife(30)
	b.SetStartingLife(15)
	a.SetLife(30)
	b.SetLife(15)
	g := NewGame(a, b)
	condition := ControllerLifeAtLeastStartingLife()
	ctxA := StateContext{ControllerID: a.PlayerID()}
	ctxB := StateContext{ControllerID: b.PlayerID()}
	if !condition.checkState(g, ctxA) || !condition.checkState(g, ctxB) {
		t.Fatal("current life at the starting total did not match")
	}
	a.SetLife(29)
	b.SetLife(16)
	if condition.checkState(g, ctxA) || !condition.checkState(g, ctxB) {
		t.Fatal("condition did not compare each controller with its own starting total")
	}
	if condition.checkState(g, StateContext{ControllerID: uuid.Nil}) || condition.checkState(g, StateContext{ControllerID: uuid.New()}) {
		t.Fatal("missing controller matched")
	}
	clone := g.Clone()
	clone.GetPlayer(a.PlayerID()).SetLife(31)
	if !condition.checkState(clone, ctxA) || condition.checkState(g, ctxA) {
		t.Fatal("clone did not keep independent current life")
	}
	if got := InspectDefinition(condition); got.Role != "state-condition" {
		t.Fatalf("inspection = %+v", got)
	}
}

func TestControllerLifeAtLeastStartingLifeRegistration(t *testing.T) {
	condition := ControllerLifeAtLeastStartingLife()
	card := NewEnchantment("Life watcher", "", WithAbility(Static(While(condition,
		ModifyEachPT(ControlledPermanents(IsCreature), Fixed(1), Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	creature := NewCreature("Subject", "", 2, 2)
	g.PutOnBattlefield(card, a.PlayerID())
	g.PutOnBattlefield(creature, a.PlayerID())
	checkPower := func(want int) {
		t.Helper()
		g.ApplyContinuousEffects()
		if got := g.FindPermanent(creature.ID()).CurrentPower(g); got != want {
			t.Fatalf("current power = %d, want %d", got, want)
		}
	}
	checkPower(3)
	a.SetLife(19)
	checkPower(2)
	a.SetLife(20)
	checkPower(3)
}
