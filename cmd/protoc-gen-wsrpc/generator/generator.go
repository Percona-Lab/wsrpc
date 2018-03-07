package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/sirupsen/logrus"
)

func stripSuffix(arg string) string {
	p := strings.Split(arg, ".")
	return p[len(p)-1]
}

func Generate(req *plugin_go.CodeGeneratorRequest, version string) *plugin_go.CodeGeneratorResponse {
	v := req.GetCompilerVersion()
	protocVersion := fmt.Sprintf("%d.%d.%d", v.GetMajor(), v.GetMinor(), v.GetPatch())
	if v.GetSuffix() != "" {
		version += v.GetSuffix()
	}
	logrus.Debugf("protoc version %s, protoc-gen-wsrpc version %s", protocVersion, version)

	var res plugin_go.CodeGeneratorResponse
	for _, ftg := range req.FileToGenerate {
		for _, pf := range req.ProtoFile {
			if ftg != pf.GetName() {
				continue
			}

			logrus.Debugf("Name: %s", pf.GetName())
			logrus.Debugf("Package: %s", pf.GetPackage())
			for _, m := range pf.GetMessageType() {
				logrus.Debugf("MessageType: %+v", m)
			}
			for _, e := range pf.GetEnumType() {
				logrus.Debugf("Enum: %+v", e)
			}
			for _, s := range pf.GetService() {
				logrus.Debugf("Service: %+v", s)
			}
			logrus.Debugf("Extension: %+v", pf.GetExtension())
			logrus.Debugf("Options: %+v", pf.GetOptions())
			logrus.Debugf("Syntax: %+v", pf.GetSyntax())

			for _, service := range pf.GetService() {
				// render prolog
				var content bytes.Buffer
				err := prologTemplate.Execute(&content, &prologTemplateData{
					SourceFile:    pf.GetName(),
					ProtocVersion: protocVersion,
					Version:       version,
					PackageName:   pf.GetPackage(),
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
					if m.GetClientStreaming() || m.GetServerStreaming() {
						res.Error = proto.String(fmt.Sprintf("streaming is not supported yet: %s.%s.%s", data.PackageName, data.ServiceName, m.GetName()))
						return &res
					}

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
