package launch

import "github.com/Rethunk-Tech/go-gameproc"

// WaitPID waits until pid exits and reports how; the status is known only for Mortar's own children.
func WaitPID(pid int) (Exit, error) {
	s, err := gameproc.WaitPID(pid)
	return Exit{Code: s.Code, Signal: s.Signal}, err
}

// WaitError maps cmd.Wait's result to an Exit.
func WaitError(err error) Exit {
	s := gameproc.StatusOf(err)
	return Exit{Code: s.Code, Signal: s.Signal}
}
