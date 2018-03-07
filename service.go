package wsrpc

type ServiceMethod struct {
	Path   string
	Method func(interface{}, []byte) ([]byte, error)
}

type ServiceDesc struct {
	Methods []ServiceMethod
}
