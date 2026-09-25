package mage

import "testing"

func TestComposableExileMetadata(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	controller := BindPlayer("controller")
	power := BindNumber("power")
	for _, effect := range []ResolvingEffect{
		Exile(target),
		Sequence(SnapshotPlayer(ControllerOf(target), controller), SnapshotNumber(PowerOf(target), power), Exile(target), GainLifeFor(FromPlayer(controller), FromNumber(power))),
	} {
		props := effect.Properties()
		if props.TargetPurposeOverride != AITargetExile || props.Outcome != OutcomeDetriment || props.Destroys {
			t.Fatalf("incorrect exile metadata: %+v", props)
		}
	}
}
