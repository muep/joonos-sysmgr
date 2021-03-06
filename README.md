Joonos system manager

# Introduction
The system manager is a service whose primary tasks are:
- Obtain per-node certificates which then can ben used to authenticate
  Joonos nodes to an MQTT server.
- Download and apply system updates

In addition to these, the tool features some limited telemetry features.

Development of the program is hosted on
[Github](https://github.com/muep/joonos-sysmgr)

# Design decisions
## Layer over MQTT
This arises from the idea of reusing some infrastructure in use cases
where MQTT is used anyway.

It might be reasonably simple to add some other mechanism directly
over TLS.

## JSON over MQTT in early messages
Some other representations for data are more efficient and possibly
more robust as well, but at least in the initial stages JSON has
some advantages that stem from its convenient extensibility.

Another thing convenient about JSON-formatted messages is that they
can easily be produced with a generic text editor and sent e.g.
with `mosquitto_pub`.

## Implement in Go
Go is not necessarily the prettiest programming language in existence,
but it is one option that meets these:

- Pointer arithmetic is not an essential tool in string manipulation
- Has pretty good support in Yocto
- Program + runtime (if any) does not take too much storage space
- Executing the program takes no more than few tens of MiB of RAM

## Subcommands instead of multiple executables
To support experimentation and potential additional use cases, the
repository needs to be able to provide multiple commands.

Most of the extra functionality does not add much code. Placing these
alternative modes into separate executables would make the space
needed for the main functionality slightly smaller, but adding a
second mode of operation would lead to additional executables which
then would need to bundle the Go runtime, or at least the libraries
for MQTT connectivity.

Deploying as a single executable is also convenient from point of view
of writing bitbake recipes or other deployment tooling.

## Include at least a minimal certificate authority
This may help with automated testing, at least. The mechanism is
simple enough that a more complete implementation can be written
separately.
