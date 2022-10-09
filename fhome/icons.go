package fhome

type Icon string

const (
	IconHeating     Icon = "icon_cell_717_white"
	IconLighting    Icon = "icon_cell_710_white"
	IconTemperature Icon = "icon_cell_706_white"
	IconNumber      Icon = "icon_cell_707_white"
	IconGate        Icon = "icon_cell_724_white"
	IconUnknown     Icon = "unknown"
)

func CreateIcon(icon string) Icon {
	switch icon {
	case string(IconHeating):
		return IconHeating
	case string(IconLighting):
		return IconLighting
	case string(IconTemperature):
		return IconTemperature
	case string(IconNumber):
		return IconNumber
	case string(IconGate):
		return IconGate
	default:
		return IconUnknown
	}
}
