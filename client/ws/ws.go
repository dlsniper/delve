package ws

import (
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "os/signal"
    sys "golang.org/x/sys/unix"
    . "github.com/derekparker/delve/client/internal"
    "github.com/derekparker/delve/command"
    "github.com/derekparker/delve/goreadline"
    "github.com/derekparker/delve/proctl"
    "github.com/gorilla/websocket"
    "bufio"
    "log"
    "sync"
)

const historyFile string = ".dbg_history"

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

type ConnectionHandler struct {
    mu  sync.Mutex
    connectionCount   int
}
//connection limiter to one client
func (ch *ConnectionHandler) shouldReject(w *http.ResponseWriter) bool {
    ch.mu.Lock()
    ch.connectionCount++
    shouldReject := false
    if ch.connectionCount>1 {
        (*w).WriteHeader(429)

        shouldReject= true
    }
    ch.mu.Unlock()
    return shouldReject
}

func wsHandler(stdoutReadFd *os.File, commandChan chan <- string) http.HandlerFunc {
    var connectionHandler ConnectionHandler

    return func(w http.ResponseWriter, r *http.Request) {

        //reject more than one connection
        if connectionHandler.shouldReject(&w) {
            return
        }

        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            Die(1, fmt.Sprintf("%q", err))
        }
        err = conn.WriteMessage(websocket.TextMessage, []byte("Conected to DLV debugger"))
        if err != nil {
            Die(1, fmt.Sprintf("%q", err))
        }
        go func() {
            for {
                stoutScanner := bufio.NewScanner(stdoutReadFd)
                for stoutScanner.Scan() {
                    err = conn.WriteMessage(websocket.TextMessage, []byte(stoutScanner.Text()+"\n"))
                    if err != nil {
                        Die(1, fmt.Sprintf("%q", err))

                    }
                }
            }
        }()
        //listens for incoming commands
        for {
            _, message, err := conn.ReadMessage()
            if err != nil {
                Die(1, fmt.Sprintf("%q", err))
            }
            cmdstr := string(message)

            if cmdstr != "" {
                goreadline.AddHistory(cmdstr)
            }
            commandChan <- cmdstr
        }

    }
}
func Run(run bool, pid int, address string, args []string) {
    var (
        dbp *proctl.DebuggedProcess
        err error
        stdoutReadFd *os.File
        stdoutWriteFd *os.File
        oldStout *os.File
        commandChan chan string
    )
    //route stdout to our pipe
    stdoutReadFd, stdoutWriteFd, _ = os.Pipe()
    oldStout = os.Stdout
    os.Stdout = stdoutWriteFd
    //transfer stdout to wsHandler
    //transfer commands from wsHandler to commands hanlder
    commandChan = make(chan string)

    cleanup := func() {
        os.Stdout=oldStout
        close(commandChan)
        stdoutReadFd.Close()
        stdoutWriteFd.Close()
    }
    defer cleanup()


    switch {
        case run:
        const debugname = "debug"
        cmd := exec.Command("go", "build", "-o", debugname, "-gcflags", "-N -l")
        err := cmd.Run()
        if err != nil {
            Die(1, "Could not compile program:", err)
        }
        defer os.Remove(debugname)
        dbp, err = proctl.Launch(append([]string{"./" + debugname}, args...))
        if err != nil {
            Die(1, "Could not launch program:", err)
        }
        case pid != 0:
        dbp, err = proctl.Attach(pid)
        if err != nil {
            Die(1, "Could not attach to process:", err)
        }
        default:
        dbp, err = proctl.Launch(args)
        if err != nil {
            Die(1, "Could not launch program:", err)
        }
    }

    ch := make(chan os.Signal)
    signal.Notify(ch, sys.SIGINT)
    go func() {
        for _ = range ch {
            if dbp.Running() {
                dbp.RequestManualStop()
            }
            cleanup()
            os.Exit(0)
        }
    }()

    //spawn websocket server
    go func() {
        log.Print("Listening on: "+address)
        http.HandleFunc("/", wsHandler(stdoutReadFd, commandChan))
        log.Fatalf("Error: %q", http.ListenAndServe(address, nil))
    }()

    //commands handler, has to be in the main process due to thread registry lookup
    cmds := command.DebugCommands()
    for {
        cmdstr, args := ParseCommand(<-commandChan)
        if cmdstr == "exit" {
            cleanup()
            HandleExit(dbp, 0, false)
        }
        cmd := cmds.Find(cmdstr)

        err = cmd(dbp, args...)
        if err != nil {
            fmt.Println(fmt.Sprintf("Command failed: %s\n", err))
        }
    }

}
