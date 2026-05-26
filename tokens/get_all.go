package tokens

import "errors"

func GetAll() ([]TokenRecord, error) {
	if tokenStore == nil {
		return nil, errors.New("token store not found")
	}
	return tokenStore.ListAll()
}
