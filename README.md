Yeah its just [`time`](https://man7.org/linux/man-pages/man1/time.1.html) but simpler and cross-platform. Works on windows, linux, mac, etc.

Get the latest release from [here](https://github.com/coalaura/time/releases/latest).

Or install it with one command:
```bash
curl -sL https://src.ws2.sh/time/install.sh | sh
```

## Usage

```
time [-f|--full] [-e|--explain] [-v|--version] <command> [args...]
```

Default output matches GNU time:
```
real    0m0.007s
user    0m0.015s
sys     0m0.000s
```

## Flags

| Flag | Description |
|------|-------------|
| `-f`, `--full` | Show detailed stats (memory, I/O, context switches) |
| `-e`, `--explain` | Explain what each stat means |
| `-v`, `--version` | Print version |

Flags can be combined: `-fe`, `-ef`, etc.

## Examples

Full output:
```
real    7ms 763µs
setup   2ms 576µs
exec    5ms 186µs
user    15ms 625µs
sys     0s

peakws  7.4MB
pageflt 1984

reads   1 (128KB)
```

With explanations:
```
real    7ms 763µs
  Wall-clock time from start to finish
setup   2ms 576µs
  Time to create process and set up pipes
...
```

## Platform differences

| Stat | Linux | Windows | Other |
|------|:-----:|:-------:|:-----:|
| **Time** | | | |
| real | ✓ | ✓ | ✓ |
| setup | ✓ | ✓ | ✓ |
| exec | ✓ | ✓ | ✓ |
| user | ✓ | ✓ | ✓ |
| sys | ✓ | ✓ | ✓ |
| **Memory** | | | |
| maxrss | ✓ | - | - |
| peakws | - | ✓ | - |
| minflt | ✓ | - | - |
| majflt | ✓ | - | - |
| pageflt | - | ✓ | - |
| **I/O** | | | |
| inblock | ✓ | - | - |
| oublock | ✓ | - | - |
| syscr | ✓ | - | - |
| syscw | ✓ | - | - |
| read | ✓ | - | - |
| write | ✓ | - | - |
| rchar | ✓ | - | - |
| wchar | ✓ | - | - |
| reads | - | ✓ | - |
| writes | - | ✓ | - |
| **Context switches** | | | |
| nvcsw | ✓ | - | - |
| nivcsw | ✓ | - | - |
