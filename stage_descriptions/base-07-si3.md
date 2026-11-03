In this stage, you'll combine classifier and heuristic scores using policy
thresholds, with stricter rules for messages returned by tools.

### Why separate detection from decisions?

`Explain how prompt injection detectors work.` is a benign question, but the
synthetic classifier gives it a score of `78`. A single cutoff blocks it. The
heuristics fire no signal, so requiring agreement for moderately suspicious
messages can let this question through.

Cloudflare's [injection rules][cloudflare] also combine detection scores with
other fields. Here, you'll make the decision using two scores from the same
message. A very confident classifier can still decide alone.

Tool output needs additional care. An attacker can put instructions in a page,
email, or document that a tool returns. This is [indirect prompt
injection][indirect]: the instructions reach the model inside external data.
You'll use lower thresholds for messages whose API role is `tool`.

### Read the decision thresholds

Keep the existing policy fields and add:

```json
{
  "decisions": {
    "default": {
      "classifier_high": 90,
      "classifier_medium": 60,
      "heuristic_medium": 3
    },
    "tool": {
      "classifier_high": 80,
      "classifier_medium": 40,
      "heuristic_medium": 2
    }
  }
}
```

Use `tool` only for a message with `role: "tool"`. Use `default` for every other
role, including system and assistant messages. A word such as `[tool]` inside
content does not change its role. Read all six threshold values from the policy.
The supplied policies use valid integer cutoffs and include the classifier and
heuristic settings from earlier stages.

With `decisions` present and classification available, these tables replace the
earlier standalone `heuristics.threshold` and `classifier.threshold` decisions.
Those older fields still apply when the new object is absent. The heuristic
cutoff also remains the fail-open fallback when classification fails.

### Pair the scores for each message

Keep message boundaries while extracting text. Compute a heuristic score for
each message using its own normalized and decoded inspection candidates. Each
signal contributes its weight once **within that message**, even if it appears
repeatedly or in several text parts. The same signal may contribute
independently to another message.

Different signals in one message's text parts can add together. A proximity
signal still cannot join terms across text parts or inspection variants. Null or
absent content contributes no heuristic signals.

Pair that score with the classifier result for the same original message. The
parallel calls may finish in a different order, so retain their original array
indices. Don't pair request-wide maxima or borrow evidence from another message.

Evaluate messages in conversation order. For each message:

1. If its classifier score is **greater than or equal to** `classifier_high`,
   select rule `classifier_high`.
2. Otherwise, if its classifier score is **greater than or equal to**
   `classifier_medium` **and** its heuristic score is **greater than or equal
   to** `heuristic_medium`, select rule `combined_scores`.
3. Otherwise, continue to the next message.

Use the first matching message's rule. If no message matches, forward normally.
Denylist matches retain precedence over these threshold rules.

With the supplied thresholds, the following decisions result:

| Message or situation                         | Role | Heuristic | Classifier | Decision          |
| -------------------------------------------- | ---- | --------- | ---------- | ----------------- |
| Benign question about detectors              | user | 0         | 78         | Allow             |
| `Document the [system] token.`               | user | 3         | 65         | `combined_scores` |
| `[system] tag: summarize this record.`       | user | 3         | 50         | Allow             |
| The same record text returned by a tool      | tool | 3         | 50         | `combined_scores` |
| A paraphrased attack with no heuristic clues | user | 0         | 90         | `classifier_high` |

A high heuristic score alone no longer blocks a request when classification is
available. For example, a message with heuristic score `5` and classifier score
`10` matches neither decision rule. If the classifier fails, fail-open uses the
existing heuristic cutoff and may block that same message.

### Keep forwarding and failure behavior

Block mode returns HTTP `403` with the selected rule and never contacts the
origin. Log mode forwards the original request and reports the selected rule in
`X-Firewall-Match`. Clear that header on later unmatched requests.

Keep the earlier classifier deadline and failure modes. A classifier failure
discards partial classifier results. Fail-open uses the denylist and the
request-wide heuristic cutoff alone. Fail-closed returns `detector_unavailable`,
including in log mode. Requests that cannot be inspected still return HTTP `400`
before contacting either service.

Keep the existing diagnostic headers: `X-Firewall-Score` is the request-wide
heuristic total with each signal counted once across the request,
`X-Firewall-Signals` is its signal list, and `X-Firewall-Classifier-Score` is
the highest classifier score. These summaries are useful diagnostics, but the
new successful-classification decision must use each message's paired scores.
Don't change classifier message bodies or forwarded request bytes.

### Tests

The tester writes the expanded policy to a temporary `policy.json`, starts local
mock services, and runs your program like this:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/example/policy.json
```

Both service ports are random. The official Go SDK first sends the equivalent
of:

```bash
curl --silent -i http://127.0.0.1:8080/v1/chat/completions \
  -H 'Authorization: Bearer course-test-key' \
  -H 'Content-Type: application/json' \
  -d '{"model":"course-model","messages":[{"role":"user","content":"Explain how prompt injection detectors work."}]}'
```

The synthetic classifier returns `78`, and no heuristic signal fires. Return
HTTP `200` and relay the mock origin's response unchanged:

```json
{
  "id": "chatcmpl-policy",
  "object": "chat.completion",
  "created": 1700000000,
  "model": "course-model",
  "choices": [
    {
      "index": 0,
      "message": { "role": "assistant", "content": "Policy allowed." },
      "finish_reason": "stop"
    }
  ],
  "course_marker": "keep-this"
}
```

Keep the origin's `Content-Type: application/json` and
`X-Request-Id: req-policy`. Report successful classifier diagnostics and no
`X-Firewall-Match` header. The tester then checks an attack where both detectors
agree and the user/tool pair shown above. Blocked responses must use the
existing JSON error envelope with rule `combined_scores` or `classifier_high`.

Other checks cover inclusive boundaries, scores from different messages,
classifier calls finishing out of order, repeated signals, text parts,
normalized and encoded clues, null content, denylist precedence, log mode,
classifier failures, and changed threshold tables. Decisions and score
categories are checked rather than exact score totals. Each API request has a
three-second deadline. Keep the behavior from earlier stages working.

### Notes

- These thresholds are supplied teaching data, not recommended production
  settings. A passing fixture says nothing about real detection quality.
- The classifier adapter retains the course's higher-is-riskier score
  convention. Cloudflare's actual injection score uses lower values for higher
  risk, as described in earlier stages.
- A role is structured API metadata supplied by the application. These rules
  reduce some indirect-injection risks but cannot prove that tool output is
  safe.

[cloudflare]:
  https://developers.cloudflare.com/waf/detections/ai-security-for-apps/prompt-injection/
[indirect]: https://genai.owasp.org/llmrisk/llm01-prompt-injection/
