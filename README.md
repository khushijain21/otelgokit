# otelgokit
otelgokit is a bridge between go-kit-log and OTel logs


## Some Notes 
go-kit-log supports structured key-value logging.

- Timestamp is extracted if the value is of type `time.Time`
- logging level is extracted only if the key matches the keyword "level"

    Currently, the following levels are supported. You may clone to add/remove/customize it according to your requirement. Follow this piece of (https://github.com/khushijain21/otelgokit/blob/7e45608feef0741c904bc6222624f3e3379602d8/log.go#L135

    Supported Levels:
    - "debug"
    - "info"
    - "warn"
    - "error"
    - "panic"
    - "fatal"
- This bridge also supports context passing (for trace correlation). 

It currently does not support minimum level logging
## Example

```go
// Use a working LoggerProvider implementation instead e.g. use go.opentelemetry.io/otel/sdk/log.
provider := noop.NewLoggerProvider()

logger := NewOTelLogger("testLog", provider)

// You can set context for trace correlation 
ctx := context.Background()

// logs at info level
logger.Log("ctx", ctx, "level", "info", testKey, testValue)  

```
