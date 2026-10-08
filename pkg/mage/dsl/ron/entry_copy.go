package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileEntryCopy(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	filter, err := compileFilter(n.Args[0])
	if err != nil {
		return nil, err
	}
	if n.Args[1].Kind != Symbol || n.Args[1].Name != "NoCopyEffects" {
		return nil, unsupported(n.Args[1], "entry copy modifications")
	}
	return func() mage.CardOption { return mage.WithAbility(mage.MayCopyAsEnters(filter)) }, nil
}
