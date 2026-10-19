package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type castStackObjectOptions struct {
	// An empty prepared division must not cause a second choice after payment.
	DamageDivisionPrepared bool
	DamageDistribution     map[uuid.UUID]int
	CapturedNumbers        map[*NumberBinding]int
	CapturedCards          map[*CardBinding]capturedCard
	RevealedHandCards      []capturedCard
	HandTargetCards        map[uuid.UUID]capturedCard
	ExileTargetCards       map[uuid.UUID]capturedCard
	AnteTargetCards        map[uuid.UUID]capturedCard
	CapturedCardSets       map[*CardsBinding][]capturedCard
	CapturedPermanents     map[*PermanentBinding]capturedPermanent
	Card                   Card
	Controller             uuid.UUID
	Targets                []uuid.UUID
	TargetSpecs            []Target
	XValue                 int
	CastZone               Zone
	SourceID               uuid.UUID
	IsCopy                 bool
	ExileOnLeaveStack      bool
	SnapshotCast           bool
	PromptTargets          bool
	ModePrompt             string
	ModalSelection         *modalSpellSelection
}

func (g *Game) prepareSpellTargetGroups(controller uuid.UUID, card Card, groups [][]uuid.UUID, x int) ([]uuid.UUID, []Target, error) {
	return g.prepareTargetGroups(controller, SpellTargetSource(card), card.CastTargets(), groups, x, "spell target group")
}

func (g *Game) prepareTargetGroups(controller uuid.UUID, card Card, specs []Target, groups [][]uuid.UUID, x int, label string) ([]uuid.UUID, []Target, error) {
	if len(groups) != len(specs) {
		return nil, nil, fmt.Errorf("%s requires %d target groups", label, len(specs))
	}
	var targets []uuid.UUID
	var aligned []Target
	for i, spec := range specs {
		chosen := append([]uuid.UUID(nil), groups[i]...)
		query := g.targetBindingsQuery(aligned, targets, nil, x)
		chosen = query.acquireOpponentChosenTargets(controller, card, []Target{spec}, chosen)
		if err := query.validateActionTargets(controller, card, []Target{spec}, chosen, x, label); err != nil {
			return nil, nil, fmt.Errorf("target group %d: %w", i+1, err)
		}
		targets = append(targets, chosen...)
		for range chosen {
			aligned = append(aligned, spec)
		}
	}
	return targets, aligned, nil
}

func (g *Game) pushCastSpellObject(opts castStackObjectOptions) (*StackObject, error) {
	card := opts.Card
	if card == nil {
		return nil, nil
	}
	controller := g.GetPlayer(opts.Controller)
	if controller == nil {
		return nil, ErrPlayerNotFound
	}

	if err := g.checkSpellTiming(card, opts.Controller); err != nil {
		return nil, err
	}

	selection := opts.ModalSelection
	if selection == nil {
		var err error
		selection, err = g.prepareModalSpellSelection(controller, card, opts.Targets, opts.XValue, opts.ModePrompt)
		if err != nil {
			return nil, err
		}
	}
	var effects []Effect
	var targets []uuid.UUID
	var modalTargets [][]uuid.UUID
	var modeChoice int
	if selection != nil {
		effects = selection.effects
		targets = selection.targets
		modalTargets = selection.modalTargets
		modeChoice = selection.mode
	} else {
		effects, targets = g.prepareSpellStackPayload(controller, card, opts.Targets, false)
		if opts.PromptTargets {
			targets = append(targets, g.promptTargetsForListWithX(opts.Controller, SpellTargetSource(card), card.CastTargets(), opts.XValue)...)
		}
	}
	if !opts.DamageDivisionPrepared {
		if selection == nil {
			specs := card.CastTargets()
			if opts.TargetSpecs != nil {
				specs = opts.TargetSpecs
			}
			if len(specs) == 0 && len(targets) != 0 {
				return nil, fmt.Errorf("too many targets for spell")
			}
			if err := g.validateActionTargets(opts.Controller, SpellTargetSource(card), specs, targets, opts.XValue, "spell"); err != nil {
				return nil, err
			}
		}
		division, err := g.prepareDamageDivision(opts.Controller, card, effects, targets, opts.XValue)
		if err != nil {
			return nil, err
		}
		opts.DamageDistribution = division
	}
	sourceID := opts.SourceID
	if sourceID == uuid.Nil {
		sourceID = card.ID()
	}

	obj := &StackObject{
		spellTextDefinition: card,
		capturedNumbers:     maps.Clone(opts.CapturedNumbers),
		targetHandCards:     maps.Clone(opts.HandTargetCards),
		targetExileCards:    maps.Clone(opts.ExileTargetCards),
		targetAnteCards:     maps.Clone(opts.AnteTargetCards),
		capturedCards:       maps.Clone(opts.CapturedCards),
		revealedHandCards:   slices.Clone(opts.RevealedHandCards),
		ownsHandReveals:     !opts.IsCopy && len(opts.RevealedHandCards) > 0,
		capturedCardSets:    cloneCardsBindings(opts.CapturedCardSets),
		capturedPermanents:  maps.Clone(opts.CapturedPermanents),
		ID:                  uuid.New(),
		Card:                card,
		Controller:          opts.Controller,
		SourceID:            sourceID,
		Effects:             effects,
		Targets:             targets,
		TargetSpecs:         expandTargetSpecs(card.CastTargets(), targets, opts.XValue),
		TargetSource:        card,
		XValue:              opts.XValue,
		ModeChoice:          modeChoice,
		ModalTargets:        modalTargets,
		IsCopy:              opts.IsCopy,
		CastZone:            opts.CastZone,
		ExileOnLeaveStack:   opts.ExileOnLeaveStack,
	}
	if modal, ok := getModalSpellAbility(card); ok {
		obj.TargetSpecs = expandTargetSpecs(modal.Modes()[modeChoice].Targets, targets, opts.XValue)
	}
	if opts.TargetSpecs != nil {
		obj.TargetSpecs = append([]Target(nil), opts.TargetSpecs...)
	}
	if opts.SnapshotCast {
		obj.CastContext = g.snapshotCastContext(opts.Controller)
	}

	if modes := card.Modes(); selection == nil && len(modes) > 0 {
		obj.ModeChoice = controller.ChooseMode(modes, card.Name())
	}

	obj.DamageDistribution = maps.Clone(opts.DamageDistribution)
	g.pushStack(obj)
	g.trackers.Turn.castSpells = append(g.trackers.Turn.castSpells, damageSourceFromSpellObject(obj))
	g.recordSpellCast(card, opts.Controller)
	instantOrdinal := 0
	if card.HasType(TypeInstant) {
		instantOrdinal = g.trackers.Turn.InstantsCast(opts.Controller)
	}
	g.FireEvent(GameEvent{
		CastInstantOrdinal: instantOrdinal,
		Type:               EvtSpellCast,
		SourceID:           sourceID,
		TargetID:           obj.ID,
		PlayerID:           opts.Controller,
	})
	g.fireBecomesTargetEvents(obj, false)
	return obj, nil
}

type modalSpellSelection struct {
	effects      []Effect
	targets      []uuid.UUID
	modalTargets [][]uuid.UUID
	mode         int
}

func (g *Game) prepareModalSpellSelection(caster Player, card Card, targets []uuid.UUID, x int, prompt string) (*modalSpellSelection, error) {
	ms, ok := getModalSpellAbility(card)
	if !ok {
		return nil, nil
	}
	if prompt == "" {
		prompt = card.Name()
	}
	var choice int
	if chooser, ok := caster.(EffectModeChooser); ok {
		choice = chooser.ChooseModeWithEffects(ms.modes, prompt, g)
	} else {
		choice = caster.ChooseMode(modalLabels(ms), prompt)
	}
	if choice < 0 || choice >= len(ms.modes) {
		return nil, fmt.Errorf("invalid spell mode")
	}
	mode := ms.modes[choice]
	source := SpellTargetSourceWithDeclarations(card, mode.Targets)
	if len(targets) == 0 {
		targets = g.promptTargetsForListWithX(caster.PlayerID(), source, mode.Targets, x)
	} else {
		targets = g.acquireRandomTargets(caster.PlayerID(), source, mode.Targets, targets, x)
		targets = g.acquireOpponentChosenTargets(caster.PlayerID(), source, mode.Targets, targets)
	}
	if len(mode.Targets) == 0 && len(targets) != 0 {
		return nil, fmt.Errorf("too many targets for spell mode")
	}
	if err := g.validateActionTargets(caster.PlayerID(), source, mode.Targets, targets, x, "spell mode"); err != nil {
		return nil, err
	}
	targets = append([]uuid.UUID(nil), targets...)
	selected := &modalSpellSelection{effects: append([]Effect(nil), mode.Effects...), targets: targets, mode: choice, modalTargets: make([][]uuid.UUID, len(ms.modes))}
	selected.modalTargets[choice] = targets
	return selected, nil
}

func (g *Game) prepareSpellStackPayload(caster Player, card Card, targets []uuid.UUID, promptTargets bool) (effects []Effect, resolvedTargets []uuid.UUID) {
	targets = append([]uuid.UUID(nil), targets...)
	for _, a := range card.Abilities() {
		sa, ok := a.(*SpellAbility)
		if !ok {
			continue
		}
		// SpellAbility and SimpleActivatedAbility share the *ActionDefinition
		// type, so filter by Kind() to keep activated abilities (Jalum Tome,
		// Jade Statue, Forcefield) from running their effects on cast.
		if sa.Kind() != ActionSpell {
			continue
		}
		effects = append(effects, sa.Effects()...)
		if promptTargets && len(sa.Targets()) > 0 {
			targets = append(targets, g.promptTargetsForList(caster.PlayerID(), SpellTargetSource(card), sa.Targets())...)
		}
	}
	return effects, targets
}

func (g *Game) recordSpellCast(card Card, playerID uuid.UUID) {
	if card == nil {
		return
	}
	if card.HasType(TypeInstant) {
		g.trackers.Turn.RecordInstantCast(playerID)
	}
	if card.HasType(TypeSorcery) {
		g.trackers.Turn.RecordSorceryCast(playerID)
	}
}

func (g *Game) checkSpellTiming(card Card, caster uuid.UUID) error {
	if g.IsSpellCastProhibited(caster, card) {
		return fmt.Errorf("can't cast %s: a continuous effect prohibits this spell", card.Name())
	}
	if err := g.checkSpellCastingConditions(card, caster); err != nil {
		return err
	}
	for _, ability := range card.Abilities() {
		if action, ok := UnwrapAbility(ability).(*ActionDefinition); ok && action.kind == ActionSpell && action.timing == TimingStepOnly && !action.allowsStep(g.turns.Step()) {
			return fmt.Errorf("can only cast %s during %s", card.Name(), (spellStepDefinition{action.stepOnly, actionStepExtras(action)}).inspectDefinition().Text)
		}
	}
	return nil
}

func actionStepExtras(action *ActionDefinition) []PhaseStep {
	var steps []PhaseStep
	for step := Untap; step <= Cleanup; step++ {
		if action.stepMask&(1<<uint(step)) != 0 {
			steps = append(steps, step)
		}
	}
	return steps
}
