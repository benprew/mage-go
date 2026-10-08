package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// RedirectNextDamageFromSelectedSource saves one source, subject and player.
// A selected permanent spell continues to the permanent it becomes.
func RedirectNextDamageFromSelectedSource(source DamageSourceRef, subject PermanentRef, player PlayerRef, filter DamageSourceFilter) ResolvingEffect {
	return &selectedSourceRedirectInstallation{source: source, subject: subject, player: player, filter: filter}
}

type selectedSourceRedirectInstallation struct {
	filter  DamageSourceFilter
	source  DamageSourceRef
	subject PermanentRef
	player  PlayerRef
}

func (*selectedSourceRedirectInstallation) Text() string {
	return "redirect next damage from selected source this turn"
}
func (*selectedSourceRedirectInstallation) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *selectedSourceRedirectInstallation) inspectDefinition() definitionNode {
	n := definitionNode{Role: "install-selected-source-redirection", Text: e.Text(), children: []definitionEdge{{"source", damageSourceDefinition{e.source}}, {"subject", e.subject}, {"player", e.player}, {"filter", e.filter}}}
	if nilDefinition(e.subject) || nilDefinition(e.player) {
		n.problem = "supply subject and player references"
	}
	if _, ok := e.subject.(permanentInformationReference); !ok {
		n.problem = "subject requires an exact permanent identity"
	}
	return n
}
func (e *selectedSourceRedirectInstallation) Apply(ctx *EffectContext) error {
	source := e.source.resolveDamageSource(ctx)
	if !source.Valid() || (source.Incarnation() != uuid.Nil) == (source.StackID() != uuid.Nil) {
		return nil
	}
	ref, ok := e.subject.(permanentInformationReference)
	if !ok {
		return nil
	}
	subject, ok := ref.informationIdentity(ctx)
	if !ok || subject.id == uuid.Nil || subject.incarnation == uuid.Nil {
		return nil
	}
	player, ok := e.player.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(player) == nil {
		return nil
	}
	information := ctx.Game.damageReferenceInformation(source)
	if !information.SameObject(source) {
		information = source
	}
	if !e.filter.Match(information) {
		return nil
	}
	ctx.Game.effects.AddReplacement(&selectedSourceRedirectReplacement{replacementBase: replacementBase{duration: core.EndOfTurn}, source: cloneDamageSource(information), filter: e.filter, subject: subject, player: player})
	return nil
}

type selectedSourceRedirectReplacement struct {
	replacementBase
	filter            DamageSourceFilter
	source, continued DamageSource
	subject           capturedPermanent
	player            uuid.UUID
	consumed          bool
}

func (r *selectedSourceRedirectReplacement) Matches(action Action, reader GameReader) bool {
	damage, ok := action.(*DamageToCreatureAction)
	if !ok || damage.Amount() <= 0 || damage.PermanentID() != r.subject.id {
		return false
	}
	g := reader.(*Game)
	subject := g.FindPermanent(r.subject.id)
	information := g.damageReferenceInformation(damage.DamageSource())
	if !information.SameObject(damage.DamageSource()) {
		information = damage.DamageSource()
	}
	return r.filter.Match(information) && subject != nil && subject.incarnationID == r.subject.incarnation && damageRedirectDestination(subject) && g.GetPlayer(r.player) != nil && (r.source.SameObject(damage.DamageSource()) || r.continued.SameObject(damage.DamageSource()))
}
func (r *selectedSourceRedirectReplacement) Replace(action Action, g *Game) Action {
	if !r.Matches(action, g) {
		return action
	}
	damage := action.(*DamageToCreatureAction)
	r.consumed = true
	return NewDamageToPlayerAction(damage.ActionSource(), r.player, damage.Amount(), damage.IsCombatDamage()).withDamageSource(damage.DamageSource())
}
func (r *selectedSourceRedirectReplacement) IsActive(GameReader) bool { return !r.consumed }
func (r *selectedSourceRedirectReplacement) Clone() ReplacementEffect {
	c := *r
	c.source = cloneDamageSource(r.source)
	c.continued = cloneDamageSource(r.continued)
	return &c
}
func (r *selectedSourceRedirectReplacement) redirectBatch(actions []Action, applied []map[ReplacementEffect]bool, g *Game) {
	for i, action := range actions {
		if action == nil || applied[i][r] || damageReplacementBlocked(action, g) || !r.Matches(action, g) {
			continue
		}
		applied[i][r] = true
		actions[i] = r.Replace(action, g)
	}
}
