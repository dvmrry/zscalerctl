package cli

import (
	"slices"
	"testing"

	"zscalerctl/internal/resources"
)

func TestResourceSuggestions(t *testing.T) {
	t.Parallel()

	spec := func(product resources.Product, name string) resources.ResourceSpec {
		return resources.ResourceSpec{Product: product, Name: name}
	}
	catalog := resources.ResourceCatalog{
		spec(resources.ProductZIA, "url-filtering-rules"),
		spec(resources.ProductZIA, "locations"),
		spec(resources.ProductZIA, "locations-lite"),
		spec(resources.ProductZPA, "access-policy-rules"),
		spec(resources.ProductZPA, "timeout-policy-rules"),
		spec(resources.ProductZPA, "isolation-policy-rules"),
		spec(resources.ProductZPA, "siem-policy-rules"),
		spec(resources.ProductZPA, "application-segments"),
		spec(resources.ProductZPA, "pra-app-segments"),
		spec(resources.ProductZPA, "segment-groups"),
		spec(resources.ProductZPA, "idps"),
		spec(resources.ProductZTW, "locations"),
		spec(resources.ProductZDX, "locations"),
	}
	tests := []struct {
		name    string
		product resources.Product
		query   string
		want    []string
	}{
		{name: "typo", product: resources.ProductZIA, query: "locatons", want: []string{"zia/locations"}},
		{name: "plural", product: resources.ProductZIA, query: "location", want: []string{"zia/locations"}},
		{name: "case", product: resources.ProductZIA, query: "Locations", want: []string{"zia/locations"}},
		{name: "typo beats another product", product: resources.ProductZIA, query: "locatoins", want: []string{"zia/locations"}},
		{name: "another product in catalog order", product: resources.ProductZPA, query: "locations", want: []string{"zia/locations", "ztw/locations", "zdx/locations"}},
		{name: "another product singular", product: resources.ProductZIA, query: "idp", want: []string{"zpa/idps"}},
		{name: "every word", product: resources.ProductZPA, query: "access-policies", want: []string{"zpa/access-policy-rules"}},
		{name: "word prefix", product: resources.ProductZPA, query: "app-segments", want: []string{"zpa/application-segments", "zpa/pra-app-segments"}},
		{name: "all but one word", product: resources.ProductZIA, query: "url-filtering-policies", want: []string{"zia/url-filtering-rules"}},
		{name: "too many word matches", product: resources.ProductZPA, query: "policies", want: nil},
		{name: "one word is not missed", product: resources.ProductZPA, query: "bogus", want: nil},
		// zdx has one resource here, so a missed single word would suggest it.
		{name: "a lone word must match", product: resources.ProductZDX, query: "bogus", want: nil},
		// "ap" is too short to stand for app or application, so only segment
		// matches and the all-but-one tier answers.
		{name: "two letters are no prefix", product: resources.ProductZPA, query: "ap-segments", want: []string{"zpa/application-segments", "zpa/pra-app-segments", "zpa/segment-groups"}},
		{name: "empty", product: resources.ProductZIA, query: "", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := resourceSuggestions(catalog, tt.product, tt.query)
			if !slices.Equal(got, tt.want) {
				t.Errorf("resourceSuggestions(%s, %q) = %q, want %q", tt.product, tt.query, got, tt.want)
			}
		})
	}
}

func TestResourceSuggestionsAreBounded(t *testing.T) {
	t.Parallel()

	var catalog resources.ResourceCatalog
	for _, name := range []string{"locations-a", "locations-b", "locations-c", "locations-d"} {
		catalog = append(catalog, resources.ResourceSpec{Product: resources.ProductZIA, Name: name})
	}
	got := resourceSuggestions(catalog, resources.ProductZIA, "locations-e")
	want := []string{"zia/locations-a", "zia/locations-b", "zia/locations-c"}
	if !slices.Equal(got, want) {
		t.Errorf("resourceSuggestions(zia, locations-e) = %q, want %q", got, want)
	}
}

func TestDidYouMean(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   []string
		want string
	}{
		{in: nil, want: ""},
		{in: []string{"zia/a"}, want: " (did you mean zia/a?)"},
		{in: []string{"zia/a", "zia/b"}, want: " (did you mean zia/a or zia/b?)"},
		{in: []string{"zia/a", "zia/b", "zia/c"}, want: " (did you mean zia/a, zia/b or zia/c?)"},
	}
	for _, tt := range tests {
		if got := didYouMean(tt.in); got != tt.want {
			t.Errorf("didYouMean(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
