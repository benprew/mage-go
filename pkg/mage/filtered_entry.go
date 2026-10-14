package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// EntersTappedFor changes entry for matching permanents of the selected players.
// The source must have this ability before entry. Its current controller supplies You.
func EntersTappedFor(players PlayerSet, filter PermanentFilter) Ability {
	return &filteredEntryAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}, players: players, filter: filter}
}

type filteredEntryAbility struct {
	BaseAbility
	players PlayerSet
	filter  PermanentFilter
}

func (a *filteredEntryAbility) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-entry-replacement", Text: "matching permanents enter tapped", battlefieldSource: true, children: []definitionEdge{{"players", a.players}}}
}

type filteredEntryKey struct{ source, incarnation, ability uuid.UUID }

type filteredEntryReplacement struct {
	replacementBase
	key     filteredEntryKey
	ability *filteredEntryAbility
}

func (r *filteredEntryReplacement) liveSource(g *Game) *Permanent {
	source := g.FindPermanent(r.key.source)
	if source == nil || source.incarnationID != r.key.incarnation || source.FaceDown {
		return nil
	}
	for _, ability := range source.RuntimeAbilities {
		if ability.AbilityID() == r.key.ability {
			return source
		}
	}
	return nil
}
func (r *filteredEntryReplacement) IsActive(reader GameReader) bool {
	g, ok := reader.(*Game)
	return ok && r.liveSource(g) != nil
}
func (r *filteredEntryReplacement) Matches(action Action, reader GameReader) bool {
	entry, ok := action.(*battlefieldEntryAction)
	g, gameOK := reader.(*Game)
	if !ok || !gameOK {
		return false
	}
	source := r.liveSource(g)
	if source == nil {
		return false
	}
	projected, view := g.projectEntryPermanent(entry.permanent)
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceIncarnation: source.incarnationID, sourceCaptured: true}
	return slices.Contains(r.ability.players.resolvePlayers(ctx), projected.ControllerID()) && r.ability.filter.Match(projected, view)
}
func (r *filteredEntryReplacement) Replace(action Action, g *Game) Action {
	if r.Matches(action, g) {
		action.(*battlefieldEntryAction).permanent.Tapped = true
	}
	return action
}
func (r *filteredEntryReplacement) Clone() ReplacementEffect { clone := *r; return &clone }
func (*filteredEntryReplacement) Text() string               { return "matching permanent enters tapped" }

func (a *battlefieldEntryAction) filteredReplacementCandidates(g *Game) []ReplacementEffect {
	var result []ReplacementEffect
	if a.filtered == nil {
		a.filtered = make(map[filteredEntryKey]*filteredEntryReplacement)
	}
	for _, source := range g.AllBattlefield() {
		if source.PhasedOut || source.FaceDown {
			continue
		}
		for _, ability := range source.RuntimeAbilities {
			rule, ok := UnwrapAbility(ability).(*filteredEntryAbility)
			if !ok {
				continue
			}
			key := filteredEntryKey{source.ID(), source.incarnationID, ability.AbilityID()}
			replacement := a.filtered[key]
			if replacement == nil {
				replacement = &filteredEntryReplacement{replacementBase: replacementBase{sourceID: source.ID(), priority: ReplacementPriorityDefault}, key: key, ability: rule}
				a.filtered[key] = replacement
			}
			if replacement.Matches(a, g) {
				result = append(result, replacement)
			}
		}
	}
	return result
}

// Entry matching must not publish the proposed permanent or change live layers.
func (g *Game) projectEntryPermanent(entrant *Permanent) (*Permanent, *Game) {
	view := g.Clone()
	projected := &Permanent{}
	clonePermanentInto(projected, entrant)
	view.zones.ClearEnteringPermanent()
	view.zones.AddPermanent(projected)
	for _, ability := range projected.RuntimeAbilities {
		holder, ok := UnwrapAbility(ability).(*StaticAbilityHolder)
		if !ok {
			continue
		}
		for _, effect := range holder.Effects {
			composed, ok := effect.(*composedLayerEffect)
			if !ok {
				continue
			}
			var modifications []*layerModification
			for _, modification := range composed.modifications {
				if _, self := modification.subject.(sourceRef); self && modification.subjects == nil {
					modifications = append(modifications, modification)
				}
			}
			if len(modifications) == 0 {
				continue
			}
			own := composed.cloneEffect().(*composedLayerEffect)
			own.modifications = modifications
			view.setEffectSource(own, projected.ID())
			view.effects.Add(printedStaticEffect(own, projected, holder))
		}
	}
	view.effects.Apply(view)
	projected = view.FindPermanent(projected.ID())
	return projected, view
}
