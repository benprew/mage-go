package mage

import "github.com/google/uuid"

type entryHistoryKey struct {
	source, entrant capturedPermanent
}

type entryHistorySet struct {
	source     PermanentRef
	candidates PermanentSet
}

// PermanentsEnteredWithSource selects exact visits recorded by RecordEntryFrom.
// A query can precede the entry instruction without saving an empty result.
func PermanentsEnteredWithSource(source PermanentRef, candidates PermanentSet) PermanentSet {
	return entryHistorySet{source, candidates}
}

func (entryHistorySet) permanentSet() {}
func (s entryHistorySet) inspectDefinition() definitionNode {
	n := permanentInformationNode(s.source, "permanent-set", "permanents entered with source")
	n.children = append(n.children, definitionEdge{"candidates", s.candidates})
	return n
}
func (s entryHistorySet) resolvePermanents(ctx *EffectContext) []*Permanent {
	ref, ok := s.source.(permanentInformationReference)
	if !ok || nilDefinition(ref) || nilDefinition(s.candidates) {
		return nil
	}
	source, ok := ref.informationIdentity(ctx)
	if !ok || source.incarnation == uuid.Nil {
		return nil
	}
	var result []*Permanent
	for _, p := range s.candidates.resolvePermanents(ctx) {
		if p != nil && !p.PhasedOut && ctx.Game.zones.entryHistory[entryHistoryKey{source, capturedPermanent{p.ID(), p.incarnationID}}] {
			result = append(result, p)
		}
	}
	return result
}

type recordEntryOption struct{ source PermanentRef }

// RecordEntryFrom links a successful entry to an exact source visit.
// The source is saved before entry choices and can have left the battlefield.
func RecordEntryFrom(source PermanentRef) BattlefieldEntryOption { return recordEntryOption{source} }
func (recordEntryOption) battlefieldEntryOption()                {}
func (o recordEntryOption) applyBattlefieldEntry(e *battlefieldEntryOptions) {
	e.historySource = o.source
	e.historySourceCount++
}

func (e battlefieldEntryOptions) captureHistorySource(ctx *EffectContext) capturedPermanent {
	if ref, ok := e.historySource.(permanentInformationReference); ok && !nilDefinition(ref) {
		if source, valid := ref.informationIdentity(ctx); valid {
			return source
		}
	}
	return capturedPermanent{}
}

func (g *Game) recordBattlefieldEntrySource(source capturedPermanent, entrant *Permanent) {
	if source.incarnation == uuid.Nil || entrant == nil {
		return
	}
	if g.zones.entryHistory == nil {
		g.zones.entryHistory = make(map[entryHistoryKey]bool)
	}
	g.zones.entryHistory[entryHistoryKey{source, capturedPermanent{entrant.ID(), entrant.incarnationID}}] = true
}
