package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	sys "golang.org/x/sys/unix"

	. "github.com/derekparker/delve/client/internal"
	"github.com/derekparker/delve/command"
	"github.com/derekparker/delve/goreadline"
	"github.com/derekparker/delve/proctl"
)


func Run(run bool, pid int, args []string) {
	var (
		dbp *proctl.DebuggedProcess
		err error
	)

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
		}
	}()

	cmds := command.DebugCommands()
	goreadline.LoadHistoryFromFile(HistoryFile)
	fmt.Println("Type 'help' for list of commands.")

	for {
		cmdstr, err := promptForInput()
		if err != nil {
			if err == io.EOF {
				HandleExit(dbp, 0, true)
			}
			Die(1, "Prompt for input failed.\n")
		}

		cmdstr, args := ParseCommand(cmdstr)

		if cmdstr == "exit" {
			HandleExit(dbp, 0, true)
		}

		cmd := cmds.Find(cmdstr)
		err = cmd(dbp, args...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Command failed: %s\n", err)
		}
	}
}


func promptForInput() (string, error) {
	prompt := "(dlv) "
	linep := goreadline.ReadLine(&prompt)
	if linep == nil {
		return "", io.EOF
	}
	line := strings.TrimSuffix(*linep, "\n")
	if line != "" {
		goreadline.AddHistory(line)
	}

	return line, nil
}
