package internal
import (
    "github.com/derekparker/delve/proctl"
    "github.com/derekparker/delve/goreadline"
    "fmt"
    "io"
    "strings"
    "os"
    sys "golang.org/x/sys/unix"
)
const HistoryFile string = ".dbg_history"


func HandleExit(dbp *proctl.DebuggedProcess, status int, shouldConfirm bool) {
    errno := goreadline.WriteHistoryToFile(HistoryFile)
    if errno != 0 {
        fmt.Println("readline:", errno)
    }

    var answer string
    if shouldConfirm {
        prompt := "Would you like to kill the process? [y/n]"
        answerp := goreadline.ReadLine(&prompt)
        if answerp == nil {
            Die(2, io.EOF)
        }
        answer = strings.TrimSuffix(*answerp, "\n")
    }
    for _, bp := range dbp.HWBreakPoints {
        if bp == nil {
            continue
        }
        if _, err := dbp.Clear(bp.Addr); err != nil {
            fmt.Printf("Can't clear breakpoint @%x: %s\n", bp.Addr, err)
        }
    }

    for pc := range dbp.BreakPoints {
        if _, err := dbp.Clear(pc); err != nil {
            fmt.Printf("Can't clear breakpoint @%x: %s\n", pc, err)
        }
    }

    fmt.Println("Detaching from process...")
    err := sys.PtraceDetach(dbp.Process.Pid)
    if err != nil {
        Die(2, "Could not detach", err)
    }

    if answer == "y" || !shouldConfirm {
        fmt.Println("Killing process", dbp.Process.Pid)

        err := dbp.Process.Kill()
        if err != nil {
            fmt.Println("Could not kill process", err)
        }
    }

    Die(status, "Hope I was of service hunting your bug!")
}

func Die(status int, args ...interface{}) {
    fmt.Fprint(os.Stderr, args)
    fmt.Fprint(os.Stderr, "\n")
    os.Exit(status)
}

func ParseCommand(cmdstr string) (string, []string) {
    vals := strings.Split(cmdstr, " ")
    return vals[0], vals[1:]
}