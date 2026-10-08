package mage

import (
	"fmt"
	"reflect"

	"github.com/google/uuid"
)

// CardDefinitionPlan retains a builder for a printed card definition.
// The builder must be pure and must create independent declarations each time.
type CardDefinitionPlan struct {
	build          CardFactory
	seed           map[any]bool
	name           string
	copiableValues *CopiableValues
}

// NewCardDefinitionPlan checks a builder before the plan can be used.
// It checks two builds to detect declarations shared by different instances.
func NewCardDefinitionPlan(build CardFactory) (*CardDefinitionPlan, error) {
	first, err := buildCardDefinition(build)
	if err != nil {
		return nil, err
	}
	second, err := buildCardDefinition(build)
	if err != nil {
		return nil, err
	}
	if first.Name() != second.Name() {
		return nil, fmt.Errorf("card definition builder must keep the printed name")
	}
	printed := printedCopiableCharacteristics(first)
	if !printed.matches(second) {
		return nil, fmt.Errorf("card definition builder must keep printed characteristics")
	}
	if err := independentCardDefinitions(first, second); err != nil {
		return nil, err
	}
	seed := cardDeclarationIdentities(first)
	for identity := range cardDeclarationIdentities(second) {
		seed[identity] = true
	}
	plan := &CardDefinitionPlan{build: build, seed: seed, name: first.Name()}
	plan.copiableValues = &CopiableValues{definition: plan, printed: printed}
	return plan, nil
}

// Build creates a new printed definition with no owner or saved game choices.
// Use CopiableValues for prototypes made from saved copy information.
func (p *CardDefinitionPlan) Build() (Card, error) {
	if p == nil {
		return nil, fmt.Errorf("card definition requires a plan")
	}
	card, err := buildCardDefinition(p.build)
	if err != nil {
		return nil, err
	}
	if !p.copiableValues.printed.matches(card) {
		return nil, fmt.Errorf("card definition builder must keep printed characteristics")
	}
	for identity := range cardDeclarationIdentities(card) {
		if p.seed[identity] {
			return nil, fmt.Errorf("%s: builder reused a card identity or declaration", p.name)
		}
	}
	card.definitionPlan = p
	card.copiableValues = p.copiableValues
	return card, nil
}

// WithCardDefinitionPlan retains the plan used to construct this definition.
func WithCardDefinitionPlan(plan *CardDefinitionPlan) CardOption {
	return func(card *BaseCard) { card.definitionPlan = plan }
}

// RebuildCardDefinition creates independent declarations from a retained plan.
// It does not use a registry or change the source card.
func RebuildCardDefinition(card Card) (Card, error) {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.definitionPlan == nil {
		return nil, fmt.Errorf("card definition has no retained plan")
	}
	return base.definitionPlan.Build()
}

func buildCardDefinition(build CardFactory) (card *BaseCard, err error) {
	defer func() {
		if problem := recover(); problem != nil {
			card, err = nil, fmt.Errorf("card definition factory: %v", problem)
		}
	}()
	if build == nil {
		return nil, fmt.Errorf("card definition requires a factory")
	}
	built := build()
	if nilDefinition(built) {
		return nil, fmt.Errorf("card definition factory requires a card")
	}
	card, ok := built.(*BaseCard)
	if !ok {
		return nil, fmt.Errorf("card definition builder requires a BaseCard")
	}
	if card.Owner() != uuid.Nil {
		return nil, fmt.Errorf("card definition builder must not retain an owner")
	}
	if card.ID() == uuid.Nil {
		return nil, fmt.Errorf("card definition builder requires a card identity")
	}
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		return nil, err
	}
	if err := visitCardDeclarations(card, func(value any) error {
		if ability, ok := value.(Ability); ok && (ability.Source() != uuid.Nil || ability.Controller() != uuid.Nil) {
			return fmt.Errorf("card definition builder must not retain ability source or controller")
		}
		if target, ok := value.(Target); ok && target.IsChosen() {
			return fmt.Errorf("card definition builder must supply targets without saved choices")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return card, nil
}

type definitionCardIdentity struct{ id uuid.UUID }
type definitionAbilityIdentity struct{ id uuid.UUID }

func independentCardDefinitions(first, second Card) error {
	if first.ID() == second.ID() {
		return fmt.Errorf("card definition builder reused a card identity")
	}
	identities := cardDeclarationIdentities(first)
	for identity := range cardDeclarationIdentities(second) {
		if identities[identity] {
			return fmt.Errorf("card definition builder reused a declaration: %T", identity)
		}
	}
	return nil
}

func cardDeclarationIdentities(card Card) map[any]bool {
	identities := map[any]bool{definitionCardIdentity{card.ID()}: true}
	_ = visitCardDeclarations(card, func(value any) error {
		if ability, ok := value.(Ability); ok {
			identities[definitionAbilityIdentity{ability.AbilityID()}] = true
		}
		if target, ok := value.(Target); ok && reflect.ValueOf(target).Comparable() {
			identities[target] = true
		}
		if cost, ok := value.(Cost); ok && reflect.ValueOf(cost).Kind() == reflect.Pointer && reflect.TypeOf(cost).Elem().Size() > 0 {
			identities[cost] = true
		}
		node := describeDefinition(value)
		if node.targetDeclaration != nil {
			identities[node.targetDeclaration] = true
		}
		if node.targetRead != nil {
			identities[node.targetRead] = true
		}
		for _, binding := range append([]savedBinding{node.bindingDeclaration, node.bindingRead, node.bindingWrite}, node.bindingWrites...) {
			if !nilDefinition(binding) {
				identities[binding] = true
			}
		}
		return nil
	})
	return identities
}

func visitCardDeclarations(card Card, inspect func(any) error) error {
	seen := make(map[any]bool)
	var visit func(any) error
	visit = func(value any) error {
		if nilDefinition(value) {
			return fmt.Errorf("card definition requires non-nil declarations")
		}
		if reflect.ValueOf(value).Comparable() {
			if seen[value] {
				return nil
			}
			seen[value] = true
		}
		if err := inspect(value); err != nil {
			return err
		}
		if target, ok := value.(Target); ok {
			if spec := UnwrapTarget(target); spec != target {
				if err := visit(spec); err != nil {
					return err
				}
			}
			if bounded, ok := target.(*countBoundTarget); ok {
				if err := visit(bounded.inner); err != nil {
					return err
				}
			}
		}
		if action, ok := value.(*ActionDefinition); ok {
			for _, cost := range action.Costs() {
				if err := visit(cost); err != nil {
					return err
				}
			}
			for _, target := range action.Targets() {
				if err := visit(target); err != nil {
					return err
				}
			}
		}
		for _, child := range describeDefinition(value).children {
			if err := visit(child.value); err != nil {
				return err
			}
		}
		return nil
	}
	for _, ability := range card.Abilities() {
		if err := visit(ability); err != nil {
			return err
		}
	}
	for _, target := range card.CastTargets() {
		if err := visit(target); err != nil {
			return err
		}
	}
	if base, ok := card.(*BaseCard); ok {
		for _, cost := range base.AdditionalCosts() {
			if err := visit(cost); err != nil {
				return err
			}
		}
	}
	return nil
}
