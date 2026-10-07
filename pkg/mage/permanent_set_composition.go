package mage

import (
	"fmt"

	"github.com/google/uuid"
)

type composedPermanentSet struct {
	members      []PermanentSet
	intersection bool
}

// UnionPermanents selects each member of any supplied set once.
func UnionPermanents(sets ...PermanentSet) PermanentSet {
	return composedPermanentSet{members: append([]PermanentSet(nil), sets...)}
}

// IntersectPermanents selects members present in every supplied set.
func IntersectPermanents(sets ...PermanentSet) PermanentSet {
	return composedPermanentSet{members: append([]PermanentSet(nil), sets...), intersection: true}
}

func (composedPermanentSet) permanentSet() {}
func (s composedPermanentSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "union"}
	if s.intersection {
		n.Text = "intersection"
	}
	if len(s.members) == 0 {
		n.problem = "supply at least one permanent set"
	}
	for i, member := range s.members {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("members[%d]", i), member})
	}
	return n
}
func (s composedPermanentSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	counts := map[uuid.UUID]int{}
	for _, member := range s.members {
		if nilDefinition(member) {
			return nil
		}
		seen := map[uuid.UUID]bool{}
		for _, p := range member.resolvePermanents(ctx) {
			current := ctx.Game.FindPermanent(p.ID())
			if current != nil && current.incarnationID == p.incarnationID && !seen[p.incarnationID] {
				seen[p.incarnationID] = true
				counts[p.incarnationID]++
			}
		}
	}
	var result []*Permanent
	for _, p := range ctx.Game.FilterBattlefield(AnyPermanent) {
		count := counts[p.incarnationID]
		if !p.PhasedOut && count > 0 && (!s.intersection || count == len(s.members)) {
			result = append(result, p)
		}
	}
	return result
}

type attachmentHostSet struct{ hosts PermanentSet }

// AttachmentsOn selects attachments on current members of the host set.
func AttachmentsOn(hosts PermanentSet) PermanentSet { return attachmentHostSet{hosts: hosts} }
func (attachmentHostSet) permanentSet()             {}
func (s attachmentHostSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set", Text: "attachments on hosts", children: []definitionEdge{{"hosts", s.hosts}}}
}
func (s attachmentHostSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.hosts) {
		return nil
	}
	hosts := map[uuid.UUID]uuid.UUID{}
	for _, host := range s.hosts.resolvePermanents(ctx) {
		current := ctx.Game.FindPermanent(host.ID())
		if current != nil && current.incarnationID == host.incarnationID {
			hosts[host.ID()] = host.incarnationID
		}
	}
	var result []*Permanent
	for _, p := range ctx.Game.FilterBattlefield(AnyPermanent) {
		incarnation, ok := hosts[p.AttachedTo]
		if !p.PhasedOut && ok && p.attachedIncarnation == incarnation {
			result = append(result, p)
		}
	}
	return result
}

type playerSetPermanents struct {
	players PlayerSet
	filter  PermanentFilter
}

// PermanentsControlledByPlayers selects members controlled by any selected player.
func PermanentsControlledByPlayers(players PlayerSet, filter PermanentFilter) PermanentSet {
	return playerSetPermanents{players: players, filter: filter}
}
func (playerSetPermanents) permanentSet() {}
func (s playerSetPermanents) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set", Text: "controlled by players", children: []definitionEdge{{"players", s.players}}}
}
func (s playerSetPermanents) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.players) {
		return nil
	}
	players := map[uuid.UUID]bool{}
	for _, id := range s.players.resolvePlayers(ctx) {
		players[id] = true
	}
	var result []*Permanent
	for _, p := range ctx.Game.FilterBattlefield(s.filter) {
		if !p.PhasedOut && players[p.ControllerID()] {
			result = append(result, p)
		}
	}
	return result
}
