package functions

import "testing"

func TestCatalogCoversEveryFunction(t *testing.T) {
	for _, info := range Catalog() {
		if info.Description == "" {
			t.Errorf("%s has no catalog entry (add it to catalog.go)", info.ImplKey)
		}
		if info.Parameters["type"] != "object" {
			t.Errorf("%s: parameters must be an object schema", info.ImplKey)
		}
	}
	for key := range catalog {
		if _, ok := StaticFunctionMap[key]; !ok {
			t.Errorf("catalog entry %s matches no function", key)
		}
	}
}
