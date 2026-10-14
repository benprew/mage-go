package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileSpellExile(subject, modifiersNode Node, ctx scope) (effectFactory, error) {
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref := subject.Args[0]
	if ref.Kind != Symbol || ref.Name != "ThisSpell" || !ctx.spellTargets || ctx.permanent || !ctx.resolving || ctx.sourceZone != nil || ctx.delayedStep || ctx.drawReplacementBody || ctx.delayedDeath != nil {
		return nil, problem(ref, "spell exile requires ThisSpell in a resolving spell body")
	}
	modifiers, err := list(modifiersNode)
	if err != nil {
		return nil, err
	}
	count := 0
	counter := core.Scream
	if len(modifiers) != 0 {
		if len(modifiers) != 1 || modifiers[0].Name != "ExilesWithNumberCounters" {
			return nil, unsupported(modifiersNode, "spell exile modifiers")
		}
		modifier := modifiers[0]
		if err := arity(modifier, 2); err != nil {
			return nil, err
		}
		value := modifier.Args[0]
		if value.Name != "Integer" {
			return nil, unsupported(value, "spell exile counter amount")
		}
		if err := arity(value, 1); err != nil {
			return nil, err
		}
		amount := value.Args[0]
		if amount.Kind != IntegerNode || amount.Integer < 0 {
			return nil, problem(amount, "spell exile requires a nonnegative integer amount")
		}
		count = amount.Integer
		counter, err = compilePermanentCounterType(modifier.Args[1])
		if err != nil {
			return nil, err
		}
	}
	return func(instanceBindings) mage.Effect { return mage.ExileSpellWithCounters(mage.ThisSpell, counter, count) }, nil
}
