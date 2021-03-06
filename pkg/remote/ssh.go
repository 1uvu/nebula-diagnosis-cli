package remote

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/nebula/nebula-diagnose/pkg/config"

	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	scid string
	*ssh.Client
}

type ExecuteResult struct {
	CMD     string
	Err     error
	StdOut  []byte
	StdErr  []byte
	Latency time.Duration
}

var (
	sshClients = make(map[string]*SSHClient)
	sshMux     sync.RWMutex
)

func GetSSHClient(scid string, conf *config.SSHConfig) (*SSHClient, error) {
	sshMux.Lock()
	if _, ok := sshClients[scid]; !ok {
		c, err := newSSHClient(conf)
		if err != nil {
			return nil, err
		}

		c.scid = scid
		sshClients[scid] = c
	}
	sshMux.Unlock()

	sshMux.RLock()
	c := sshClients[scid]
	sshMux.RUnlock()

	return c, nil
}

func newSSHClient(conf *config.SSHConfig) (*SSHClient, error) {
	timeout, err := time.ParseDuration(conf.Timeout)
	if err != nil {
		return nil, err
	}

	sshConf := &ssh.ClientConfig{
		Timeout:         timeout,
		User:            conf.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshConf.Auth = []ssh.AuthMethod{ssh.Password(conf.Password)}

	sshHost := fmt.Sprintf("%s:%d", conf.Address, conf.Port)
	sshClient, err := ssh.Dial("tcp", sshHost, sshConf)
	if err != nil {
		return nil, err
	}

	return &SSHClient{
		Client: sshClient,
	}, nil
}

func (c *SSHClient) ExecuteAsync(cmd string, ch chan<- ExecuteResult) {
	res, _ := c.Execute(cmd)
	ch <- res
}

func (c *SSHClient) Execute(cmd string) (ExecuteResult, bool) {
	now := time.Now()
	session, err := c.NewSession()
	if err != nil {
		return ExecuteResult{cmd, err, []byte{}, []byte{}, time.Since(now)}, false
	}
	defer session.Close()

	var stdOut bytes.Buffer
	var stdErr bytes.Buffer

	session.Stdout = &stdOut
	session.Stderr = &stdErr

	err = session.Run(cmd)
	if err != nil || len(stdErr.Bytes()) > 0 {
		return ExecuteResult{cmd, err, stdOut.Bytes(), stdErr.Bytes(), time.Since(now)}, false
	}

	return ExecuteResult{cmd, err, stdOut.Bytes(), stdErr.Bytes(), time.Since(now)}, true
}

func (c *SSHClient) Close() {
	sshMux.Lock()
	defer sshMux.Unlock()
	c.Client.Close()
	delete(sshClients, c.scid)
}
