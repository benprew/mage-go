package mage

import (
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type castStackObjectOptions struct {
	TargetGroups       [][]uuid.UUID
	Mode               *int
	Card               *Card
	Controller         uuid.UUID
	Targets            []uuid.UUID
	XValue             int
	CastZone           Zone
	SourceID           uuid.UUID
	IsCopy             bool
	ExileOnLeaveStack  bool
	SnapshotCast       bool
	PromptTargets      bool
	ModePrompt         string
	damageDistribution map[uuid.UUID]int
	damageTotal        int
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

	effects, targets, modalTargets, modeChoice := g.prepareSpellStackPayload(controller, card, opts.Targets, opts.PromptTargets, opts.ModePrompt, opts.Mode, opts.TargetGroups, opts.XValue)
	sourceID := opts.SourceID
	if sourceID == uuid.Nil() {
		sourceID = card.ID()
	}

	obj := &StackObject{
		ID:                 uuid.New(),
		Card:               card,
		Controller:         opts.Controller,
		SourceID:           sourceID,
		Effects:            effects,
		Targets:            targets,
		TargetSpecs:        expandTargetSpecs(card.CastTargets(), targets, opts.XValue),
		TargetSource:       card,
		XValue:             opts.XValue,
		ModeChoice:         modeChoice,
		ModalTargets:       modalTargets,
		IsCopy:             opts.IsCopy,
		CastZone:           opts.CastZone,
		ExileOnLeaveStack:  opts.ExileOnLeaveStack,
		DamageDistribution: opts.damageDistribution,
		damageTotal:        opts.damageTotal,
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

	if opts.PromptTargets {
		var err error
		obj.DamageDistribution, obj.damageTotal, err = g.announceDividedDamage(effects, card, opts.Controller, targets, opts.XValue)
		if err != nil {
			return nil, err
		}
	}
	g.pushStack(obj)
	g.recordSpellCast(card, opts.Controller)
	g.FireEvent(GameEvent{
		Type:          EvtSpellCast,
		SourceID:      sourceID,
		StackObjectID: obj.ID,
		PlayerID:      opts.Controller,
	})
	g.fireBecomesTargetEvents(obj, false)
	return obj, nil
}

func stackSpellForCastEvent(evt *GameEvent, g GameReader) *StackObject {
	if evt == nil {
		return nil
	}
	id := evt.StackObjectID
	if id == uuid.Nil() {
		id = evt.SourceID
	}
	if obj := g.FindStackObject(id); obj != nil && !obj.IsAbility && obj.Card != nil {
		return obj
	}
	return nil
}

func spellViewForCastEvent(evt *GameEvent, g GameReader) ObjectView {
	if evt == nil {
		return nil
	}
	if obj := stackSpellForCastEvent(evt, g); obj != nil {
		return stackObjectView{Card: obj.Card, object: obj}
	}
	if evt.StackObjectID == uuid.Nil() {
		if card := g.FindCardAnywhere(evt.SourceID); card != nil {
			return spellObjectView{Card: card, controller: evt.PlayerID}
		}
	}
	return nil
}

func (g *Game) prepareSpellStackPayload(caster Player, card *Card, targets []uuid.UUID, promptTargets bool, modePrompt string, modeOverride *int, suppliedGroups [][]uuid.UUID, x int) (effects []Effect, resolvedTargets []uuid.UUID, modalTargets [][]uuid.UUID, modeChoice int) {
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
			targets = g.promptTargetsForList(caster.PlayerID(), card, mode.Targets, x)
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
			targets = append(targets, g.promptTargetsForList(caster.PlayerID(), card, sa.Targets(), x)...)
		}
	}
	return effects, targets, nil, modeChoice
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
