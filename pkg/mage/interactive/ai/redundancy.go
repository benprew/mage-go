package ai

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type keywordGrantEffect interface {
	KeywordGrantTarget() mage.TargetSelector
}

// AbilityActivationIsRedundant reports whether an activation would only grant
// keywords that every recipient already has or is already set to receive from
// an object on the stack. Effects with any additional strategic result are
// never classified as redundant.
func AbilityActivationIsRedundant(g *mage.Game, controllerID, sourceID uuid.UUID, effects []mage.Effect, targets []uuid.UUID) bool {
	grants, ok := pureKeywordGrants(effects)
	if !ok {
		return false
	}
	for i, grant := range grants {
		recipients, ok := keywordGrantRecipients(g, controllerID, sourceID, grant.KeywordGrantTarget(), targets)
		if !ok || len(recipients) == 0 {
			return false
		}
		keyword := effects[i].Properties().GrantedKeyword
		for _, targetID := range recipients {
			if !keywordAlreadyGranted(g, targetID, keyword) {
				return false
			}
		}
	}
	return true
}

// NonRedundantKeywordGrantTargets removes candidates for which a pure keyword-
// granting activation would have no effect. Other kinds of abilities return
// the candidates unchanged.
func NonRedundantKeywordGrantTargets(g *mage.Game, controllerID, sourceID uuid.UUID, effects []mage.Effect, candidates []uuid.UUID) []uuid.UUID {
	if _, ok := pureKeywordGrants(effects); !ok {
		return candidates
	}
	result := make([]uuid.UUID, 0, len(candidates))
	for _, candidate := range candidates {
		if !AbilityActivationIsRedundant(g, controllerID, sourceID, effects, []uuid.UUID{candidate}) {
			result = append(result, candidate)
		}
	}
	return result
}

func pureKeywordGrants(effects []mage.Effect) ([]keywordGrantEffect, bool) {
	if len(effects) == 0 {
		return nil, false
	}
	grants := make([]keywordGrantEffect, 0, len(effects))
	for _, effect := range effects {
		props := effect.Properties()
		grant, ok := effect.(keywordGrantEffect)
		if !ok || props.GrantedKeyword == 0 || props.DamageValue != nil || props.DrawCount != 0 ||
			props.LifeGain != 0 || props.PowerBoost != 0 || props.ToughnessBoost != 0 || props.IsBounce ||
			props.Taps || props.Regenerates || props.TokenPower != 0 || props.TokenToughness != 0 {
			return nil, false
		}
		grants = append(grants, grant)
	}
	return grants, true
}

func keywordAlreadyGranted(g *mage.Game, targetID uuid.UUID, keyword core.Attr) bool {
	if target := g.FindPermanent(targetID); target != nil && target.HasKeyword(keyword) {
		return true
	}
	for _, obj := range g.StackObjects() {
		for _, effect := range obj.Effects {
			grant, ok := effect.(keywordGrantEffect)
			if !ok || effect.Properties().GrantedKeyword != keyword {
				continue
			}
			recipients, ok := keywordGrantRecipients(g, obj.Controller, obj.SourceID, grant.KeywordGrantTarget(), obj.Targets)
			if ok && containsID(recipients, targetID) {
				return true
			}
		}
	}
	return false
}

func keywordGrantRecipients(g *mage.Game, controllerID, sourceID uuid.UUID, selector mage.TargetSelector, targets []uuid.UUID) ([]uuid.UUID, bool) {
	switch selector.Kind {
	case mage.KindTarget:
		if len(targets) == 0 {
			return nil, false
		}
		return targets[:1], true
	case mage.KindAllTargets:
		return targets, len(targets) > 0
	case mage.KindSource:
		return []uuid.UUID{sourceID}, sourceID != uuid.Nil
	case mage.KindAttached:
		source := g.FindPermanent(sourceID)
		if source == nil || source.AttachedTo == uuid.Nil {
			return nil, false
		}
		return []uuid.UUID{source.AttachedTo}, true
	case mage.KindMatching, mage.KindAllMatching:
		var recipients []uuid.UUID
		for _, permanent := range g.AllBattlefield() {
			if selector.Kind == mage.KindMatching && permanent.Controller != controllerID {
				continue
			}
			if permanent.HasType(core.TypeCreature) && selector.Filter.Match(permanent, g) {
				recipients = append(recipients, permanent.ID())
			}
		}
		return recipients, true
	default:
		return nil, false
	}
}

func containsID(ids []uuid.UUID, want uuid.UUID) bool {
	return slices.Contains(ids, want)
}
