package hash

import "golang.org/x/crypto/bcrypt"

type Bcrypt struct{ cost int }

func NewBcrypt(cost int) *Bcrypt { return &Bcrypt{cost: cost} }

func (b *Bcrypt) Hash(password string) (string, error) {
	value, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	return string(value), err
}

func (b *Bcrypt) Compare(passwordHash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil
}
