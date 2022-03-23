package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"io"
	"net"
	"sync"
	"time"
)

// User represents user connection.
// It contains logic of receiving and sending messages.
// That is, there are no active reader or writer. Some other layer of the
// application should call Receive() to read user's incoming message.
type User struct {
	io   sync.Mutex
	conn io.ReadWriteCloser
	uid  string
	chat *Chat
}

// Receive reads next message from user's underlying connection.
// It blocks until full message received.
func (u *User) Receive() error {

	req, err := u.readRequest()

	if err != nil {

		u.conn.Close()
		return err
	}
	if req == nil {
		// Handled some control message.
		return nil
	}
	c := u.chat
	fmt.Println("gagaga  " + req.TOID)
	//ux, ok := c.ns[req.TOID]
	//if ok {
	//
	//	c.pool.Schedule(func() {
	//		ux.write(req)
	//	})
	//}

	for _, value := range c.ns {
		c.pool.Schedule(func() {
			value.write(req)
		})
	}

	return nil
}

// readRequests reads json-rpc request from connection.
// It takes io mutex.
func (u *User) readRequest() (*Request, error) {
	u.io.Lock()
	defer u.io.Unlock()

	h, r, err := wsutil.NextReader(u.conn, ws.StateServerSide)
	if err != nil {
		return nil, err
	}
	if h.OpCode.IsControl() {
		return nil, wsutil.ControlFrameHandler(u.conn, ws.StateServerSide)(h, r)
	}

	req := &Request{}
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(req); err != nil {
		return nil, err
	}

	return req, nil
}

func (u *User) writeErrorTo(req *Request, err string) error {
	return u.write(Error{
		ID:    req.ID,
		Error: err,
	})
}

func (u *User) writeResultTo(req *Request, result string) error {
	return u.write(Response{
		ID:     req.ID,
		Result: result,
	})
}

func (u *User) writeNotice(method string, params string) error {
	return u.write(Request{
		TOID:   method,
		Params: params,
	})
}

func (u *User) write(x interface{}) error {
	w := wsutil.NewWriter(u.conn, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	u.io.Lock()
	defer u.io.Unlock()

	if err := encoder.Encode(x); err != nil {
		return err
	}

	return w.Flush()
}

func (u *User) writeRaw(p []byte) error {
	u.io.Lock()
	defer u.io.Unlock()

	_, err := u.conn.Write(p)

	return err
}

// Chat contains logic of user interaction.
type Chat struct {
	mu   sync.RWMutex
	ns   map[string]*User
	pool *Pool
	out  chan []byte
}

func NewChat(pool *Pool) *Chat {
	chat := &Chat{
		pool: pool,
		ns:   make(map[string]*User),
		out:  make(chan []byte, 1),
	}

	//go chat.writer()

	return chat
}

// Register registers new connection as a User.
func (c *Chat) Register(conn net.Conn, uid string) *User {
	user := &User{
		chat: c,
		conn: conn,
		uid:  uid,
	}

	c.mu.Lock()
	{
		c.ns[user.uid] = user
	}
	c.mu.Unlock()

	//user.writeNotice("hello", "dd")
	//c.Broadcast("greet", "you")

	return user
}

// Remove removes user from chat.
func (c *Chat) Remove(user *User) {
	c.mu.Lock()
	removed := c.remove(user)
	c.mu.Unlock()

	if !removed {
		return
	}

	//c.Broadcast("goodbye", "gg")
}

// Broadcast sends message to all alive users.
func (c *Chat) Broadcast(method string, params string) error {
	var buf bytes.Buffer

	w := wsutil.NewWriter(&buf, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	r := Request{TOID: method, Params: params}
	if err := encoder.Encode(r); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	c.out <- buf.Bytes()

	return nil
}

// mutex must be held.
func (c *Chat) remove(user *User) bool {
	if _, has := c.ns[user.uid]; !has {
		return false
	}

	delete(c.ns, user.uid)

	return true
}

func timestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
