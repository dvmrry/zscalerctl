package agenteval_test

import (
	"encoding/json"
	"testing"

	"github.com/dvmrry/zscalerctl/internal/agenteval"
)

// TestEnvelopeParserGoldens pins the §2.1 last-block answer-envelope extractor
// against tricky raw inputs (§5.2 TestEnvelopeParserGoldens): a single block,
// MULTIPLE blocks (last wins), CRLF line endings, JSON with nested braces in the
// answer, trailing prose after the block, a missing block (ok=false), and
// malformed JSON (ok=false). These are authored to the REAL envelope spec
// (<<<ZSCTL_ANSWER … ZSCTL_ANSWER), never the earlier ad-hoc pilot format.
func TestEnvelopeParserGoldens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		agentText  string
		wantOK     bool
		wantAnswer string // raw JSON of the parsed answer; checked only when wantOK
	}{
		{
			name: "single block",
			agentText: "Here is my answer.\n" +
				"<<<ZSCTL_ANSWER\n" +
				`{"answer": 12, "evidence": ["zscalerctl zia locations list"]}` + "\n" +
				"ZSCTL_ANSWER\n",
			wantOK:     true,
			wantAnswer: "12",
		},
		{
			name: "multiple blocks last wins",
			agentText: "Let me think out loud first.\n" +
				"<<<ZSCTL_ANSWER\n" +
				`{"answer": 1, "evidence": []}` + "\n" +
				"ZSCTL_ANSWER\n" +
				"Wait, I miscounted. Final:\n" +
				"<<<ZSCTL_ANSWER\n" +
				`{"answer": 3, "evidence": ["zscalerctl zia rule-labels list"]}` + "\n" +
				"ZSCTL_ANSWER\n",
			wantOK:     true,
			wantAnswer: "3",
		},
		{
			name: "CRLF line endings",
			agentText: "answer below\r\n" +
				"<<<ZSCTL_ANSWER\r\n" +
				`{"answer": "US", "evidence": []}` + "\r\n" +
				"ZSCTL_ANSWER\r\n",
			wantOK:     true,
			wantAnswer: `"US"`,
		},
		{
			name: "nested braces in answer object",
			agentText: "<<<ZSCTL_ANSWER\n" +
				`{"answer": {"country": "US", "geo": {"lat": 1}}, "evidence": ["x"]}` + "\n" +
				"ZSCTL_ANSWER",
			wantOK:     true,
			wantAnswer: `{"country": "US", "geo": {"lat": 1}}`,
		},
		{
			name: "trailing prose after block",
			agentText: "<<<ZSCTL_ANSWER\n" +
				`{"answer": ["id", "name"], "evidence": []}` + "\n" +
				"ZSCTL_ANSWER\n" +
				"(I hope that is right — let me know if you need more detail.)\n",
			wantOK:     true,
			wantAnswer: `["id", "name"]`,
		},
		{
			name:      "missing block",
			agentText: "I ran the command but forgot to emit the envelope. The answer is twelve.",
			wantOK:    false,
		},
		{
			name: "malformed JSON body",
			agentText: "<<<ZSCTL_ANSWER\n" +
				`{"answer": 12, "evidence": [}` + "\n" +
				"ZSCTL_ANSWER\n",
			wantOK: false,
		},
		{
			name: "opener with no closer",
			agentText: "<<<ZSCTL_ANSWER\n" +
				`{"answer": 12, "evidence": []}` + "\n",
			wantOK: false,
		},
		{
			name: "empty body between delimiters",
			agentText: "<<<ZSCTL_ANSWER\n" +
				"\n" +
				"ZSCTL_ANSWER\n",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env, ok := agenteval.ParseAnswer(tc.agentText)
			if ok != tc.wantOK {
				t.Fatalf("ParseAnswer ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if !jsonEqual(t, string(env.Answer), tc.wantAnswer) {
				t.Fatalf("ParseAnswer answer = %s, want %s", env.Answer, tc.wantAnswer)
			}
		})
	}
}

// jsonEqual reports whether two JSON snippets are semantically equal (ignoring
// insignificant whitespace), so goldens need not match byte-for-byte spacing.
func jsonEqual(t *testing.T, a, b string) bool {
	t.Helper()
	var av, bv any
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		t.Fatalf("unmarshal got %q: %v", a, err)
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		t.Fatalf("unmarshal want %q: %v", b, err)
	}
	ab, _ := json.Marshal(av)
	bb, _ := json.Marshal(bv)
	return string(ab) == string(bb)
}
