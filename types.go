package conditions

// DataType represents the primitive data types available.
type DataType string

const (
	Unknown = DataType("")
	Number  = DataType("number")
	Boolean = DataType("boolean")
	String  = DataType("string")
)

// InspectDataType returns the data type of a given value.
func InspectDataType(v interface{}) DataType {
	switch v.(type) {
	case float64:
		return Number
	case bool:
		return Boolean
	case string:
		return String
	default:
		return Unknown
	}
}
