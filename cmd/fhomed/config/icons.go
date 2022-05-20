package config

type Icon string

const (
	Heating     Icon = "icon_cell_717_white"
	Lighting    Icon = "icon_cell_710_white"
	Temperature Icon = "icon_cell_706_white"
	Number      Icon = "icon_cell_707_white"
	Gate        Icon = "icon_cell_724_white"
	Unknown     Icon = "unknown"
)

func CreateIcon(icon string) Icon {
	switch icon {
	case string(Heating):
		return Heating
	case string(Lighting):
		return Lighting
	case string(Temperature):
		return Temperature
	case string(Number):
		return Number
	case string(Gate):
		return Gate
	default:
		return Unknown
	}
}
