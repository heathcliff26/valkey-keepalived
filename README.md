[![CI](https://github.com/heathcliff26/valkey-keepalived/actions/workflows/ci.yaml/badge.svg?event=push)](https://github.com/heathcliff26/valkey-keepalived/actions/workflows/ci.yaml)
[![Coverage Status](https://coveralls.io/repos/github/heathcliff26/valkey-keepalived/badge.svg)](https://coveralls.io/github/heathcliff26/valkey-keepalived)
[![Editorconfig Check](https://github.com/heathcliff26/valkey-keepalived/actions/workflows/editorconfig-check.yaml/badge.svg?event=push)](https://github.com/heathcliff26/valkey-keepalived/actions/workflows/editorconfig-check.yaml)
[![Generate go test cover report](https://github.com/heathcliff26/valkey-keepalived/actions/workflows/go-testcover-report.yaml/badge.svg)](https://github.com/heathcliff26/valkey-keepalived/actions/workflows/go-testcover-report.yaml)
[![Renovate](https://github.com/heathcliff26/valkey-keepalived/actions/workflows/renovate.yaml/badge.svg)](https://github.com/heathcliff26/valkey-keepalived/actions/workflows/renovate.yaml)

# valkey-keepalived

Achieve simple high availability with valkey without needing to ensure you have enough nodes for a qorum. Use keepalived for ensuring availability and choosing the correct master node.

Does not ensure data integrity and can lead to split brain.

## Table of Contents

- [valkey-keepalived](#valkey-keepalived)
  - [Table of Contents](#table-of-contents)
  - [How does it work](#how-does-it-work)
  - [Container Images](#container-images)
    - [Image location](#image-location)
    - [Tags](#tags)
  - [Usage](#usage)
  - [Examples](#examples)
  - [What am i using it for](#what-am-i-using-it-for)
  - [Liability and warranty](#liability-and-warranty)

## How does it work

The failover is facilitaded through a simple method:
1. Check all the nodes if they are up and retrieve their unique "run_id"
2. Check the "run_id" for the valkey instance behind the keepalived IP
3. Promote that valkey instance to master
4. Ensure all other nodes are slaves of the new master

Since the answer to which valkey instance is behind the keepalived IP does not change, it does not matter how many instances of valkey-keepalived are doing this, as the result should always be the same.

## Container Images

### Image location

| Container Registry                                                                             | Image                              |
| ---------------------------------------------------------------------------------------------- | ---------------------------------- |
| [Github Container](https://github.com/users/heathcliff26/packages/container/package/valkey-keepalived) | `ghcr.io/heathcliff26/valkey-keepalived`   |
| [Docker Hub](https://hub.docker.com/r/heathcliff26/valkey-keepalived)                  | `docker.io/heathcliff26/valkey-keepalived` |

### Tags

There are different flavors of the image:

| Tag(s)      | Description                                                 |
| ----------- | ----------------------------------------------------------- |
| **latest**  | Last released version of the image                          |
| **rolling** | Rolling update of the image, always build from main branch. |
| **vX.Y.Z**  | Released version of the image                               |

## Usage

For deploying it simply run:
```
podman run -d -v config.yaml:/config/config.yaml ghcr.io/heathcliff26/valkey-keepalived
```

Available commands and flags:
```
$ valkey-keepalived help
valkey-keepalived failover a group of valkey databases based on a virtual ip

Usage:
  valkey-keepalived [flags]
  valkey-keepalived [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  version     Print version information and exit

Flags:
  -c, --config string   Path to config file
      --env             Expand enviroment variables in config file
  -h, --help            help for valkey-keepalived

Use "valkey-keepalived [command] --help" for more information about a command.
```

## Examples

An example configuration can be found [here](examples/config.yaml)

## What am i using it for

I'm using this in my homelab since i want to be able to scale down to 1 node for saving electricity, but still have easy HA when starting additional nodes.

I'm not using valkey in any application where i can't deal with a split brain or corrupted data.

## Liability and warranty

Normally it is enough that this is in the license, but since using this could lead to data loss, let me reiterate;

This project is provided as is, it comes with no warranty and i do not accept any liability for damages, problems, etc. caused by the use of this project. Use at your own risk.
