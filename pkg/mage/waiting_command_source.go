package mage

import (
	"slices"

	"github.com/google/uuid"
)

type waitingPreventionExactReferences interface {
	waitingPreventionExactSources() []DamageSource
}

type waitingRedirectionExactReferences interface {
	waitingRedirectionExactSources() []DamageSource
}

func (r *sourcePreventionReplacement) waitingPreventionExactSources() []DamageSource {
	return []DamageSource{r.exactSource}
}

func (r *selectedSourceRedirectReplacement) waitingRedirectionExactSources() []DamageSource {
	return []DamageSource{r.source}
}

// WaitingPreventionCommandSources reads exact visits in active prevention rules.
// A returned command object cannot replace the saved visit.
func (g *Game) WaitingPreventionCommandSources() []DamageSource {
	return g.waitingReplacementZoneSources(func(replacement ReplacementEffect) []DamageSource {
		refs, ok := replacement.(waitingPreventionExactReferences)
		if !ok {
			return nil
		}
		return refs.waitingPreventionExactSources()
	}, DamageSource.CommandVisit, g.commandReferenceInformation)
}

// WaitingRedirectionCommandSources reads exact visits in active redirection rules.
// Consumption and cleanup end the reference lifetime.
func (g *Game) WaitingRedirectionCommandSources() []DamageSource {
	return g.waitingReplacementZoneSources(func(replacement ReplacementEffect) []DamageSource {
		refs, ok := replacement.(waitingRedirectionExactReferences)
		if !ok {
			return nil
		}
		return refs.waitingRedirectionExactSources()
	}, DamageSource.CommandVisit, g.commandReferenceInformation)
}

func (g *Game) waitingReplacementZoneSources(references func(ReplacementEffect) []DamageSource, visit func(DamageSource) uuid.UUID, information func(DamageSource) DamageSource) []DamageSource {
	var sources []DamageSource
	for _, list := range [2][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			if !replacement.IsActive(g) {
				continue
			}
			for _, saved := range references(replacement) {
				if !saved.hasExactIdentity() || visit(saved) == uuid.Nil {
					continue
				}
				source := information(saved)
				if source.SameObject(saved) && !slices.ContainsFunc(sources, source.SameObject) {
					sources = append(sources, cloneDamageSource(source))
				}
			}
		}
	}
	return sources
}

// ChooseDamageSourceFromWaitingCommandReferences uses visits saved by waiting rules.
// Other source categories need separate enumeration services.
func ChooseDamageSourceFromWaitingCommandReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingCommands: true}
}
