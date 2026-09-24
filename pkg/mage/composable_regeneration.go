package mage

// Regenerate protects the subject from its next destruction this turn.
func Regenerate(subject PermanentRef) ResolvingEffect {
	return &subjectAction{permanent: subject, effect: &regenerateTargetEffect{}, verb: "regenerate"}
}
