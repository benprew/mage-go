package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PutThenSacrificeFor retains a matching entry, then requires a sacrifice.
// The actor's count is checked before entry. The actor can sacrifice the entrant.
func PutThenSacrificeFor(players PlayerSet, entrant, counted PermanentFilter, minimum ResolutionValue) Ability {
	return &retainedEntryAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}, players: players, entrant: entrant, counted: counted, minimum: minimum}
}

type retainedEntryAbility struct {
	BaseAbility
	players          PlayerSet
	entrant, counted PermanentFilter
	minimum          ResolutionValue
}

func (a *retainedEntryAbility) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-entry-sequence", Text: "put then sacrifice", battlefieldSource: true, children: []definitionEdge{{"players", a.players}, {"minimum", a.minimum}}}
}

type retainedEntryFollowUp struct {
	actor  uuid.UUID
	filter PermanentFilter
}

type retainedEntryReplacement struct {
	replacementBase
	key     filteredEntryKey
	ability *retainedEntryAbility
}

func (r *retainedEntryReplacement) liveSource(g *Game) *Permanent {
	source := g.FindPermanent(r.key.source)
	if source == nil || source.PhasedOut || source.FaceDown || source.incarnationID != r.key.incarnation {
		return nil
	}
	for _, ability := range source.RuntimeAbilities {
		if ability.AbilityID() == r.key.ability {
			return source
		}
	}
	return nil
}
func (r *retainedEntryReplacement) IsActive(reader GameReader) bool {
	g, ok := reader.(*Game)
	return ok && r.liveSource(g) != nil
}
func (r *retainedEntryReplacement) Matches(action Action, reader GameReader) bool {
	entry, ok := action.(*battlefieldEntryAction)
	g, gameOK := reader.(*Game)
	if !ok || !gameOK {
		return false
	}
	source := r.liveSource(g)
	if source == nil {
		return false
	}
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceIncarnation: source.incarnationID, sourceCaptured: true}
	if !slices.Contains(r.ability.players.resolvePlayers(ctx), entry.context.actor) {
		return false
	}
	count := 0
	for _, p := range g.AllBattlefield() {
		if !p.PhasedOut && p.ControllerID() == entry.context.actor && r.ability.counted.Match(p, g) {
			count++
		}
	}
	if count < r.ability.minimum.resolveValue(ctx) {
		return false
	}
	projected, view := g.projectEntryPermanent(entry.permanent)
	return r.ability.entrant.Match(projected, view)
}
func (r *retainedEntryReplacement) Replace(action Action, g *Game) Action {
	if r.Matches(action, g) {
		entry := action.(*battlefieldEntryAction)
		entry.permanent.entryFollowUps = append(entry.permanent.entryFollowUps, retainedEntryFollowUp{entry.context.actor, r.ability.counted})
	}
	return action
}
func (r *retainedEntryReplacement) Clone() ReplacementEffect { clone := *r; return &clone }
func (*retainedEntryReplacement) Text() string               { return "put then sacrifice" }

func (a *battlefieldEntryAction) retainedReplacementCandidates(g *Game) []ReplacementEffect {
	if a.retained == nil {
		a.retained = make(map[filteredEntryKey]*retainedEntryReplacement)
	}
	var result []ReplacementEffect
	for _, source := range g.AllBattlefield() {
		if source.PhasedOut || source.FaceDown {
			continue
		}
		for _, ability := range source.RuntimeAbilities {
			rule, ok := UnwrapAbility(ability).(*retainedEntryAbility)
			if !ok {
				continue
			}
			key := filteredEntryKey{source.ID(), source.incarnationID, ability.AbilityID()}
			replacement := a.retained[key]
			if replacement == nil {
				replacement = &retainedEntryReplacement{replacementBase: replacementBase{sourceID: source.ID(), priority: ReplacementPriorityDefault}, key: key, ability: rule}
				a.retained[key] = replacement
			}
			if replacement.Matches(a, g) {
				result = append(result, replacement)
			}
		}
	}
	return result
}

func (g *Game) finishRetainedEntry(p *Permanent) {
	if len(p.entryFollowUps) == 0 {
		return
	}
	followUps := p.entryFollowUps
	if current := g.MutablePermanentIncludingPhased(p.ID()); current != nil && current.incarnationID == p.incarnationID {
		followUps = current.entryFollowUps
		current.entryFollowUps = nil
	}
	for _, followUp := range slices.Backward(followUps) {
		effect := SacrificeChosenBy(You, You, MatchingPermanents(followUp.filter))
		_ = ApplyEffect(g, effect, uuid.Nil, followUp.actor, nil)
	}
}
