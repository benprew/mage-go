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
