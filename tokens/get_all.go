package tokens

import "context"

func GetAll() ([]TokenRecord, error) {
	cur, err := tokenStore.GetAll()
	if err != nil {
		return nil, err
	}

	var tokenRecords []TokenRecord
	for cur.Next(context.TODO()) {
		var d TokenRecord
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		tokenRecords = append(tokenRecords, d)
	}

	return tokenRecords, nil
}
