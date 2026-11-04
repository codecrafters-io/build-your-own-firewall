# Stage 6: Call the classifier

## Learning goal

Integrate a classifier service to catch an attack that the supplied heuristics
miss. Send each original message to `POST /classify`, validate its integer score
in 0..100, and compare the highest score with the policy cutoff. Keep scores in
message order so Stage 7 can combine evidence within each message.

The optional `classifier` policy object contains only `threshold`. Without it,
make no classifier calls. Preserve original message fields, keep the origin's
credentials away from the classifier, and preserve forwarded request bytes.

## Simplified scope

Sequential calls are sufficient. Set a 200 ms HTTP timeout per call. If any
call fails, discard classifier scores and retain the existing denylist and
heuristic decision. This is one fixed fail-open fallback; block/log mode still
controls input detection matches. Report detector availability and, on success,
the highest classifier score through response headers.

The base course no longer requires parallel calls, a shared batch deadline,
worker cancellation, slow-body deadline enforcement, or configurable failure
modes. Those are possible service-resilience extensions, rather than additional
requirements inherited by later stages.

## Verification scope

The classifier fixture supplies deterministic synthetic responses. Checks cover
the paraphrased attack, complete conversation history, inclusive and changed
cutoffs, a deliberate false positive, original roles and text parts, null
content, empty conversations, malformed or invalid scores, HTTP and transport
failures, a single stalled response, partial-score discard, heuristic fallback,
denylist precedence, log mode, and clean traffic after failures and matches.

A sequential reference must pass. A reference without a timeout must fail the
single stalled-response case, with scheduling headroom around the 200 ms timeout.
No relative-latency benchmark or dripping response is used. Earlier-stage
solutions must fail because they never contact the classifier. Both Go and
Python cumulative solutions must retain earlier stage behavior.

## Fidelity

This is a local course adapter, not a Cloudflare API. Its score increases with
risk, unlike Cloudflare's documented injection score. Synthetic scores teach
integration and decision flow; they do not measure model detection quality.
Learners need no model account, training, or paid API calls.

Sources:

- https://developers.cloudflare.com/waf/detections/ai-security-for-apps/prompt-injection/
- https://pkg.go.dev/net/http#Client
- https://docs.python.org/3/library/urllib.request.html
