# read-mongo-logs

Want to see what queries are being run against your Mongo instance? Run
this binary to enable verbose Mongo logs on the database in question, then
immediately start tailing the logs. Example output:

```
2017-08-29T11:45:41-07:00 "" 127.0.0.1 QUERY invites {"$and":[{"inviteeEmail":"kevin@burke.services"},{"$or":[{"status":1},{"status":2}]}]}
2017-08-29T11:45:41-07:00 "" 127.0.0.1 result: time:0s returned:1
2017-08-29T13:03:52-07:00 "" 127.0.0.1 REMOVE locks {"ownerId":"81fcaadb-5c26-4863-8a11-dca0b70bbf79"}
2017-08-29T13:03:52-07:00 "" 127.0.0.1 result: time:0s deleted:0
```

Note: This tool works by setting the database's profiling level to 2, which logs
data about every query to the database's `system.profile` collection. We then
stream the data. Be careful about running this in production, since this will
slow down your database and may log sensitive data to the system profile.

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

If you enabled query logs and you want to disable them, use the `--disable`
flag.

```
read-mongo-logs --disable 192.168.0.5:9999/foo
```


## Installation

Find your target operating system (darwin, windows, linux) and desired bin
directory, and modify the command below as appropriate:

    curl --silent --location https://github.com/kevinburke/read-mongo-logs/releases/download/0.3/read-mongo-logs-linux-amd64 > /usr/local/bin/read-mongo-logs && chmod 755 /usr/local/bin/read-mongo-logs

On Travis, you may want to create `$HOME/bin` and write to that, since
/usr/local/bin isn't writable with their container-based infrastructure.

The latest version is 0.3.

If you have a Go development environment, you can also install via source code:

    go get -u github.com/kevinburke/read-mongo-logs
