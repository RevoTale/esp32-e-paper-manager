package provision

import "encoding/binary"

func encodeConfig(destination []byte, config Config) {
	destination[16] = byte(config.Auth)
	destination[17], destination[18] = byte(len(config.SSID)), byte(len(config.Passphrase))
	destination[19] = byte(len(config.Timezone))
	binary.BigEndian.PutUint16(destination[20:22], uint16(len(config.Manager)))
	copy(destination[24:40], config.DeviceID[:])
	copy(destination[40:72], config.DeviceKey[:])
	copy(destination[72:327], config.Manager)
	copy(destination[327:359], config.SSID)
	copy(destination[359:422], config.Passphrase)
	copy(destination[422:486], config.Timezone)
}

func decodeConfig(source []byte) (Config, bool) {
	return (Codec{}).decodeConfig(source)
}

func (codec Codec) decodeConfig(source []byte) (Config, bool) {
	managerLength := int(binary.BigEndian.Uint16(source[20:22]))
	ssidLength, passLength, timezoneLength := int(source[17]), int(source[18]), int(source[19])
	if managerLength > MaxManager || ssidLength > MaxSSID || passLength > MaxPassphrase || timezoneLength > MaxTimezone {
		return Config{}, false
	}
	config := Config{Auth: AuthMode(source[16]), Manager: string(source[72 : 72+managerLength]),
		SSID: string(source[327 : 327+ssidLength]), Passphrase: string(source[359 : 359+passLength]),
		Timezone: string(source[422 : 422+timezoneLength])}
	copy(config.DeviceID[:], source[24:40])
	copy(config.DeviceKey[:], source[40:72])
	return config, config.ValidateFor(codec) == nil
}
