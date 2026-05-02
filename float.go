package conditions

import "math"

// Epsilon for float comparison. Set before first use via SetDefaultEpsilon.
var defaultEpsilon = 1e-6

// SetDefaultEpsilon sets the epsilon used for floating-point equality comparisons.
// Call this before any concurrent Evaluate calls if you need a non-default value.
func SetDefaultEpsilon(ep float64) {
	defaultEpsilon = ep
}

// float64Equal compares two floats with epsilon tolerance.
func float64Equal(a, b float64) bool {
	if a == b {
		return true
	}
	diff := math.Abs(a - b)
	if diff > defaultEpsilon {
		return false
	}
	// Near-zero: use absolute tolerance scaled by smallest representable float
	if a == 0 || b == 0 {
		return diff < defaultEpsilon*math.SmallestNonzeroFloat32
	}
	// Relative error check for well-separated values
	return diff/math.Max(math.Abs(a), math.Abs(b)) < defaultEpsilon
}
