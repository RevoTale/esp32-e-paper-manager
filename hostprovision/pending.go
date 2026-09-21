package hostprovision

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

var (
	ErrPending         = errors.New("host provisioning: recovery=1 unresolved .pending exists; retain both enrollment files, reopen USB and inspect; do not retry mutation")
	ErrStaging         = errors.New("host provisioning: recovery=2 staging failed; active enrollment preserved; inspect and retain any .pending before retry")
	ErrAcknowledgement = errors.New("host provisioning: recovery=3 ACK does not confirm candidate; retain active and .pending, reopen USB and inspect; do not retry mutation")
	ErrPromotion       = errors.New("host provisioning: recovery=4 promotion failed; retain active and .pending, reopen USB and inspect; do not retry mutation")
	ErrDurability      = errors.New("host provisioning: recovery=5 active candidate installed but directory sync failed; retain active, verify storage and reopen USB to inspect; do not retry mutation")
)

// PendingEnrollment holds one exclusive candidate until a matching USB ACK.
// Close releases the directory handle; it never discards the candidate.
type PendingEnrollment struct {
	root       *os.Root
	name       string
	enrollment Enrollment
	previous   enrollmentSnapshot
	candidate  enrollmentSnapshot
	ops        registryOps
}

type registryOps struct {
	write  func(*os.File, []byte) error
	sync   func(*os.Root) error
	rename func(*os.Root, string, string) error
}

func StageEnrollment(path string, enrollment Enrollment) (*PendingEnrollment, error) {
	return stageEnrollment(path, enrollment, registryOps{write: writeEnrollmentFile, sync: syncRegistry,
		rename: func(root *os.Root, old, next string) error { return root.Rename(old, next) }})
}

func stageEnrollment(path string, enrollment Enrollment, ops registryOps) (*PendingEnrollment, error) {
	data, err := candidateBytes(path, enrollment)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, ErrStaging
	}
	pending := &PendingEnrollment{root: root, name: filepath.Base(path), enrollment: enrollment, ops: ops}
	if err = pending.stage(data); err != nil {
		_ = pending.Close()
		return nil, err
	}
	return pending, nil
}

func candidateBytes(path string, enrollment Enrollment) ([]byte, error) {
	if path == "" || !validEnrollment(enrollment) {
		return nil, ErrStaging
	}
	data, err := json.Marshal(enrollment)
	if err != nil {
		return nil, ErrStaging
	}
	return append(data, '\n'), nil
}

func validEnrollment(enrollment Enrollment) bool {
	return enrollment.DeviceID != ([16]byte{}) && enrollment.DeviceKey != ([32]byte{}) &&
		len(enrollment.Timezone) > 0 && len(enrollment.Timezone) <= provision.MaxTimezone
}

func (p *PendingEnrollment) stage(data []byte) error {
	if _, err := p.root.Lstat(p.name + ".pending"); err == nil {
		return ErrPending
	} else if !errors.Is(err, os.ErrNotExist) {
		return ErrStaging
	}
	previous, err := readEnrollmentSnapshot(p.root, p.name)
	if err != nil {
		return ErrStaging
	}
	p.previous = previous
	file, err := p.root.OpenFile(p.name+".pending", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return ErrStaging
	}
	if err = p.ops.write(file, data); err != nil {
		return ErrStaging
	}
	if err = p.ops.sync(p.root); err != nil {
		return ErrStaging
	}
	p.candidate, err = readEnrollmentSnapshot(p.root, p.name+".pending")
	if err != nil || string(p.candidate.data) != string(data) {
		return ErrStaging
	}
	return nil
}

func (p *PendingEnrollment) Close() error { return p.root.Close() }

// Confirm promotes only the staged request after a canonical successful ACK.
// Public inspect metadata alone is insufficient to recover an ambiguous write.
func (p *PendingEnrollment) Confirm(request provision.Request, response provision.Response) error {
	return p.ConfirmFor(provision.Codec{}, request, response)
}

func (p *PendingEnrollment) ConfirmFor(codec provision.Codec, request provision.Request, response provision.Response) error {
	if !p.matchesFor(codec, request, response) {
		return ErrAcknowledgement
	}
	if !unchangedEnrollment(p.root, p.name, p.previous) ||
		!unchangedEnrollment(p.root, p.name+".pending", p.candidate) {
		return ErrPromotion
	}
	if err := p.ops.rename(p.root, p.name+".pending", p.name); err != nil {
		return ErrPromotion
	}
	if err := p.ops.sync(p.root); err != nil {
		return ErrDurability
	}
	return nil
}

func (p *PendingEnrollment) matchesFor(codec provision.Codec, request provision.Request, response provision.Response) bool {
	c := request.Config
	if c.ValidateFor(codec) != nil || c.DeviceID != p.enrollment.DeviceID || c.DeviceKey != p.enrollment.DeviceKey ||
		c.Timezone != p.enrollment.Timezone {
		return false
	}
	return matchingResponse(request, response)
}

func matchingResponse(request provision.Request, response provision.Response) bool {
	if request.Operation != provision.OperationProvision && request.Operation != provision.OperationRotate {
		return false
	}
	if response.Operation != request.Operation || response.Code != provision.CodeOK ||
		response.State != provision.StateProvisioned || response.Generation == 0 {
		return false
	}
	c := request.Config
	return response.DeviceID == c.DeviceID && matchingMetadata(c, response)
}

func matchingMetadata(config provision.Config, response provision.Response) bool {
	return response.Auth == config.Auth && response.SSID == config.SSID &&
		response.Manager == config.Manager && response.Timezone == config.Timezone
}
