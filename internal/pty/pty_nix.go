//go:build !windows
// +build !windows

package pty

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"

	"github.com/creack/pty"
	"github.com/mohammed90/caddy-ssh/internal/session"
	"go.uber.org/zap"
)

type caddyPty struct {
	pty *os.File
}

func (s Shell) openPty(sess session.Session, cmd []string) (sshPty, error) {
	ptyReq, winCh, isPty := sess.Pty()
	if s.ForcePTY && !isPty {
		return nil, fmt.Errorf("ssh: not pty")
	}
	s.logger.Info("openPty", zap.String("TERM", ptyReq.Term))
	args := []string{}
	if len(cmd) > 0 {
		args = append(args, "-c")
		args = append(args, cmd...)
	}

	shell := "/bin/sh"
	if s.Shell != "" {
		shell = s.Shell
	}
	execCmd := exec.Command(shell, args...)

	env := make([]string, len(s.Env))
	for k, v := range s.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	execCmd.Env = append(execCmd.Env, sess.Environ()...)
	execCmd.Env = append(execCmd.Env, env...)
	execCmd.Env = append(execCmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))

	usr, err := user.Lookup(sess.User())
	if err != nil {
		return nil, err
	}
	if usr.HomeDir != "" {
		execCmd.Dir = usr.HomeDir
	}
	// thanks @mholt!
	// jailCommand(execCmd, u)
	// run as unprivileged user
	uid, _ := strconv.ParseUint(usr.Uid, 10, 32)
	gid, _ := strconv.ParseUint(usr.Gid, 10, 32)
	f, err := pty.StartWithAttrs(execCmd, &pty.Winsize{}, &syscall.SysProcAttr{
		Setsid: true,
		Credential: &syscall.Credential{
			Uid:         uint32(uid), // <-- other user's ID
			Gid:         uint32(gid),
			NoSetGroups: true,
		},
	})
	if err != nil {
		return nil, err
	}
	spty := &caddyPty{f}
	go func() {
		for win := range winCh {
			spty.SetWindowsSize(win.Height, win.Width)
		}
	}()
	return spty, nil
}

func (p *caddyPty) Communicate(peer io.ReadWriter) {
	go func() {
		io.Copy(p.pty, peer) // stdin
	}()
	io.Copy(peer, p.pty) // stdout
}

func (p *caddyPty) SetWindowsSize(h, w int) {
	pty.Setsize(p.pty, &pty.Winsize{Rows: uint16(h), Cols: uint16(w)}) //nolint
}

func (p *caddyPty) Close() error {
	return p.pty.Close()
}

var _ sshPty = (*caddyPty)(nil)
