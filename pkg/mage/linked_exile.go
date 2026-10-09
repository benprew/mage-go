package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ExileLink identifies one group of linked exile abilities.
// A declaration has no mutable game state.
type ExileLink struct{ id uuid.UUID }

// NewExileLink creates a distinct declaration for linked exile instructions.
func NewExileLink() *ExileLink { return &ExileLink{id: uuid.New()} }

func (l *ExileLink) inspectDefinition() definitionNode {
	n := definitionNode{Role: "exile-link"}
	if l == nil || l.id == uuid.Nil {
		n.problem = "supply a non-nil exile link declaration"
	}
	return n
}

// ExilePermanentLinked saves the link before the permanent's zone event.
func ExilePermanentLinked(subject PermanentRef, link *ExileLink) ResolvingEffect {
	return &linkedExileEffect{subject: subject, link: link}
}

// ExileLibraryTopLinked exiles the current top without a draw or reveal.
// Face-down exile gives no player permission to inspect the card.
func ExileLibraryTopLinked(player PlayerRef, link *ExileLink, faceDown bool) ResolvingEffect {
	return &linkedExileEffect{player: player, link: link, library: true, faceDown: faceDown}
}

type linkedExileEffect struct {
	subject           PermanentRef
	player            PlayerRef
	link              *ExileLink
	library, faceDown bool
}

func (*linkedExileEffect) Text() string { return "exile with an ability link" }
func (*linkedExileEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, TargetPurposeOverride: AITargetExile}
}
func (e *linkedExileEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", children: []definitionEdge{{"link", e.link}, {"source", This}}}
	if e.library {
		n.children = append(n.children, definitionEdge{"player", e.player})
	} else {
		n.children = append(n.children, definitionEdge{"subject", e.subject})
	}
	return n
}
func (e *linkedExileEffect) refersToSourcePermanent() bool { return true }
func (e *linkedExileEffect) Apply(ctx *EffectContext) error {
	source, ok := This.informationIdentity(ctx)
	if !ok || e.link == nil || e.link.id == uuid.Nil {
		return nil
	}
	if !e.library {
		if e.subject != nil {
			if p, present := e.subject.resolvePermanent(ctx); present {
				ctx.Game.exilePermanentWithLink(p, source.id, e.link, source)
			}
		}
		return nil
	}
	player := handOperationPlayer(e.player, ctx)
	if player == nil || len(player.Library()) == 0 {
		return nil
	}
	card := player.Library()[0]
	player.SetLibrary(player.Library()[1:])
	if e.faceDown {
		ctx.Game.zones.ExileCardFaceDown(card, source.id)
	} else {
		ctx.Game.zones.ExileCard(card, source.id)
	}
	ctx.Game.setExileLink(card.ID(), e.link, source)
	ctx.Game.FireEvent(core.GameEvent{Type: core.EvtZoneChange, SourceID: card.ID(), PlayerID: player.PlayerID(), FromZone: core.ZoneLibrary, ToZone: core.ZoneExile})
	ctx.Game.consumePendingExileZoneChange(card.ID())
	return nil
}

func (g *Game) setExileLink(cardID uuid.UUID, link *ExileLink, source capturedPermanent) {
	if exiled := g.FindExiledCard(cardID); exiled != nil {
		exiled.link = link
		exiled.linkedSource = source
	}
}

// ExiledCardSet selects exile visits without permission to inspect the cards.
type ExiledCardSet interface {
	exiledCardSet()
	resolveExiledCards(*EffectContext) []capturedCard
}

// CardsExiledWith selects current exile visits from this link and source.
func CardsExiledWith(link *ExileLink, source PermanentRef) ExiledCardSet {
	return linkedExileSet{link: link, source: source}
}

type linkedExileSet struct {
	link   *ExileLink
	source PermanentRef
}

func (linkedExileSet) exiledCardSet() {}
func (s linkedExileSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "exiled-card-set", Text: "linked exile visits", children: []definitionEdge{{"link", s.link}, {"source", s.source}}}
}
func (s linkedExileSet) resolveExiledCards(ctx *EffectContext) []capturedCard {
	ref, ok := s.source.(permanentInformationReference)
	if !ok || nilDefinition(ref) || s.link == nil || s.link.id == uuid.Nil {
		return nil
	}
	source, ok := ref.informationIdentity(ctx)
	if !ok || source.id == uuid.Nil || source.incarnation == uuid.Nil {
		return nil
	}
	var cards []capturedCard
	for _, exiled := range ctx.Game.zones.exile {
		if exiled.link == s.link && exiled.linkedSource == source {
			cards = append(cards, capturedCard{id: exiled.Card.ID(), visit: exiled.Visit, zone: core.ZoneExile})
		}
	}
	return cards
}

// CountExiledCards counts current exile visits without reading characteristics.
func CountExiledCards(subjects ExiledCardSet) ResolutionValue {
	return exiledCardCountValue{subjects: subjects}
}

type exiledCardCountValue struct{ subjects ExiledCardSet }

func (exiledCardCountValue) Text() string { return "number of selected exile visits" }
func (v exiledCardCountValue) resolveValue(ctx *EffectContext) int {
	return len(v.subjects.resolveExiledCards(ctx))
}
func (v exiledCardCountValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "exiled-card-count", children: []definitionEdge{{"subjects", v.subjects}}}
}
