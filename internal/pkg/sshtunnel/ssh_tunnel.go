package sshtunnel

import (
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"time"
)

type SSHTunnel struct {
	Local  *Endpoint
	Server *Endpoint
	Remote *Endpoint
	Config *ssh.ClientConfig
	Log    *log.Logger
}

func (tunnel *SSHTunnel) logf(fmt string, args ...interface{}) {
	if tunnel.Log != nil {
		tunnel.Log.Printf(fmt, args...)
	}
}

func (tunnel *SSHTunnel) Start() error {
	listener, err := net.Listen("tcp", tunnel.Local.String())
	if err != nil {
		return err
	}
	defer listener.Close()

	tunnel.Local.Port = listener.Addr().(*net.TCPAddr).Port

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		tunnel.logf("accepted connection")
		go tunnel.forward(conn)
	}
}

func (tunnel *SSHTunnel) forward(localConn net.Conn) {

	var serverConn *ssh.Client
	var err error

	for i := 5; i >= 0; i-- {
		serverConn, err = ssh.Dial("tcp", tunnel.Server.String(), tunnel.Config)
		if err != nil {
			tunnel.logf("server dial error (%d retries left): %s", i, err)
			time.Sleep(3 * time.Second)
			continue
		}

		break
	}

	tunnel.logf("connected to %s (1 of 2)\n", tunnel.Server.String())

	var remoteConn net.Conn
	for i := 5; i >= 0; i-- {
		remoteConn, err = serverConn.Dial("tcp", tunnel.Remote.String())
		if err != nil {
			tunnel.logf("remote dial error (%d retries left): %s", i, err)
			time.Sleep(3 * time.Second)
			continue
		}

		break
	}

	tunnel.logf("connected to %s (2 of 2)\n", tunnel.Remote.String())

	copyConn := func(writer, reader net.Conn) {
		_, err := io.Copy(writer, reader)
		if err != nil {
			tunnel.logf("io.Copy error: %s", err)
		}
	}

	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
}

func NewSSHTunnel(tunnel string, auth ssh.AuthMethod, destination string) *SSHTunnel {
	// A random port will be chosen for us.
	localEndpoint := NewEndpoint("localhost:0")

	server := NewEndpoint(tunnel)
	if server.Port == 0 {
		server.Port = 22
	}

	sshTunnel := &SSHTunnel{
		Config: &ssh.ClientConfig{
			User: server.User,
			Auth: []ssh.AuthMethod{auth},
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				// Always accept key.
				return nil
			},
		},
		Local:  localEndpoint,
		Server: server,
		Remote: NewEndpoint(destination),
	}

	return sshTunnel
}
