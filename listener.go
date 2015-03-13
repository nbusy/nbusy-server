package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

var (
	ping   = []byte("ping")
	closed = []byte("close")
)

// Listener accepts connections from devices.
type Listener struct {
	debug    bool
	listener net.Listener
}

// Listen creates a TCP listener with the given PEM encoded X.509 certificate and the private key on the local network address laddr.
// Debug mode logs all server activity.
func Listen(cert, privKey []byte, laddr string, debug bool) (*Listener, error) {
	tlsCert, err := tls.X509KeyPair(cert, privKey)
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(cert)
	if err != nil || !ok {
		return nil, fmt.Errorf("failed to parse the certificate or the private key: %v", err)
	}

	conf := tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		ClientCAs:    pool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
	}

	l, err := tls.Listen("tcp", laddr, &conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS listener no network address %v: %v", laddr, err)
	}
	if debug {
		log.Printf("Listener created with local network address: %v\n", laddr)
	}

	return &Listener{
		debug:    debug,
		listener: l,
	}, nil
}

// Session is a generic session data store for client handlers.
type Session struct {
	UserID string
	Closed bool
	Data   interface{}
}

// Accept waits for incoming connections and forwards the client connect/message/disconnect events to provided handlers in a new goroutine.
// This function blocks and never returns, unless there is an error while accepting a new connection.
func (l *Listener) Accept(handleMsg func(conn *tls.Conn, session *Session, msg []byte), handleDisconn func(conn *tls.Conn, session *Session)) error {
	for {
		conn, err := l.listener.Accept()
		if err != nil {
			return fmt.Errorf("error while accepting a new connection from a client: %v", err)
			// todo: it might not be appropriate to break the loop on recoverable errors (like client disconnect during handshake)
			// the underlying fd.accept() does some basic recovery though we might need more: http://golang.org/src/net/fd_unix.go
		}
		tlsconn, _ := conn.(*tls.Conn) // todo: check ok
		if l.debug {
			log.Println("Client connected: listening for messages from client IP:", conn.RemoteAddr())
		}
		go handleClient(tlsconn, l.debug, handleMsg, handleDisconn)
	}
}

// handleClient waits for messages from the connected client and forwards the client message/disconnect
// events to provided handlers in a new goroutine.
// This function never returns, unless there is an error while reading from the channel or the client disconnects.
func handleClient(conn *tls.Conn, debug bool, handleMsg func(conn *tls.Conn, session *Session, msg []byte), handleDisconn func(conn *tls.Conn, session *Session)) {
	defer conn.Close()
	if debug {
		defer log.Println("Closed connection to client with IP:", conn.RemoteAddr())
	}

	session := &Session{UserID: ""}
	reader := bufio.NewReader(conn)

	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute * 5))

		// read the content length header
		line, err := reader.ReadSlice('\n')
		if err != nil {
			log.Fatalln("Client read error: ", err)
			break
		}

		// calculate the content length
		n, err := strconv.Atoi(string(line[:len(line)-1]))
		if err != nil || n == 0 {
			log.Fatalln("Client read error: invalid content lenght header sent or content lenght mismatch: ", err)
			break
		}

		// read the message content
		if debug {
			log.Println("Starting to read message content of bytes: ", n)
		}
		msg := make([]byte, n)
		total := 0
		for total < n {
			// todo: log here in case it gets stuck, pumping up cpu usage!
			i, err := reader.Read(msg)
			if err != nil {
				log.Fatalln("Error while reading incoming message: ", err)
				break
			}
			total += i
		}
		if err != nil {
			log.Fatalln("Error while reading incoming message: ", err)
			break
		}
		if debug {
			log.Printf("Read %v bytes client IP %v. Incoming message: %v\n", n, conn.RemoteAddr(), string(msg))
		}

		if n == 4 && bytes.Equal(msg, ping) {
			continue
		}

		if n == 5 && bytes.Equal(msg, closed) {
			go handleDisconn(conn, session)
			return
		}

		go handleMsg(conn, session, msg)
	}
}

// Close closes the listener.
func (l *Listener) Close() error {
	if l.debug {
		defer log.Println("Listener was closed on local network address:", l.listener.Addr())
	}
	return l.listener.Close()
}
