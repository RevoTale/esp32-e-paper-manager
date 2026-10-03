package provision

import "testing"

func validConfig() Config {
	return Config{
		SSID: "example-wpa3", Passphrase: "correct horse battery staple",
		Manager: "tcp://manager.example:9757", Timezone: "Europe/Kyiv",
		DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}, Auth: AuthWPA3SAE,
	}
}

func TestConfigAcceptsDNSIPv4AndIPv6Managers(t *testing.T) {
	for _, manager := range []string{
		"tcp://manager.example:9757", "tcp://192.0.2.1:443", "tcp://[2001:db8::1]:443",
	} {
		config := validConfig()
		config.Manager = manager
		if err := config.Validate(); err != nil {
			t.Fatalf("Validate(%q) error=%v", manager, err)
		}
	}
}

func TestConfigRejectsUnsafeOrIncompleteValues(t *testing.T) {
	tests := []func(*Config){
		func(c *Config) { c.Auth = 0 }, func(c *Config) { c.SSID = "" },
		func(c *Config) { c.Passphrase = "short" }, func(c *Config) { c.Manager = "http://example.com" },
		func(c *Config) { c.Manager = "tcp://user@example.com:1" }, func(c *Config) { c.Manager = "tcp://-bad.example:1" },
		func(c *Config) { c.Timezone = "../UTC" }, func(c *Config) { c.DeviceID = [16]byte{} },
		func(c *Config) { c.DeviceKey = [32]byte{} },
	}
	for index, mutate := range tests {
		config := validConfig()
		mutate(&config)
		if err := config.Validate(); err == nil {
			t.Fatalf("case %d accepted", index)
		}
	}
}

func TestConfigRejectsBoundaryAndCharacterViolations(t *testing.T) {
	tests := []func(*Config){
		func(c *Config) { c.SSID = "bad\x00ssid" },
		func(c *Config) { c.SSID = string(make([]byte, 33)) },
		func(c *Config) { c.Passphrase = string(make([]byte, 64)) },
		func(c *Config) { c.Manager = "tcp://bad_label.example:1" },
		func(c *Config) { c.Manager = "tcp://example..com:1" },
		func(c *Config) { c.Manager = "tcp://example.com:0" },
		func(c *Config) { c.Timezone = "Europe//Kyiv" },
		func(c *Config) { c.Timezone = "Europe/Kyiv?" },
	}
	for index, mutate := range tests {
		config := validConfig()
		mutate(&config)
		if err := config.Validate(); err == nil {
			t.Fatalf("case %d accepted", index)
		}
	}
}
