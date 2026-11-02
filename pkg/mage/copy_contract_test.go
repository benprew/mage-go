package mage

import (
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestRulesCopyDoesNotCopyActivationState(t *testing.T) {
	source := NewCreature("Limited Source", "{R}", 1, 1,
		WithAbility(NewActivatedAbility(GainLife(1), ManaCostOf("{0}"), WithOncePerTurn(), WithMaxActivationsPerTurn(2))))
	action := source.Abilities()[0].(*ActionDefinition)
	action.activatedThisTurn = true
	action.activationsThisTurn = 2
	owner := uuid.New()
	receiver := NewCreature("Receiver", "{U}", 0, 0)
	receiver.SetOwner(owner)
	id := receiver.ID()
	receiver.CloneFrom(source)
	copied := receiver.Abilities()[0].(*ActionDefinition)
	if copied.activatedThisTurn || copied.activationsThisTurn != 0 {
		t.Fatal("rules copy inherited activation state")
	}
	if receiver.ID() != id || receiver.Owner() != owner || copied.AbilityID() == action.AbilityID() {
		t.Fatal("rules copy lost receiver identity or reused ability identity")
	}
	branch := source.CloneInstance().Abilities()[0].(*ActionDefinition)
	if !branch.activatedThisTurn || branch.activationsThisTurn != 2 || branch.AbilityID() != action.AbilityID() {
		t.Fatal("simulation clone did not preserve activation state and identity")
	}
}

func TestRulesCopyCastingProperties(t *testing.T) {
	def, err := NewCardDefinition(CardDefinitionSpec{Name: "Receiver", TypeLine: TypeLine{CardTypes: []CardType{TypeCreature}}})
	if err != nil {
		t.Fatal(err)
	}
	source := NewCreature("Source", "{G}", 1, 1, WithUncounterable(),
		WithAdditionalCost(LifePayCost(1)), WithAlternateCost(ZoneHand, ParseManaCost("{0}")),
		WithAuraAIProfile(AuraAIProfile{PowerBoost: 2}))
	for _, receiver := range []*Card{NewCreature("Legacy Receiver", "{0}", 0, 0), def.NewInstance(uuid.New())} {
		receiver.CloneFrom(source)
		if !receiver.IsUncounterable() || len(receiver.AlternateCosts()) != 1 || len(receiver.AdditionalCosts()) != 1 {
			t.Fatal("rules copy lost casting properties")
		}
		if profile, ok := receiver.AuraAIProfile(); !ok || profile.PowerBoost != 2 {
			t.Fatal("rules copy lost AI metadata")
		}
		receiver.CloneFrom(NewCreature("Plain", "{0}", 1, 1))
		if receiver.IsUncounterable() || len(receiver.AlternateCosts()) != 0 || len(receiver.AdditionalCosts()) != 0 {
			t.Fatal("rules copy kept old casting properties")
		}
	}
}

func TestGameCloneLegacyCardAliases(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	card := NewCreature("Legacy", "{0}", 1, 1,
		WithAbility(NewActivatedAbility(GainLife(1), ManaCostOf("{0}"), WithOncePerTurn())))
	card.SetOwner(owner)
	g.players[0].AddToHand(card)
	g.stack.Push(&StackObject{Card: card, TargetSource: card})
	branch := g.Clone()
	cloned := branch.players[0].Hand()[0]
	if cloned == card || cloned != branch.stack.Peek().Card || cloned != branch.stack.Peek().TargetSource || cloned.ID() != card.ID() {
		t.Fatal("simulation clone did not isolate a legacy card and preserve aliases")
	}
	cloned.SetBasePT(8, 9)
	cloned.SetOwner(uuid.New())
	cloned.Abilities()[0].(*ActionDefinition).activatedThisTurn = true
	if card.Power() != 1 || card.Owner() != owner || card.Abilities()[0].(*ActionDefinition).activatedThisTurn {
		t.Fatal("simulation changed the original card")
	}
}

func TestGameClonePendingAbilityAliases(t *testing.T) {
	for _, removed := range []bool{false, true} {
		t.Run(map[bool]string{false: "granted", true: "source removed"}[removed], func(t *testing.T) {
			g := newPriorityTestGame()
			owner := g.players[0].PlayerID()
			host := g.PutOnBattlefield(NewCreature("Host", "{0}", 1, 1), owner)
			ability := WheneverSpellCastTrigger(GainLife(1), false)
			host.RuntimeAbilities = append(host.RuntimeAbilities, WrapGrantedAbility(ability))
			g.FireEvent(GameEvent{Type: EvtSpellCast, PlayerID: owner})
			if len(g.triggers.pending) != 1 {
				t.Fatal("expected one pending trigger")
			}
			if removed {
				g.RemoveFromBattlefield(host)
			}
			branch := g.Clone()
			pending := branch.triggers.pending[0].ability
			if pending == ability || pending.AbilityID() != ability.AbilityID() {
				t.Fatal("pending trigger did not get private state with the same identity")
			}
			if !removed && pending != UnwrapAbility(branch.FindPermanent(host.ID()).RuntimeAbilities[0]) {
				t.Fatal("pending trigger lost its runtime ability alias")
			}
			pending.SetController(uuid.New())
			if ability.Controller() != owner {
				t.Fatal("pending trigger changed the original ability")
			}
		})
	}
}

func TestCopiableValuesSnapshot(t *testing.T) {
	source := NewCreature("Source", "{U}", 2, 3,
		WithAbility(NewActivatedAbility(GainLife(1), ManaCostOf("{0}"), WithOncePerTurn(), WithTarget(TargetCreature()))))
	action := source.Abilities()[0].(*ActionDefinition)
	action.activatedThisTurn = true
	action.SetController(uuid.New())
	target := action.Targets()[0]
	if err := target.Choose(uuid.Nil(), source, nil, []uuid.UUID{uuid.New()}); err != nil {
		t.Fatal(err)
	}
	values := source.CopiableValues()
	source.SetBasePT(8, 9)
	first, second := NewCreature("First", "{0}", 0, 0), NewCreature("Second", "{0}", 0, 0)
	first.ApplyCopiableValues(values)
	second.ApplyCopiableValues(values)
	for _, copied := range []*Card{first, second, first.NewCopy(), first.Copy()} {
		ability := copied.Abilities()[0].(*ActionDefinition)
		if copied.Power() != 2 || ability.activatedThisTurn || ability.Controller() != uuid.Nil() || ability.Targets()[0].IsChosen() {
			t.Fatal("copy retained source state or read changed source characteristics")
		}
	}
	first.Abilities()[0].(*ActionDefinition).activatedThisTurn = true
	first.SetBasePT(4, 5)
	third := NewCreature("Third", "{0}", 0, 0)
	third.ApplyCopiableValues(values)
	if third.Power() != 2 || second.Abilities()[0].(*ActionDefinition).activatedThisTurn || !action.activatedThisTurn || !target.IsChosen() {
		t.Fatal("copy modified the snapshot, another copy, or the source")
	}
}

func TestGameCloneTargetAndCastContextAliases(t *testing.T) {
	g := newPriorityTestGame()
	target := TargetCreature()
	card := NewInstant("Aliased Spell", "{0}", NewTargetedSpell(target, GainLife(1)), WithCastTarget(target))
	g.players[0].AddToHand(card)
	context := &CastContext{RevealedAtCast: []*Card{card}, ColorsSpent: map[Color]int{Red: 1}}
	g.stack.Push(&StackObject{Card: card, TargetSource: card, TargetSpecs: []Target{target}, CastContext: context})
	g.resolution.resolvingCard = card
	g.resolution.resolvingCastContext = context
	branch := g.Clone()
	object := branch.stack.Peek()
	clonedTarget := object.Card.CastTargets()[0]
	if clonedTarget == target || clonedTarget != object.TargetSpecs[0] || clonedTarget != object.Card.Abilities()[0].(*ActionDefinition).Targets()[0] {
		t.Fatal("clone did not preserve target aliases")
	}
	if object.CastContext == context || object.CastContext != branch.resolution.resolvingCastContext || object.CastContext.RevealedAtCast[0] != object.Card || branch.resolution.resolvingCard != object.Card {
		t.Fatal("clone did not preserve card and cast context aliases")
	}
	object.CastContext.ColorsSpent[Red] = 3
	if context.ColorsSpent[Red] != 1 {
		t.Fatal("clone changed the original cast context")
	}
}

func TestGameClonePreservesPermanentAbilityAliases(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	card := NewCreature("Limited Source", "{0}", 1, 1,
		WithAbility(NewActivatedAbility(GainLife(1), ManaCostOf("{0}"), WithOncePerTurn())))
	permanent := g.PutOnBattlefield(card, owner)
	branch := g.Clone()
	cloned := branch.FindPermanent(permanent.ID())
	if cloned.Card == card || cloned.RuntimeAbilities[0] == cloned.Card.Abilities()[0] {
		t.Fatal("payment clone shares runtime and card abilities")
	}
	cloned.RuntimeAbilities[0].(*ActionDefinition).activatedThisTurn = true
	if permanent.RuntimeAbilities[0].(*ActionDefinition).activatedThisTurn {
		t.Fatal("payment validation changed the source ability")
	}
}

func TestGameClonePendingOnlyAbility(t *testing.T) {
	g := newPriorityTestGame()
	ability := WheneverSpellCastTrigger(GainLife(1), false)
	event := &GameEvent{Type: EvtSpellCast, Amount: 3}
	g.triggers.pending = []*pendingTrigger{
		{ability: ability, event: event},
		{ability: ability, event: event},
	}
	branch := g.Clone()
	first, second := branch.triggers.pending[0], branch.triggers.pending[1]
	if first.ability == ability || first.ability != second.ability || first.event == event || first.event != second.event {
		t.Fatal("pending-only trigger state was shared or lost its aliases")
	}
	first.event.Amount = 7
	if event.Amount != 3 {
		t.Fatal("cloned event changed the original event")
	}
}

func TestGameCloneStaticEffectAlias(t *testing.T) {
	g := newPriorityTestGame()
	effect := FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil })
	card := NewCreature("Static Source", "{0}", 1, 1, WithAbility(StaticAbility(effect)))
	permanent := g.PutOnBattlefield(card, g.players[0].PlayerID())
	branch := g.Clone()
	cloned := branch.FindPermanent(permanent.ID()).RuntimeAbilities[0].(*StaticAbilityHolder).Effects[0]
	if cloned == effect {
		t.Fatal("static ability effect is shared with the original")
	}
	found := false
	for _, active := range branch.effects.effects {
		if printed, ok := active.(*printedAbilityContinuousEffect); ok && printed.ContinuousEffect == cloned {
			found = true
		}
	}
	if !found {
		t.Fatal("active effect lost its static ability alias")
	}
	cloned.SetSourceID(uuid.New())
	if permanent.RuntimeAbilities[0].(*StaticAbilityHolder).Effects[0].SourceID() != permanent.ID() {
		t.Fatal("cloned effect changed the original source")
	}
}

func TestGameCloneZoneBatchSnapshot(t *testing.T) {
	g := newPriorityTestGame()
	card := NewCreature("Snapshot Card", "{0}", 1, 1)
	g.players[0].AddToHand(card)
	before := g.Clone()
	g.triggers.zoneBatch = &zoneChangeBatch{before: before}
	branch := g.Clone()
	snapshot := branch.triggers.zoneBatch.before
	if snapshot == before {
		t.Fatal("simulation retained the original zone batch snapshot")
	}
	snapshot.players[0].Hand()[0].SetBasePT(7, 8)
	if before.players[0].Hand()[0].Power() != 1 || card.Power() != 1 {
		t.Fatal("snapshot mutation changed another branch")
	}
}

func TestGameCloneCopyEffectState(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	source := g.PutOnBattlefield(NewCreature("Copy Source", "{0}", 1, 1), owner)
	first := g.PutOnBattlefield(NewCreature("First Form", "{0}", 2, 2), owner)
	second := g.PutOnBattlefield(NewCreature("Second Form", "{0}", 3, 3), owner)
	g.effects.AddCopyEffect(source.ID(), first)
	branch := g.Clone()
	branch.effects.UpdateCopyEffect(source.ID(), branch.FindPermanent(second.ID()))
	if g.effects.CopyEffectCurrentName(source.ID()) != first.Name() {
		t.Fatal("simulation changed the original copy effect")
	}
}
