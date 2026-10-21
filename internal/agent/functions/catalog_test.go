package functions

import (
	"testing"

	"github.com/gohead-cms/gohead/internal/agent/tools"
)

func TestCatalogCoversEveryFunction(t *testing.T) {
	for _, info := range Catalog() {
		if info.Description == "" {
			t.Errorf("%s has no catalog entry (add it to internal/agent/tools)", info.ImplKey)
		}
		if info.Parameters["type"] != "object" {
			t.Errorf("%s: parameters must be an object schema", info.ImplKey)
		}
	}
	for _, info := range tools.All() {
		if _, ok := StaticFunctionMap[info.ImplKey]; !ok {
			t.Errorf("catalog entry %s matches no function", info.ImplKey)
		}
	}
}
