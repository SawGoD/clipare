package ui

const (
	devicesTitle    = "Устройства"
	addDeviceTitle  = "Добавить устройство"
	additionalTitle = "Дополнительно"
	advancedTitle   = "Расширенные параметры"
)

// Platform-independent visibility policy; adapters own geometry and materials.
type settingsPresentation struct {
	EmptyDevices, ShowPeerList, ShowRemove, ShowCompactAdd bool
	ShowPreferences                                        bool
}

func presentSettings(peerCount int, expanded bool) settingsPresentation {
	hasPeers := peerCount > 0
	return settingsPresentation{EmptyDevices: !hasPeers, ShowPeerList: hasPeers, ShowRemove: hasPeers, ShowCompactAdd: hasPeers, ShowPreferences: expanded}
}
