package main

import (
	"flag"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/sirupsen/logrus"

	"github.com/Percona-Lab/wsrpc/cmd/protoc-gen-wsrpc/generator"
)

var Version = "0.1.0-dev"

func main() {
	debugF := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		logrus.Fatal(err)
	}
	var req plugin_go.CodeGeneratorRequest
	if err = proto.Unmarshal(b, &req); err != nil {
		logrus.Fatal(err)
	}

	params := req.GetParameter()
	if params != "" {
		for _, p := range strings.Split(params, ",") {
			parts := strings.Split(p, "=")
			if len(parts) != 2 {
				logrus.Fatalf("Failed to parse parameter %q.", p)
			}
			if err = flag.Set(parts[0], parts[1]); err != nil {
				logrus.Fatalf("Failed to parse parameter %q: %s", p, err)
			}
		}
	}

	if *debugF {
		logrus.SetLevel(logrus.DebugLevel)
	}

	res := generator.Generate(&req, Version)

	if b, err = proto.Marshal(res); err != nil {
		logrus.Fatal(err)
	}
	if _, err = os.Stdout.Write(b); err != nil {
		logrus.Fatal(err)
	}
}
