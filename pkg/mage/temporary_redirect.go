package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type permanentDamageRedirectInstallation struct {
	source, destination PermanentRef
	player              PlayerRef
}

// RedirectDamageFromPermanentThisTurn saves a source, player and destination.
// Control changes cannot change the saved player or permanent identities.
func RedirectDamageFromPermanentThisTurn(source PermanentRef, player PlayerRef, destination PermanentRef) ResolvingEffect {
	return &permanentDamageRedirectInstallation{source: source, player: player, destination: destination}
}
func (*permanentDamageRedirectInstallation) Text() string {
	return "redirect damage from saved permanent this turn"
}
func (*permanentDamageRedirectInstallation) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *permanentDamageRedirectInstallation) inspectDefinition() definitionNode {
	n := definitionNode{Role: "install-permanent-damage-redirection", Text: e.Text(), children: []definitionEdge{{"source", e.source}, {"player", e.player}, {"destination", e.destination}}}
	if nilDefinition(e.source) || nilDefinition(e.player) || nilDefinition(e.destination) {
		n.problem = "supply source, player and destination references"
	}
	if _, ok := e.source.(permanentInformationReference); !ok {
		n.problem = "source requires an exact permanent identity"
	}
	if _, ok := e.destination.(permanentInformationReference); !ok {
		n.problem = "destination requires an exact permanent identity"
	}
	return n
}
func (e *permanentDamageRedirectInstallation) Apply(ctx *EffectContext) error {
	sourceRef, ok := e.source.(permanentInformationReference)
	if !ok {
		return nil
	}
	source, ok := sourceRef.informationIdentity(ctx)
	if !ok || source.id == uuid.Nil || source.incarnation == uuid.Nil {
		return nil
	}
	destinationRef, ok := e.destination.(permanentInformationReference)
	if !ok {
		return nil
	}
	destination, ok := destinationRef.informationIdentity(ctx)
	if !ok || destination.id == uuid.Nil || destination.incarnation == uuid.Nil {
		return nil
	}
	player, ok := e.player.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(player) == nil {
		return nil
	}
	ctx.Game.effects.AddReplacement(&permanentDamageRedirectReplacement{
		replacementBase: replacementBase{duration: core.EndOfTurn},
		from:            source,
		to:              destination,
		player:          player,
	})
	return nil
}

type permanentDamageRedirectReplacement struct {
	replacementBase
	from, to capturedPermanent
	player   uuid.UUID
}

func damageRedirectDestination(p *Permanent) bool {
	return p.HasType(core.TypeCreature) || p.HasType(core.TypePlaneswalker)
}
func (r *permanentDamageRedirectReplacement) destination(g *Game) *Permanent {
	p := g.FindPermanent(r.to.id)
	if p == nil || p.incarnationID != r.to.incarnation || !damageRedirectDestination(p) {
		return nil
	}
	return p
}
func (r *permanentDamageRedirectReplacement) Matches(action Action, reader GameReader) bool {
	damage, ok := action.(*DamageToPlayerAction)
	if !ok || damage.Amount() <= 0 || damage.PlayerID() != r.player {
		return false
	}
	g := reader.(*Game)
	source := damage.DamageSource()
	return source.ID() == r.from.id && source.Incarnation() == r.from.incarnation && r.destination(g) != nil && g.GetPlayer(r.player) != nil
}
func (r *permanentDamageRedirectReplacement) Replace(action Action, g *Game) Action {
	if !r.Matches(action, g) {
		return action
	}
	damage := action.(*DamageToPlayerAction)
	return NewDamageToCreatureAction(damage.ActionSource(), r.to.id, damage.Amount(), damage.IsCombatDamage()).withDamageSource(damage.DamageSource())
}
func (*permanentDamageRedirectReplacement) IsActive(GameReader) bool   { return true }
func (r *permanentDamageRedirectReplacement) Clone() ReplacementEffect { c := *r; return &c }
