package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ChooseNumberAsEnters saves the controller's number before battlefield entry.
// Return requires a new choice. Phasing keeps the saved number.
func ChooseNumberAsEnters(minimum, maximum int) AsEntersBattlefieldAbility {
	return &numericEntryChoice{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}, minimum: minimum, maximum: maximum}
}

type numericEntryChoice struct {
	BaseAbility
	minimum, maximum int
}

func (a *numericEntryChoice) inspectDefinition() definitionNode {
	n := definitionNode{Role: "entry-number-choice", Text: "choose an entry number", battlefieldSource: true}
	if a.minimum < 0 || a.maximum < a.minimum {
		n.problem = "number bounds require zero or greater and minimum at most maximum"
	}
	return n
}
func (a *numericEntryChoice) OnEnter(g *Game, p *Permanent) bool {
	choice := a.minimum
	if player := g.GetPlayer(p.ControllerID()); player != nil {
		choice = validNumericChoice(player, a.minimum, a.maximum)
	}
	p.ChosenNumber, p.HasChosenNumber = choice, true
	return true
}
func validNumericChoice(player Player, minimum, maximum int) int {
	answer := player.ChooseNumber(minimum, maximum, fmt.Sprintf("Choose a number from %d through %d", minimum, maximum))
	if answer < minimum || answer > maximum {
		return minimum
	}
	return answer
}

// ChosenNumberOf reads the exact visit's current or last-known number.
// A missing choice supplies zero and does not make a choice.
func ChosenNumberOf(subject PermanentRef) ResolutionValue { return chosenNumberValue{subject} }

type chosenNumberValue struct{ subject PermanentRef }

func (chosenNumberValue) Text() string { return "the chosen number" }
func (v chosenNumberValue) inspectDefinition() definitionNode {
	return permanentInformationNode(v.subject, "permanent-value", v.Text())
}
func (v chosenNumberValue) resolveValue(ctx *EffectContext) int {
	p, lki := readPermanentInformation(v.subject, ctx)
	if lki != nil {
		p = lki.Snapshot
	}
	if p == nil || !p.HasChosenNumber {
		return 0
	}
	return p.ChosenNumber
}

// MayChooseNumber offers a new number to the resolving ability's controller.
// The saved subject must remain present through both choices.
func MayChooseNumber(subject PermanentRef, minimum, maximum int) ResolvingEffect {
	return &numericUpdateChoice{subject: subject, minimum: minimum, maximum: maximum}
}

type numericUpdateChoice struct {
	subject          PermanentRef
	minimum, maximum int
}

func (*numericUpdateChoice) Text() string                 { return "may choose a new number" }
func (*numericUpdateChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *numericUpdateChoice) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
	if e.minimum < 0 || e.maximum < e.minimum {
		n.problem = "number bounds require zero or greater and minimum at most maximum"
	}
	return n
}
func (e *numericUpdateChoice) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	identity := capturedPermanent{p.ID(), p.incarnationID}
	player := ctx.Game.GetPlayer(ctx.Controller)
	if player == nil || !player.ChooseMayAbility(e.Text()) {
		return nil
	}
	live := ctx.Game.FindPermanent(identity.id)
	if live == nil || live.PhasedOut || live.incarnationID != identity.incarnation {
		return nil
	}
	choice := validNumericChoice(player, e.minimum, e.maximum)
	live = ctx.Game.FindPermanent(identity.id)
	if live == nil || live.PhasedOut || live.incarnationID != identity.incarnation {
		return nil
	}
	current := ctx.Game.MutablePermanent(identity.id)
	current.ChosenNumber, current.HasChosenNumber = choice, true
	ctx.Game.ApplyContinuousEffects()
	return nil
}
