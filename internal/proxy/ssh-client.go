package proxy

import (
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"strings"
	"time"
)

type BastionSSHSession struct {
	logger       *zap.Logger
	client       *ssh.Client
	session      *ssh.Session
	remoteStdin  io.Writer
	remoteStdout io.Reader
	remoteStderr io.Reader
}

func NewSSHSession(logger *zap.Logger, address string, login, password, privateKey string) (*BastionSSHSession, error) {
	var bastionClient BastionSSHSession
	var err error

	config := &ssh.ClientConfig{
		User: login,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Нужно проверять ключ хоста
		Timeout:         time.Second * 5,
	}

	if len(privateKey) > 0 {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			logger.Debug("")
			return nil, err
		}
		config.Auth = append(config.Auth, ssh.PublicKeys(signer))
	}

	bastionClient.client, err = dialWithDeadline("tcp", address, config)
	if err != nil {
		return nil, err
	}

	bastionClient.session, err = bastionClient.client.NewSession()
	if err != nil {
		return nil, err
	}

	bastionClient.remoteStdin, err = bastionClient.session.StdinPipe()
	if err != nil {
		return nil, err
	}
	bastionClient.remoteStdout, err = bastionClient.session.StdoutPipe()
	if err != nil {
		return nil, err
	}
	bastionClient.remoteStderr, err = bastionClient.session.StderrPipe()
	if err != nil {
		return nil, err
	}
	bastionClient.logger = logger

	return &bastionClient, nil
}

func dialWithDeadline(network string, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	conn, err := net.DialTimeout(network, addr, config.Timeout)
	if err != nil {
		return nil, err
	}
	if config.Timeout > 0 {
		err = conn.SetReadDeadline(time.Now().Add(config.Timeout))
		if err != nil {
			return nil, err
		}
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, err
	}
	if config.Timeout > 0 {
		err = conn.SetReadDeadline(time.Time{})
		if err != nil {
			return nil, err
		}
	}
	return ssh.NewClient(c, chans, reqs), nil
}

func (sess BastionSSHSession) StartShell(term string, h int, w int, environment []string) error {
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	}

	sess.SetEnvironment(environment)
	if err := sess.session.RequestPty(term, h, w, modes); err != nil {
		sess.logger.Error("Error requesting \"pty-req\"")
		return err
	}
	if err := sess.session.Shell(); err != nil {
		sess.logger.Error("Error requesting \"shell\"")
		return err
	}
	return nil
}

func (sess BastionSSHSession) SetEnvironment(envs []string) {
	for _, env := range envs {
		kv := strings.Split(env, "=")
		if len(kv) != 2 {
			sess.logger.Error("Error splitting string to key/value", zap.String("arg", env))
			continue
		}
		if err := sess.session.Setenv(kv[0], kv[1]); err != nil {
			sess.logger.Error("Error setting environment variable", zap.String("arg_name", kv[0]), zap.String("arg_value", kv[1]), zap.String("error", err.Error()))
			continue
		}
	}
}

func (sess BastionSSHSession) ResizeWindow(h, w int) {
	if err := sess.session.WindowChange(h, w); err != nil {
		sess.logger.Error(err.Error())
	}
}

func (sess BastionSSHSession) Close() {
	err := sess.session.Close()
	if err != nil {
		sess.logger.Error(err.Error())
	}
	err = sess.client.Close()
	if err != nil {
		sess.logger.Error(err.Error())
	}
	sess.logger.Debug("SSH session to target host was closed")
}

func (sess BastionSSHSession) Stdin() io.Writer {
	return sess.remoteStdin
}

func (sess BastionSSHSession) Stdout() io.Reader {
	return sess.remoteStdout
}

func (sess BastionSSHSession) Stderr() io.Reader {
	return sess.remoteStderr
}
