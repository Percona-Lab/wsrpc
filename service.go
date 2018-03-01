package wsrpc

type ServiceMethod struct {
	Name   string
	Method func(interface{}, []byte) ([]byte, error)
}

type ServiceDesc struct {
	Methods []ServiceMethod
}
