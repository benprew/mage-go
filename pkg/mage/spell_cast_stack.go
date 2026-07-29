package mage

import (
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

	effects, targets, modalTargets, modeChoice := g.prepareSpellStackPayload(controller, card, opts.Targets, opts.PromptTargets, opts.ModePrompt)
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
		XValue:            opts.XValue,
		ModeChoice:        modeChoice,
		ModalTargets:      modalTargets,
		IsCopy:            opts.IsCopy,
		CastZone:          opts.CastZone,
		ExileOnLeaveStack: opts.ExileOnLeaveStack,
	}
	if opts.SnapshotCast {
		obj.CastContext = g.snapshotCastContext(opts.Controller)
	}

	if modes := card.Modes(); len(modes) > 0 {
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

func (g *Game) prepareSpellStackPayload(caster Player, card Card, targets []uuid.UUID, promptTargets bool, modePrompt string) (effects []Effect, resolvedTargets []uuid.UUID, modalTargets [][]uuid.UUID, modeChoice int) {
	if ms, ok := getModalSpellAbility(card); ok {
		if modePrompt == "" {
			modePrompt = card.Name()
		}
		if chooser, ok := caster.(EffectModeChooser); ok {
			modeChoice = chooser.ChooseModeWithEffects(ms.modes, modePrompt, g)
		} else {
			modeChoice = caster.ChooseMode(modalLabels(ms), modePrompt)
		}
		if modeChoice < 0 || modeChoice >= len(ms.modes) {
			modeChoice = 0
		}
		mode := ms.modes[modeChoice]
		effects = append(effects, mode.Effects...)
		targets = g.promptTargetsForList(caster.PlayerID(), card, mode.Targets)
		modalTargets = make([][]uuid.UUID, len(ms.modes))
		modalTargets[modeChoice] = targets
		return effects, targets, modalTargets, modeChoice
	}

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
	return effects, targets, nil, modeChoice
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
		g.instantsCastThisTurn[playerID]++
	}
	if card.HasType(TypeSorcery) {
		g.sorceriesCastThisTurn[playerID]++
	}
}
