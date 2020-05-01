package server

import (
	"default-cource-work/chat/protocol"
	"io"
	"log"
	"net"
	"sync"
)

type ChatServer interface {
	Listen(address string) error
	Broadcast(command interface{}) error
	Start()
	Close() error
}

type TcpChatServer struct {
	listener net.Listener
	clients []*client
	mutex   *sync.Mutex
}

type client struct {
	conn   net.Conn
	name   string
	writer *protocol.CommandWriter
}

func (s *TcpChatServer) Listen(address string) error {
	l, err := net.Listen("tcp", address)
	if err == nil {
		s.listener = l
		log.Printf("Listening on %v", address)
	}
	return err
}

func (s *TcpChatServer) Close() error {
	return s.listener.Close()
}

func (s *TcpChatServer) Start() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Print(err)
		} else {
			client := s.accept(conn)
			go s.serve(client)
		}
	}
}

func (s *TcpChatServer) accept(conn net.Conn) *client {
	log.Printf("Accepting connection from %v, total clients: %v", conn.RemoteAddr().String(), len(s.clients)+1)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	client := &client{
		conn:   conn,
		writer: protocol.NewCommandWriter(conn),
	}
	s.clients = append(s.clients, client)
	return client
}

func (s *TcpChatServer) remove(client *client) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for i, check := range s.clients {
		if check == client {
			s.clients = append(s.clients[:i], s.clients[i+1:]...)
		}
	}
	log.Printf("Closing connection from %v", client.conn.RemoteAddr().String())
	client.conn.Close()
}

func (s *TcpChatServer) serve(client *client) {
	cmdReader := protocol.NewCommandReader(client.conn)
	defer s.remove(client)
	for {
		cmd, err := cmdReader.Read()
		if err != nil && err != io.EOF {
			log.Printf("Read error: %v", err)
		}
		if cmd != nil {
			switch v := cmd.(type) {
				case protocol.SendCommand:
					go s.Broadcast(protocol.MessageCommand{
						Message: v.Message,
						Name:    client.name,
					})
				case protocol.NameCommand:
					client.name = v.Name
			}
		}
		if err == io.EOF {
			break
		}
	}
}

func (s *TcpChatServer) Broadcast(command interface{}) error {
	for _, client := range s.clients {
		// TODO: handle error here?
		client.writer.Write(command)
	}
	return nil
}