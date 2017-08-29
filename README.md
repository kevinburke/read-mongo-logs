# read-mongo-logs

Want to see what queries are being run against your Mongo instance? Run
this binary to enable verbose Mongo logs on the database in question, then
immediately start tailing the logs. Example output:

```
2017-08-29T11:45:41-07:00 "" 127.0.0.1 QUERY {"filter":{"$and":[{"inviteeEmail":"dkimmel@gmail.com"},{"$or":[{"status":1},{"status":2}]}]},"find":"invites"}
2017-08-29T11:45:41-07:00 "" 127.0.0.1 result: time:0s returned:1
2017-08-29T13:03:52-07:00 "" 127.0.0.1 REMOVE locks {"ownerId":"81fcaadb-5c26-4863-8a11-dca0b70bbf79"}
2017-08-29T13:03:52-07:00 "" 127.0.0.1 result: time:0s deleted:0
```

### Usage

The database syntax uses the same syntax as the mongo shell client. To tail the
`accounts` database on the local machine:

```
read-mongo-logs accounts
```

Or on the `foo` database on host 192.168.0.5, port 9999:

```
read-mongo-logs 192.168.0.5:9999/foo
```

## Installation

Find your target operating system (darwin, windows, linux) and desired bin
directory, and modify the command below as appropriate:

    curl --silent --location https://github.com/kevinburke/read-mongo-logs/releases/download/0.2/read-mongo-logs-linux-amd64 > /usr/local/bin/read-mongo-logs && chmod 755 /usr/local/bin/read-mongo-logs

On Travis, you may want to create `$HOME/bin` and write to that, since
/usr/local/bin isn't writable with their container-based infrastructure.

The latest version is 0.2.

If you have a Go development environment, you can also install via source code:

    go get -u github.com/kevinburke/read-mongo-logs