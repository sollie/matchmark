package wa1500

import (
	"encoding/json"
	"strings"
)

type Classification int

const (
	Unclassified Classification = iota
	Marksman
	Sharpshooter
	Expert
	Master
	HighMaster
)

var classificationNames = [...]string{"Unclassified", "Marksman", "Sharpshooter", "Expert", "Master", "High Master"}

func (c Classification) Name() string {
	if c < 0 || int(c) >= len(classificationNames) {
		return ""
	}
	return classificationNames[c]
}

func (c Classification) Valid() bool { return c >= Unclassified && c <= HighMaster }

func ParseClassification(s string) (Classification, bool) {
	s = strings.Join(strings.Fields(s), " ")
	for i, name := range classificationNames {
		if strings.EqualFold(s, name) {
			return Classification(i), true
		}
	}
	return 0, false
}

func (c Classification) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Value int    `json:"value"`
		Name  string `json:"name"`
	}{int(c), c.Name()})
}
