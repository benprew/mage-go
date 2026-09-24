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

type indexedKeywordGrantEffect interface {
	KeywordGrantTargetIndex([]mage.Target) (int, bool)
}

type keywordGrant struct {
	selector mage.TargetSelector
	index    int
	indexed  bool
}

func (grant keywordGrant) recipients(g *mage.Game, controllerID, sourceID uuid.UUID, targets []uuid.UUID) ([]uuid.UUID, bool) {
	if grant.indexed {
		if grant.index < 0 || grant.index >= len(targets) || targets[grant.index] == uuid.Nil {
			return nil, false
		}
		return targets[grant.index : grant.index+1], true
	}
	return keywordGrantRecipients(g, controllerID, sourceID, grant.selector, targets)
}

// AbilityActivationIsRedundant reports whether an activation would only grant
// keywords that every recipient already has or is already set to receive from
// an object on the stack. Effects with any additional strategic result are
// never classified as redundant.
func AbilityActivationIsRedundant(g *mage.Game, controllerID, sourceID uuid.UUID, effects []mage.Effect, targets []uuid.UUID, targetSpecs ...mage.Target) bool {
	grants, ok := pureKeywordGrants(effects, targetSpecs)
	if !ok {
		return false
	}
	for i, grant := range grants {
		recipients, ok := grant.recipients(g, controllerID, sourceID, targets)
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
func NonRedundantKeywordGrantTargets(g *mage.Game, controllerID, sourceID uuid.UUID, effects []mage.Effect, candidates []uuid.UUID, targetSpecs ...mage.Target) []uuid.UUID {
	if len(targetSpecs) > 1 {
		return candidates
	}
	if _, ok := pureKeywordGrants(effects, targetSpecs); !ok {
		return candidates
	}
	result := make([]uuid.UUID, 0, len(candidates))
	for _, candidate := range candidates {
		if !AbilityActivationIsRedundant(g, controllerID, sourceID, effects, []uuid.UUID{candidate}, targetSpecs...) {
			result = append(result, candidate)
		}
	}
	return result
}

func pureKeywordGrants(effects []mage.Effect, targetSpecs []mage.Target) ([]keywordGrant, bool) {
	if len(effects) == 0 {
		return nil, false
	}
	grants := make([]keywordGrant, 0, len(effects))
	for _, effect := range effects {
		props := effect.Properties()
		if props.GrantedKeyword == 0 || props.DamageValue != nil || props.DrawCount != 0 ||
			props.LifeGain != 0 || props.PowerBoost != 0 || props.ToughnessBoost != 0 || props.IsBounce ||
			props.Taps || props.Regenerates || props.TokenPower != 0 || props.TokenToughness != 0 {
			return nil, false
		}
		switch grant := effect.(type) {
		case keywordGrantEffect:
			grants = append(grants, keywordGrant{selector: grant.KeywordGrantTarget()})
		case indexedKeywordGrantEffect:
			index, ok := grant.KeywordGrantTargetIndex(targetSpecs)
			if !ok {
				return nil, false
			}
			grants = append(grants, keywordGrant{index: index, indexed: true})
		default:
			return nil, false
		}
	}
	return grants, true
}

func keywordAlreadyGranted(g *mage.Game, targetID uuid.UUID, keyword core.Attr) bool {
	if target := g.FindPermanent(targetID); target != nil && target.HasKeyword(keyword) {
		return true
	}
	for _, obj := range g.StackObjects() {
		for _, effect := range obj.Effects {
			grants, ok := pureKeywordGrants([]mage.Effect{effect}, obj.TargetSpecs)
			if !ok || effect.Properties().GrantedKeyword != keyword {
				continue
			}
			recipients, ok := grants[0].recipients(g, obj.Controller, obj.SourceID, obj.Targets)
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
			if selector.Kind == mage.KindMatching && permanent.ControllerID() != controllerID {
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
