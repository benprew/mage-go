package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCollectionAttackRule(n, operation Node) (optionFactory, error) {
	if operation.Kind != Symbol {
		return nil, unsupported(operation, "collection attack operation")
	}
	subjects, err := compileStaticPermanentFilter(n.Args[0])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption { return mage.WithAbility(mage.Static(mage.CantAttackEach(subjects.build()))) }, nil
}
