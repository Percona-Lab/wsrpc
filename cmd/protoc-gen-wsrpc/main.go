package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/plugin"

	"github.com/Percona-Lab/wsrpc/cmd/protoc-gen-wsrpc/generator"
)

func main() {
	flag.Parse()

	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	var req plugin_go.CodeGeneratorRequest
	if err = proto.Unmarshal(b, &req); err != nil {
		log.Fatal(err)
	}

	res := generator.Generate(&req)

	if b, err = proto.Marshal(res); err != nil {
		log.Fatal(err)
	}
	if _, err = os.Stdout.Write(b); err != nil {
		log.Fatal(err)
	}
}
