package mage

import "github.com/google/uuid"

// CreatedTokenLink identifies one token producer and its linked consumers.
// The declaration contains no mutable game state.
type CreatedTokenLink struct{ id uuid.UUID }

// NewCreatedTokenLink creates a distinct token link declaration.
func NewCreatedTokenLink() *CreatedTokenLink { return &CreatedTokenLink{id: uuid.New()} }

func (l *CreatedTokenLink) inspectDefinition() definitionNode {
	n := definitionNode{Role: "created-token-link"}
	if l == nil || l.id == uuid.Nil {
		n.problem = "supply a non-nil created token link declaration"
	}
	return n
}

// CreateTokenCopiesLinked links accepted tokens to the source's exact visit.
// The link is present before entry observations.
func CreateTokenCopiesLinked(player PlayerRef, count ResolutionValue, token TokenDefinition, link *CreatedTokenLink) ResolvingEffect {
	return &createLinkedTokensEffect{player: player, count: count, token: token, link: link}
}

type createLinkedTokensEffect struct {
	player PlayerRef
	count  ResolutionValue
	token  TokenDefinition
	link   *CreatedTokenLink
}

func (*createLinkedTokensEffect) Text() string { return "create linked tokens" }
func (e *createLinkedTokensEffect) Properties() EffectProperties {
	return (&createTokenCopiesEffect{token: e.token}).Properties()
}
func (*createLinkedTokensEffect) refersToSourcePermanent() bool { return true }
func (e *createLinkedTokensEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"count", e.count}, {"token", e.token}, {"link", e.link}, {"source", This}}}
}
func (e *createLinkedTokensEffect) Apply(ctx *EffectContext) error {
	source, ok := This.informationIdentity(ctx)
	if !ok {
		return nil
	}
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	count := e.count.resolveValue(ctx)
	if count <= 0 {
		return nil
	}
	_, err := createDefinitionTokenBatchLinked(ctx.Game, player, count, e.token.(*tokenDefinition), e.link, source)
	return err
}

// TokensCreatedWith selects surviving, unphased tokens from one source visit.
// Source departure does not remove the link. A returned source has a new visit.
func TokensCreatedWith(link *CreatedTokenLink, source PermanentRef) PermanentSet {
	return linkedCreatedTokenSet{link: link, source: source}
}

type linkedCreatedTokenSet struct {
	link   *CreatedTokenLink
	source PermanentRef
}

func (linkedCreatedTokenSet) permanentSet() {}
func (s linkedCreatedTokenSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "linked-created-token-set", children: []definitionEdge{{"link", s.link}, {"source", s.source}}}
}
func (s linkedCreatedTokenSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	ref, ok := s.source.(permanentInformationReference)
	if !ok || nilDefinition(ref) {
		return nil
	}
	source, ok := ref.informationIdentity(ctx)
	if !ok {
		return nil
	}
	var result []*Permanent
	for _, p := range ctx.Game.FilterBattlefield(IsToken) {
		if p.createdTokenLink == s.link && p.createdTokenSource == source {
			result = append(result, p)
		}
	}
	return result
}
