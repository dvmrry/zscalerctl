package agenteval

// prompt.go is the PURE prompt-composition helper (docs/AGENTIC_COVERAGE_PLAN.md
// §2.1, §6.3). It assembles the full text the live runner hands an agent: the
// question prompt, the mandatory machine-extractable answer-envelope instructions
// (§2.1), and the §6.3 hard-rule line confining the agent to `./zscalerctl` and
// the two staged docs. It never execs, dials the network, reads the clock, or
// calls an LLM — it is a deterministic function of the Question, so the exact
// composed prompt is unit-testable without invoking any agent.

import "strings"

// answerEnvelopeInstructions is the verbatim §2.1 answer-protocol block appended
// to every question. It instructs the agent to emit EXACTLY one fenced
// <<<ZSCTL_ANSWER … ZSCTL_ANSWER block, last, with a typed `answer` and an
// `evidence` list. The grader extracts the LAST such block (ParseAnswer); the
// envelope is the agent's own — the harness never injects markers (§2.1).
//
// It is built from the AnswerOpen/AnswerClose delimiter constants (contract.go)
// so the instructed delimiters can never drift from the ones ParseAnswer scans
// for.
var answerEnvelopeInstructions = strings.Join([]string{
	"When you have the answer, output EXACTLY one block, last, with nothing after it:",
	AnswerOpen,
	`{"answer": <typed value>, "evidence": ["<command you ran>", ...]}`,
	AnswerClose,
}, "\n")

// hardRuleLine is the §6.3 confinement instruction: the agent may ONLY run the
// local `./zscalerctl` tool and read the two staged docs, and must discover the
// surface from the tool itself rather than from prior knowledge. This is what
// makes the eval measure self-describability (could the agent discover the
// answer from the surface) rather than recall.
const hardRuleLine = "Hard rule: only run ./zscalerctl and read the two docs (AGENTS.md, the skill file); discover everything else from the tool itself — do not answer from prior knowledge or guess resource names."

// ComposePrompt assembles the full agent-facing prompt for one question (§2.1,
// §6.3): the question Prompt, then the hard-rule confinement line, then the
// answer-envelope instructions LAST (so the envelope block is the final thing the
// agent reads, reinforcing "output the block last"). The pieces are separated by
// blank lines. It is a total, deterministic function of q.Prompt.
func ComposePrompt(q Question) string {
	return strings.Join([]string{
		strings.TrimSpace(q.Prompt),
		hardRuleLine,
		answerEnvelopeInstructions,
	}, "\n\n")
}
