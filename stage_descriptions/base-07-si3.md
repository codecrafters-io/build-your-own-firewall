In this stage, you'll combine rule and ML scores using thresholds for each
application route.

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

Use the first route whose `path` exactly matches the once-decoded request path,
case-sensitively. The query string is excluded. If no route matches, use
`default`. `/articles/extra` and `/Articles` do not match `/articles`.

### Expected decisions

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
A low ML score never cancels strong rule evidence; otherwise, an attacker who
lowers the model's score would also bypass your rules. The ML score can add a
block, and only the selected route's `heuristic_high` decides when rule evidence
alone blocks. Keep the earlier block/log behavior and diagnostic headers.

If the detector is unavailable, retain the earlier denylist and standalone
heuristic cutoff. Route thresholds do not apply to that fallback. Without
`decisions`, retain Stage 6 behavior.

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

The last request can still be blocked by the standalone heuristic cutoff if the
detector fails. That difference is part of the specified fallback.

An allowed request must reach the origin unchanged. A combined block must return
HTTP `403` with:

```json
{ "error": { "rule": "combined_scores" } }
```

### Notes

- These are teaching thresholds, not production recommendations. A route
  override changes score thresholds; it cannot disable denylist matches or the
  later size, rate, response, and virtual-patch controls.
- Each thresholds object supplies all four fields as positive integers.
- Route matching uses the original decoded path even when inspection uses a
  case-folded copy. Changing the inspection representation must not change
  application routing.
- For the operational tradeoff, see
  [Cloudflare's guidance on false positives and combined score rules](https://developers.cloudflare.com/waf/detections/attack-score/).
