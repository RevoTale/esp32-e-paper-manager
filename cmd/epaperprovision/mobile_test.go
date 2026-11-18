package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

func TestMobileInitAndUSBEndpoint(t *testing.T) {
	dir := t.TempDir()
	reg := filepath.Join(dir, "device.json")
	if err := hostprovision.SaveEnrollment(reg, hostprovision.Enrollment{DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}, Timezone: "UTC"}); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "mobile")
	args := []string{"-registry", reg, "mobile-init", dst, "paper.example", "192.0.2.4"}
	var out bytes.Buffer
	if err := run(args, nil, &out); err != nil || out.Len() != 0 {
		t.Fatal(err, "unexpected output or setup failure")
	}
	crt := filepath.Join(dst, "device-tls.crt")
	checkMobileNames(t, dst, crt, args[4:])
	checkMobileArgument(t, crt)
	checkMobileCard(t, dst)
	if err := run(args, nil, &out); err == nil {
		t.Fatal("replaced identity")
	}
	checkBadMobileArgs(t, dir, reg, dst, crt)
}

func checkMobileNames(t *testing.T, dst, crt string, hosts []string) {
	t.Helper()
	pair, err := tls.LoadX509KeyPair(crt, filepath.Join(dst, "device-tls.key"))
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, host := range hosts {
		if err := leaf.VerifyHostname(host); err != nil {
			t.Fatal(err)
		}
		endpoint, err := mobileEndpoint(host, crt)
		if err != nil || !provision.ValidESP32Endpoint(endpoint) {
			t.Fatal(endpoint, err)
		}
	}
}

func checkMobileCard(t *testing.T, dst string) {
	t.Helper()
	info, err := os.Stat(filepath.Join(dst, "phone-setup.txt"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal(err)
	}
	card, err := os.ReadFile(filepath.Join(dst, "phone-setup.txt"))
	if err != nil || !bytes.HasPrefix(card, []byte("Wi-Fi: Epaper-010000\nPassword: ")) {
		t.Fatal("invalid setup card")
	}
}

func checkBadMobileArgs(t *testing.T, dir, reg, dst, crt string) {
	t.Helper()
	var err error
	var out bytes.Buffer
	for _, bad := range [][]string{{"mobile-init"}, {"-registry", reg, "mobile-init", dst, "bad@host"}, {"-registry", "missing", "mobile-init", dst, "paper"}} {
		if run(bad, nil, &out) == nil {
			t.Fatal(bad)
		}
	}
	if _, err = mobileEndpoint("wrong.example", crt); err == nil {
		t.Fatal("wrong name")
	}
	if _, err = mobileEndpoint("paper:443:bad", crt); err == nil {
		t.Fatal("wrong address")
	}
	if _, err = mobileEndpoint("paper", "missing"); err == nil {
		t.Fatal("missing certificate")
	}
	for _, data := range [][]byte{[]byte("bad"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("bad")})} {
		path := filepath.Join(dir, "invalid")
		if err = os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err = mobileEndpoint("paper", path); err == nil {
			t.Fatal("bad certificate")
		}
	}
}

func TestMobilePinBounds(t *testing.T) {
	for _, suffix := range []string{"", strings.Repeat("0", 64), strings.Repeat("g", 64), strings.Repeat("f", 65)} {
		if provision.ValidESP32Endpoint("tls://example.com:443#" + suffix) {
			t.Fatal(suffix)
		}
	}
}

func checkMobileArgument(t *testing.T, crt string) {
	t.Helper()
	result, err := managerArgument([]string{"set-manager", "paper.example", crt})
	if err != nil || !strings.HasPrefix(result, "tls://paper.example:443#") {
		t.Fatal(result, err)
	}
	if _, err = managerArgument([]string{"set-manager", "other.example", crt}); err == nil {
		t.Fatal("accepted wrong TLS name")
	}
}
