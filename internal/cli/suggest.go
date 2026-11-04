package cli

import (
	"fmt"
	"slices"
	"strings"

	"zscalerctl/internal/resources"
)

// maxResourceSuggestions bounds a suggestion list; more word matches than this
// mean the name was not specific enough to point anywhere.
const maxResourceSuggestions = 3

// resourceSuggestions returns the catalog resources, as product/name, that an
// unknown name in product most likely meant. It tries, in order:
//
//   - a near typo of a name in the same product (edit distance 1, or 2 for
//     longer names),
//   - the same name in another product, in catalog product order
//     (zpa locations → zia/locations),
//   - a same-product name containing every word of the request
//     (access-policies → access-policy-rules),
//   - a same-product name containing all but one word of a multi-word request
//     (url-filtering-policies → url-filtering-rules).
//
// Words are compared singular, and a word of three or more letters also
// matches a longer word it starts (app → application). Only the first tier
// with matches is used; a word tier with more than maxResourceSuggestions
// matches yields nothing. The result holds catalog names only.
func resourceSuggestions(catalog resources.ResourceCatalog, product resources.Product, name string) []string {
	query := resourceWords(name)
	if len(query) == 0 {
		return nil
	}
	queryKey := strings.Join(query, "-")
	type typo struct {
		name     string
		distance int
	}
	var typos []typo
	var elsewhere, allWords, allButOne []string
	for _, spec := range catalog {
		words := resourceWords(spec.Name)
		id := string(spec.Product) + "/" + spec.Name
		if spec.Product != product {
			if slices.Equal(words, query) {
				elsewhere = append(elsewhere, id)
			}
			continue
		}
		if d := editDistance(queryKey, strings.Join(words, "-")); d <= typoBudget(queryKey) {
			typos = append(typos, typo{name: id, distance: d})
			continue
		}
		switch missing := missingWords(query, words); {
		case missing == 0:
			allWords = append(allWords, id)
		case missing == 1 && len(query) >= 2:
			allButOne = append(allButOne, id)
		}
	}
	if len(typos) > 0 {
		slices.SortFunc(typos, func(a, b typo) int {
			if a.distance != b.distance {
				return a.distance - b.distance
			}
			return strings.Compare(a.name, b.name)
		})
		out := make([]string, 0, min(len(typos), maxResourceSuggestions))
		for _, t := range typos[:min(len(typos), maxResourceSuggestions)] {
			out = append(out, t.name)
		}
		return out
	}
	if len(elsewhere) > 0 {
		products := knownProducts(catalog)
		slices.SortStableFunc(elsewhere, func(a, b string) int {
			return slices.Index(products, productOf(a)) - slices.Index(products, productOf(b))
		})
		return elsewhere[:min(len(elsewhere), maxResourceSuggestions)]
	}
	for _, tier := range [][]string{allWords, allButOne} {
		if len(tier) == 0 {
			continue
		}
		if len(tier) > maxResourceSuggestions {
			return nil
		}
		slices.Sort(tier)
		return tier
	}
	return nil
}

// specsSupporting returns the catalog resources that support any of ops.
func specsSupporting(catalog resources.ResourceCatalog, ops ...string) resources.ResourceCatalog {
	var out resources.ResourceCatalog
	for _, spec := range catalog {
		if slices.ContainsFunc(ops, spec.SupportsReadOperation) {
			out = append(out, spec)
		}
	}
	return out
}

// productOf returns the product of a product/name id.
func productOf(id string) resources.Product {
	product, _, _ := strings.Cut(id, "/")
	return resources.Product(product)
}

// didYouMean renders suggestions as " (did you mean a, b or c?)", or "" when
// there are none.
func didYouMean(suggestions []string) string {
	switch len(suggestions) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf(" (did you mean %s?)", suggestions[0])
	default:
		last := len(suggestions) - 1
		return fmt.Sprintf(" (did you mean %s or %s?)", strings.Join(suggestions[:last], ", "), suggestions[last])
	}
}

// resourceWords splits a resource name into lowercase singular words.
func resourceWords(name string) []string {
	var words []string
	for _, word := range strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	}) {
		words = append(words, singular(word))
	}
	return words
}

// singular strips a plural ending well enough to compare resource names:
// policies → policy, addresses → address, rules → rule. Short words such as
// dns and ips are left alone.
func singular(word string) string {
	switch {
	case len(word) > 4 && strings.HasSuffix(word, "ies"):
		return strings.TrimSuffix(word, "ies") + "y"
	case strings.HasSuffix(word, "sses"):
		return strings.TrimSuffix(word, "es")
	case len(word) > 3 && strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss"):
		return strings.TrimSuffix(word, "s")
	}
	return word
}

// missingWords counts the query words that match no candidate word, exactly or
// as a prefix of at least three letters.
func missingWords(query, candidate []string) int {
	missing := 0
	for _, q := range query {
		if !slices.ContainsFunc(candidate, func(c string) bool {
			return c == q || (len(q) >= 3 && strings.HasPrefix(c, q))
		}) {
			missing++
		}
	}
	return missing
}

// typoBudget is the edit distance still read as a typo of a name this long.
func typoBudget(name string) int {
	if len(name) >= 8 {
		return 2
	}
	return 1
}

// editDistance is the Levenshtein distance between a and b, by bytes;
// resource names are ASCII.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}
