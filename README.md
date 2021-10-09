JoonOS system manager

# Introduction
The system manager is especially intended to obtain per-node
certificates which then can ben used to authenticate JoonOS nodes to
an MQTT server.

It is envisioned that the system manager may also publish telemetry to
the server, but this is yet to be defined.

# Design decisions
## Implement in Go
Go is not necessarily the prettiest programming language in existence,
but it is one option that meets these:

- Pointer arithmetic is not an essential tool in string manipulation
- Has pretty good support in Yocto
- Program + runtime (if any) does not take too much storage space
- Executing the program takes no more than few tens of MiB of RAM

## Subcommands insteaf of multiple executables
To support experimentation and potential additional use cases, the
repository needs to be able to provide multiple commands.

Since most of the extra functionality does not add much code but
likely does reuse the same runtime. Placing these alternative modes
into separate executables would make the space needed for the main
functionality slightly smaller, but adding a second mode of operation
could even double the storage space requirement.
