package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type castStackObjectOptions struct {
	Card              Card
	Controller        uuid.UUID
	Targets           []uuid.UUID
	XValue            int
	CastZone          Zone
	SourceID          uuid.UUID
	IsCopy            bool
	ExileOnLeaveStack bool
	SnapshotCast      bool
	PromptTargets     bool
	ModePrompt        string
	ModalSelection    *modalSpellSelection
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
		effects, targets = g.prepareSpellStackPayload(controller, card, opts.Targets, opts.PromptTargets)
	}
	sourceID := opts.SourceID
	if sourceID == uuid.Nil {
		sourceID = card.ID()
	}

	obj := &StackObject{
		ID:                uuid.New(),
		Card:              card,
		Controller:        opts.Controller,
		SourceID:          sourceID,
		Effects:           effects,
		Targets:           targets,
		TargetSpecs:       expandTargetSpecs(card.CastTargets(), targets, opts.XValue),
		TargetSource:      card,
		XValue:            opts.XValue,
		ModeChoice:        modeChoice,
		ModalTargets:      modalTargets,
		IsCopy:            opts.IsCopy,
		CastZone:          opts.CastZone,
		ExileOnLeaveStack: opts.ExileOnLeaveStack,
	}
	if modal, ok := getModalSpellAbility(card); ok {
		obj.TargetSpecs = expandTargetSpecs(modal.Modes()[modeChoice].Targets, targets, opts.XValue)
	}
	if opts.SnapshotCast {
		obj.CastContext = g.snapshotCastContext(opts.Controller)
	}

	if modes := card.Modes(); selection == nil && len(modes) > 0 {
		obj.ModeChoice = controller.ChooseMode(modes, card.Name())
	}

	g.chooseDividedDamageForSpell(obj, card, controller)
	g.pushStack(obj)
	g.recordSpellCast(card, opts.Controller)
	g.FireEvent(GameEvent{
		Type:     EvtSpellCast,
		SourceID: sourceID,
		PlayerID: opts.Controller,
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
	if len(targets) == 0 {
		targets = g.promptTargetsForListWithX(caster.PlayerID(), card, mode.Targets, x)
	} else {
		targets = g.acquireRandomTargets(caster.PlayerID(), card, mode.Targets, targets, x)
		targets = g.acquireOpponentChosenTargets(caster.PlayerID(), card, mode.Targets, targets)
	}
	if len(mode.Targets) == 0 && len(targets) != 0 {
		return nil, fmt.Errorf("too many targets for spell mode")
	}
	if err := g.validateActionTargets(caster.PlayerID(), card, mode.Targets, targets, x, "spell mode"); err != nil {
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
			targets = append(targets, g.promptTargetsForList(caster.PlayerID(), card, sa.Targets())...)
		}
	}
	return effects, targets
}

func (g *Game) chooseDividedDamageForSpell(obj *StackObject, card Card, controller Player) {
	if obj == nil || card == nil || controller == nil {
		return
	}
	for _, eff := range obj.Effects {
		if !IsDividedDamageEffect(eff) {
			continue
		}
		total := DividedDamageTotal(eff).Resolve(g, obj.SourceID, obj.Controller, obj.Targets)
		if total > 0 && len(obj.Targets) > 0 {
			dist := controller.ChooseDamageDistribution(obj.Targets, total, card.Name(), g)
			obj.DamageDistribution = sanitizeDamageDistribution(dist, obj.Targets, total)
		}
		break
	}
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
