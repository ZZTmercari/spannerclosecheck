package analyzer

import "fmt"

type ResourceType struct {
	Name        string
	CloseMethod string
}

func (rt ResourceType) CloseMessage() string {
	return fmt.Sprintf("%s.%s() must be deferred", rt.Name, rt.CloseMethod)
}

var spannerResourceTypes = map[string]ResourceType{
	"ReadOnlyTransaction":      {"ReadOnlyTransaction", "Close"},
	"BatchReadOnlyTransaction": {"BatchReadOnlyTransaction", "Close"},
	"RowIterator":              {"RowIterator", "Stop"},
}
