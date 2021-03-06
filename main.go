package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"text/template"
	"time"

	_ "embed"

	"github.com/gorilla/websocket"
)

const (
	StdoutTimeout = 10 * time.Minute
)

type TemplatePageData struct {
	Title  string
	WsHost string
}

type StreamType string

const (
	None   StreamType = "none"
	Stdin  StreamType = "stdin"
	Stdout StreamType = "stdout"
	Stderr StreamType = "stderr"
)

type WsMessage struct {
	Type StreamType `json:"type"`
	Arg  string     `json:"arg"`
}

type WsProcess struct {
	Cmd    *exec.Cmd
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
	Stderr io.ReadCloser
}

var Address = flag.String("address", "127.0.0.1", "sockd service ip address")
var Port = flag.Int("port", 8080, "sockd service port")
var Script = flag.String("script", "ls", "path to script or executable for sockd service to run")

//go:embed index.html
var htmlContents string
var htmlTemplate = template.Must(template.New("").Parse(htmlContents))

func newProcess(command string, args ...string) (*WsProcess, error) {
	cmd := exec.Command(command, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	return &WsProcess{
		Cmd:    cmd,
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}, nil
}

func (ws *WsProcess) Close() {
	ws.Stdin.Close()
	ws.Stdout.Close()
	ws.Stderr.Close()
}

func (ws *WsProcess) Start() error {
	if err := ws.Cmd.Start(); err != nil {
		return err
	} else {
		return nil
	}
}

func (ws *WsProcess) Wait() error {
	if err := ws.Cmd.Wait(); err != nil {
		return err
	} else {
		return nil
	}
}

// WsProcess - handles websocket connections
func WsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != "http://"+r.Host {
		http.Error(w, "Incorrect host origin", 403)
		return
	}

	conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
	if err != nil {
		http.Error(w, "Failed to open websocket connection", http.StatusBadRequest)
	}

	defer conn.Close()

	Log(conn)
}

// Log - Stdin/Stdout/Stderr websocket logger
func Log(conn *websocket.Conn) {
	ws, err := newProcess(*Script)
	if err != nil {
		log.Fatal(err)
	}

	defer ws.Close()

	reader := bufio.NewReader(ws.Stdout)

	// Stdout timeout
	conn.SetReadDeadline(time.Now().Add(StdoutTimeout))

	// Handle stdout
	//		Stdout -> sockd -> Browser -> Stdout
	//
	go func(reader io.Reader) {
		scanner := bufio.NewScanner(reader)

		for scanner.Scan() {
			if err = conn.WriteJSON(WsMessage{
				Type: Stdout,
				Arg:  fmt.Sprintf("[%s] %s\n", time.Now().Format(time.RFC850), scanner.Text()),
			}); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("JSON Write error: %v\n", err)
				}
			}
		}
	}(reader)

	// Handle stdin
	//		Stdin -> Browser -> sockd -> Stdin
	//
	go func() {
		for {
			message := WsMessage{}

			err := conn.ReadJSON(&message)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("JSON Read error: %v", err)
				}
				break
			}

			if message.Type == Stdin {
				log.Println("Received: ", message.Arg)

				ws.Stdin.Write([]byte(message.Arg + "\n"))
			} else {
				log.Printf("Wrong message type: %s : Arg: %s", message.Type, message.Arg)
			}
		}
	}()

	if err = ws.Start(); err != nil {
		log.Fatal(err)
	}

	if err = ws.Wait(); err != nil {
		log.Fatal(err)
	}
}

// HtmlHandler - sample websocket html client
func HtmlHandler(w http.ResponseWriter, r *http.Request) {
	data := TemplatePageData{
		Title:  "Sockd client",
		WsHost: "ws://" + r.Host + "/ws",
	}

	htmlTemplate.Execute(w, data)
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	http.HandleFunc("/ws", WsHandler)
	http.HandleFunc("/", HtmlHandler)

	log.Printf("Starting server on %s:%d\n", *Address, *Port)

	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", *Address, *Port), nil))
}
