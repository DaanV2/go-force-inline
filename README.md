# go-force-inline

A Go CLI tool that generates synthetic PGO (Profile-Guided Optimization) profiles from source code directives. Annotate call sites with `//pgogen:hot` comments to force the Go compiler's PGO inliner to treat them as hot, increasing the inlining budget from 80 to 2000 AST nodes.

> [!IMPORTANT]
> NOTE: While this tool can help with speeding up, some notes:

1. The tool generates synthetic profiles, This can't really replace real PGO profiles, and is potentially alot more work.
   1. Whoevere with fast moving codebases, this can help with keeping up fast
   2. It can collide with real PGO profiles, so be careful when using both
2. The tool is not perfect, and can miss some call sites, or generate invalid profiles


## Setup

Add the tool to your project:

```bash
go get -tool github.com/daanv2/go-force-inline@latest
```

## Usage

### Annotate your code

Place `//pgogen:hot` on the line immediately above a function call:

```go
func handler(r *Request) *Response {
    //pgogen:hot weight=10000
    result := processRequest(r)

    //pgogen:hot
    validated := validateInput(r.Body)

    return buildResponse(result, validated)
}
```

The `weight` parameter is optional (default: `10000`).

### Generate the profile

```bash
go tool github.com/daanv2/go-force-inline generate ./...
```

| Flag           | Description                               |
| -------------- | ----------------------------------------- |
| `-o, --output` | Output file path (default: `default.pgo`) |

### Build with PGO

The Go toolchain auto-detects `default.pgo` in the module root:

```bash
go build ./...
```

Or specify it explicitly:

```bash
go build -pgo=default.pgo ./...
```

### Verify a profile

Inspect which edges in a profile are considered hot:

```bash
go tool github.com/daanv2/go-force-inline verify default.pgo
```

| Flag          | Description                                    |
| ------------- | ---------------------------------------------- |
| `--threshold` | CDF hot threshold percentage (default: `99.0`) |

Output:

```
Edge                                          Weight    CDF%   Hot?
----                                          ------    ----   ----
main.handler → main.processRequest           10000    50.0%   yes
main.handler → main.validateInput            10000   100.0%   yes

Total weight: 20000
Hot threshold: 99% (cumulative weight < 19800)
```

### Global flags

| Flag                  | Description                                                            |
| --------------------- | ---------------------------------------------------------------------- |
| `--log-level`         | Log level: `debug`, `info`, `warn`, `error`, `fatal` (default: `info`) |
| `--log-format`        | Log format: `text`, `json`, `logfmt` (default: `text`)                 |
| `--log-file`          | Write logs to a file                                                   |
| `--log-report-caller` | Include source file in log output                                      |

## Warnings

The tool warns about situations where forced inlining won't work:

- Directive not followed by a call expression
- Callee has `//go:noinline` (PGO won't override it)
- Interface method calls (cannot be inlined)
- Calls inside closures (fragile naming)
- Chained calls on one line (ambiguous target)
