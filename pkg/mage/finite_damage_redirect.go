package mage

import (
	"fmt"
	"maps"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// RedirectNextDamageToPlayer saves a finite damage budget until cleanup.
// The subject keeps its exact visit and the destination keeps its player.
func RedirectNextDamageToPlayer(subject PermanentRef, player PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &finiteDamageRedirectInstallation{subject, player, amount}
}

type finiteDamageRedirectInstallation struct {
	subject PermanentRef
	player  PlayerRef
	amount  ResolutionValue
}

func (*finiteDamageRedirectInstallation) Text() string {
	return "redirect next damage to a player this turn"
}
func (*finiteDamageRedirectInstallation) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *finiteDamageRedirectInstallation) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"player", e.player}, {"amount", e.amount}}}
	if _, ok := e.subject.(permanentInformationReference); !ok {
		n.problem = "subject requires an exact permanent identity"
	}
	if nilDefinition(e.subject) || nilDefinition(e.player) || nilDefinition(e.amount) {
		n.problem = "supply subject, player and amount"
	}
	return n
}
func (e *finiteDamageRedirectInstallation) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject) || playerRefReadsSourcePermanent(e.player) || resolutionValueReadsSourcePermanent(e.amount)
}
func (e *finiteDamageRedirectInstallation) Apply(ctx *EffectContext) error {
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
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	ctx.Game.effects.AddReplacement(&finiteDamageRedirectReplacement{replacementBase: replacementBase{duration: core.EndOfTurn}, subject: subject, player: player, remaining: amount})
	return nil
}

type finiteDamageRedirectReplacement struct {
	replacementBase
	subject   capturedPermanent
	player    uuid.UUID
	remaining int
}

func (r *finiteDamageRedirectReplacement) Matches(action Action, reader GameReader) bool {
	damage, ok := action.(*DamageToCreatureAction)
	if !ok || damage.Amount() <= 0 || r.remaining <= 0 || damage.PermanentID() != r.subject.id {
		return false
	}
	g := reader.(*Game)
	subject := g.FindPermanent(r.subject.id)
	return subject != nil && !subject.PhasedOut && subject.incarnationID == r.subject.incarnation && damageRedirectDestination(subject) && g.GetPlayer(r.player) != nil && (damage.recipientIncarnation == uuid.Nil || damage.recipientIncarnation == r.subject.incarnation)
}
func (r *finiteDamageRedirectReplacement) IsActive(GameReader) bool { return r.remaining > 0 }
func (r *finiteDamageRedirectReplacement) Clone() ReplacementEffect { c := *r; return &c }
func (r *finiteDamageRedirectReplacement) Replace(action Action, g *Game) Action {
	actions, _ := r.splitDamageEvent([]Action{action}, []map[ReplacementEffect]bool{{}}, g)
	return &expandedDamageAction{actions: actions}
}
func (r *finiteDamageRedirectReplacement) splitDamageEvent(actions []Action, applied []map[ReplacementEffect]bool, g *Game) (result []Action, uses []map[ReplacementEffect]bool) {
	var indices []int
	for i, action := range actions {
		if action != nil && !applied[i][r] && !damageReplacementBlocked(action, g) && r.Matches(action, g) {
			indices = append(indices, i)
		}
	}
	if len(indices) == 0 {
		return actions, applied
	}
	capacity := make([]int, len(indices)+1)
	for j := len(indices) - 1; j >= 0; j-- {
		capacity[j] = capacity[j+1] + min(damageActionAmount(actions[indices[j]]), r.remaining-capacity[j+1])
	}
	budget := capacity[0]
	chooser := replacementAffectedPlayer(actions[indices[0]], g)
	for j, i := range indices {
		action := actions[i].(*DamageToCreatureAction)
		minimum, maximum := max(0, budget-capacity[j+1]), min(budget, action.Amount())
		amount := maximum
		if minimum < maximum && chooser != nil {
			amount = chooser.ChooseNumber(minimum, maximum, fmt.Sprintf("Choose damage to redirect from %s (packet %d, %d damage)", damageSourceLabel(action, g), i+1, action.Amount()))
			if amount < minimum || amount > maximum {
				amount = maximum
			}
		}
		applied[i][r] = true
		r.remaining -= amount
		budget -= amount
		actions[i] = damageActionWithAmount(action, action.Amount()-amount)
		if amount > 0 {
			redirected := NewDamageToPlayerAction(action.ActionSource(), r.player, amount, action.IsCombatDamage()).withDamageSource(action.DamageSource())
			keepDamageCause(action, redirected)
			actions = append(actions, redirected)
			applied = append(applied, maps.Clone(applied[i]))
		}
	}
	return actions, applied
}
