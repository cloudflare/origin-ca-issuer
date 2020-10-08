package cfapi

type Factory interface {
	APIWith([]byte) (Interface, error)
}

type FactoryFunc func([]byte) (Interface, error)

func (f FactoryFunc) APIWith(serviceKey []byte) (Interface, error) {
	return f(serviceKey)
}
