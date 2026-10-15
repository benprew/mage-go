package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileLinkedEnchantGrant(n Node, ctx scope, subject permanentReferencePlan, parts []Node) (effectFactory, error) {
	if !ctx.resolving || ctx.drawReplacementBody || !ctx.graveyardEnchant || subject.kind != permanentSource {
		return nil, problem(n, "linked enchant change requires the resolving graveyard Aura source")
	}
	if len(parts) != 2 || parts[0].Name != "LosesAbility" || parts[1].Name != "AddAbility" {
		return nil, problem(n, "supply graveyard enchant removal followed by one linked enchant grant")
	}
	if _, err := compileCreatureEnchantRemoval(parts[0], subject); err != nil {
		return nil, err
	}
	if err := arity(parts[1], 1); err != nil {
		return nil, err
	}
	abilities, err := list(parts[1].Args[0])
	if err != nil {
		return nil, err
	}
	if len(abilities) != 1 || abilities[0].Name != "EnchantPermanent" {
		return nil, unsupported(parts[1], "linked enchant grant")
	}
	enchant := abilities[0]
	if err := arity(enchant, 1); err != nil {
		return nil, err
	}
	filter := enchant.Args[0]
	if filter.Name != "And" {
		return nil, unsupported(filter, "linked creature enchant filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	clauses, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	if len(clauses) != 2 {
		return nil, problem(filter, "supply creature type and exact entry source")
	}
	creature := clauses[0]
	if creature.Name != "IsCardtype" {
		return nil, unsupported(creature, "linked enchant creature type")
	}
	if err := arity(creature, 1); err != nil {
		return nil, err
	}
	if creature.Args[0].Kind != Symbol || creature.Args[0].Name != "Creature" {
		return nil, unsupported(creature, "linked enchant creature type")
	}
	if clauses[1].Name != "PutOntoBattlefieldByPermanent" {
		return nil, unsupported(clauses[1], "linked enchant entry relation")
	}
	if _, err := compileEntryHistory(clauses[1], ctx); err != nil {
		return nil, err
	}
	hosts, err := compileContextualCollection(filter, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.Until(core.Indefinite,
			mage.RemoveGraveyardCreatureEnchant(subject.build(bindings)),
			mage.GrantPermanentEnchant(subject.build(bindings), hosts(bindings)),
		)
	}, nil
}
