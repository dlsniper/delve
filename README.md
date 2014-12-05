# Delve

### What is Delve?

Delve is a (Beta) Go debugger, written in Go.

This project is currently in beta. Most of the functionality is there, but there are various improvements to be made.

### Building

Delve requires Go 1.4 to build.

```
go get github.com/derekparker/delve/cmd/dlv
```

You will need readline installed on your system. With apt simply: `sudo apt-get install libreadline-dev` .

### Features

* Attach to an already running process
* Launch a process and begin debug session
* Set breakpoints, single step, step over functions, print variable contents, print thread and goroutine information

### Usage

The debugger can be launched in three ways:

* Compile, run, and attach in one step:

	```
	$ dlv -run
	```

* Provide the name of the program you want to debug, and the debugger will launch it for you.

	```
	$ dlv path/to/program
	```

* Provide the pid of a currently running process, and the debugger will attach and begin the session.

	```
	$ sudo dlv -pid 44839
	```

### Breakpoints

Delve can insert breakpoints via the `breakpoint` command once inside a debug session, however for ease of debugging, you can also call `runtime.Breakpoint()` and Delve will handle the breakpoint and stop the program at the next source line.

### Commands

Once inside a debugging session, the following commands may be used:

* `break` - Set break point at the entry point of a function, or at a specific file/line. Example: `break foo.go:13`.

* `continue` - Run until breakpoint or program termination.

* `step` - Single step through program.

* `next` - Step over to next source line.

* `threads` - Print status of all traced threads.

* `goroutines` - Print status of all goroutines.

* `print $var` - Evaluate a variable.

* `exit` - Exit the debugger.


### Upcoming features

* In-scope variable setting
* Support for OS X
* Editor integration

### License

MIT
