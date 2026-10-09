package mage

import "github.com/google/uuid"

// ProposedDamageValue reads an amount when a prevention rule applies.
// It cannot read resolution bindings or retain game state.
type ProposedDamageValue interface {
	proposedDamageValue(int) int
}

type proposedDamageAmount struct{}
type halfProposedDamageAmount struct{}

// ProposedDamageAmount reads all damage in the matching source event.
func ProposedDamageAmount() ProposedDamageValue { return proposedDamageAmount{} }

// HalfProposedDamageRoundedDown reads half the damage in the matching event.
func HalfProposedDamageRoundedDown() ProposedDamageValue { return halfProposedDamageAmount{} }

func (proposedDamageAmount) proposedDamageValue(amount int) int     { return amount }
func (halfProposedDamageAmount) proposedDamageValue(amount int) int { return amount / 2 }
func (proposedDamageAmount) inspectDefinition() definitionNode {
	return definitionNode{Role: "proposed-damage-value", Text: "proposed damage amount"}
}
func (halfProposedDamageAmount) inspectDefinition() definitionNode {
	return definitionNode{Role: "proposed-damage-value", Text: "half proposed damage rounded down"}
}

// SourcePreventionOperation specifies prevention or its additional life gain.
// The operation keeps no binding or result outside its matching event.
type SourcePreventionOperation interface {
	sourcePreventionOperation()
}

type preventProposedDamage struct{ amount ProposedDamageValue }
type gainLifeFromPreventedDamage struct{ recipient PlayerRef }

// PreventProposedDamage prevents an amount read from the proposed source event.
func PreventProposedDamage(amount ProposedDamageValue) SourcePreventionOperation {
	return preventProposedDamage{amount: amount}
}

// GainLifeFromPreventedDamage gains life for a separately saved player.
// The amount is equal to this rule's actual prevention.
// It must follow one prevention operation.
func GainLifeFromPreventedDamage(recipient PlayerRef) SourcePreventionOperation {
	return gainLifeFromPreventedDamage{recipient: recipient}
}
func (preventProposedDamage) sourcePreventionOperation()       {}
func (gainLifeFromPreventedDamage) sourcePreventionOperation() {}

// PreventNextDamageFromSelectedSourceWithOperations saves one source event rule.
// Supply one prevention operation and an optional life gain operation after it.
// Matching simultaneous packets share an amount. No prevention keeps the rule.
func PreventNextDamageFromSelectedSourceWithOperations(source DamageSourceRef, recipient PlayerRef, filter DamageSourceFilter, operations ...SourcePreventionOperation) ResolvingEffect {
	effect := &exactSourcePreventionInstallation{source: source, recipient: recipient, filter: filter, continuePermanentSpell: true}
	if len(operations) < 1 || len(operations) > 2 {
		effect.operationProblem = "supply one prevention operation and at most one life gain operation"
		return effect
	}
	prevention, ok := operations[0].(preventProposedDamage)
	if !ok || nilDefinition(prevention.amount) {
		effect.operationProblem = "supply a prevention operation with a proposed damage value first"
		return effect
	}
	effect.amount = prevention.amount
	if len(operations) == 2 {
		gain, ok := operations[1].(gainLifeFromPreventedDamage)
		if !ok || nilDefinition(gain.recipient) {
			effect.operationProblem = "supply a life gain operation after prevention"
			return effect
		}
		effect.gainLifeRecipient = gain.recipient
	}
	return effect
}

func (r *sourcePreventionReplacement) preventProposedAmount(amount int) int {
	prevented := min(amount, max(0, r.amount.proposedDamageValue(amount)))
	r.consumed = r.consumed || prevented > 0
	return prevented
}

// These outcomes wait until the complete damage event has finished (CR 615.5).
type preventionLifeGain struct {
	playerID uuid.UUID
	amount   int
}

func (r *sourcePreventionReplacement) lifeGainForPrevention(amount int) preventionLifeGain {
	if r.gainLifePlayerID != uuid.Nil && amount > 0 {
		return preventionLifeGain{playerID: r.gainLifePlayerID, amount: amount}
	}
	return preventionLifeGain{}
}

func (outcome preventionLifeGain) apply(g *Game) {
	if outcome.amount > 0 {
		if player := g.GetPlayer(outcome.playerID); player != nil {
			g.gainLifeWithEvent(player, outcome.amount)
		}
	}
}

func applyPreventionLifeGains(g *Game, outcomes []preventionLifeGain) {
	for _, outcome := range outcomes {
		outcome.apply(g)
	}
}

func (r *sourcePreventionReplacement) preventAction(action Action) (Action, preventionLifeGain) {
	prevented := r.preventProposedAmount(damageActionAmount(action))
	return damageActionWithAmount(action, damageActionAmount(action)-prevented), r.lifeGainForPrevention(prevented)
}
