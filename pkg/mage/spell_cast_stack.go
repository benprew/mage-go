package mage

import (
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type castStackObjectOptions struct {
	TargetGroups      [][]uuid.UUID
	Mode              *int
	Card              *Card
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

	effects, targets, modalTargets, modeChoice := g.prepareSpellStackPayload(controller, card, opts.Targets, opts.PromptTargets, opts.ModePrompt, opts.Mode, opts.TargetGroups)
	sourceID := opts.SourceID
	if sourceID == uuid.Nil() {
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
	specs := card.CastTargets()
	if action, ok := getModalSpellAbility(card); ok {
		specs = action.modes[modeChoice].Targets
		obj.TargetSpecs = expandTargetSpecs(specs, targets, opts.XValue)
	}
	obj.TargetGroups = groupTargets(specs, targets, opts.XValue)
	if opts.TargetGroups != nil {
		obj.TargetGroups = cloneTargetGroups(opts.TargetGroups)
	}
	obj.TargetSpecs = expandedTargetGroups(specs, obj.TargetGroups)
	obj.declaredTargets = len(specs) > 0
	if opts.SnapshotCast {
		obj.CastContext = g.snapshotCastContext(opts.Controller)
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

func (g *Game) prepareSpellStackPayload(caster Player, card *Card, targets []uuid.UUID, promptTargets bool, modePrompt string, modeOverride *int, suppliedGroups [][]uuid.UUID) (effects []Effect, resolvedTargets []uuid.UUID, modalTargets [][]uuid.UUID, modeChoice int) {
	if ms, ok := getModalSpellAbility(card); ok {
		if modePrompt == "" {
			modePrompt = card.Name()
		}
		if modeOverride != nil {
			modeChoice = *modeOverride
		} else if chooser, ok := caster.(EffectModeChooser); ok {
			modeChoice = chooser.ChooseModeWithEffects(ms.modes, modePrompt, g)
		} else {
			modeChoice = caster.ChooseMode(modalLabels(ms), modePrompt)
		}
		if modeChoice < 0 || modeChoice >= len(ms.modes) {
			modeChoice = 0
		}
		mode := ms.modes[modeChoice]
		effects = append(effects, mode.Effects...)
		if len(targets) == 0 && suppliedGroups == nil {
			targets = g.promptTargetsForList(caster.PlayerID(), card, mode.Targets)
		}
		modalTargets = make([][]uuid.UUID, len(ms.modes))
		modalTargets[modeChoice] = targets
		return effects, targets, modalTargets, modeChoice
	}

	targets = append([]uuid.UUID(nil), targets...)
	for _, a := range card.abilities {
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
	return effects, targets, nil, modeChoice
}

func (g *Game) chooseDividedDamageForSpell(obj *StackObject, card *Card, controller Player) {
	if obj == nil || card == nil || controller == nil {
		return
	}
	for _, eff := range obj.Effects {
		if !IsDividedDamageEffect(eff) {
			continue
		}
		total := DividedDamageTotal(eff).Resolve(NewEffectContext(g, obj.SourceID, obj.Controller, obj.Targets))
		if total > 0 && len(obj.Targets) > 0 {
			dist := controller.ChooseDamageDistribution(obj.Targets, total, card.Name(), g)
			obj.DamageDistribution = sanitizeDamageDistribution(dist, obj.Targets, total)
		}
		break
	}
}

func (g *Game) recordSpellCast(card *Card, playerID uuid.UUID) {
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
