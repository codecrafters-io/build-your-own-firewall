In this stage, you'll combine rule and ML scores using thresholds for each
application route.

### Why tune decisions by route?

A detector can flag a legitimate security article because it contains attack
examples. Blocking it is a false positive. Combine moderate ML scores with rule
evidence, keep a high cutoff for strong evidence from either signal, and use
explicit route thresholds so tuning one endpoint does not exempt the whole
application.

### Policy

Add an optional `decisions` object:

```json
{
  "decisions": {
    "default": {
      "ml_high": 90,
      "ml_medium": 60,
      "heuristic_high": 5,
      "heuristic_medium": 3
    },
    "routes": [
      {
        "path": "/articles",
        "ml_high": 95,
        "ml_medium": 80,
        "heuristic_high": 8,
        "heuristic_medium": 5
      }
    ]
  }
}
```

Exact matching confines route tuning to its intended endpoint; decoding ensures
an encoded spelling such as `/articl%65s` selects the same policy as `/articles`.
Use the first route whose `path` exactly matches the once-percent-decoded
request path, case-sensitively, without NFKC normalization or case folding. This
applies even when inspection normalization is disabled. The query string is
excluded. If no route matches, use `default`. `/articles/extra` and `/Articles`
do not match `/articles`.

### Expected decisions

If a low ML score could cancel strong rule evidence, lowering the model's score
would also bypass your rules. Let either strong signal block on its own, and
require both signals when neither is strong enough alone.

When classification succeeds, these route decisions replace the separate
heuristic and classifier cutoffs. Denylist rules still win first. Otherwise,
evaluate the selected thresholds in this order:

| Condition                                                                               | Rule              |
| --------------------------------------------------------------------------------------- | ----------------- |
| Heuristic score is at least `heuristic_high`                                            | `heuristic_high`  |
| ML score is at least `ml_high`                                                          | `ml_high`         |
| ML score is at least `ml_medium` **and** heuristic score is at least `heuristic_medium` | `combined_scores` |
| None of these conditions                                                                | Allow             |

All comparisons are inclusive. The two scores describe the same HTTP request.
Only the selected route's `heuristic_high` decides when rule evidence alone
blocks. Keep the earlier block/log behavior and diagnostic headers.

If the detector is unavailable, retain the earlier denylist and standalone
heuristic cutoff. Route thresholds do not apply to that fallback. Without
`decisions`, retain the earlier standalone heuristic and classifier cutoffs.

### Examples

With the policy above:

| Path              | Heuristic score | ML score | Decision          |
| ----------------- | --------------- | -------- | ----------------- |
| `/search`         | 3               | 70       | `combined_scores` |
| `/articles`       | 3               | 70       | Allow             |
| `/articles/extra` | 3               | 70       | `combined_scores` |
| `/articles`       | 0               | 95       | `ml_high`         |
| `/search`         | 5               | 10       | `heuristic_high`  |
| `/articles`       | 5               | 10       | Allow             |

The final example (`/articles` with heuristic score `5`) can still be blocked by
the standalone heuristic cutoff if the detector fails. That difference is part
of the specified fallback.

An allowed request must reach the origin unchanged. A combined block must return
HTTP `403` with:

```json
{ "error": { "rule": "combined_scores" } }
```

### Tests

The tester starts the protected application and detector, writes the trusted
policy, and starts your firewall with this interface. Addresses and the absolute
policy path vary between runs:

```bash
./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

Run the exercise with:

```bash
codecrafters test
```

The application accepts legitimate security articles containing attack examples.
The tester submits this article using the real classifier:

```bash
curl --include http://127.0.0.1:8080/articles \
  -H 'Content-Type: text/plain' \
  --data-binary 'A security article explaining or 1=1 --'
```

Its heuristic score is `5` and ML score is `69`. First, the tester omits the
route override: default `heuristic_high: 5` selects `heuristic_high`, blocking
the legitimate article. It then restarts with the `/articles` override shown
above. Your firewall must preserve the article, returning HTTP `200`,
`X-WAF-Score: 5`, `X-WAF-ML-Score: 69`, and:

```json
{ "article": "A security article explaining or 1=1 --", "published": true }
```

The tester checks that the application receives the original article bytes. A
clean product search must still work. This demonstrates a false positive that
route thresholds can address. A hard denylist match still wins on `/articles`.
Raising score thresholds cannot exempt it.

Additional scripted cases check exact route matching, inclusive cutoffs, the
heuristic floor, denylist precedence, and detector-outage fallback. With
heuristic score `3` and scripted ML score `70`, `/search` selects
`combined_scores` while `/articles` passes under its route thresholds.

The earlier ML-only probe scoring `77` has no heuristic evidence. Under these
combined default thresholds, it is allowed. Raising a cutoff can lose detection
coverage. Check legitimate traffic as well as attack probes when tuning policy.

### Notes

- These are teaching thresholds, not production recommendations. A route
  override changes score thresholds. It cannot disable denylist matches or the
  size, response, and virtual-patch controls covered in later stages.
- Each thresholds object supplies all four fields as positive integers. Route
  matching uses the original decoded path even when inspection uses a
  case-folded copy. Changing the inspection representation must not change
  application routing.
- For the operational tradeoff, see
  [Cloudflare's guidance on false positives and combined score rules](https://developers.cloudflare.com/waf/detections/attack-score/).
