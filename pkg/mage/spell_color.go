package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// SpellOrPermanentTargetBinding declares one target across two public zones.
type SpellOrPermanentTargetBinding struct{ *targetDeclaration }

// DeclareSpellOrPermanentTarget permits copies to choose either target category.
func DeclareSpellOrPermanentTarget(name string) *SpellOrPermanentTargetBinding {
	return &SpellOrPermanentTargetBinding{&targetDeclaration{name: name, spec: &typedSpellOrPermanentTarget{BaseTarget: BaseTarget{min: 1, max: 1}}}}
}
func (b *SpellOrPermanentTargetBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "spell-or-permanent-ref")
}

// SetObjectColor keeps a color change on the selected object's original identity.
func SetObjectColor(object *SpellOrPermanentTargetBinding, color core.Color) ResolvingEffect {
	return &objectTargetColorEffect{object: object, color: color}
}

type objectTargetColorEffect struct {
	object *SpellOrPermanentTargetBinding
	color  core.Color
}

func (*objectTargetColorEffect) Text() string                 { return "change target spell or permanent's color" }
func (*objectTargetColorEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *objectTargetColorEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"object", e.object}}}
	if e.color < core.White || e.color > core.Green {
		n.problem = "supply one actual color"
	}
	return n
}
func (e *objectTargetColorEffect) Apply(ctx *EffectContext) error {
	value, ok := firstTargetValue(ctx.targetBindings[e.object.targetDeclaration])
	if !ok {
		return nil
	}
	if value.incarnation != uuid.Nil {
		p := ctx.Game.FindPermanent(value.id)
		if p == nil || p.PhasedOut || p.incarnationID != value.incarnation {
			return nil
		}
	} else {
		spell := ctx.Game.stack.FindByID(value.id)
		if spell == nil || spell.IsAbility || spell.Card == nil {
			return nil
		}
	}
	ctx.Game.ChangeObjectColor(value.id, e.color)
	return nil
}

type typedSpellOrPermanentTarget struct{ BaseTarget }

func (*typedSpellOrPermanentTarget) Possible(controller uuid.UUID, source Card, g *Game) []uuid.UUID {
	permanents := TargetPermanent().Possible(controller, source, g)
	spells := (&typedSpellTarget{}).Possible(controller, source, g)
	return uniqueTargetCandidates(append(permanents, spells...))
}
func (*typedSpellOrPermanentTarget) Choose(uuid.UUID, Card, *Game, []uuid.UUID) error { return nil }
