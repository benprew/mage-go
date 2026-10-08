package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// DiscardAction represents a proposed discard.
type DiscardAction struct {
	source                  uuid.UUID
	playerID                uuid.UUID
	cardID                  uuid.UUID
	destination             Zone
	effectCause             bool
	requiresCharacteristics bool
}

// NewDiscardAction creates an effect-caused discard whose normal destination
// is its owner's graveyard.
func NewDiscardAction(source, playerID, cardID uuid.UUID) *DiscardAction {
	return &DiscardAction{source: source, playerID: playerID, cardID: cardID, destination: ZoneGraveyard, effectCause: true}
}

func newCostDiscardAction(source, playerID, cardID uuid.UUID) *DiscardAction {
	return &DiscardAction{source: source, playerID: playerID, cardID: cardID, destination: ZoneGraveyard}
}

func (a *DiscardAction) ActionSource() uuid.UUID { return a.source }
func (a *DiscardAction) PlayerID() uuid.UUID     { return a.playerID }
func (a *DiscardAction) CardID() uuid.UUID       { return a.cardID }
func (a *DiscardAction) Destination() Zone       { return a.destination }
func (a *DiscardAction) CausedByEffect() bool    { return a.effectCause }

func (a *DiscardAction) withDestination(destination Zone) *DiscardAction {
	cp := *a
	cp.destination = destination
	return &cp
}

type discardToLibraryReplacement struct {
	replacementBase
	incarnation uuid.UUID
	ruleID      uuid.UUID
}

func (r *discardToLibraryReplacement) currentSource(g GameReader) *Permanent {
	source := g.FindPermanent(r.sourceID)
	if source == nil || source.PhasedOut || source.incarnationID != r.incarnation {
		return nil
	}
	for _, ability := range source.RuntimeAbilities {
		if holder, ok := UnwrapAbility(ability).(*StaticAbilityHolder); ok {
			for _, effect := range holder.Effects {
				if rule, ok := effect.(*discardToLibraryContinuous); ok && rule.ruleID == r.ruleID {
					return source
				}
			}
		}
	}
	return nil
}

func (r *discardToLibraryReplacement) Matches(action Action, g GameReader) bool {
	discard, ok := action.(*DiscardAction)
	source := r.currentSource(g)
	return ok && source != nil && discard.CausedByEffect() && discard.PlayerID() == source.ControllerID() && discard.Destination() == ZoneGraveyard
}

func (r *discardToLibraryReplacement) Replace(action Action, g *Game) Action {
	source := r.currentSource(g)
	if source == nil {
		return action
	}
	player := g.GetPlayer(source.ControllerID())
	if player == nil || !player.ChooseMayAbility("put discarded card on top of your library instead") {
		return action
	}
	return action.(*DiscardAction).withDestination(ZoneLibrary)
}

func (r *discardToLibraryReplacement) IsActive(g GameReader) bool {
	return r.currentSource(g) != nil
}

func (r *discardToLibraryReplacement) Clone() ReplacementEffect {
	clone := *r
	return &clone
}

// DiscardToLibraryReplacement permits effect-caused discards to go to the library.
// The source's current controller makes each optional choice.
func DiscardToLibraryReplacement() ContinuousEffect {
	return &discardToLibraryContinuous{ruleID: uuid.New()}
}

type discardToLibraryContinuous struct {
	effectSource
	ruleID uuid.UUID
}

func (*discardToLibraryContinuous) Text() string {
	return "you may discard to the top of your library instead"
}
func (*discardToLibraryContinuous) Properties() EffectProperties { return EffectProperties{} }
func (*discardToLibraryContinuous) GetLayer() Layer              { return LayerAbility }
func (*discardToLibraryContinuous) GetDuration() Duration        { return WhileOnBattlefield }
func (e *discardToLibraryContinuous) IsActive(g *Game) bool {
	source := g.FindPermanent(e.sourceID)
	return source != nil && !source.PhasedOut
}
func (e *discardToLibraryContinuous) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-discard-replacement", Text: e.Text(), battlefieldSource: true}
}
func (e *discardToLibraryContinuous) ApplyContinuous(ctx *EffectContext) error {
	source := ctx.Game.FindPermanent(e.sourceID)
	if source != nil && !source.PhasedOut {
		ctx.Game.effects.AddCycleReplacement(&discardToLibraryReplacement{
			replacementBase: replacementBase{sourceID: source.ID(), duration: WhileOnBattlefield},
			incarnation:     source.incarnationID, ruleID: e.ruleID,
		})
	}
	return nil
}
func (e *discardToLibraryContinuous) cloneEffect() ContinuousEffect {
	clone := *e
	return &clone
}
