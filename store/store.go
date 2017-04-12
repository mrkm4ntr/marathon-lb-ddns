package store

type Store interface {
	GetIPAddresses() ([]string, error)
	SetIPAddresses([]string) error
	ListCNames() ([]string, error)
	AddCName(string) error
	RemoveCName(string) error
}
