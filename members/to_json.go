package members

import "encoding/json"

func (m *Member) ToJSON() ([]byte, error) {
	j, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return j, nil
}
