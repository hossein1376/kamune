package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/hossein1376/kamune"
)

var errCh = make(chan error)
var stop = make(chan struct{})

type Program struct {
	*tea.Program
	transport *kamune.Transport
}

func NewProgram(p *tea.Program) *Program {
	return &Program{Program: p}
}

func main() {
	args := os.Args[1:]
	if len(args) != 2 {
		return
	}

	switch addr := args[1]; args[0] {
	case "dial":
		go func() {
			client(addr)
		}()
	case "serve":
		go func() {
			server(addr)
		}()
	default:
		panic(fmt.Errorf("invalid command: %s", args[0]))
	}

	select {
	case err := <-errCh:
		fmt.Println("error:", err)
	case <-stop:
	}
}

func serveHandler(t *kamune.Transport) error {
	p := NewProgram(tea.NewProgram(initialModel(t), tea.WithAltScreen()))
	go func() {
		if _, err := p.Run(); err != nil {
			panic(err)
		}
		stop <- struct{}{}
	}()

	for {
		b := kamune.Bytes(nil)
		metadata, err := t.Receive(b)
		if err != nil {
			if errors.Is(err, kamune.ErrConnClosed) {
				p.Quit()
				return nil
			}
			errCh <- fmt.Errorf("receiving: %w", err)
			return nil
		}
		p.Send(NewMessage(metadata.Timestamp(), b.Value))
	}
}

func server(addr string) {
	srv, err := kamune.NewServer(addr, serveHandler, kamune.ServeWithUDP())
	if err != nil {
		errCh <- fmt.Errorf("starting server: %w", err)
		return
	}
	errCh <- srv.ListenAndServe()
}

func client(addr string) {
	var t *kamune.Transport
	for {
		var opErr *net.OpError
		var err error
		t, err = kamune.Dial(addr, kamune.DialWithUDPConn())
		if err == nil {
			break
		}
		if errors.As(err, &opErr) && errors.Is(opErr.Err, syscall.ECONNREFUSED) {
			time.Sleep(2 * time.Second)
			continue
		}
		log.Printf("dial err: %v", err)
		time.Sleep(5 * time.Second)
	}
	defer t.Close()

	p := NewProgram(tea.NewProgram(initialModel(t), tea.WithAltScreen()))
	go func() {
		if _, err := p.Run(); err != nil {
			errCh <- err
		}
		stop <- struct{}{}
	}()

	for {
		b := kamune.Bytes(nil)
		metadata, err := t.Receive(b)
		if err != nil {
			if errors.Is(err, kamune.ErrConnClosed) {
				p.Quit()
				return
			}
			errCh <- fmt.Errorf("receiving: %w", err)
			return
		}
		p.Send(NewMessage(metadata.Timestamp(), b.Value))
	}
}

type Message struct {
	prefix string
	text   string
}

func NewMessage(timestamp time.Time, text []byte) Message {
	return Message{
		prefix: fmt.Sprintf("[%s] Peer: ", timestamp.Format(time.DateTime)),
		text:   string(text),
	}
}
