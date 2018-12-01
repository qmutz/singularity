// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	osexec "os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/pkg/util/copy"

	"github.com/kr/pty"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/pkg/ociruntime"
	"github.com/sylabs/singularity/pkg/util/rlimit"
	"github.com/sylabs/singularity/pkg/util/unix"

	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/util/exec"

	"github.com/sylabs/singularity/internal/pkg/security"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func setRlimit(rlimits []specs.POSIXRlimit) error {
	var resources []string

	for _, rl := range rlimits {
		if err := rlimit.Set(rl.Type, rl.Soft, rl.Hard); err != nil {
			return err
		}
		for _, t := range resources {
			if t == rl.Type {
				return fmt.Errorf("%s was already set", t)
			}
		}
		resources = append(resources, rl.Type)
	}

	return nil
}

func (engine *EngineOperations) emptyProcess(masterConn net.Conn) error {
	// pause process, by sending data to Smaster the process will
	// be paused with SIGSTOP signal
	if _, err := masterConn.Write([]byte("t")); err != nil {
		return fmt.Errorf("failed to pause process: %s", err)
	}

	// block on read waiting SIGCONT signal
	data := make([]byte, 1)
	if _, err := masterConn.Read(data); err != nil {
		return fmt.Errorf("failed to receive ack from Smaster: %s", err)
	}

	masterConn.Close()

	var status syscall.WaitStatus
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGCHLD, syscall.SIGINT, syscall.SIGTERM)

	if err := security.Configure(&engine.EngineConfig.OciConfig.Spec); err != nil {
		return fmt.Errorf("failed to apply security configuration: %s", err)
	}

	for {
		s := <-signals
		switch s {
		case syscall.SIGCHLD:
			for {
				if pid, _ := syscall.Wait4(-1, &status, syscall.WNOHANG, nil); pid <= 0 {
					break
				}
			}
		case syscall.SIGINT, syscall.SIGTERM:
			os.Exit(0)
		}
	}
}

// StartProcess starts the process
func (engine *EngineOperations) StartProcess(masterConn net.Conn) error {
	cwd := engine.EngineConfig.OciConfig.Process.Cwd

	if cwd == "" {
		cwd = "/"
	}

	if !filepath.IsAbs(cwd) {
		return fmt.Errorf("cwd property must be an absolute path")
	}

	if err := os.Chdir(cwd); err != nil {
		return fmt.Errorf("can't enter in current working directory: %s", err)
	}

	if err := setRlimit(engine.EngineConfig.OciConfig.Process.Rlimits); err != nil {
		return err
	}

	if engine.EngineConfig.EmptyProcess {
		return engine.emptyProcess(masterConn)
	}

	args := engine.EngineConfig.OciConfig.Process.Args
	env := engine.EngineConfig.OciConfig.Process.Env

	for _, e := range engine.EngineConfig.OciConfig.Process.Env {
		if strings.HasPrefix(e, "PATH=") {
			os.Setenv("PATH", e[5:])
		}
	}

	bpath, err := osexec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	args[0] = bpath

	if engine.EngineConfig.MasterPts != -1 {
		slaveFd := engine.EngineConfig.SlavePts
		if err := syscall.Dup3(slaveFd, int(os.Stdin.Fd()), 0); err != nil {
			return err
		}
		if err := syscall.Dup3(slaveFd, int(os.Stdout.Fd()), 0); err != nil {
			return err
		}
		if err := syscall.Dup3(slaveFd, int(os.Stderr.Fd()), 0); err != nil {
			return err
		}
		if err := syscall.Close(engine.EngineConfig.MasterPts); err != nil {
			return err
		}
		if err := syscall.Close(slaveFd); err != nil {
			return err
		}
		if _, err := syscall.Setsid(); err != nil {
			return err
		}
		if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, os.Stdin.Fd(), uintptr(syscall.TIOCSCTTY), 1); err != 0 {
			return fmt.Errorf("failed to set crontrolling terminal: %s", err.Error())
		}
	} else if engine.EngineConfig.OutputStreams[1] != -1 && engine.EngineConfig.ErrorStreams[1] != -1 {
		if err := syscall.Dup3(engine.EngineConfig.OutputStreams[1], int(os.Stdout.Fd()), 0); err != nil {
			return err
		}
		if err := syscall.Close(engine.EngineConfig.OutputStreams[1]); err != nil {
			return err
		}
		if err := syscall.Close(engine.EngineConfig.OutputStreams[0]); err != nil {
			return err
		}

		if err := syscall.Dup3(engine.EngineConfig.ErrorStreams[1], int(os.Stderr.Fd()), 0); err != nil {
			return err
		}
		if err := syscall.Close(engine.EngineConfig.ErrorStreams[1]); err != nil {
			return err
		}
		if err := syscall.Close(engine.EngineConfig.ErrorStreams[0]); err != nil {
			return err
		}
		os.Stdin.Close()
	}

	if !engine.EngineConfig.Exec {
		// pause process, by sending data to Smaster the process will
		// be paused with SIGSTOP signal
		if _, err := masterConn.Write([]byte("t")); err != nil {
			return fmt.Errorf("failed to pause process: %s", err)
		}

		// block on read waiting SIGCONT signal
		data := make([]byte, 1)
		if _, err := masterConn.Read(data); err != nil {
			return fmt.Errorf("failed to receive ack from Smaster: %s", err)
		}
	}

	if err := security.Configure(&engine.EngineConfig.OciConfig.Spec); err != nil {
		return fmt.Errorf("failed to apply security configuration: %s", err)
	}

	err = syscall.Exec(args[0], args, env)

	if !engine.EngineConfig.Exec {
		// write data to just tell Smaster to not execute PostStartProcess
		// in case of failure
		if _, err := masterConn.Write([]byte("t")); err != nil {
			sylog.Errorf("fail to send data to Smaster: %s", err)
		}
	}

	return fmt.Errorf("exec %s failed: %s", args[0], err)
}

// PreStartProcess will be executed in smaster context
func (engine *EngineOperations) PreStartProcess(pid int, masterConn net.Conn, fatalChan chan error) error {
	// stop container process
	syscall.Kill(pid, syscall.SIGSTOP)

	hooks := engine.EngineConfig.OciConfig.Hooks
	if hooks != nil {
		for _, h := range hooks.Prestart {
			if err := exec.Hook(&h, &engine.EngineConfig.State); err != nil {
				return err
			}
		}
	}

	file, err := instance.Get(engine.CommonConfig.ContainerID)
	socket := filepath.Join(filepath.Dir(file.Path), "attach.sock")
	engine.EngineConfig.State.Annotations[ociruntime.AnnotationAttachSocket] = socket

	attach, err := unix.CreateSocket(socket)
	if err != nil {
		return err
	}

	socket = filepath.Join(filepath.Dir(file.Path), "control.sock")
	engine.EngineConfig.State.Annotations[ociruntime.AnnotationControlSocket] = socket
	control, err := unix.CreateSocket(socket)
	if err != nil {
		return err
	}

	logPath := engine.EngineConfig.GetLogPath()
	if logPath == "" {
		containerID := engine.CommonConfig.ContainerID
		dir, err := instance.GetDirPrivileged(containerID)
		if err != nil {
			return err
		}
		logPath = filepath.Join(dir, containerID+".log")
	}

	format := engine.EngineConfig.GetLogFormat()
	formatter, ok := instance.LogFormats[format]
	if !ok {
		return fmt.Errorf("log format %s is not supported", format)
	}

	logger, err := instance.NewLogger(logPath, formatter)
	if err != nil {
		return err
	}

	go engine.handleControl(control, logger, fatalChan)
	go engine.handleStream(attach, logger, fatalChan)

	pidFile := engine.EngineConfig.GetPidFile()
	if pidFile != "" {
		if err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
			return err
		}
	}

	if err := engine.updateState("created"); err != nil {
		return err
	}

	// since paused process block on read, send it an
	// ACK so when it will receive SIGCONT, the process
	// will continue execution normally
	if _, err := masterConn.Write([]byte("s")); err != nil {
		return fmt.Errorf("failed to send ACK to start process: %s", err)
	}

	// wait container process execution
	data := make([]byte, 1)

	if _, err := masterConn.Read(data); err != io.EOF {
		return err
	}

	return nil
}

// PostStartProcess will execute code in smaster context after execution of container
// process, typically to write instance state/config files or execute post start OCI hook
func (engine *EngineOperations) PostStartProcess(pid int) error {
	if err := engine.updateState("running"); err != nil {
		return err
	}

	hooks := engine.EngineConfig.OciConfig.Hooks
	if hooks != nil {
		for _, h := range hooks.Poststart {
			if err := exec.Hook(&h, &engine.EngineConfig.State); err != nil {
				sylog.Warningf("%s", err)
			}
		}
	}

	return nil
}

func (engine *EngineOperations) handleStream(l net.Listener, logger *instance.Logger, fatalChan chan error) {
	var stdout io.ReadWriter
	var stderr io.Reader
	var outputWriters *copy.MultiWriter
	var errorWriters *copy.MultiWriter
	var tbuf *copy.TerminalBuffer

	hasTerminal := engine.EngineConfig.OciConfig.Process.Terminal

	defer l.Close()

	outputWriters = &copy.MultiWriter{}
	outputWriters.Add(logger.NewWriter("stdout", false))

	if hasTerminal {
		stdout = os.NewFile(uintptr(engine.EngineConfig.MasterPts), "stream-master-pts")
		tbuf = copy.NewTerminalBuffer()
		outputWriters.Add(tbuf)
	} else {
		outputStream := os.NewFile(uintptr(engine.EngineConfig.OutputStreams[0]), "stdout-stream")
		errorStream := os.NewFile(uintptr(engine.EngineConfig.ErrorStreams[0]), "error-stream")
		outputWriters.Add(os.Stdout)
		stdout = outputStream
		stderr = errorStream
	}

	go func() {
		io.Copy(outputWriters, stdout)
	}()

	if stderr != nil {
		errorWriters = &copy.MultiWriter{}
		errorWriters.Add(logger.NewWriter("stderr", false))
		errorWriters.Add(os.Stderr)

		go func() {
			io.Copy(errorWriters, stderr)
		}()
	}

	for {
		c, err := l.Accept()
		if err != nil {
			fatalChan <- err
			return
		}

		go func() {
			outputWriters.Add(c)
			if errorWriters != nil {
				errorWriters.Add(c)
			}
			if hasTerminal {
				tbuf.Lock()
				c.Write(tbuf.Line())
				tbuf.Unlock()
				io.Copy(stdout, c)
			} else {
				io.Copy(ioutil.Discard, c)
			}
			outputWriters.Del(c)
			if errorWriters != nil {
				errorWriters.Del(c)
			}
			c.Close()
		}()
	}
}

func (engine *EngineOperations) handleControl(l net.Listener, logger *instance.Logger, fatalChan chan error) {
	var master *os.File

	if engine.EngineConfig.OciConfig.Process.Terminal {
		master = os.NewFile(uintptr(engine.EngineConfig.MasterPts), "control-master-pts")
	}

	for {
		ctrl := &ociruntime.Control{}

		c, err := l.Accept()
		if err != nil {
			fatalChan <- err
			return
		}
		dec := json.NewDecoder(c)
		if err := dec.Decode(ctrl); err != nil {
			fatalChan <- err
			return
		}

		c.Close()

		if ctrl.ConsoleSize != nil && master != nil {
			size := &pty.Winsize{
				Cols: uint16(ctrl.ConsoleSize.Width),
				Rows: uint16(ctrl.ConsoleSize.Height),
			}
			if err := pty.Setsize(master, size); err != nil {
				fatalChan <- err
				return
			}
		}
		if ctrl.ReopenLog {
			logger.ReOpenFile()
		}
	}
}
