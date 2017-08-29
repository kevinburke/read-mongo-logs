package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type ProfileResult struct {
	Was    int
	SlowMS int
	OK     bool
}

func init() {
	flag.Usage = func() {
		os.Stderr.WriteString(`usage: read-mongo-logs [mongo-url]

Enable verbose Mongo logs on the provided database, and then tail the logs. We
parse Mongo URL's the same way that the mongo shell client parses them, for
example, specify "read-mongo-logs accounts" to connect to the accounts database
on localhost.
`)
	}
}

type MongoDuration time.Duration

func (m *MongoDuration) SetBSON(raw bson.Raw) error {
	if raw.Kind != 0x10 {
		return fmt.Errorf("unknown kind for millis argument: %v (want 0x10)", raw.Kind)
	}
	if len(raw.Data) != 4 {
		return fmt.Errorf("wrong length for millis argument: %v (want 4)", raw.Data)
	}
	i := binary.LittleEndian.Uint32(raw.Data)
	*m = MongoDuration(time.Duration(i) * time.Millisecond)
	return nil
}

// Documentation is here: https://docs.mongodb.com/manual/reference/database-profiler/
type LogResult struct {
	AppName     string        `bson:"appName"`
	Command     bson.M        `bson:"command"`
	Client      string        `bson:"client"`
	Duration    MongoDuration `bson:"millis"`
	NumDeleted  int           `bson:"ndeleted"`
	NumMatched  int           `bson:"nMatched"`
	NumModified int           `bson:"nModified"`
	NumReturned int           `bson:"nreturned"`
	Namespace   string        `bson:"ns"`
	Op          string        `bson:"op"`
	Query       bson.M        `bson:"query"`
	Size        int64         `bson:"responseLength"`
	Time        time.Time     `bson:"ts"`
	Update      bson.M        `bson:"updateobj"`
	User        string        `bson:"user"`
}

func writePrefix(buf *bytes.Buffer, result *LogResult) {
	buf.WriteString(result.Time.Format(time.RFC3339))
	buf.WriteByte(' ')
	if result.User == "" {
		buf.WriteString(`""`)
	} else {
		buf.WriteString(result.User)
	}
	buf.WriteByte(' ')
	buf.WriteString(result.Client)
	buf.WriteByte(' ')
}

func debugLoop(iter *mgo.Iter, db string, w io.Writer) error {
	// useful for debugging and getting the raw query
	result := new(bson.M)
	count := 0
	for iter.Next(result) {
		if op, ok := (*result)["op"]; ok && op == "remove" {
			if op, ok := (*result)["ns"]; !ok || op != "accounts.invites" {
				continue
			}
			data, err := json.MarshalIndent(result, " ", "    ")
			if err != nil {
				return err
			}
			os.Stdout.Write(data)
			os.Stdout.Write([]byte{'\n', '\n'})
			count++
			if count > 30 {
				break
			}
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if err := iter.Close(); err != nil {
		return err
	}
	return nil
}

func loop(iter *mgo.Iter, db string, w io.Writer) error {
	result := new(LogResult)
	buf := new(bytes.Buffer)  // query line
	buf2 := new(bytes.Buffer) // result line
	for iter.Next(result) {
		buf.Reset()
		buf2.Reset()
		writePrefix(buf, result)
		writePrefix(buf2, result)
		buf.WriteString(strings.ToUpper(result.Op))
		buf.WriteByte(' ')
		buf2.WriteString("result: ")
		fmt.Fprintf(buf2, "time:%s size:%d ", time.Duration(result.Duration).String(), result.Size)
		switch result.Op {
		case "query":
			find, ok := result.Query["find"].(string)
			if !ok {
				return errors.New("query: could not convert find argument to string")
			}
			data, err := json.Marshal(result.Query["filter"])
			if err != nil {
				log.Fatal(err)
			}
			buf.WriteString(find)
			buf.WriteByte(' ')
			buf.Write(data)
			fmt.Fprintf(buf2, "returned:%d ", result.NumReturned)
		case "update":
			data, err := json.Marshal(result.Query)
			if err != nil {
				log.Fatal(err)
			}
			buf.WriteString(strings.TrimPrefix(result.Namespace, db+"."))
			buf.WriteByte(' ')
			buf.Write(data)
			buf.WriteByte(' ')
			// TODO: how to add two different documents here? newline?
			data2, err2 := json.Marshal(result.Update)
			if err2 != nil {
				log.Fatal(err2)
			}
			buf.Write(data2)
			fmt.Fprintf(buf2, "matched:%d modified:%d ", result.NumMatched, result.NumModified)
		case "remove":
			if result.Query == nil {
				fmt.Fprintf(buf, "%s {} ", strings.TrimPrefix(result.Namespace, db+"."))
			} else {
				data, err := json.Marshal(result.Query)
				if err != nil {
					return err
				}
				buf.WriteString(strings.TrimPrefix(result.Namespace, db+"."))
				buf.WriteByte(' ')
				buf.Write(data)
			}
			fmt.Fprintf(buf2, "deleted:%d ", result.NumDeleted)
		case "insert":
			collection, ok := result.Query["insert"].(string)
			if !ok {
				return errors.New("insert: could not convert collection argument to string")
			}
			data, err := json.Marshal(result.Query["documents"])
			if err != nil {
				log.Fatal(err)
			}
			buf.WriteString(collection)
			buf.WriteByte(' ')
			buf.Write(data)
		case "command":
			data, err := json.Marshal(result.Command)
			if err != nil {
				return err
			}
			buf.Write(data)
		}
		buf.WriteByte('\n')
		buf2.WriteByte('\n')
		if _, err := w.Write(buf.Bytes()); err != nil {
			return err
		}
		if _, err := w.Write(buf2.Bytes()); err != nil {
			return err
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if err := iter.Close(); err != nil {
		return err
	}
	return nil
}

var query = &bson.M{
	"op":                  bson.RegEx{Pattern: "^((?!(getmore|killcursors)).)"},
	"ns":                  bson.RegEx{Pattern: `^((?!(admin\.\$cmd|\.system|\.tmp\.)).)*$`},
	"command.profile":     bson.M{"$exists": false},
	"command.listIndexes": bson.M{"$exists": false},
}

const Version = "0.1"

func main() {
	version := flag.Bool("version", false, "Print the version string and exit")
	flag.Parse()
	if *version {
		fmt.Fprintf(os.Stderr, "read-mongo-logs version %s\n", Version)
		os.Exit(2)
	}
	if flag.NArg() != 1 {
		os.Stderr.WriteString("error: Please supply a database argument\n\n")
		flag.Usage()
	}
	// logic taken from
	// https://github.com/mongodb/mongo/blob/master/src/mongo/shell/mongo.js#L352
	var url = strings.TrimSpace(flag.Arg(0))
	if !strings.HasPrefix(url, "mongodb://") {
		colon := strings.LastIndex(url, ":")
		slash := strings.LastIndex(url, "/")
		if colon == -1 && slash == -1 {
			url = "mongodb://localhost:27017/" + url
		} else if slash != -1 {
			url = "mongodb://" + url
		}
	}
	info, err := mgo.ParseURL(url)
	if err != nil {
		log.Fatal(err)
	}
	// TODO
	//if ssl {
	//info.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
	//return tls.Dial("tcp", addr.String(), &tls.Config{})
	//}
	//}
	client, err := mgo.DialWithInfo(info)
	if err != nil {
		log.Fatal(err)
	}
	client.SetSafe(&mgo.Safe{
		FSync: true,
		WMode: "majority",
	})
	db := client.DB(info.Database)
	res := new(ProfileResult)
	if err := db.Run(bson.D{{Name: "profile", Value: 2}, {Name: "slowms", Value: 0}}, res); err != nil {
		log.Fatal(err)
	}
	if !res.OK {
		log.Fatal("Could not enable verbose logging on " + info.Database)
	}
	iter := db.C("system.profile").Find(query).Tail(-1)
	if err := loop(iter, info.Database, os.Stdout); err != nil {
		log.Fatal(err)
	}
	//if err := debugLoop(iter, info.Database, os.Stdout); err != nil {
	//log.Fatal(err)
	//}
}
