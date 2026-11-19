In this stage, you'll combine rule and ML scores using thresholds for each
application route.

### Why tune decisions by route?

Blocking a legitimate security article because it contains attack examples is
a **false positive**. Route thresholds let you tune that endpoint while keeping
stricter thresholds elsewhere.

### Keep a legitimate article working

The application accepts legitimate security articles containing attack examples.
The tester submits this article using the supplied classifier:

```bash
curl --include http://127.0.0.1:8080/articles \
  -H 'Content-Type: text/plain' \
  --data-binary 'A security article explaining or 1=1 --'
```

Its heuristic score is `5` and ML score is `69`. First, the tester omits the
route override: default `heuristic_high: 5` selects `heuristic_high`, blocking
the legitimate article. It then restarts with the `/articles` override shown
below. Your firewall must preserve the article, returning HTTP `200`,
`X-WAF-Score: 5`, `X-WAF-ML-Score: 69`, and:

```json
{"article":"A security article explaining or 1=1 --","published":true}
```

The application must receive the original article bytes, and a clean product
search must still work. A hard denylist match still wins on `/articles`.
Raising score thresholds cannot exempt it.

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

Use the first route whose `path` exactly matches the once-percent-decoded
request path, case-sensitively, without NFKC normalization or case folding. This
applies even when inspection normalization is disabled. The query string is
excluded. If no route matches, use `default`. `/articl%65s` matches `/articles`.
`/articles/extra` and `/Articles` do not.

### Expected decisions

Let either strong signal block on its own. A low ML score must not cancel
strong rule evidence. Require both signals when neither is strong enough alone.

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

The earlier ML-only probe scores `77` with no heuristic evidence, below the
default `ml_high` of `90`, so it now passes. Lowering `ml_high` would catch it
again.

An allowed request must reach the origin unchanged. A combined block must return
HTTP `403` with:

```json
{ "error": { "rule": "combined_scores" } }
```

### Tests

Run `codecrafters test`. Expect HTTP `200` and
`{"article":"A security article explaining or 1=1 --","published":true}` for the
article once the route override applies. The tester verifies that:

- The article above is blocked under default thresholds and reaches the
  application unchanged with the `/articles` override.
- Route matching is exact, cutoffs are inclusive, and combined decisions require
  the heuristic floor.
- With heuristic score `3` and scripted ML score `70`, `/search` selects
  `combined_scores` while `/articles` passes under its route thresholds.
- Denylist matches retain precedence, and detector outages use the earlier
  fallback rules.

Raising a cutoff can lose detection coverage. Check legitimate traffic as well
as attack probes when tuning policy.

### Notes

- These are teaching thresholds. They are not production recommendations. A
  route override changes score thresholds. It cannot disable denylist matches or
  the size, response, and virtual-patch controls covered in later stages.
- Each thresholds object supplies all four fields as positive integers.
  Inspection normalization must not change application routing.
- For the operational tradeoff, see
  [Cloudflare's guidance on false positives and combined score rules](https://developers.cloudflare.com/waf/detections/attack-score/).
