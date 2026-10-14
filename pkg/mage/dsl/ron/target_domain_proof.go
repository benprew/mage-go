package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileTargetSubtypeDomain(n Node) (mage.PermanentTargetDomain, error) {
	switch n.Name {
	case "And", "Or":
		if err := arity(n, 1); err != nil {
			return mage.PermanentTargetDomain{}, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentTargetDomain{}, err
		}
		if len(parts) == 0 {
			return mage.PermanentTargetDomain{}, problem(n, "target domain requires operands")
		}
		domains := make([]mage.PermanentTargetDomain, len(parts))
		for i, part := range parts {
			domains[i], err = compileTargetSubtypeDomain(part)
			if err != nil {
				return mage.PermanentTargetDomain{}, err
			}
		}
		if n.Name == "And" {
			return mage.AllPermanentDomains(domains...), nil
		}
		return mage.AnyPermanentDomain(domains...), nil
	case "IsCreatureType", "IsNonCreatureType":
		if err := arity(n, 1); err != nil {
			return mage.PermanentTargetDomain{}, err
		}
		subtype, err := symbol(n.Args[0])
		if err != nil {
			return mage.PermanentTargetDomain{}, err
		}
		if n.Name == "IsNonCreatureType" {
			return mage.NonSubtypeDomain(subtypeName(subtype)), nil
		}
		return mage.SubtypeDomain(subtypeName(subtype)), nil
	}
	return mage.UnknownPermanentDomain(), nil
}
