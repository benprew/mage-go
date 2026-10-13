package mage

import (
	"slices"

	"github.com/google/uuid"
)

// LeastHandSize reads the smallest selected player's current hand size.
// An empty player set supplies zero.
func LeastHandSize(players PlayerSet) ResolutionValue {
	return leastHandSize{players}
}

type leastHandSize struct{ players PlayerSet }

func (leastHandSize) Text() string { return "least hand size" }
func (v leastHandSize) resolveValue(ctx *EffectContext) int {
	selected := v.players.resolvePlayers(ctx)
	least, found := 0, false
	for _, id := range EachPlayer.resolvePlayers(ctx) {
		if !slices.Contains(selected, id) {
			continue
		}
		count := len(ctx.Game.GetPlayer(id).Hand())
		if !found || count < least {
			least = count
			found = true
		}
	}
	return least
}
func (v leastHandSize) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"players", v.players}}}
}

// DiscardAllButEach makes each selected player discard down to a kept count.
// All choices precede replacements and the shared simultaneous discard batch.
func DiscardAllButEach(players PlayerSet, keep ResolutionValue) ResolvingEffect {
	return &discardAllButEach{players, keep}
}

type discardAllButEach struct {
	players PlayerSet
	keep    ResolutionValue
}

func (*discardAllButEach) Text() string { return "each player discards all but a number of cards" }
func (*discardAllButEach) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *discardAllButEach) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"players", e.players}, {"keep", e.keep}}}
}
func (e *discardAllButEach) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.players) || resolutionValueReadsSourcePermanent(e.keep)
}
func (e *discardAllButEach) Apply(ctx *EffectContext) error {
	keep := max(0, e.keep.resolveValue(ctx))
	groups := snapshotHandGroups(ctx, e.players)
	total := 0
	for _, group := range groups {
		total += max(0, len(group.actions)-keep)
	}
	actions := make([]*DiscardAction, 0, total)
	for _, group := range groups {
		count := max(0, len(group.actions)-keep)
		actions = append(actions, chooseHandDiscards(ctx, group, count)...)
	}
	ctx.Game.discardActionsTogether(actions)
	return nil
}

type handChoiceGroup struct {
	player  Player
	actions []*DiscardAction
}

func snapshotHandGroup(ctx *EffectContext, player Player) handChoiceGroup {
	group := handChoiceGroup{player: player}
	for _, card := range player.Hand() {
		action := NewDiscardAction(ctx.SourceID, player.PlayerID(), card.ID())
		action.handVisit = ctx.Game.handVisit(card.ID())
		group.actions = append(group.actions, action)
	}
	return group
}

func snapshotHandGroups(ctx *EffectContext, players PlayerSet) []handChoiceGroup {
	selected := players.resolvePlayers(ctx)
	var groups []handChoiceGroup
	for _, id := range EachPlayer.resolvePlayers(ctx) {
		if slices.Contains(selected, id) {
			groups = append(groups, snapshotHandGroup(ctx, ctx.Game.GetPlayer(id)))
		}
	}
	return groups
}

func chooseHandDiscards(ctx *EffectContext, group handChoiceGroup, count int) []*DiscardAction {
	count = min(max(0, count), len(group.actions))
	if count == 0 {
		return nil
	}
	chosen := group.player.ChooseCardsFromHand(count, "Choose cards to discard", ctx.Game)
	byID := make(map[uuid.UUID]*DiscardAction, len(group.actions))
	for _, action := range group.actions {
		byID[action.CardID()] = action
	}
	actions := make([]*DiscardAction, 0, count)
	for _, card := range chosen {
		if card == nil {
			continue
		}
		if action := byID[card.ID()]; action != nil {
			actions = append(actions, action)
			delete(byID, card.ID())
			if len(actions) == count {
				return actions
			}
		}
	}
	for _, action := range group.actions {
		if byID[action.CardID()] != nil {
			actions = append(actions, action)
			delete(byID, action.CardID())
			if len(actions) == count {
				break
			}
		}
	}
	return actions
}
