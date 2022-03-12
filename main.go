package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gobwas/ws"
	"github.com/mailru/easygo/netpoll"

	"net/http"
	_ "net/http/pprof"
)

var (
	addr      = flag.String("listen", ":3333", "address to bind to")
	debug     = flag.String("pprof", "", "address for pprof http")
	workers   = flag.Int("workers", 128, "max workers count")
	queue     = flag.Int("queue", 1, "workers task queue size")
	ioTimeout = flag.Duration("io_timeout", time.Millisecond*100, "i/o operations timeout")
)

func main() {
	flag.Parse()

	// Initialize netpoll instance. We will use it to be noticed about incoming
	// events from listener of user connections.
	poller, err := netpoll.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	var (
		// Make pool of X size, Y sized work queue and one pre-spawned
		// goroutine.
		pool = NewPool(*workers, *queue, 1)
		chat = NewChat(pool)
		exit = make(chan struct{})
	)
	// handle is a new incoming connection handler.
	// It upgrades TCP connection to WebSocket, registers netpoll listener on
	// it and stores it as a chat user in Chat instance.
	//
	// We will call it below within accept() loop.
	handle := func(conn net.Conn, uid string) {
		// NOTE: we wrap conn here to show that ws could work with any kind of
		// io.ReadWriter.
		safeConn := deadliner{conn, *ioTimeout}

		// Zero-copy upgrade to WebSocket connection.
		//hs, err := ws.Upgrade(safeConn)
		//if err != nil {
		//	log.Printf("%s: upgrade error: %v", nameConn(conn), err)
		//	connconn.Close()
		//	return
		//}

		//	log.Printf("%s: established websocket connection: %+v", nameConn(conn), hs)

		// Register incoming user in chat.
		user := chat.Register(safeConn, uid)

		// Create netpoll event descriptor for conn.
		// We want to handle only read events of it.
		desc := netpoll.Must(netpoll.HandleRead(conn))

		// Subscribe to events about conn.
		poller.Start(desc, func(ev netpoll.Event) {
			if ev&(netpoll.EventReadHup|netpoll.EventHup) != 0 {
				// When ReadHup or Hup received, this mean that client has
				// closed at least write end of the connection or connections
				// itself. So we want to stop receive events about such conn
				// and remove it from the chat registry.
				log.Println("fuckyou33")
				poller.Stop(desc)
				chat.Remove(user)
				return
			}
			// Here we can read some new message from connection.
			// We can not read it right here in callback, because then we will
			// block the poller's inner loop.
			// We do not want to spawn a new goroutine to read single message.
			// But we want to reuse previously spawned goroutine.
			pool.Schedule(func() {
				if err := user.Receive(); err != nil {
					// When receive failed, we can only disconnect broken
					// connection and stop to receive events about it.
					log.Println("fuckyou22")
					poller.Stop(desc)
					chat.Remove(user)
				}
			})
		})
	}

	// Create incoming connections listener.
	//ln, err := net.Listen("tcp", *addr)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//log.Printf("websocket is listening on %s", ln.Addr().String())

	http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			err := conn.Close()
			if err != nil {
				return
			}
			return
			// handle error
		}
		a1 := r.URL.Query()
		a2, ok := a1["myid"]
		if !ok {
			fmt.Println("error user")
			err := conn.Close()
			if err != nil {
				return
			}
			return
		}
		a3 := a2[0]
		fmt.Println(r.RemoteAddr)
		handle(conn, a3)
	}))

	<-exit
}

func nameConn(conn net.Conn) string {
	return conn.LocalAddr().String() + " > " + conn.RemoteAddr().String()
}

// deadliner is a wrapper around net.Conn that sets read/write deadlines before
// every Read() or Write() call.
type deadliner struct {
	net.Conn
	t time.Duration
}

func (d deadliner) Write(p []byte) (int, error) {
	if err := d.Conn.SetWriteDeadline(time.Now().Add(d.t)); err != nil {
		return 0, err
	}
	return d.Conn.Write(p)
}

func (d deadliner) Read(p []byte) (int, error) {
	if err := d.Conn.SetReadDeadline(time.Now().Add(d.t)); err != nil {
		return 0, err
	}
	return d.Conn.Read(p)
}
