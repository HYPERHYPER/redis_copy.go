redis_copy
==========

Rapidly copies all keys from one Redis instance to another. Uses SCAN to read keys and pipelines DUMP/RESTORE to move values.

Dependencies
------------

```bash
$ go get gopkg.in/redis.v5
```

Running
-------

Edit the file to set the Redis connection info, and maybe tune the number of records per chunk (currently 7500) and number of goroutines (10), then:

```bash
$ go run redis_copy.go
```

### New to Go?

```bash
$ mkdir -p ~/go/src
$ export GOPATH=~/go
$ go get gopkg.in/redis.v5
$ cd ~/go/src
$ git clone $THISREPO
```
