package plugins

import (
	"os/exec"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/loop"
)

// RegistrarConfig generates contains static configuration inher
type RegistrarConfig interface {
	RegisterLOOP(loopId string, cmdName string) (func() *exec.Cmd, loop.GRPCOpts, error)
}

type registarConfig struct {
	grpcOpts           loop.GRPCOpts
	loopRegistrationFn func(loopId string) (*RegisteredLoop, error)
}

// NewRegistrarConfig creates a RegistarConfig
// loopRegistrationFn must act as a global registry function of LOOPs and must be idempotent.
// The [func() *exec.Cmd] for a LOOP should be generated by calling [RegistrarConfig.RegisterLOOP]
func NewRegistrarConfig(grpcOpts loop.GRPCOpts, loopRegistrationFn func(loopId string) (*RegisteredLoop, error)) RegistrarConfig {
	return &registarConfig{
		grpcOpts:           grpcOpts,
		loopRegistrationFn: loopRegistrationFn,
	}
}

// RegisterLOOP calls the configured loopRegistrationFn. The loopRegistrationFn must act as a global registry for LOOPs and must be idempotent.
func (pc *registarConfig) RegisterLOOP(loopID string, cmdName string) (func() *exec.Cmd, loop.GRPCOpts, error) {
	cmdFn, err := NewCmdFactory(pc.loopRegistrationFn, CmdConfig{
		ID:  loopID,
		Cmd: cmdName,
	})
	if err != nil {
		return nil, loop.GRPCOpts{}, err
	}
	return cmdFn, pc.grpcOpts, nil
}
