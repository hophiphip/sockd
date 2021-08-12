package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/gorilla/websocket"
)

const Port = ":8080"

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

func WsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != "http://"+r.Host {
		http.Error(w, "Incorrect host origin", 403)
		return
	}

	conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
	if err != nil {
		http.Error(w, "Failed to open websocket connection", http.StatusBadRequest)
	}

	go Log(conn)
}

func Log(conn *websocket.Conn) {
	ws, err := newProcess("./sub.sh")
	if err != nil {
		log.Fatal(err)
	}

	defer ws.Close()

	reader := bufio.NewReader(ws.Stdout)

	// Handle stdout
	go func(reader io.Reader) {
		scanner := bufio.NewScanner(reader)

		for scanner.Scan() {
			if err = conn.WriteJSON(WsMessage{
				Type: Stdout,
				Arg:  fmt.Sprintf("[%s] %s\n", time.Now().Format(time.RFC850), scanner.Text()),
			}); err != nil {
				log.Println("Failed to send JSON")
			}
		}
	}(reader)

	// Handle stdin
	go func() {
		for {
			message := WsMessage{}

			err := conn.ReadJSON(&message)
			if err != nil {
				log.Println("Error parsing JSON: ", err)
			}

			if message.Type == Stdin {
				log.Println("Received: ", message.Arg)

				ws.Stdin.Write([]byte(message.Arg + "\n"))
			} else {
				log.Println("Wrong message type: ", message.Type)
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

func main() {

	http.HandleFunc("/ws", WsHandler)
	http.HandleFunc("/", serveHtml)

	log.Printf("Starting server on %s\n", Port)

	log.Fatal(http.ListenAndServe(Port, nil))
}

var html = []byte(`
<html>
<head>
    <title>WebSocket demo</title>
</head>
<body>

    <div>
        <form>
            <label for="stdin">Stdin</label>
            <input type="text" id="stdinfield"/><br />
            <button type="button" id="sendBtn">Send</button>
        </form>
    </div>
    <div id="container"></div>

    <script type="text/javascript" src="http://ajax.googleapis.com/ajax/libs/jquery/1.10.2/jquery.min.js"></script>
    <script type="text/javascript">
        $(function () {
            var ws;

            if (window.WebSocket === undefined) {
                $("#container").append("Your browser does not support WebSockets");
                return;
            } else {
                ws = initWS();
            }

            function initWS() {
                var socket = new WebSocket("ws://localhost:8080/ws"),
                    container = $("#container")
                socket.onopen = function() {
                    container.append("<p>Socket is open</p>");
                };
                socket.onmessage = function (e) {
										var parsed = JSON.parse(e.data)
                    container.append("<p>" + parsed["arg"] + "</p>");
                }
                socket.onclose = function () {
                    container.append("<p>Socket closed</p>");
                }

                return socket;
            }

            $("#sendBtn").click(function (e) {
                e.preventDefault();

                ws.send(JSON.stringify({
										type: "stdin",
										arg: $("#stdinfield").val(), 
								}));
            });
        });
    </script>
</body>
</html>
`)

func serveHtml(w http.ResponseWriter, r *http.Request) {
	w.Write(html)
}
