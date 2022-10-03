package proxy

import (
	"bastion/internal/log"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/gliderlabs/ssh"
	expect "github.com/google/goexpect"
	"github.com/plyul/telnet"
	"go.uber.org/zap"
)

type proxySessionData struct {
	logger         *zap.Logger
	pty            ssh.Pty
	winCh          <-chan ssh.Window
	clientStdin    io.Writer
	clientStdout   io.Reader
	clientStderr   io.ReadWriter
	env            []string
	targetAddress  string
	targetLogin    string
	targetPassword string
	targetPrivKey  string
}

// /--------\ stdout -> R /--------\ W ->  stdin /--------\
// | client |             | proxy  |             | target |
// \--------/ stdin  <- W \--------/ R <- stdout \--------/
//
//	stderr <- W(!)      (!)R <- stderr
func (app *BastionProxy) SessionHandler(clientSession ssh.Session) {
	clientAddress := strings.Split(clientSession.RemoteAddr().String(), ":")[0]
	token := clientSession.User()

	sessionLogger := log.Get().With(zap.String("client", clientAddress), zap.String("token", token))
	ptyReq, winCh, isPty := clientSession.Pty()
	if !isPty {
		sessionLogger.Error("Client has not requested PTY, will not serve request")
		return
	}

	session, err := app.apiClient.GetSession(token)
	if err != nil {
		sessionLogger.Error("Error getting session data", zap.String("error", err.Error()))
		return
	}

	if !strings.EqualFold(session.TargetNetwork, app.config.GuardedNetwork) {
		sessionLogger.Error("Wrong target network", zap.String("target_network", session.TargetNetwork))
		return
	}

	_, err = io.WriteString(clientSession, fmt.Sprintf("Бастион приветствует тебя!\nЕсли ты будешь хулиганить, то я вычислю тебя по IP!\n\n"))
	if err != nil {
		sessionLogger.Error(err.Error())
		return
	}
	data := proxySessionData{
		logger:         sessionLogger,
		pty:            ptyReq,
		winCh:          winCh,
		clientStdin:    clientSession,
		clientStdout:   clientSession,
		clientStderr:   clientSession.Stderr(),
		env:            clientSession.Environ(),
		targetAddress:  session.TargetHost + ":" + session.TargetPort,
		targetLogin:    session.TargetLogin,
		targetPassword: session.TargetPassword,
		targetPrivKey:  session.TargetPrivKey,
	}

	switch strings.ToLower(session.TargetProtocol) {
	case "ssh":
		sessionLogger.Debug("Establishing SSH session to target host")
		err = app.proxyToSSH(data)
	case "telnet":
		sessionLogger.Debug("Establishing Telnet session to target host")
		err = app.proxyToTelnet(data)
	default:
		sessionLogger.Error("Unknown protocol", zap.String("protocol", session.TargetProtocol))
		return
	}
	if err != nil {
		sessionLogger.Error("Error while connecting to target host", zap.String("error", err.Error()))
	}
	_, _ = io.WriteString(clientSession, "\nДо свидания\n")
	_ = clientSession.Close()
	_ = sessionLogger.Sync()
}

func (app *BastionProxy) ConnCallback(conn net.Conn) net.Conn {
	_ = conn.SetDeadline(time.Now().Add(time.Second * time.Duration(app.config.ConnectTimeoutSec)))
	return conn
}

func (app *BastionProxy) proxyToSSH(sessData proxySessionData) error {
	target, err := NewSSHSession(sessData.logger, sessData.targetAddress, sessData.targetLogin, sessData.targetPassword, sessData.targetPrivKey)
	if err != nil {
		sessData.logger.Error(err.Error())
		return err
	}
	defer target.Close()

	go func() { // проксирование запроса "window-change"
		for win := range sessData.winCh {
			target.ResizeWindow(win.Height, win.Width)
		}
	}()

	done := make(chan bool)
	go app.connectStreams("target stdin", sessData.logger, sessData.clientStdout, target.Stdin(), done)
	go app.connectStreams("target stdout", sessData.logger, target.Stdout(), sessData.clientStdin, done)
	go app.connectStreams("target stderr", sessData.logger, target.Stderr(), sessData.clientStderr, done)

	if err := target.StartShell(sessData.pty.Term, sessData.pty.Window.Height, sessData.pty.Window.Width, sessData.env); err != nil {
		sessData.logger.Error(err.Error())
		return err
	}
	<-done
	return nil
}

func (app *BastionProxy) proxyToTelnet(sessData proxySessionData) error {
	target, err := telnet.Connect(sessData.targetAddress)
	if err != nil {
		sessData.logger.Error(err.Error())
		return err
	}

	target.SetWindowSize(sessData.pty.Window.Width, sessData.pty.Window.Height)
	go func() { // проксирование запроса "window-change"
		for win := range sessData.winCh {
			target.SetWindowSize(win.Width, win.Height)
			sessData.logger.Debug(fmt.Sprintf("Window size changed to (%d, %d)", win.Width, win.Height))
		}
	}()

	_, _ = io.WriteString(sessData.clientStdin, app.telnetLoginToTarget(target, sessData.logger, sessData.targetLogin, sessData.targetPassword))

	done := make(chan bool)
	go app.connectStreams("target stdin", sessData.logger, sessData.clientStdout, target, done)
	go app.connectStreams("target stdout", sessData.logger, target, sessData.clientStdin, done)
	<-done
	return target.Close()
}

func (app *BastionProxy) connectStreams(id string, logger *zap.Logger, reader io.Reader, writer io.Writer, done chan bool) {
	logger.Debug("Connecting stream", zap.String("stream_id", id))
	buffer := make([]byte, 1024)
	bl := log.NewBufferedLogger(logger, id)
	defer bl.Close()
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			_, blerr := bl.Write(buffer[:n])
			if blerr != nil {
				logger.Error(blerr.Error())
			}
			_, werr := writer.Write(buffer[:n])
			if werr != nil {
				logger.Error(werr.Error())
				done <- true
				return
			}
		}
		if err != nil {
			if err == io.EOF {
				logger.Debug("Stream EOF", zap.String("stream_id", id))
			} else {
				logger.Error(err.Error())
			}
			done <- true
			return
		}
	}
}

func (app *BastionProxy) telnetLoginToTarget(session *telnet.Connection, logger *zap.Logger, login, password string) string {
	const expectTimeout = time.Second * 5
	var capturedOutput string

	userRE := regexp.MustCompile("ogin: ")
	passRE := regexp.MustCompile("assword: ")

	expecting := true
	resCh := make(chan error)

	opts := &expect.GenOptions{
		In:  session,
		Out: session,
		Wait: func() error {
			return <-resCh
		},
		Close: func() error {
			close(resCh)
			return nil
		},
		Check: func() bool {
			return expecting
		},
	}

	expector, _, err := expect.SpawnGeneric(opts, -1)
	if err != nil {
		return capturedOutput
	}

	output, match, err := expector.Expect(userRE, expectTimeout)
	if err != nil {
		logger.Error(err.Error())
		return ""
	}
	capturedOutput += output
	logger.Debug("Login expect", zap.String("output", output), zap.String("match", match[0]))
	err = expector.Send(login + "\n")
	if err != nil {
		logger.Error(err.Error())
		return ""
	}

	output, match, err = expector.Expect(passRE, expectTimeout)
	if err != nil {
		logger.Error(err.Error())
		return ""
	}
	capturedOutput += output + "\n"
	logger.Debug("Password expect", zap.String("output", output), zap.String("match", match[0]))
	err = expector.Send(password + "\n")
	if err != nil {
		logger.Error(err.Error())
		return ""
	}

	expecting = false
	_ = expector.Close()
	capturedOutput = strings.ReplaceAll(capturedOutput, login, "<BASTION-WAS-HERE>")
	capturedOutput = strings.ReplaceAll(capturedOutput, password, "<BASTION-WAS-HERE>")
	return capturedOutput
}
