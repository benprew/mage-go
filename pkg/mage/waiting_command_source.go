package mage

import (
	"slices"

	"github.com/google/uuid"
)

type waitingPreventionCommandReferences interface {
	waitingPreventionCommands() []DamageSource
}

type waitingRedirectionCommandReferences interface {
	waitingRedirectionCommands() []DamageSource
}

func (r *sourcePreventionReplacement) waitingPreventionCommands() []DamageSource {
	return []DamageSource{r.exactSource}
}

func (r *selectedSourceRedirectReplacement) waitingRedirectionCommands() []DamageSource {
	return []DamageSource{r.source}
}

// WaitingPreventionCommandSources reads exact visits in active prevention rules.
// A returned command object cannot replace the saved visit.
func (g *Game) WaitingPreventionCommandSources() []DamageSource {
	return g.waitingReplacementCommandSources(func(replacement ReplacementEffect) []DamageSource {
		refs, ok := replacement.(waitingPreventionCommandReferences)
		if !ok {
			return nil
		}
		return refs.waitingPreventionCommands()
	})
}

// WaitingRedirectionCommandSources reads exact visits in active redirection rules.
// Consumption and cleanup end the reference lifetime.
func (g *Game) WaitingRedirectionCommandSources() []DamageSource {
	return g.waitingReplacementCommandSources(func(replacement ReplacementEffect) []DamageSource {
		refs, ok := replacement.(waitingRedirectionCommandReferences)
		if !ok {
			return nil
		}
		return refs.waitingRedirectionCommands()
	})
}

func (g *Game) waitingReplacementCommandSources(references func(ReplacementEffect) []DamageSource) []DamageSource {
	var sources []DamageSource
	for _, list := range [2][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			if !replacement.IsActive(g) {
				continue
			}
			for _, saved := range references(replacement) {
				if !saved.hasExactIdentity() || saved.CommandVisit() == uuid.Nil {
					continue
				}
				source := g.commandReferenceInformation(saved)
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
