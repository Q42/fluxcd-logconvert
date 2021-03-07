# Flux log convert
FluxCD v1 has no options to configure logging. Customizing the logging ingestion is cumbersome.
Instead we can pipe the logs of Flux through this tool, which converts it to valid StackDriver-ingestible JSON.

Take a look at the log streams:

- [before](./.snapshots/fluxlogs-source.ndjson)
- [after](./.snapshots/fluxlog-TestSnapshot.ndjson)

## Examples

```javascript
// this debug log
{ "somefield": "foo", "ts": "2021-03-06T21:14:44.870397172Z" }
// becomes
{"severity":"DEBUG","timestamp":"2021-03-06T21:14:44.870397172Z","message":"","serviceContext":{"service":"fluxcd"},"somefield":"foo"}

// this error
{ "err": "something went wrong", "test": "foo", "caller": "main.go:20" }
// becomes
{"severity":"ERROR","timestamp":null,"message":"something went wrong","serviceContext":{"service":"fluxcd"},"caller":"main.go:20","err":"something went wrong","test":"foo","logging.googleapis.com/sourceLocation":{"file":"main.go","line":"20","function":"unknown"}}
```

## Commands
```bash
go build ./
go test ./
```
