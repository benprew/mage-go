package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.Sequence(dsl.OnWin(dsl.Draw(dsl.You, dsl.Fixed(1)))) // ERROR ResolvingEffect
