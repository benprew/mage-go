package ron

import (
	"maps"
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileCreatedTokenAssociation(rules []Node, permanent bool) (*mage.CreatedTokenLink, error) {
	producers := 0
	var consumer *Node
	var walk func(Node)
	walk = func(n Node) {
		if n.Name == "TheCreatedTokens" {
			declaration := n
			consumer = &declaration
		}
		if n.Name == "CreateTokens" {
			producers++
			return
		}
		for _, arg := range n.Args {
			walk(arg)
		}
		for _, key := range slices.Sorted(maps.Keys(n.Fields)) {
			walk(n.Fields[key])
		}
	}
	for _, rule := range rules {
		walk(rule)
	}
	if consumer == nil {
		return nil, nil
	}
	if consumer.Kind != Symbol || !permanent || producers != 1 {
		return nil, problem(*consumer, "created token link requires a symbol and one unambiguous permanent-source producer")
	}
	return mage.NewCreatedTokenLink(), nil
}
