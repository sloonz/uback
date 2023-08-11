<p align="center">
  <a href="https://github.com/sloonz/uback/actions/workflows/check.yml"><img alt="CI status" src="https://github.com/sloonz/uback/actions/workflows/check.yml/badge.svg"></a>
  <a href="https://goreportcard.com/report/github.com/sloonz/uback"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/sloonz/uback"></a>
</p>

# uback

There are many backup tools out there. Or should I say, there are may
backup *scripts* out there. Most of them will focus on a specific usage
(say, backup `btrfs` snapshots to a `sftp` server). All of them will
have different feature matrices (encryption/incremental backups/pruning)
and different means of configuration/deployment.

![Situation: there is gogolplex backup tools...](https://imgs.xkcd.com/comics/standards.png)

`uback` try to solve this by defining a simple model of backup producers
(called "sources") and backup stores (called "destinations"), and writing
the intermediate between them once for all. One can now focus on writing
each specific source (`tar`, `mysqldump`, `btrfs`,...) or destination
(`s3`, `sftp`, local files,...) in a relatively straightforward
way. After that, from the point of view of the system administrator,
the configuration can be made in a uniform way accros different backup
workflows.

Key features are encryption, compression, incremental backups (if the
source allow it) and retention policy.

## Quickstart and Documentation

All the documentation is in the [doc/](doc/) directory. You should start
by the [tutorial](doc/tutorial.md) and then jump to advanced topics ([file
format](doc/file-format.md), [custom sources](doc/custom-sources.md),
[custom destinations](doc/custom-destinations.md)) or the documentation
specific to each source or destination.

## Supported Sources

* [tar](doc/src-tar.md)
* [mariabackup](doc/src-mariabackup.md): MariaDB backup system, supports
incremental backups

## Supported Destintations

* [fs](doc/dest-fs.md): local filesystem
* [object-storage](doc/dest-object-storage.md): S3-compatible object
storage

## Current Status and Planned Features

`uback` is in a preliminary stage, quite lacking feature-wide, but fairly
stable with a good test suite. Here is a rough sketch of the roadmap :

### 0.1

* [x] Core features:
  * [x] Backups & Incremental Backups
  * [x] Restoration
  * [x] Encryption
  * [x] Compression
  * [x] Pruning/Retention Policy
* [x] Sources: `tar`, `mariabackup`
* [x] Destinations: local filesystem, S3 compatible object storage
* [x] Documentation
* [x] CI/Release Management

### 0.2 (released)

* [x] Custom sources
* [x] Custom destinations

### 0.3 (released)

* [x] switch to [age](https://age-encryption.org/) for encryption. This
will be the first (and hopefully the last) breaking change for the file
format and keys format.

### 0.4 (next)

* [x] `btrfs` source
* [x] `btrfs` destination
* [x] remove mariabackup footguns
  * [x] add the option to use a dockerized mariabackup in the restoration
  process to have an exect version match
  * [x] returns an error when attempting to create an incremental backup
  based on a full backup created by a different version

### 0.5

* [ ] Proxy support

### 0.6

* [ ] Should be suitable for production

### 1.0

* Community feedback 
