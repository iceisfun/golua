# exec — Process Execution Module

The `exec` module provides modern process execution capabilities for GoLua,
going beyond `os.execute` with support for streaming I/O, stdin interaction,
and timed waits.

## Quick Start

```lua
-- Simple execution
local result = exec.run("ls", "-al")
print(result.stdout)

-- Streaming
local p = exec.spawn("ping", "-c", "5", "example.com")
for line in p:readlines() do
    print(line)
end
p:wait()
```

## Module Functions

### exec.run(cmd, args..., [opts])

Runs a command synchronously and returns the result. An optional options table
can be passed as the last argument.

```lua
local result = exec.run("echo", "hello")
-- result.success = true
-- result.code    = 0
-- result.stdout  = "hello\n"
-- result.stderr  = ""

-- Merge stderr into stdout
local r = exec.run("sh", "-c", "echo out; echo err >&2", {merge_stderr = true})
-- r.stdout contains both "out\n" and "err\n"
-- r.stderr is ""
```

**Returns** a table with:
| Field     | Type    | Description                    |
|-----------|---------|--------------------------------|
| `success` | boolean | true if exit code is 0         |
| `code`    | number  | exit code                      |
| `stdout`  | string  | captured standard output       |
| `stderr`  | string  | captured standard error        |

### exec.spawn(cmd, args..., [opts])

Spawns a process and returns a process handle with stdin, stdout, and stderr
pipes. The process runs asynchronously. An optional options table can be
passed as the last argument.

```lua
local p = exec.spawn("sort")
p:write("b\na\n")
p:close_stdin()
print(p:readline())  -- "a"
p:wait()

-- Merge stderr into stdout for unified streaming
local p2 = exec.spawn("sh", "-c", "echo out; echo err >&2", {merge_stderr = true})
for line in p2:readlines() do
    print(line)  -- prints both "out" and "err"
end
p2:wait()
```

### exec.run_shell(cmdline, [opts])

Runs a command through the system shell (`sh -c`), enabling pipes, redirects,
and variable expansion. An optional options table can be passed as the second
argument.

```lua
local result = exec.run_shell("echo hello | tr a-z A-Z")
print(result.stdout)  -- "HELLO\n"

-- Merge stderr into stdout
local r = exec.run_shell("make 2>&1", {merge_stderr = true})
```

## Process Object Methods

Process objects are returned by `exec.spawn`.

### p:read()

Reads available data from stdout. Returns a string, or nil at EOF.

### p:readline()

Reads a single line from stdout (strips trailing newline). Returns nil at EOF.

### p:readlines()

Returns an iterator for streaming stdout line by line.

```lua
for line in p:readlines() do
    print(line)
end
```

### p:write(data)

Writes data to the process's stdin. Returns the process for method chaining.

### p:close_stdin()

Closes the stdin pipe. Required for processes that read until EOF (e.g., `sort`).

### p:wait([ms])

Without arguments, blocks until the process completes and returns a result table.

With a timeout in milliseconds, returns `result, done` where `done` is false
if the timeout expired.

```lua
-- Blocking
local result = p:wait()

-- With timeout
local result, done = p:wait(5000)
if not done then
    p:kill()
end
```

### p:is_complete()

Returns true if the process has finished.

### p:kill()

Sends SIGKILL to the process and all its children (process group kill).

### p:exit_code()

Returns the exit code, or nil if the process hasn't completed.

### p:stderr()

Reads a line from stderr. Returns nil at EOF.

## Options Table

All module functions accept an optional options table as the last argument:

| Field          | Type    | Default | Description                          |
|----------------|---------|---------|--------------------------------------|
| `merge_stderr` | boolean | false   | Merge stderr into stdout (like 2>&1) |
| `cwd`          | string  | inherit | Working directory for the process     |
| `env`          | table   | inherit | Environment variables (replaces all)  |
| `timeout`      | number  | none    | Timeout in milliseconds              |

### merge_stderr

When `true`, all stderr output is interleaved into stdout. The `stderr`
field in result tables will be empty, and `p:stderr()` on spawned processes
will return nil.

```lua
local r = exec.run("make", "all", {merge_stderr = true})
print(r.stdout)  -- contains both stdout and stderr
```

### cwd

Sets the working directory for the spawned process.

```lua
local r = exec.run("ls", {cwd = "/tmp"})
print(r.stdout)
```

### env

Replaces the process's entire environment. Keys and values must be strings.
If not set, the process inherits the parent's environment.

```lua
local r = exec.run("sh", "-c", "echo $GREETING", {
    env = {GREETING = "hello", PATH = "/usr/bin:/bin"}
})
print(r.stdout)  -- "hello\n"
```

### timeout

Kills the process (and all its children) if it doesn't complete within the
given number of milliseconds. For `exec.run` and `exec.run_shell`, the
function returns with `success = false`. For `exec.spawn`, the process is
automatically killed when the timeout expires.

```lua
-- Run with 5-second timeout
local r = exec.run("slow_command", {timeout = 5000})
if not r.success then
    print("timed out or failed")
end

-- Spawn with timeout
local p = exec.spawn("long_task", {timeout = 10000})
local result = p:wait()  -- returns when done or killed by timeout
```

### Combining options

```lua
local r = exec.run("make", "all", {
    cwd = "/path/to/project",
    env = {PATH = "/usr/bin:/bin", CC = "gcc"},
    merge_stderr = true,
    timeout = 30000
})
```

## Provider Architecture

The exec module requires a `LuaProcessProvider` to be set on the VM.

### Go Setup

```go
v := vm.New()
if err := v.SetProcessProvider(vm.NewDefaultProcessProvider()); err != nil {
    log.Fatal(err)
}
stdlib.Open(v)
```

### Custom Provider

Implement `LuaProcessProvider` to control process execution:

```go
type LuaProcessProvider interface {
    Spawn(ctx context.Context, cmd string, args []string, opts ProcessOptions) (LuaProcess, error)
}
```

This allows hosts to:
- Restrict which commands can be executed
- Sandbox working directories
- Filter environment variables
- Log process usage
- Disable execution entirely (don't set the provider)

### ProcessOptions

```go
type ProcessOptions struct {
    Env         map[string]string  // nil = inherit parent
    Dir         string             // empty = inherit parent
    Stdin       bool               // create stdin pipe
    Stdout      bool               // capture stdout
    Stderr      bool               // capture stderr
    MergeStderr bool               // merge stderr into stdout
}
```

## Security

- The exec module is **disabled by default**. It is only available when a
  `ProcessProvider` is registered on the VM.
- `DefaultProcessProvider` uses `os/exec` with context cancellation support.
- Hosts can implement custom providers to restrict, sandbox, or audit process
  execution.

## Comparison with os.execute

| Feature           | os.execute              | exec module              |
|-------------------|------------------------|--------------------------|
| Output capture    | No                     | Yes (stdout + stderr)    |
| Stdin interaction | No                     | Yes                      |
| Streaming         | No                     | Yes (readlines iterator) |
| Timed wait        | No                     | Yes (wait with timeout)  |
| Process control   | No                     | Yes (kill, is_complete)  |
| Merge stderr      | No                     | Yes (merge_stderr opt)   |
| Working directory | No                     | Yes (cwd option)         |
| Custom env        | No                     | Yes (env option)         |
| Auto-timeout      | No                     | Yes (timeout option)     |
| Shell mode        | Always                 | Optional (run_shell)     |
| Return value      | ok, exittype, code     | Result table             |
| Provider          | LuaExecProvider        | LuaProcessProvider       |
