package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
)

func stripSuffix(arg string) string {
	p := strings.Split(arg, ".")
	return p[len(p)-1]
}

func Generate(req *plugin_go.CodeGeneratorRequest) *plugin_go.CodeGeneratorResponse {
	var res plugin_go.CodeGeneratorResponse

	for _, ftg := range req.FileToGenerate {
		for _, pf := range req.ProtoFile {
			if ftg != pf.GetName() {
				continue
			}

			log.Printf("Name: %s", pf.GetName())
			log.Printf("Package: %s", pf.GetPackage())
			for _, m := range pf.GetMessageType() {
				log.Printf("MessageType: %+v", m)
			}
			for _, e := range pf.GetEnumType() {
				log.Printf("Enum: %+v", e)
			}
			for _, s := range pf.GetService() {
				log.Printf("Service: %+v", s)
			}
			log.Printf("%+v", pf.GetExtension())
			log.Printf("%+v", pf.GetOptions())
			log.Printf("%+v", pf.GetSyntax())

			for _, service := range pf.GetService() {
				// render prolog
				var content bytes.Buffer
				err := prologTemplate.Execute(&content, &prologTemplateData{
					SourceFile:  pf.GetName(),
					PackageName: pf.GetPackage(),
				})
				if err != nil {
					res.Error = proto.String(err.Error())
					return &res
				}

				// prepare template data
				data := &templateData{
					PackageName: pf.GetPackage(),
					ServiceName: service.GetName(),
				}
				data.ServiceNameUnexported = strings.ToLower(data.ServiceName[0:1]) + data.ServiceName[1:]
				for _, m := range service.GetMethod() {
					data.Methods = append(data.Methods, method{
						Name:       m.GetName(),
						InputType:  stripSuffix(m.GetInputType()),
						OutputType: stripSuffix(m.GetOutputType()),
					})
				}

				// render client and server
				if err = clientTemplate.Execute(&content, data); err != nil {
					res.Error = proto.String(err.Error())
					return &res
				}
				if err = serverTemplate.Execute(&content, data); err != nil {
					res.Error = proto.String(err.Error())
					return &res
				}

				// render and format the whole file
				ext := filepath.Ext(pf.GetName())
				fileName := fmt.Sprintf("%s.%s.wsrpc.go", strings.TrimSuffix(pf.GetName(), ext), service.GetName())
				resFile := &plugin_go.CodeGeneratorResponse_File{
					Name: proto.String(fileName),
				}
				b := content.Bytes()
				if b, err = format.Source(b); err != nil {
					res.Error = proto.String(fmt.Sprintf("failed to format source code: %s", err))
					return &res
				}
				resFile.Content = proto.String(string(b))

				res.File = append(res.File, resFile)
			}
		}
	}

	return &res
}
