# Proxying

## Overview

Sometimes, you may want to produce backup data on a remote host or store
backup data to a remote host while still keeping `uback` configuration
on a third host. For example, you may want to use a `btrfs` source
on a remote host to produce the backup but store it locally on a `fs`
destination, or use a `tar` source locally and use a `fs` destination
on another host. You may combine both requirements, and on host A
get a backup from a `btrfs` source from host B and store it in a `fs`
destination on host C.

Proxying means using another `uback` process to provide the source and/or
the destination while doing a backup. The other `uback` process may run
on another user, or in a container, or a remote host.

Note that encryption and compression is done on the local process (not
the remote one). Also, restoring with proxy is not supported ; you must
use a direct source.

## Usage

For both sources and destinations :

1. Use `proxy` as a source or destination type.

2. Spawn the other `uback` instance by setting the `command` option to
`uback proxy`.

3. Specify the proxyfied `type` and/or `command` option by prefixying
it with `proxy-`.

## Examples

Proxy a custom destination using ssh :

```
type=proxy,command="ssh root@example.com uback proxy",proxy-type=command,proxy-command=uback-custom-dest
```

Proxy a `btrfs` source using sudo :

```
type=proxy,command="sudo uback proxy",proxy-type=btrfs
```
