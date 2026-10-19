package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type spellDestructionInstruction struct {
	nextUntaps           *int
	allCombatPrevention  bool
	objectColor          *targetDeclaration
	deathReturn          bool
	turnAttackRule       bool
	optionalPayment      bool
	temporaryLayer       bool
	sourcePrevention     bool
	deferred             bool
	libraryProcedure     *[2]PlayerCollectionPredicate
	librarySearch        bool
	graveyardMove        *targetDeclaration
	manaCharacteristics  []TargetCharacteristicValuePredicate
	counterSubject       *targetDeclaration
	playerActionSubjects *PlayerCollectionPredicate
	drawCount            *int
	lifePermanent        *targetDeclaration
	target               *targetDeclaration
	subjects             ContextualPermanentCollectionPredicate
	cannotRegenerate     bool
	returnToHand         bool
	exilesPermanent      bool
	tapOperation         TapOperationKind
	lifeSubject          *DamageRecipientPredicate
	randomDiscardSubject *DamageRecipientPredicate
	drawSubject          *DamageRecipientPredicate
	manaSubject          *DamageRecipientPredicate
	preventionRecipient  *DamageRecipientPredicate
	damageRecipient      *DamageRecipientPredicate
	damageSubjects       *DamageRecipientCollectionPredicate
	damageAmount         *LivePermanentCountPredicate
	historyAmount        *PlayerDamageHistoryPredicate
	handSizeAmount       *PlayerHandSizePredicate
	characteristicAmount *TargetCharacteristicValuePredicate
	recipientAmount      *RecipientPermanentCountPredicate
}

// InspectSpellDestruction checks native destruction, return, tap, damage, life,
// draw, discard, counter, exile, mana changes and finite, global or source combat prevention.
// Layers must be current. Singular subjects must be complete native targets.
// All target declarations and instruction children must support pure inspection.
// The query checks resolution legality before it checks destruction destinations.
// True proves that a watched visit must move to the graveyard. False with a true
// known result means no proof exists in the supported tree. An unknown result
// must not be used as proof that the spell cannot destroy a permanent.
// A return can exclude exact watched visits from later destruction. Other
// nonempty earlier groups, counter, exile, damage, draw, life, mana and shield actions
// block later state proofs. Tap and destruction prefixes permit later proofs
// only when no selected visit can change state.
// Inspection does not execute callbacks, use shields, make choices or change state.
func (g *Game) InspectSpellDestruction(obj *StackObject, watched []PermanentVisit) (certain, known bool) {
	if g == nil || obj == nil || obj.ID == uuid.Nil || g.drawn || obj.leftStack || obj.IsAbility ||
		nilDefinition(obj.Card) || (!obj.Card.HasType(core.TypeInstant) && !obj.Card.HasType(core.TypeSorcery)) ||
		g.GetPlayer(obj.Controller) == nil || g.stack.FindByID(obj.ID) != obj || len(obj.TargetSpecs) != len(obj.Targets) {
		return false, false
	}
	instructions, ok := inspectSpellPermanentInstructions(obj.Effects)
	if !ok || !validCastingManaReads(obj, instructions) {
		return false, false
	}
	targets := make(map[*targetDeclaration]PermanentTargetPredicate, len(obj.TargetSpecs))
	damageTargets := make(map[*targetDeclaration]DamageTargetPredicate)
	damageGroups := make(map[*targetDeclaration]DamageTargetsPredicate)
	playerTargets := make(map[*targetDeclaration]PlayerTargetPredicate)
	spellTargets := make(map[*targetDeclaration]SpellTargetPredicate)
	objectTargets := make(map[*targetDeclaration]ObjectTargetPredicate)
	pluralTargets := make(map[*targetDeclaration]PermanentTargetsPredicate)
	graveyardTargets := make(map[*targetDeclaration]GraveyardTargetPredicate)
	for i, spec := range obj.TargetSpecs {
		declaration, ok := spec.(*targetDeclaration)
		if !ok || declaration == nil || obj.Targets[i] == uuid.Nil {
			return false, false
		}
		if _, duplicate := objectTargets[declaration]; duplicate {
			return false, false
		}
		if predicate, ok := InspectObjectTarget(&SpellOrPermanentTargetBinding{declaration}); ok {
			objectTargets[declaration] = predicate
			continue
		}
		if _, duplicate := graveyardTargets[declaration]; duplicate {
			return false, false
		}
		if _, duplicate := spellTargets[declaration]; duplicate {
			return false, false
		}
		if _, seen := damageGroups[declaration]; seen {
			continue
		}
		if _, seen := pluralTargets[declaration]; seen {
			continue
		}
		if _, duplicate := damageTargets[declaration]; duplicate {
			return false, false
		}
		if _, duplicate := targets[declaration]; duplicate {
			return false, false
		}
		if _, duplicate := playerTargets[declaration]; duplicate {
			return false, false
		}
		if predicate, ok := InspectGraveyardTarget(&CardTargetBinding{declaration}); ok {
			graveyardTargets[declaration] = predicate
			continue
		}
		if predicate, ok := InspectSpellTarget(&SpellTargetBinding{declaration}); ok {
			spellTargets[declaration] = predicate
			continue
		}
		if predicate, ok := InspectPlayerTarget(&PlayerTargetBinding{declaration}); ok {
			playerTargets[declaration] = predicate
			continue
		}
		if predicate, ok := InspectDamageTargets(&DamageTargetsBinding{declaration}); ok && (declaration.Min() != 1 || declaration.Max() != 1) {
			damageGroups[declaration] = predicate
			continue
		}
		if predicate, ok := InspectDamageTarget(&DamageTargetBinding{declaration}); ok {
			damageTargets[declaration] = predicate
			continue
		}
		if predicate, ok := InspectPermanentTarget(&PermanentTargetBinding{declaration}); ok {
			targets[declaration] = predicate
		} else if predicate, ok := InspectPermanentTargets(&PermanentTargetsBinding{declaration}); ok {
			pluralTargets[declaration] = predicate
		} else {
			return false, false
		}
	}
	for _, spec := range obj.Card.CastTargets() {
		declaration, ok := spec.(*targetDeclaration)
		if !ok || declaration == nil {
			continue
		}
		if predicate, ok := InspectDamageTargets(&DamageTargetsBinding{declaration}); ok {
			damageGroups[declaration] = predicate
			continue
		}
		minimum, maximum := TargetBounds(declaration, obj.XValue)
		if minimum == 0 && maximum == 0 {
			predicate, ok := InspectPermanentTargets(&PermanentTargetsBinding{declaration})
			if !ok {
				return false, false
			}
			pluralTargets[declaration] = predicate
		}
	}
	plans := make([]spellDestructionInstruction, 0, len(instructions))
	for _, instruction := range instructions {
		for _, declaration := range instruction.layerValueTargets {
			if _, declared := targets[declaration]; !declared {
				return false, false
			}
		}
		for _, read := range instruction.delayedManaReads {
			if _, declared := spellTargets[read.targetDeclaration]; !declared {
				return false, false
			}
		}
		plan := spellDestructionInstruction{drawCount: instruction.drawCount, nextUntaps: instruction.nextUntaps, allCombatPrevention: instruction.allCombatPrevention, deathReturn: instruction.deathReturn, turnAttackRule: instruction.turnAttackRule, optionalPayment: instruction.optionalPayment, deferred: instruction.deferred, temporaryLayer: instruction.temporaryLayer, sourcePrevention: instruction.sourcePrevention, librarySearch: instruction.librarySearch, manaCharacteristics: instruction.manaCharacteristics, handSizeAmount: instruction.handSizeAmount, recipientAmount: instruction.recipientAmount, characteristicAmount: instruction.characteristicAmount, historyAmount: instruction.historyAmount, damageAmount: instruction.damageAmount, cannotRegenerate: instruction.cannotRegenerate, returnToHand: instruction.returnToHand, tapOperation: instruction.tapOperation, exilesPermanent: instruction.exilesPermanent}
		if instruction.optionalPayment && instruction.counterSubject != nil {
			declaration := instruction.counterSubject.targetDeclaration
			if _, declared := spellTargets[declaration]; !declared {
				return false, false
			}
			plan.counterSubject = declaration
		}
		if instruction.objectColor != nil {
			declaration := instruction.objectColor.targetDeclaration
			if _, declared := objectTargets[declaration]; !declared {
				return false, false
			}
			plan.objectColor = declaration
		} else if instruction.libraryProcedure != nil {
			players := [2]PlayerCollectionPredicate{}
			for i, reference := range []PlayerRef{instruction.libraryProcedure.Chooser, instruction.libraryProcedure.Library} {
				predicate, ok := InspectPlayerCollection(OnePlayer(reference))
				if !ok {
					return false, false
				}
				if predicate.target != nil {
					if _, declared := playerTargets[predicate.target]; !declared {
						return false, false
					}
				}
				players[i] = predicate
			}
			plan.libraryProcedure = &players
		} else if instruction.allCombatPrevention || instruction.deathReturn || instruction.turnAttackRule || instruction.optionalPayment {
			plans = append(plans, plan)
			continue
		} else if instruction.librarySearch {
			plans = append(plans, plan)
			continue
		} else if instruction.graveyardMove != nil {
			declaration := instruction.graveyardMove.targetDeclaration
			if _, declared := graveyardTargets[declaration]; !declared {
				return false, false
			}
			plan.graveyardMove = declaration
		} else if instruction.counterSubject != nil {
			declaration := instruction.counterSubject.targetDeclaration
			if _, declared := spellTargets[declaration]; !declared {
				return false, false
			}
			plan.counterSubject = declaration
		} else if instruction.playerActionSubjects != nil {
			predicate, ok := InspectPlayerCollection(instruction.playerActionSubjects)
			if !ok {
				return false, false
			}
			plan.playerActionSubjects = &predicate
		} else if instruction.lifePermanent != nil {
			declaration := instruction.lifePermanent.targetDeclaration
			if _, declared := targets[declaration]; !declared {
				return false, false
			}
			plan.lifePermanent = declaration
		} else if instruction.randomDiscardSubject != nil {
			predicate, ok := InspectDamageRecipient(instruction.randomDiscardSubject)
			if !ok {
				return false, false
			}
			plan.randomDiscardSubject = &predicate
		} else if instruction.drawSubject != nil {
			predicate, ok := InspectDamageRecipient(instruction.drawSubject)
			if !ok {
				return false, false
			}
			plan.drawSubject = &predicate
		} else if instruction.manaSubject != nil {
			predicate, ok := InspectDamageRecipient(instruction.manaSubject)
			if !ok {
				return false, false
			}
			plan.manaSubject = &predicate
		} else if instruction.preventionRecipient != nil {
			predicate, ok := InspectDamageRecipient(instruction.preventionRecipient)
			if !ok {
				return false, false
			}
			plan.preventionRecipient = &predicate
		} else if instruction.lifeSubject != nil {
			predicate, ok := InspectDamageRecipient(instruction.lifeSubject)
			if !ok {
				return false, false
			}
			plan.lifeSubject = &predicate
		} else if instruction.damageRecipient != nil {
			predicate, ok := InspectDamageRecipient(instruction.damageRecipient)
			if !ok {
				return false, false
			}
			plan.damageRecipient = &predicate
		} else if instruction.damageRecipients != nil {
			predicate, ok := InspectDamageRecipientCollection(instruction.damageRecipients)
			if !ok {
				return false, false
			}
			plan.damageSubjects = &predicate
		} else if instruction.deferred && instruction.subject == nil && instruction.subjects == nil {
			plans = append(plans, plan)
			continue
		} else if instruction.subject != nil {
			target, ok := instruction.subject.(*PermanentTargetBinding)
			if !ok || target == nil {
				return false, false
			}
			if _, declared := targets[target.targetDeclaration]; !declared {
				return false, false
			}
			plan.target = target.targetDeclaration
		} else {
			subjects, ok := InspectContextualPermanentCollection(instruction.subjects)
			if !ok {
				return false, false
			}
			plan.subjects = subjects
		}
		plans = append(plans, plan)
	}
	selectedSpells := make(map[*targetDeclaration]uuid.UUID, len(spellTargets))
	selections := make(map[*targetDeclaration][]PermanentVisit, len(targets))
	anyLegal := false
	selectedObjects := make(map[*targetDeclaration]ObjectTargetSelection, len(objectTargets))
	for declaration, predicate := range objectTargets {
		selected, known := predicate.Select(g, obj)
		if !known {
			return false, false
		}
		selectedObjects[declaration] = selected
		anyLegal = anyLegal || !selected.Empty()
	}
	selectedGraveyard := make(map[*targetDeclaration]uuid.UUID, len(graveyardTargets))
	for declaration, predicate := range graveyardTargets {
		selected, known := predicate.Select(g, obj)
		if !known {
			return false, false
		}
		selectedGraveyard[declaration] = selected
		anyLegal = anyLegal || selected != uuid.Nil
	}
	for declaration, predicate := range spellTargets {
		selected, known := predicate.Select(g, obj)
		if !known {
			return false, false
		}
		selectedSpells[declaration] = selected
		anyLegal = anyLegal || selected != uuid.Nil
	}
	for declaration, predicate := range targets {
		selected, known := predicate.Select(g, obj)
		if !known {
			return false, false
		}
		selections[declaration] = selected
		anyLegal = anyLegal || len(selected) != 0
	}
	for _, predicate := range damageTargets {
		selected, known := predicate.Select(g, obj)
		if !known {
			return false, false
		}
		anyLegal = anyLegal || !selected.Empty()
	}
	playerBindings := make(map[*targetDeclaration][]capturedPermanent, len(playerTargets))
	for declaration, predicate := range playerTargets {
		id, known := predicate.Select(g, obj)
		if !known {
			return false, false
		}
		playerBindings[declaration] = nil
		if id != uuid.Nil {
			playerBindings[declaration] = []capturedPermanent{{id: id}}
			anyLegal = true
		}
	}
	for declaration, selected := range selections {
		playerBindings[declaration] = nil
		for _, visit := range selected {
			playerBindings[declaration] = append(playerBindings[declaration], capturedPermanent{id: visit.ID, incarnation: visit.Incarnation})
		}
	}
	for declaration, predicate := range pluralTargets {
		selected, known := predicate.Select(g, obj)
		if !known {
			return false, false
		}
		playerBindings[declaration] = nil
		for _, visit := range selected {
			playerBindings[declaration] = append(playerBindings[declaration], capturedPermanent{id: visit.ID, incarnation: visit.Incarnation})
		}
		anyLegal = anyLegal || len(selected) != 0
	}
	for declaration, predicate := range damageGroups {
		selected, known := predicate.Select(g, obj)
		if !known {
			return false, false
		}
		playerBindings[declaration] = nil
		for _, recipient := range selected {
			value := capturedPermanent{id: recipient.Player}
			if recipient.Permanent.ID != uuid.Nil {
				value = capturedPermanent{id: recipient.Permanent.ID, incarnation: recipient.Permanent.Incarnation}
			}
			playerBindings[declaration] = append(playerBindings[declaration], value)
		}
		anyLegal = anyLegal || len(selected) != 0
	}
	ctx := &EffectContext{Game: g, Controller: obj.Controller, SourceID: obj.SourceID, targetBindings: playerBindings}
	recipients := make([]DamageTargetSelection, len(plans))
	for i, plan := range plans {
		if plan.optionalPayment || plan.deferred && plan.target == nil && plan.subjects.root == nil {
			continue
		}
		for _, amount := range plan.manaCharacteristics {
			if !amount.bindingsKnown(ctx) {
				return false, false
			}
		}
		if plan.recipientAmount != nil {
			if _, known := plan.recipientAmount.Read(ctx, obj.Controller); !known {
				return false, false
			}
		}
		if plan.characteristicAmount != nil && !plan.characteristicAmount.bindingsKnown(ctx) {
			return false, false
		}
		if plan.handSizeAmount != nil && !plan.handSizeAmount.bindingsKnown(ctx) {
			return false, false
		}
		if plan.historyAmount != nil && !plan.historyAmount.bindingsKnown(ctx) {
			return false, false
		}
		if plan.damageAmount != nil {
			if _, known := plan.damageAmount.Read(ctx); !known {
				return false, false
			}
		}
		if plan.libraryProcedure != nil {
			for _, player := range plan.libraryProcedure {
				if !player.bindingsKnown(ctx) {
					return false, false
				}
			}
		}
		recipient := plan.damageRecipient
		if plan.randomDiscardSubject != nil {
			recipient = plan.randomDiscardSubject
		}
		if plan.drawSubject != nil {
			recipient = plan.drawSubject
		}
		if plan.preventionRecipient != nil {
			recipient = plan.preventionRecipient
		}
		if plan.lifeSubject != nil {
			recipient = plan.lifeSubject
		}
		if plan.manaSubject != nil {
			recipient = plan.manaSubject
		}
		if recipient != nil {
			selected, known := recipient.Select(g, obj)
			if !known {
				return false, false
			}
			recipients[i] = selected
		}
		if plan.playerActionSubjects != nil && !plan.playerActionSubjects.bindingsKnown(ctx) {
			return false, false
		}
		if plan.damageSubjects != nil && !plan.damageSubjects.bindingsKnown(ctx) {
			return false, false
		}
		if !plan.allCombatPrevention && plan.objectColor == nil && !plan.optionalPayment && !plan.deathReturn && !plan.turnAttackRule && plan.libraryProcedure == nil && !plan.librarySearch && plan.graveyardMove == nil && plan.counterSubject == nil && plan.playerActionSubjects == nil && plan.target == nil && plan.lifePermanent == nil && plan.randomDiscardSubject == nil && plan.drawSubject == nil && plan.preventionRecipient == nil && plan.lifeSubject == nil && plan.manaSubject == nil && plan.damageRecipient == nil && plan.damageSubjects == nil && !plan.subjects.root.bindingsKnown(ctx) {
			return false, false
		}
	}
	if len(obj.Targets) != 0 && !anyLegal {
		return false, true
	}
	visits := make(map[PermanentVisit]bool, len(watched))
	for _, visit := range watched {
		visits[visit] = true
	}
	laterDestruction := make([]bool, len(plans))
	for i := len(plans) - 2; i >= 0; i-- {
		laterDestruction[i] = laterDestruction[i+1] || (plans[i+1].nextUntaps == nil && !plans[i+1].allCombatPrevention && plans[i+1].objectColor == nil && !plans[i+1].optionalPayment && !plans[i+1].deathReturn && !plans[i+1].turnAttackRule && !plans[i+1].deferred && !plans[i+1].temporaryLayer && !plans[i+1].sourcePrevention && plans[i+1].libraryProcedure == nil && !plans[i+1].librarySearch && plans[i+1].graveyardMove == nil && plans[i+1].counterSubject == nil && plans[i+1].playerActionSubjects == nil && !plans[i+1].exilesPermanent && !plans[i+1].returnToHand && plans[i+1].tapOperation == 0 && plans[i+1].lifePermanent == nil && plans[i+1].damageRecipient == nil && plans[i+1].lifeSubject == nil && plans[i+1].randomDiscardSubject == nil && plans[i+1].drawSubject == nil && plans[i+1].manaSubject == nil && plans[i+1].preventionRecipient == nil && plans[i+1].damageSubjects == nil)
	}
	for i, plan := range plans {
		if plan.libraryProcedure != nil {
			active := true
			for _, player := range plan.libraryProcedure {
				selected, known := player.Select(ctx)
				if !known {
					return false, false
				}
				active = active && len(selected) != 0
			}
			if active && laterDestruction[i] {
				return false, false
			}
			continue
		}
		if plan.objectColor != nil {
			if !selectedObjects[plan.objectColor].Empty() && laterDestruction[i] {
				return false, false
			}
			continue
		}
		if plan.optionalPayment && plan.counterSubject != nil && selectedSpells[plan.counterSubject] == uuid.Nil {
			continue
		}
		if plan.allCombatPrevention || plan.optionalPayment || plan.deathReturn || plan.turnAttackRule || plan.librarySearch {
			if laterDestruction[i] {
				return false, false
			}
			continue
		}
		if plan.graveyardMove != nil {
			if selectedGraveyard[plan.graveyardMove] != uuid.Nil && laterDestruction[i] {
				return false, false
			}
			continue
		}
		if plan.counterSubject != nil {
			if selectedSpells[plan.counterSubject] != uuid.Nil && laterDestruction[i] {
				return false, false
			}
			continue
		}
		if plan.playerActionSubjects != nil {
			selected, known := plan.playerActionSubjects.Select(ctx)
			if !known || len(selected) != 0 && laterDestruction[i] && (plan.drawCount == nil || *plan.drawCount > 0) {
				return false, false
			}
			continue
		}
		if plan.lifePermanent != nil {
			if len(selections[plan.lifePermanent]) != 0 && laterDestruction[i] {
				return false, false
			}
			continue
		}
		if plan.damageSubjects != nil {
			selected, known := plan.damageSubjects.Select(ctx)
			if !known || len(selected) != 0 && laterDestruction[i] {
				return false, false
			}
			continue
		}
		if plan.randomDiscardSubject != nil || plan.drawSubject != nil || plan.damageRecipient != nil || plan.lifeSubject != nil || plan.manaSubject != nil || plan.preventionRecipient != nil {
			if !recipients[i].Empty() && laterDestruction[i] {
				return false, false
			}
			continue
		}
		if plan.deferred {
			continue
		}
		selected := selections[plan.target]
		if plan.target == nil {
			selected = plan.subjects.Select(ctx)
		}
		if plan.nextUntaps != nil {
			if *plan.nextUntaps > 0 && len(selected) != 0 && laterDestruction[i] {
				return false, false
			}
			continue
		}
		if plan.temporaryLayer || plan.sourcePrevention {
			if len(selected) != 0 && laterDestruction[i] {
				return false, false
			}
			continue
		}
		if plan.tapOperation != 0 {
			if laterDestruction[i] {
				mayChange, known := g.InspectTapStateChange(plan.tapOperation, selected)
				if !known || mayChange {
					return false, false
				}
			}
			continue
		}
		if plan.exilesPermanent {
			if len(selected) != 0 && laterDestruction[i] {
				return false, false
			}
			continue
		}
		if plan.returnToHand {
			// A later zone entry cannot restore a returned battlefield visit.
			for _, visit := range selected {
				delete(visits, visit)
			}
			if len(visits) == 0 || !laterDestruction[i] {
				return false, true
			}
			if len(selected) != 0 {
				return false, false
			}
			continue
		}
		proof := false
		for _, visit := range selected {
			destinations, known := g.InspectDestructionDestinations(visit, plan.cannotRegenerate)
			if !known {
				return false, false
			}
			if visits[visit] && destinations == DestructionToGraveyard {
				proof = true
			}
		}
		if proof {
			return true, true
		}
		if laterDestruction[i] {
			mayChange, known := g.InspectDestructionStateChange(selected)
			if !known || mayChange {
				return false, false
			}
		}
	}
	return false, true
}

type spellPermanentInstruction struct {
	nextUntaps           *int
	allCombatPrevention  bool
	objectColor          *SpellOrPermanentTargetBinding
	unpaidWrite          *PermanentSetBinding
	deathReturn          bool
	turnAttackRule       bool
	optionalPayment      bool
	deathWrite           *PermanentSetBinding
	deathRead            *PermanentSetBinding
	deathMember          *PermanentBinding
	temporaryLayer       bool
	sourcePrevention     bool
	deferred             bool
	layerValueTargets    []*targetDeclaration
	libraryProcedure     *LibraryProcedureInstruction
	librarySearch        bool
	graveyardMove        *CardTargetBinding
	delayedManaReads     []*SpellTargetBinding
	lostManaWrite        *LostManaBinding
	lostManaRead         *LostManaBinding
	manaCharacteristics  []TargetCharacteristicValuePredicate
	castingManaReads     []*NumberBinding
	counterSubject       *SpellTargetBinding
	playerActionSubjects PlayerSet
	drawCount            *int
	discardWrite         *PlayerNumbersBinding
	discardRead          *PlayerNumbersBinding
	damageWrite          *DamageResultsBinding
	singularDamage       bool
	lifeResultRead       *NumberBinding
	lifeOutcomeRead      *DamageResultsBinding
	lifePermanent        *PermanentTargetBinding
	resultWrite          *NumberBinding
	resultReads          []*NumberBinding
	resultRead           *NumberBinding
	subject              PermanentRef
	subjects             PermanentSet
	cannotRegenerate     bool
	returnToHand         bool
	exilesPermanent      bool
	tapOperation         TapOperationKind
	lifeSubject          PlayerRef
	randomDiscardSubject PlayerRef
	drawSubject          PlayerRef
	manaSubject          PlayerRef
	preventionRecipient  DamageRecipientRef
	damageRecipient      DamageRecipientRef
	damageRecipients     DamageRecipientSet
	damageAmount         *LivePermanentCountPredicate
	historyAmount        *PlayerDamageHistoryPredicate
	handSizeAmount       *PlayerHandSizePredicate
	characteristicAmount *TargetCharacteristicValuePredicate
	recipientAmount      *RecipientPermanentCountPredicate
}

func inspectSpellPermanentInstructions(effects []ResolvingEffect) ([]spellPermanentInstruction, bool) {
	instructions, ok := inspectSpellPermanentInstructionsUnchecked(effects)
	if !ok {
		return nil, false
	}
	unpaidResults := make(map[*PermanentSetBinding]bool)
	deathResults := make(map[*PermanentSetBinding]bool)
	deathMembers := make(map[*PermanentBinding]bool)
	lostMana := make(map[*LostManaBinding]bool)
	produced := make(map[*NumberBinding]bool)
	discardResults := make(map[*PlayerNumbersBinding]bool)
	discardMembers := make(map[*NumberBinding]bool)
	damageNumbers := make(map[*NumberBinding]bool)
	damageOutcomes := make(map[*DamageResultsBinding]bool)
	for _, instruction := range instructions {
		if instruction.unpaidWrite != nil {
			if unpaidResults[instruction.unpaidWrite] {
				return nil, false
			}
			unpaidResults[instruction.unpaidWrite] = true
		}
		if instruction.deathRead != nil {
			if !deathResults[instruction.deathRead] || deathMembers[instruction.deathMember] {
				return nil, false
			}
			deathMembers[instruction.deathMember] = true
		}
		if instruction.deathWrite != nil {
			if deathResults[instruction.deathWrite] {
				return nil, false
			}
			deathResults[instruction.deathWrite] = true
		}
		if instruction.lostManaRead != nil && !lostMana[instruction.lostManaRead] {
			return nil, false
		}
		if instruction.lostManaWrite != nil {
			if lostMana[instruction.lostManaWrite] {
				return nil, false
			}
			lostMana[instruction.lostManaWrite] = true
		}
		if instruction.discardRead != nil && !discardResults[instruction.discardRead] {
			return nil, false
		}
		if instruction.discardWrite != nil {
			if discardResults[instruction.discardWrite] {
				return nil, false
			}
			for _, member := range instruction.discardWrite.members {
				if discardMembers[member] || produced[member] {
					return nil, false
				}
				discardMembers[member] = true
			}
			discardResults[instruction.discardWrite] = true
		}
		if instruction.lifeResultRead != nil && !damageNumbers[instruction.lifeResultRead] {
			return nil, false
		}
		if instruction.lifeOutcomeRead != nil && !damageOutcomes[instruction.lifeOutcomeRead] {
			return nil, false
		}
		if instruction.damageWrite != nil {
			if _, duplicate := damageOutcomes[instruction.damageWrite]; duplicate {
				return nil, false
			}
			damageOutcomes[instruction.damageWrite] = instruction.singularDamage
			damageNumbers[instruction.resultWrite] = true
		}
		for _, read := range instruction.resultReads {
			if !produced[read] {
				return nil, false
			}
		}
		if instruction.resultRead != nil && !produced[instruction.resultRead] {
			return nil, false
		}
		if instruction.resultWrite != nil {
			if produced[instruction.resultWrite] || discardMembers[instruction.resultWrite] {
				return nil, false
			}
			produced[instruction.resultWrite] = true
		}
	}
	return instructions, true
}

func inspectSpellPermanentInstructionsUnchecked(effects []ResolvingEffect) ([]spellPermanentInstruction, bool) {
	var out []spellPermanentInstruction
	for _, effect := range effects {
		if instruction, ok := InspectManaPaymentInstruction(effect); ok {
			plan, ok := inspectOptionalCounterPayment(instruction)
			if !ok {
				return nil, false
			}
			out = append(out, plan)
			continue
		}
		if instruction, ok := InspectPermanentLifePaymentInstruction(effect); ok {
			if !inspectOptionalUnpaidDestruction(instruction) {
				return nil, false
			}
			out = append(out, spellPermanentInstruction{optionalPayment: true, unpaidWrite: instruction.Unpaid})
			continue
		}
		if instruction, ok := InspectSavedPlayerInstruction(effect); ok {
			if !inspectSavedPlayerDelayedDestruction(instruction) {
				return nil, false
			}
			out = append(out, spellPermanentInstruction{deferred: true})
			continue
		}
		var children []ResolvingEffect
		switch effect := effect.(type) {
		case *sequenceEffect:
			if effect == nil {
				return nil, false
			}
			children = effect.effects
		case *compositeEffect:
			if effect == nil {
				return nil, false
			}
			children = effect.effects
		}
		switch effect.(type) {
		case *sequenceEffect, *compositeEffect:
			nested, ok := inspectSpellPermanentInstructionsUnchecked(children)
			if !ok {
				return nil, false
			}
			out = append(out, nested...)
		default:
			if instructions, ok := InspectDirectDestruction(effect); ok {
				for _, instruction := range instructions {
					out = append(out, spellPermanentInstruction{subject: instruction.Subject, subjects: instruction.Subjects, cannotRegenerate: instruction.CannotRegenerate})
				}
			} else if instruction, ok := InspectTurnAttackRuleInstruction(effect); ok {
				if !pureTurnAttackRule(instruction) {
					return nil, false
				}
				out = append(out, spellPermanentInstruction{turnAttackRule: true})
			} else if instruction, ok := InspectNextCombatThisTurnInstruction(effect); ok {
				nested, known := inspectCombatThisTurnDestruction(instruction.Body)
				if !known {
					return nil, false
				}
				out = append(out, nested...)
			} else if instruction, ok := InspectNextEndStepInstruction(effect); ok {
				nested, known := inspectDelayedPermanentDestruction(instruction.Body)
				if !known {
					return nil, false
				}
				out = append(out, nested...)
			} else if instruction, ok := InspectTemporaryMixedLayerInstruction(effect); ok {
				plan := spellPermanentInstruction{temporaryLayer: true, subject: instruction.Subject}
				for _, adjustment := range instruction.Adjustments {
					for _, amount := range []ResolutionValue{adjustment.Power, adjustment.Toughness} {
						reads, known := inspectTemporaryPTValue(amount)
						if !known {
							return nil, false
						}
						plan.layerValueTargets = append(plan.layerValueTargets, reads...)
					}
				}
				out = append(out, plan)
			} else if instruction, ok := InspectPermanentBranchInstruction(effect); ok {
				nested, known := inspectTemporaryLayerBranch(instruction)
				if !known {
					return nil, false
				}
				out = append(out, nested...)
			} else if instruction, ok := InspectTemporaryRampageInstruction(effect); ok {
				out = append(out, spellPermanentInstruction{temporaryLayer: true, subject: instruction.Subject})
			} else if instruction, ok := InspectTemporaryPTInstruction(effect); ok {
				out = append(out, spellPermanentInstruction{temporaryLayer: true, subject: instruction.Subject, subjects: instruction.Subjects})
			} else if instruction, ok := InspectTemporaryPTValueInstruction(effect); ok {
				plan := spellPermanentInstruction{temporaryLayer: true, subject: instruction.Subject}
				for _, adjustment := range instruction.Adjustments {
					for _, amount := range []ResolutionValue{adjustment.Power, adjustment.Toughness} {
						reads, known := inspectTemporaryPTValue(amount)
						if !known {
							return nil, false
						}
						plan.layerValueTargets = append(plan.layerValueTargets, reads...)
					}
				}
				out = append(out, plan)
			} else if instruction, ok := InspectObjectColorInstruction(effect); ok {
				out = append(out, spellPermanentInstruction{objectColor: instruction.Subject})
			} else if instruction, ok := InspectTemporaryColorInstruction(effect); ok {
				out = append(out, spellPermanentInstruction{temporaryLayer: true, subjects: instruction.Subjects})
			} else if instruction, ok := InspectTemporaryKeywordInstruction(effect); ok {
				out = append(out, spellPermanentInstruction{temporaryLayer: true, subject: instruction.Subject, subjects: instruction.Subjects})
			} else if instruction, ok := InspectLibraryProcedureInstruction(effect); ok {
				out = append(out, spellPermanentInstruction{libraryProcedure: &instruction})
			} else if _, ok := InspectLibrarySearchInstruction(effect); ok {
				out = append(out, spellPermanentInstruction{librarySearch: true})
			} else if instruction, ok := InspectGraveyardBattlefieldInstruction(effect); ok {
				if _, ok := instruction.Controller.(controllerRef); !ok {
					return nil, false
				}
				subject, ok := instruction.Subject.(*CardTargetBinding)
				if !ok || subject == nil {
					return nil, false
				}
				if _, ok := InspectGraveyardTarget(subject); !ok {
					return nil, false
				}
				out = append(out, spellPermanentInstruction{graveyardMove: subject})
			} else if instruction, ok := InspectGraveyardHandInstruction(effect); ok {
				subject, ok := instruction.Subject.(*CardTargetBinding)
				if !ok || subject == nil {
					return nil, false
				}
				if _, ok := InspectGraveyardTarget(subject); !ok {
					return nil, false
				}
				out = append(out, spellPermanentInstruction{graveyardMove: subject})
			} else if instructions, ok := InspectCounterInstructions(effect); ok {
				for _, instruction := range instructions {
					subject, ok := instruction.Subject.(*SpellTargetBinding)
					if !ok || subject == nil {
						return nil, false
					}
					if _, ok := InspectSpellTarget(subject); !ok {
						return nil, false
					}
					out = append(out, spellPermanentInstruction{counterSubject: subject})
				}
			} else if instruction, ok := InspectCreatureDeathInstruction(effect); ok {
				out = append(out, spellPermanentInstruction{subjects: instruction.Subjects, cannotRegenerate: instruction.CannotRegenerate, deathWrite: instruction.Result})
			} else if instruction, ok := InspectCreatureDeathReturnInstruction(effect); ok {
				reads, known := inspectCreatureDeathReturnPlayer(instruction)
				if !known {
					return nil, false
				}
				out = append(out, spellPermanentInstruction{deathReturn: true, deathRead: instruction.Result, deathMember: instruction.Member, layerValueTargets: reads})
			} else if instruction, ok := InspectCreatureDeathCountInstruction(effect); ok {
				out = append(out, spellPermanentInstruction{subjects: instruction.Subjects, cannotRegenerate: instruction.CannotRegenerate, resultWrite: instruction.Result})
			} else if instruction, ok := InspectDestructionResult(effect); ok {
				out = append(out, spellPermanentInstruction{subjects: instruction.Subjects, cannotRegenerate: instruction.CannotRegenerate, resultWrite: instruction.Result})
			} else if instructions, ok := InspectDirectReturn(effect); ok {
				for _, instruction := range instructions {
					out = append(out, spellPermanentInstruction{subject: instruction.Subject, subjects: instruction.Subjects, returnToHand: true})
				}
			} else if instructions, ok := InspectPermanentExileInstructions(effect); ok {
				for _, instruction := range instructions {
					out = append(out, spellPermanentInstruction{subject: instruction.Subject, subjects: instruction.Subjects, exilesPermanent: true})
				}
			} else if instruction, ok := InspectNextUntapInstruction(effect); ok {
				out = append(out, spellPermanentInstruction{nextUntaps: &instruction.Count, subject: instruction.Subject, subjects: instruction.Subjects})
			} else if instructions, ok := InspectTapInstructions(effect); ok {
				for _, instruction := range instructions {
					out = append(out, spellPermanentInstruction{subject: instruction.Subject, subjects: instruction.Subjects, tapOperation: instruction.Operation})
				}
			} else if instruction, ok := InspectDamageResultInstruction(effect); ok {
				if !pureSourceDamageOperands(instruction.Source, instruction.Amount) {
					return nil, false
				}
				plan := spellPermanentInstruction{resultWrite: instruction.Total, damageWrite: instruction.Results, singularDamage: instruction.Recipient != nil}
				if instruction.Recipient != nil {
					if _, ok := InspectDamageRecipient(instruction.Recipient); !ok {
						return nil, false
					}
					plan.damageRecipient = instruction.Recipient
				} else {
					if _, ok := InspectDamageRecipientCollection(instruction.Recipients); !ok {
						return nil, false
					}
					plan.damageRecipients = instruction.Recipients
				}
				out = append(out, plan)
			} else if instruction, ok := InspectDiscardUpToInstruction(effect); ok {
				maximum, fixed := instruction.Maximum.(fixedValue)
				if !fixed || maximum.n < 0 {
					return nil, false
				}
				if _, ok := InspectPlayerCollection(instruction.Subjects); !ok {
					return nil, false
				}
				out = append(out, spellPermanentInstruction{playerActionSubjects: instruction.Subjects, discardWrite: instruction.Result})
			} else if instruction, ok := InspectPlayerCollectionDamageInstruction(effect); ok {
				if _, ok := instruction.Source.(sourceRef); !ok {
					return nil, false
				}
				if _, ok := InspectPlayerCollection(instruction.Subjects); !ok {
					return nil, false
				}
				plan := spellPermanentInstruction{damageRecipients: PlayerDamageRecipients(instruction.Subjects)}
				if amount, ok := InspectRecipientPermanentCount(instruction.Amount, instruction.Member); ok {
					plan.recipientAmount = &amount
				} else if amount, ok := InspectRecipientDiscardAmount(instruction.Amount, instruction.Member); ok {
					plan.discardRead = amount.Result
				} else {
					return nil, false
				}
				out = append(out, plan)
			} else if instructions, ok := InspectEvenDamageInstructions(effect); ok {
				for _, instruction := range instructions {
					if !pureSourceDamageOperands(instruction.Source, instruction.Total) {
						return nil, false
					}
					if _, ok := InspectDamageRecipientCollection(instruction.Recipients); !ok {
						return nil, false
					}
					out = append(out, spellPermanentInstruction{damageRecipients: instruction.Recipients})
				}
			} else if instructions, ok := InspectCollectionDamageInstructions(effect); ok {
				for _, instruction := range instructions {
					plan := spellPermanentInstruction{damageRecipients: instruction.Recipients}
					if !pureSourceDamageOperands(instruction.Source, instruction.Amount) {
						if _, ok := instruction.Source.(sourceRef); !ok {
							return nil, false
						}
						binding, ok := destructionResultRead(instruction.Amount)
						if !ok {
							return nil, false
						}
						plan.resultRead = binding
					}
					out = append(out, plan)
				}
			} else if instructions, ok := InspectPacketDamageInstructions(effect); ok {
				for _, instruction := range instructions {
					for _, packet := range instruction.Packets {
						if _, ok := InspectDamageRecipient(packet.recipient); !ok {
							return nil, false
						}
						plan := spellPermanentInstruction{damageRecipient: packet.recipient}
						if !pureSourceDamageOperands(instruction.Source, packet.amount) {
							if _, ok := instruction.Source.(sourceRef); !ok {
								return nil, false
							}
							amount, ok := InspectLivePermanentCount(packet.amount)
							if !ok {
								return nil, false
							}
							plan.damageAmount = &amount
						}
						out = append(out, plan)
					}
				}
			} else if _, ok := InspectGlobalCombatPreventionInstruction(effect); ok {
				out = append(out, spellPermanentInstruction{allCombatPrevention: true})
			} else if instruction, ok := InspectCombatSourcePreventionInstruction(effect); ok {
				out = append(out, spellPermanentInstruction{sourcePrevention: true, subject: instruction.Subject})
			} else if instructions, ok := InspectFuturePreventionInstructions(effect); ok {
				for _, instruction := range instructions {
					if _, ok := InspectDamageRecipient(instruction.Recipient); !ok || !pureFixedOrXAmount(instruction.Amount) {
						return nil, false
					}
					out = append(out, spellPermanentInstruction{preventionRecipient: instruction.Recipient})
				}
			} else if instruction, ok := InspectForcedManaLossInstruction(effect); ok {
				if _, ok := InspectDamageRecipient(instruction.Subject); !ok {
					return nil, false
				}
				if _, ok := inspectContextualPermanentCollectionFilter(instruction.Filter); !ok {
					return nil, false
				}
				out = append(out, spellPermanentInstruction{manaSubject: instruction.Subject, lostManaWrite: instruction.Result})
			} else if instruction, ok := InspectLostManaInstruction(effect); ok {
				if _, ok := InspectDamageRecipient(instruction.Subject); !ok {
					return nil, false
				}
				plan := spellPermanentInstruction{manaSubject: instruction.Subject}
				if instruction.Restore {
					plan.lostManaRead = instruction.Result
				} else {
					plan.lostManaWrite = instruction.Result
				}
				out = append(out, plan)
			} else if instruction, ok := InspectNextMainPhaseInstruction(effect); ok {
				plan, ok := inspectNextMainMana(instruction)
				if !ok {
					return nil, false
				}
				out = append(out, plan)
			} else if instruction, ok := InspectRequiredTapInstruction(effect); ok {
				plans, ok := inspectRequiredTapMana(instruction)
				if !ok {
					return nil, false
				}
				out = append(out, plans...)
			} else if instructions, ok := InspectManaProductionInstructions(effect); ok {
				for _, instruction := range instructions {
					plan, ok := inspectSpellManaProduction(instruction)
					if !ok {
						return nil, false
					}
					out = append(out, plan)
				}
			} else if instructions, ok := InspectManaLossInstructions(effect); ok {
				for _, instruction := range instructions {
					if _, ok := InspectDamageRecipient(instruction.Subject); !ok {
						return nil, false
					}
					out = append(out, spellPermanentInstruction{manaSubject: instruction.Subject})
				}
			} else if instructions, ok := InspectRandomDiscardInstructions(effect); ok {
				for _, instruction := range instructions {
					if _, ok := InspectDamageRecipient(instruction.Subject); !ok || !pureFixedOrXAmount(instruction.Amount) {
						return nil, false
					}
					out = append(out, spellPermanentInstruction{randomDiscardSubject: instruction.Subject})
				}
			} else if instruction, ok := InspectWholeHandDiscardInstruction(effect); ok {
				players := instruction.Subjects
				if instruction.Subject != nil {
					players = OnePlayer(instruction.Subject)
				}
				if _, ok := InspectPlayerCollection(players); !ok {
					return nil, false
				}
				out = append(out, spellPermanentInstruction{playerActionSubjects: players})
			} else if instruction, ok := InspectPlayerDrawInstruction(effect); ok {
				value := instruction.Amount
				if legacy, ok := value.(legacyResolutionValue); ok {
					fixed, ok := legacy.value.(fixedValue)
					if !ok {
						return nil, false
					}
					value = fixed
				}
				amount, fixed := value.(fixedValue)
				if !fixed || amount.n < 0 {
					return nil, false
				}
				if _, ok := InspectPlayerCollection(instruction.Subjects); !ok {
					return nil, false
				}
				out = append(out, spellPermanentInstruction{playerActionSubjects: instruction.Subjects, drawCount: &amount.n})
			} else if instructions, ok := InspectDrawInstructions(effect); ok {
				for _, instruction := range instructions {
					if _, ok := InspectDamageRecipient(instruction.Subject); !ok || !pureFixedOrXAmount(instruction.Amount) {
						return nil, false
					}
					out = append(out, spellPermanentInstruction{drawSubject: instruction.Subject})
				}
			} else if instructions, ok := InspectLifeInstructions(effect); ok {
				for _, instruction := range instructions {
					if permanent, ok := InspectPermanentLifeInstruction(instruction.Effect()); ok {
						out = append(out, spellPermanentInstruction{lifePermanent: permanent.Subject})
						continue
					}
					if _, ok := InspectDamageRecipient(instruction.Subject); !ok {
						return nil, false
					}
					plan := spellPermanentInstruction{lifeSubject: instruction.Subject}
					if !pureFixedOrXAmount(instruction.Amount) {
						switch amount := instruction.Amount.(type) {
						case savedNumberValue:
							if amount.binding == nil || amount.binding.name == "" {
								return nil, false
							}
							plan.lifeResultRead = amount.binding
						case clampedDamageAmount:
							if amount.results == nil || amount.results.name == "" {
								return nil, false
							}
							plan.lifeOutcomeRead = amount.results
						default:
							if characteristic, ok := InspectTargetCharacteristicValue(instruction.Amount); ok {
								plan.characteristicAmount = &characteristic
								break
							}
							history, ok := InspectPlayerDamageHistory(instruction.Amount)
							if !ok {
								return nil, false
							}
							plan.historyAmount = &history
						}
					}
					out = append(out, plan)
				}
			} else if instructions, ok := InspectSingularDamageInstructions(effect); ok {
				for _, instruction := range instructions {
					_, ok := InspectDamageRecipient(instruction.Recipient)
					if !ok {
						return nil, false
					}
					plan := spellPermanentInstruction{damageRecipient: instruction.Recipient}
					if !pureSourceDamageOperands(instruction.Source, instruction.Amount) {
						if _, ok := instruction.Source.(sourceRef); !ok {
							return nil, false
						}
						if reads, ok := inspectSavedNumberSum(instruction.Amount); ok {
							plan.resultReads = reads
						} else if hand, ok := InspectPlayerHandSize(instruction.Amount); ok {
							plan.handSizeAmount = &hand
						} else {
							history, ok := InspectPlayerDamageHistory(instruction.Amount)
							if !ok {
								return nil, false
							}
							plan.historyAmount = &history
						}
					}
					out = append(out, plan)
				}
			} else {
				return nil, false
			}
		}
	}
	return out, true
}
