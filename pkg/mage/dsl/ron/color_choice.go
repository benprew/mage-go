package ron

import (
	"strings"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileColorChoice(choice, continuation Node, ctx scope) (effectFactory, error) {
	plural := choice.Name == "ChooseColors"
	if plural {
		if _, err := symbol(choice); err != nil {
			return nil, err
		}
	} else {
		if err := arity(choice, 1); err != nil {
			return nil, err
		}
		if choice.Args[0].Kind != Symbol || choice.Args[0].Name != "AnyColor" {
			return nil, unsupported(choice.Args[0], "color choice filter")
		}
	}
	if continuation.Name != "CreatePermanentLayerEffect" {
		return nil, unsupported(continuation, "color choice continuation")
	}
	if err := arity(continuation, 2); err != nil {
		return nil, err
	}
	parts, err := list(continuation.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) != 1 || parts[0].Name != "SetColor" {
		return nil, problem(continuation, "color choice requires one color-setting continuation")
	}
	if err := arity(parts[0], 1); err != nil {
		return nil, err
	}
	result := "TheChosenColor"
	if plural {
		result = "TheChosenColors"
	}
	if ref := parts[0].Args[0]; ref.Kind != Symbol || ref.Name != result {
		return nil, problem(ref, "unbound or incompatible color choice result")
	}
	subject, err := compilePermanentRef(continuation.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	colors := []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green}
	var plans []effectFactory
	var labels []string
	for mask := range 32 {
		var chosen []core.Color
		var names []string
		for i, color := range colors {
			if mask&(1<<i) != 0 {
				chosen = append(chosen, color)
				names = append(names, color.String())
			}
		}
		if !plural && len(chosen) != 1 {
			continue
		}
		// Fixed branches keep each choice inside its resolution frame.
		plans = append(plans, func(bindings instanceBindings) mage.Effect {
			return mage.Until(core.Indefinite, mage.SetColors(subject.build(bindings), chosen...))
		})
		label := strings.Join(names, ", ")
		if len(names) == 0 {
			label = "colorless"
		}
		labels = append(labels, label)
	}
	return buildResolutionChoice(plans, labels), nil
}
