package util

import (
	"os/exec"
	"time"
	"bytes"
	"errors"
	"context"
)

func ExecBinary(binPath string, args []string, timeout time.Duration) (output string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.Command(binPath, args...)

	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf

	err = cmd.Start()
	if err == exec.ErrNotFound {
		err = errors.New(binPath + " binary not found")
		return
	} else if err != nil {
		return
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		err = cmd.Process.Kill()
		if err == nil {
			err = errors.New(binPath + " process timoeut exceeded")
		}
		return
	case err = <-done:
		if err != nil {
			return
		}
	}
	output = outputBuf.String()
	return
}
