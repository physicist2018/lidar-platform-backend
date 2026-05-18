package ports

type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hashed, plain string) error
}
