# Flux log convert
FluxCD v1 has no options to configure logging. Customizing the logging ingestion is cumbersome.
Instead we can pipe the logs of Flux through this tool, which converts it to valid StackDriver-ingestible JSON.

Take a look at the log streams:

- [before](./.snapshots/fluxlogs-source.ndjson)
- [after](./.snapshots/fluxlog-TestSnapshot.ndjson)

## Commands
```bash
go build ./
go test ./
```
