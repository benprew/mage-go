package core

// Keyword is a type alias for Attr.
// All keyword constants (Flying, Trample, etc.) are defined in attr.go.
// This alias exists for backward compatibility: existing code using
// Keyword as a type, or passing Keyword values, continues to compile unchanged.
//
// Deprecated: Use Attr directly. Keyword = Attr.
type Keyword = Attr
