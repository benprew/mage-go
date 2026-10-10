package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func activatedObjectTextContext(card Card) error {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || !isEntryCopyPermanent(base) || base.HasType(core.TypeLand) || base.HasType(core.TypeInstant) || base.HasType(core.TypeSorcery) || base.HasSubType("Aura") || len(base.abilities) == 0 || len(base.modes) != 0 || len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 0 {
		return fmt.Errorf("activated object text requires complete permanent actions without casting costs or targets")
	}
	for keyword, count := range base.attrSeeds {
		if count < 1 || !slices.Contains([]core.Attr{core.Flying, core.FirstStrike, core.Trample, core.Vigilance, core.Haste, core.Reach, core.Banding, core.Defender, core.Fear, core.Deathtouch, core.Lifelink, core.Shroud, core.Hexproof, core.Menace, core.Indestructible}, keyword) {
			return fmt.Errorf("activated object text requires fixed intrinsic keywords")
		}
	}
	for _, ability := range base.abilities {
		if !layeredActivatedEligible(ability) {
			return fmt.Errorf("activated object text requires marked ordinary printed actions")
		}
	}
	return nil
}

// AddPrintedActivatedObjectTextEffect checks every printed action before installation.
// Each action keeps its own baseline, payment, targets and activation state.
func (g *Game) AddPrintedActivatedObjectTextEffect(source *Permanent, duration core.Duration, changes ...TextChange) error {
	if source == nil || len(changes) == 0 || (duration != core.Indefinite && duration != core.EndOfTurn) {
		return fmt.Errorf("activated object text requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("activated object text requires the original face-up source")
	}
	if err := activatedObjectTextContext(current.Card); err != nil {
		return err
	}
	revision, err := BuildTextDefinitionRevision(current.Card, changes...)
	if err != nil {
		return err
	}
	if err := activatedObjectTextContext(revision.Definition()); err != nil {
		return err
	}
	old, fresh := current.Card.(*BaseCard), revision.Definition().(*BaseCard)
	if !maps.Equal(old.attrSeeds, fresh.attrSeeds) || !slices.Equal(old.subTypes, fresh.subTypes) {
		return fmt.Errorf("activated object text changed intrinsic rules or subtypes")
	}
	for _, ability := range old.abilities {
		next, paired := revision.ReplacementAbility(ability.AbilityID())
		action, ok := next.(*ActionDefinition)
		if !paired || !ok || !sameActivatedObjectTextContext(ability.(*ActionDefinition), action) {
			return fmt.Errorf("activated object text changed its declaration context")
		}
	}
	probe := g.Clone()
	pending := make([]ContinuousEffect, 0, len(old.abilities))
	for _, ability := range old.abilities {
		id := ability.AbilityID()
		if err := probe.AddPrintedActivatedTextEffect(probe.FindPermanent(current.ID()), id, duration, changes...); err != nil {
			return err
		}
		pending = append(pending, &printedActivatedTextEffect{subject: capturedPermanent{id: current.ID(), incarnation: current.incarnationID}, abilityID: id, duration: duration, changes: slices.Clone(changes)})
	}
	for _, effect := range pending {
		effect.SetSourceID(current.ID())
		g.effects.Add(effect)
	}
	g.ApplyContinuousEffects()
	return nil
}

func sameActivatedObjectTextContext(old, fresh *ActionDefinition) bool {
	if !sameActivatedTextContext(old, fresh) || len(old.effects) != len(fresh.effects) {
		return false
	}
	for i, mode := range old.modes {
		if len(mode.Targets) != len(fresh.modes[i].Targets) || len(mode.Effects) != len(fresh.modes[i].Effects) {
			return false
		}
	}
	return true
}
