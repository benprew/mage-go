package mage

import (
	"maps"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AddSpellDamageToNextSelectedSourceEvent saves one damage replacement this turn.
// The selected source deals its damage. The saved spell deals the same proposed
// amount to that source's controller in the same event.
func AddSpellDamageToNextSelectedSourceEvent(source DamageSourceRef, recipient PlayerRef, spell DamageSourceRef, filter DamageSourceFilter) ResolvingEffect {
	return &continuedDamageInstallation{source: source, recipient: recipient, spell: spell, filter: filter}
}

type continuedDamageInstallation struct {
	source, spell DamageSourceRef
	recipient     PlayerRef
	filter        DamageSourceFilter
}

func (*continuedDamageInstallation) Text() string {
	return "continue damage and add spell damage this turn"
}
func (*continuedDamageInstallation) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *continuedDamageInstallation) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"source", damageSourceDefinition{e.source}}, {"recipient", e.recipient}, {"spell", damageSourceDefinition{e.spell}}, {"filter", e.filter}}}
}
func (e *continuedDamageInstallation) refersToSourcePermanent() bool {
	return damageReadsSourcePermanent(e.source) || damageReadsSourcePermanent(e.spell) || playerRefReadsSourcePermanent(e.recipient)
}
func (e *continuedDamageInstallation) Apply(ctx *EffectContext) error {
	if nilDefinition(e.source) || nilDefinition(e.spell) || nilDefinition(e.recipient) {
		return nil
	}
	source, spell := e.source.resolveDamageSource(ctx), e.spell.resolveDamageSource(ctx)
	player, ok := e.recipient.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(player) == nil || !source.hasExactIdentity() || spell.StackID() == uuid.Nil || spell.Incarnation() != uuid.Nil {
		return nil
	}
	information := ctx.Game.damageSourceRuleInformation(source)
	if !information.SameObject(source) {
		information = source
	}
	if !e.filter.Match(information) {
		return nil
	}
	ctx.Game.effects.AddReplacement(&continuedDamageReplacement{replacementBase: replacementBase{duration: core.EndOfTurn, sourceID: spell.ID()}, source: cloneDamageSource(information), spell: cloneDamageSource(spell), player: player, filter: e.filter, cause: ctx.Game.currentDamageCause()})
	return nil
}

type continuedDamageReplacement struct {
	replacementBase
	source, continued, spell DamageSource
	player                   uuid.UUID
	filter                   DamageSourceFilter
	cause                    damageCause
	consumed                 bool
}

func (r *continuedDamageReplacement) Matches(action Action, reader GameReader) bool {
	g := reader.(*Game)
	damage, ok := action.(*DamageToPlayerAction)
	if !ok || damage.Amount() <= 0 || damage.PlayerID() != r.player || g.GetPlayer(r.player) == nil {
		return false
	}
	source := damage.DamageSource()
	information := g.damageSourceRuleInformation(source)
	if !information.SameObject(source) {
		information = source
	}
	return r.filter.Match(information) && (r.source.SameObject(source) || r.continued.SameObject(source))
}
func (r *continuedDamageReplacement) IsActive(GameReader) bool { return !r.consumed }
func (r *continuedDamageReplacement) Clone() ReplacementEffect {
	c := *r
	c.source = cloneDamageSource(r.source)
	c.continued = cloneDamageSource(r.continued)
	c.spell = cloneDamageSource(r.spell)
	c.cause = r.cause.clone()
	return &c
}
func (r *continuedDamageReplacement) Replace(action Action, g *Game) Action {
	if !r.Matches(action, g) {
		return action
	}
	actions, _ := r.expandDamageEvent([]Action{action}, []map[ReplacementEffect]bool{{}}, g)
	return &expandedDamageAction{actions: actions}
}

// damageEventExpander keeps new damage in the shared replacement pipeline.
// Each descendant retains prior uses, so a rule cannot repeat on its own event.
type damageEventExpander interface {
	expandDamageEvent([]Action, []map[ReplacementEffect]bool, *Game) ([]Action, []map[ReplacementEffect]bool)
}

func (r *continuedDamageReplacement) expandDamageEvent(actions []Action, applied []map[ReplacementEffect]bool, g *Game) (expanded []Action, uses []map[ReplacementEffect]bool) {
	type extraPacket struct {
		player uuid.UUID
		amount int
		used   map[ReplacementEffect]bool
	}
	var packets []extraPacket
	for i, action := range actions {
		if action == nil || applied[i][r] || !r.Matches(action, g) {
			continue
		}
		applied[i][r] = true
		player := damageActionSource(action).Controller()
		index := -1
		for j := range packets {
			if packets[j].player == player {
				index = j
				break
			}
		}
		if index < 0 {
			packets = append(packets, extraPacket{player: player, used: make(map[ReplacementEffect]bool)})
			index = len(packets) - 1
		}
		packets[index].amount += damageActionAmount(action)
		maps.Copy(packets[index].used, applied[i])
		r.consumed = true
	}
	for _, packet := range packets {
		damage := NewDamageToPlayerAction(r.spell.ID(), packet.player, packet.amount, false).withDamageSource(r.spell)
		damage.cause = r.cause.clone()
		damage.origin = -1
		actions = append(actions, damage)
		applied = append(applied, packet.used)
	}
	return actions, applied
}

type expandedDamageAction struct {
	actions []Action
}

func (a *expandedDamageAction) ActionSource() uuid.UUID {
	for _, action := range a.actions {
		if action != nil {
			return action.ActionSource()
		}
	}
	return uuid.Nil
}

// WaitingDamageExpansionSources reads saved references in active damage expansions.
// A returned object cannot supply the saved reference.
func (g *Game) WaitingDamageExpansionSources() []DamageSource {
	var sources []DamageSource
	for _, list := range [2][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			rule, ok := replacement.(*continuedDamageReplacement)
			if !ok || !rule.IsActive(g) {
				continue
			}
			for _, saved := range []DamageSource{rule.source, rule.continued, rule.spell} {
				if !saved.hasExactIdentity() {
					continue
				}
				information := g.damageSourceRuleInformation(saved)
				if !information.SameObject(saved) {
					information = saved
				}
				sources = append(sources, cloneDamageSource(information))
			}
		}
	}
	return sources
}
