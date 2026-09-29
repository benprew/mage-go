package core

//go:generate enumer -type=Layer,Duration -trimprefix=Layer -output=layer_enumer.go

// Layer represents a layer in the layer system.
type Layer int

const (
	LayerCopy    Layer = 1
	LayerControl Layer = 2
	LayerText    Layer = 3
	LayerType    Layer = 4
	LayerColor   Layer = 5
	LayerAbility Layer = 6
	LayerPT      Layer = 7
)

// SublayerPT represents the sublayers within Layer 7 (CR 613.4).
type SublayerPT int

const (
	SublayerPT_CDA    SublayerPT = 1 // 7a: Characteristic-defining abilities (CR 613.4a)
	SublayerPT_Set    SublayerPT = 2 // 7b: Setting base P/T (CR 613.4b)
	SublayerPT_Modify SublayerPT = 3 // 7c: Modify P/T without setting (CR 613.4c)
	SublayerPT_Switch SublayerPT = 4 // 7d: Switch P/T (CR 613.4d)
)

// Duration represents how long an effect lasts.
type Duration int

const (
	WhileOnBattlefield Duration = iota
	EndOfTurn
	EndOfCombat
	UntilYourNextTurn
	Indefinite // persists until target leaves battlefield (e.g. Sleight of Mind)
)

// AttachType distinguishes aura vs equipment attachment.
type AttachType int

const (
	AttachAura AttachType = iota
	AttachEquipment
)
