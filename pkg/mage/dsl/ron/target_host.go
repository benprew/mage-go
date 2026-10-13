package ron

import "slices"

func targetRequiresAura(n Node) bool {
	if n.Name == "IsEnchantmentType" {
		return n.Kind == Tuple && len(n.Args) == 1 && n.Args[0].Kind == Symbol && n.Args[0].Name == "Aura"
	}
	if n.Name == "And" && n.Kind == Tuple && len(n.Args) == 1 && n.Args[0].Kind == List {
		if slices.ContainsFunc(n.Args[0].Args, targetRequiresAura) {
			return true
		}
	}
	return false
}
