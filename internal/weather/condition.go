package weather

// Condition is the normalized weather condition displayed by the app.
type Condition string

const (
	// Clear indicates cloud-free conditions.
	Clear Condition = "clear"
	// MainlyClear indicates mostly cloud-free conditions.
	MainlyClear Condition = "mainly clear"
	// PartlyCloudy indicates mixed sun and cloud.
	PartlyCloudy Condition = "partly cloudy"
	// Overcast indicates full cloud cover.
	Overcast Condition = "overcast"
	// Fog indicates fog or rime fog.
	Fog Condition = "fog"
	// Drizzle indicates light liquid precipitation.
	Drizzle Condition = "drizzle"
	// FreezingRain indicates rain that may freeze on contact.
	FreezingRain Condition = "freezing rain"
	// Rain indicates steady liquid precipitation.
	Rain Condition = "rain"
	// Snow indicates snowfall or snow grains.
	Snow Condition = "snow"
	// Showers indicates intermittent rain showers.
	Showers Condition = "showers"
	// Thunderstorm indicates thunderstorm conditions.
	Thunderstorm Condition = "thunderstorm"
	// Unknown indicates an unmapped weather code.
	Unknown Condition = "unknown"
)

// ConditionFor maps an Open-Meteo WMO weather code to a normalized condition.
func ConditionFor(code int, _ bool) Condition {
	switch code {
	case 0:
		return Clear
	case 1:
		return MainlyClear
	case 2:
		return PartlyCloudy
	case 3:
		return Overcast
	case 45, 48:
		return Fog
	case 51, 53, 55, 56, 57:
		return Drizzle
	case 61, 63, 65:
		return Rain
	case 66, 67:
		return FreezingRain
	case 71, 73, 75, 77:
		return Snow
	case 80, 81, 82:
		return Showers
	case 85, 86:
		return Snow
	case 95, 96, 99:
		return Thunderstorm
	default:
		return Unknown
	}
}
