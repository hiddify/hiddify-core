package config

import (
	"encoding/json"
	"time"
)

type DurationInSeconds int

func (d DurationInSeconds) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(d))
}

func (d *DurationInSeconds) UnmarshalJSON(bytes []byte) error {
	var v int64
	if err := json.Unmarshal(bytes, &v); err != nil {
		return err
	}
	*d = DurationInSeconds(v)
	return nil
}

func (d DurationInSeconds) Duration() time.Duration {
	return time.Duration(d) * time.Second
}
