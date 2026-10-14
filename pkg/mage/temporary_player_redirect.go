package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// MayRedirectDamageFromPermanentsToPlayerThisTurn checks present recipients.
// The saved player chooses for each damage event until cleanup.
func MayRedirectDamageFromPermanentsToPlayerThisTurn(filter PermanentFilter, player PlayerRef) ResolvingEffect {
	return &playerDamageRedirectInstallation{filter: filter, player: player}
}

// RedirectSpellDamageToSourceControllerThisTurn saves one spell identity.
// Each damage event supplies its source controller, including after departure.
func RedirectSpellDamageToSourceControllerThisTurn(spell StackObjectRef) ResolvingEffect {
	return &playerDamageRedirectInstallation{spell: spell, bySource: true}
}

type playerDamageRedirectInstallation struct {
	spell    StackObjectRef
	player   PlayerRef
	filter   PermanentFilter
	bySource bool
}

func (*playerDamageRedirectInstallation) Text() string {
	return "redirect damage to a player this turn"
}
func (*playerDamageRedirectInstallation) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *playerDamageRedirectInstallation) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text()}
	if e.bySource {
		n.children = []definitionEdge{{"spell", DamageSourceOfSpell(e.spell)}}
	} else {
		n.children = []definitionEdge{{"player", e.player}}
		if nilDefinition(e.player) || e.filter.IsZero() {
			n.problem = "supply a player and recipient filter"
		}
	}
	return n
}
func (e *playerDamageRedirectInstallation) refersToSourcePermanent() bool {
	return !e.bySource && playerRefReadsSourcePermanent(e.player)
}
func (e *playerDamageRedirectInstallation) Apply(ctx *EffectContext) error {
	rule := &playerDamageRedirectThisTurn{replacementBase: replacementBase{duration: core.EndOfTurn}, filter: e.filter, bySource: e.bySource}
	if e.bySource {
		rule.source = DamageSourceOfSpell(e.spell).resolveDamageSource(ctx)
		if !rule.source.hasExactIdentity() {
			return nil
		}
	} else {
		player, ok := e.player.resolvePlayer(ctx)
		if !ok || ctx.Game.GetPlayer(player) == nil {
			return nil
		}
		rule.player = player
	}
	ctx.Game.effects.AddReplacement(rule)
	return nil
}

type playerDamageRedirectThisTurn struct {
	replacementBase
	source   DamageSource
	filter   PermanentFilter
	player   uuid.UUID
	bySource bool
}

func (r *playerDamageRedirectThisTurn) destination(action Action) uuid.UUID {
	if !r.bySource {
		return r.player
	}
	return damageActionSource(action).Controller()
}
func (r *playerDamageRedirectThisTurn) Matches(action Action, reader GameReader) bool {
	g := reader.(*Game)
	if damageActionAmount(action) <= 0 || g.GetPlayer(r.destination(action)) == nil {
		return false
	}
	if damage, ok := action.(*DamageToCreatureAction); ok {
		subject := g.FindPermanent(damage.PermanentID())
		if subject == nil || (damage.recipientIncarnation != uuid.Nil && subject.incarnationID != damage.recipientIncarnation) {
			return false
		}
	}
	if r.bySource {
		switch damage := action.(type) {
		case *DamageToPlayerAction:
			return r.source.SameObject(damage.DamageSource()) && damage.PlayerID() != r.destination(action)
		case *DamageToCreatureAction:
			return r.source.SameObject(damage.DamageSource())
		}
		return false
	}
	damage, ok := action.(*DamageToCreatureAction)
	if !ok {
		return false
	}
	subject := g.FindPermanent(damage.PermanentID())
	return subject != nil && damageRedirectDestination(subject) && r.filter.Match(subject, g)
}
func (r *playerDamageRedirectThisTurn) Replace(action Action, g *Game) Action {
	if !r.Matches(action, g) {
		return action
	}
	player := r.destination(action)
	if !r.bySource {
		if !g.GetPlayer(player).ChooseMayAbility("take this damage instead") || !r.Matches(action, g) {
			return action
		}
	}
	switch damage := action.(type) {
	case *DamageToPlayerAction:
		return NewDamageToPlayerAction(damage.ActionSource(), player, damage.Amount(), damage.IsCombatDamage()).withDamageSource(damage.DamageSource())
	case *DamageToCreatureAction:
		return NewDamageToPlayerAction(damage.ActionSource(), player, damage.Amount(), damage.IsCombatDamage()).withDamageSource(damage.DamageSource())
	}
	return action
}
func (*playerDamageRedirectThisTurn) IsActive(GameReader) bool { return true }
func (r *playerDamageRedirectThisTurn) Clone() ReplacementEffect {
	c := *r
	c.source = cloneDamageSource(r.source)
	return &c
}
