package mage

import (
	"fmt"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// counterSpellEffect counters a target spell on the stack.
type counterSpellEffect struct{}

// CounterSpell creates an effect that counters a target spell on the stack.
func CounterSpell() Effect {
	return &counterSpellEffect{}
}

func (e *counterSpellEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target spell to counter")
	}
	g.CounterSpellOnStack(targets[0])
	return nil
}

func (e *counterSpellEffect) Text() string { return "counter target spell" }
func (e *counterSpellEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// counterSpellIfColorEffect counters a target spell only if it matches a specific color.
type counterSpellIfColorEffect struct {
	color Color
}

// CounterSpellIfColor creates an effect that counters a target spell only if it matches the given color
// (e.g. Blue Elemental Blast, Red Elemental Blast).
func CounterSpellIfColor(c Color) Effect {
	return &counterSpellIfColorEffect{color: c}
}

func (e *counterSpellIfColorEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	// Find the spell on the stack
	obj := g.FindStackObject(targets[0])
	if obj == nil || obj.Card == nil {
		return nil
	}
	// Check if the spell has the required color
	for _, c := range obj.Card.ManaCost().Colors() {
		if c == e.color {
			g.CounterSpellOnStack(targets[0])
			return nil
		}
	}
	// Color doesn't match, spell is NOT countered
	return nil
}

func (e *counterSpellIfColorEffect) Text() string {
	return fmt.Sprintf("Counter target %s spell", e.color)
}
func (e *counterSpellIfColorEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// counterSpellIfXMeetsOrExceedsCMCEffect counters a spell only if X >= its CMC.
type counterSpellIfXMeetsOrExceedsCMCEffect struct{}

// CounterSpellIfXMeetsCMC creates an effect that counters a target spell only if X >= its mana value
// (e.g. Spell Blast).
func CounterSpellIfXMeetsCMC() Effect {
	return &counterSpellIfXMeetsOrExceedsCMCEffect{}
}

func (e *counterSpellIfXMeetsOrExceedsCMCEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	obj := g.FindStackObject(targets[0])
	if obj == nil || obj.Card == nil {
		return nil
	}
	cmc := obj.Card.ManaCost().CMC()
	if g.XValue() >= cmc {
		g.CounterSpellOnStack(targets[0])
	}
	return nil
}

func (e *counterSpellIfXMeetsOrExceedsCMCEffect) Text() string {
	return "Counter target spell if X >= its mana value"
}
func (e *counterSpellIfXMeetsOrExceedsCMCEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// powerSinkEffect counters a spell unless its controller pays X mana.
type powerSinkEffect struct{}

// PowerSinkEffect creates an effect that counters a spell unless its controller pays X mana,
// draining their pool either way (Power Sink).
func PowerSinkEffect() Effect {
	return &powerSinkEffect{}
}

func (e *powerSinkEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	obj := g.FindStackObject(targets[0])
	if obj == nil {
		return nil
	}
	// Check if the spell's controller can pay X mana
	spellController := g.GetPlayer(obj.Controller)
	if spellController == nil {
		return nil
	}
	// If they have enough mana in pool, they pay and spell resolves
	totalMana := spellController.ManaPool().TotalMana()
	if totalMana >= g.XValue() {
		// Opponent can pay - drain X mana but don't counter
		spellController.ManaPool().DrainGeneric(g.XValue())
		return nil
	}
	// Can't pay - counter the spell and drain all mana
	spellController.ManaPool().Clear()
	g.CounterSpellOnStack(targets[0])
	return nil
}

func (e *powerSinkEffect) Text() string {
	return "Counter target spell unless its controller pays {X}"
}
func (e *powerSinkEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// addManaEffect adds mana to the controller's pool.
type addManaEffect struct {
	color  Color
	amount int
}

// AddMana creates an effect that adds mana of the given color to the controller's pool.
func AddMana(color Color, amount int) Effect {
	return &addManaEffect{color: color, amount: amount}
}

func (e *addManaEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	p.ManaPool().Add(e.color, e.amount)
	return nil
}

func (e *addManaEffect) Text() string {
	return fmt.Sprintf("add %d %s mana", e.amount, e.color)
}
func (e *addManaEffect) Properties() EffectProperties { return EffectProperties{} }

// addAnyManaEffect adds mana of any one color to the controller's pool.
type addAnyManaEffect struct {
	amount int
}

// AddAnyMana creates an effect that adds mana of any one color (player chooses) to the controller's pool.
func AddAnyMana(amount int, _ Color) Effect {
	return &addAnyManaEffect{amount: amount}
}

func (e *addAnyManaEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	color := p.ChooseManaColor("add mana")
	p.ManaPool().Add(color, e.amount)
	return nil
}

func (e *addAnyManaEffect) Text() string {
	return fmt.Sprintf("add %d mana of any one color", e.amount)
}
func (e *addAnyManaEffect) Properties() EffectProperties { return EffectProperties{} }

// createTokenEffect creates a token creature on the battlefield.
type createTokenEffect struct {
	name      string
	power     int
	toughness int
	types     []CardType
	subTypes  []string
	keywords  []Keyword
	colors    []Color
}

// CreateToken creates an effect that puts a token creature onto the battlefield.
func CreateToken(name string, power, toughness int, types []CardType, subTypes []string, keywords ...Keyword) Effect {
	return &createTokenEffect{
		name:      name,
		power:     power,
		toughness: toughness,
		types:     types,
		subTypes:  subTypes,
		keywords:  keywords,
	}
}

// CreateColoredToken creates an effect that puts a colored token creature onto the battlefield.
func CreateColoredToken(name string, power, toughness int, colors []Color, types []CardType, subTypes []string, keywords ...Keyword) Effect {
	return &createTokenEffect{
		name:      name,
		power:     power,
		toughness: toughness,
		types:     types,
		subTypes:  subTypes,
		keywords:  keywords,
		colors:    colors,
	}
}

func (e *createTokenEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	token := NewToken(e.name, e.power, e.toughness, e.types, e.subTypes, e.keywords...)
	token.SetOwner(controller)
	if len(e.colors) > 0 {
		token.colorOverride = make([]Color, len(e.colors))
		copy(token.colorOverride, e.colors)
	}
	g.PutOnBattlefield(token, controller)
	return nil
}

func (e *createTokenEffect) Text() string {
	return fmt.Sprintf("create a %d/%d %s token", e.power, e.toughness, e.name)
}
func (e *createTokenEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, TokenPower: e.power, TokenToughness: e.toughness}
}

// cloneTargetEffect copies target permanent's characteristics onto the source
// card. If additionalTypes are provided, they are added after cloning (e.g.
// Copy Artifact adds TypeEnchantment). Target filtering is handled by the
// Target passed to NewTargetedSpell, not by this effect.
type cloneTargetEffect struct {
	additionalTypes []CardType
}

// CloneTarget creates an effect that copies a target permanent's characteristics onto the source.
// Additional types (e.g. TypeEnchantment for Copy Artifact) are added after cloning.
func CloneTarget(additionalTypes ...CardType) Effect {
	return &cloneTargetEffect{additionalTypes: additionalTypes}
}

func (e *cloneTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 || g.GetResolvingCard() == nil {
		return nil
	}
	target := g.FindPermanent(targets[0])
	if target == nil {
		return nil
	}
	g.GetResolvingCard().CloneFrom(target.Card)
	for _, t := range e.additionalTypes {
		g.GetResolvingCard().AddType(t)
	}
	return nil
}

func (e *cloneTargetEffect) Text() string {
	return "enters the battlefield as a copy of target permanent"
}
func (e *cloneTargetEffect) Properties() EffectProperties { return EffectProperties{} }

// copySpellOnStackEffect copies the top spell on the stack. Used by Fork.
type copySpellOnStackEffect struct{}

// CopySpellOnStack creates an effect that copies the target spell on the stack.
func CopySpellOnStack() Effect {
	return &copySpellOnStackEffect{}
}

func (e *copySpellOnStackEffect) Apply(g *Game, _, ctrl uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	// Find the target spell on the stack
	original := g.FindStackObject(targets[0])
	if original == nil {
		return nil
	}
	// Create a copy of the stack object
	cp := &StackObject{
		ID:         uuid.New(),
		Card:       original.Card,
		Controller: ctrl,
		SourceID:   original.SourceID,
		Effects:    make([]Effect, len(original.Effects)),
		Targets:    make([]uuid.UUID, len(original.Targets)),
		IsAbility:  original.IsAbility,
		XValue:     original.XValue,
	}
	copy(cp.Effects, original.Effects)
	copy(cp.Targets, original.Targets)
	g.PushStack(cp)
	return nil
}

func (e *copySpellOnStackEffect) Text() string {
	return "copy target instant or sorcery spell"
}
func (e *copySpellOnStackEffect) Properties() EffectProperties { return EffectProperties{} }

// attachToTargetEffect attaches the source (aura/equipment) to the target.
type attachToTargetEffect struct{}

// AttachToTarget creates an effect that attaches the source (aura or equipment) to the target permanent.
func AttachToTarget() Effect {
	return &attachToTargetEffect{}
}

func (e *attachToTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for attach")
	}
	g.Attach(sourceID, targets[0])
	return nil
}

func (e *attachToTargetEffect) Text() string                 { return "attach to target" }
func (e *attachToTargetEffect) Properties() EffectProperties { return EffectProperties{} }

// controlChangeTargetEffect gains control of a target permanent.
type controlChangeTargetEffect struct{}

// ControlChangeTarget creates an effect that gives the controller permanent control of a target (e.g. Control Magic).
func ControlChangeTarget() Effect {
	return &controlChangeTargetEffect{}
}

func (e *controlChangeTargetEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target for control change")
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	perm.Controller = controller
	return nil
}

func (e *controlChangeTargetEffect) Text() string { return "gain control of target permanent" }
func (e *controlChangeTargetEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// extraTurnEffect gives the controller an extra turn.
type extraTurnEffect struct{}

// ExtraTurn creates an effect that gives the controller an extra turn (e.g. Time Walk).
func ExtraTurn() Effect {
	return &extraTurnEffect{}
}

func (e *extraTurnEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	g.GrantExtraTurn(controller)
	return nil
}

func (e *extraTurnEffect) Text() string { return "take an extra turn after this one" }
func (e *extraTurnEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

// changeColorEffect changes a target permanent's color.
type changeColorEffect struct {
	color Color
}

// ChangeColorEffect creates an effect that changes a target permanent's color.
func ChangeColorEffect(color Color) Effect {
	return &changeColorEffect{color: color}
}

func (e *changeColorEffect) Apply(g *Game, sourceID, _ uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	perm := g.FindPermanent(targets[0])
	if perm == nil {
		return nil
	}
	// Register as a continuous effect so the color change persists
	ce := ColorOverride(perm.ID(), e.color)
	ce.SetSourceID(sourceID)
	g.AddContinuousEffect(ce)
	return nil
}

func (e *changeColorEffect) Text() string {
	return fmt.Sprintf("Target permanent becomes %s", e.color)
}
func (e *changeColorEffect) Properties() EffectProperties { return EffectProperties{} }

// counterUnlessPayEffect counters a target spell unless its controller pays a cost.
type counterUnlessPayEffect struct {
	cost string
}

// CounterUnlessPay creates an effect that counters a target spell unless its
// controller pays the specified mana cost (e.g. Force Spike, Mana Leak).
func CounterUnlessPay(cost string) Effect {
	return &counterUnlessPayEffect{cost: cost}
}

func (e *counterUnlessPayEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	if len(targets) == 0 {
		return nil
	}
	obj := g.FindStackObject(targets[0])
	if obj == nil {
		return nil
	}
	spellController := g.GetPlayer(obj.Controller)
	if spellController == nil {
		return nil
	}
	if g.TryPayCostFromLands(obj.Controller, e.cost) {
		return nil // paid, spell resolves
	}
	g.CounterSpellOnStack(targets[0])
	return nil
}

func (e *counterUnlessPayEffect) Text() string {
	return fmt.Sprintf("Counter target spell unless its controller pays %s", e.cost)
}
func (e *counterUnlessPayEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

// forcefieldEffect activates a Forcefield shield on the controller for this turn.
type forcefieldEffect struct{}

// ForcefieldEffect creates an effect that reduces all unblocked combat damage to the controller
// to 1 for this turn (Forcefield).
func ForcefieldEffect() Effect {
	return &forcefieldEffect{}
}

func (e *forcefieldEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	g.AddForcefieldShield(controller)
	return nil
}

func (e *forcefieldEffect) Text() string {
	return "prevent all but 1 combat damage from each unblocked creature this turn"
}
func (e *forcefieldEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
