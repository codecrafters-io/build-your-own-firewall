In this stage, you'll combine the heuristic and detector scores into one decision, with different thresholds for different parts of the application.

### False Positives

Blocking a harmless request is called a [false positive](https://developers.google.com/machine-learning/glossary#false-positive-fp). The origin's articles page shows how they happen. A security article can quote attacks, like `A security article explaining or 1=1 --`. That text fires both `sql-or` and `sql-comment`, for a heuristic score of 3 + 2 = `5`, and the detector scores it `69`. With the heuristic threshold at `5`, your firewall blocks the article.

Raising the threshold everywhere would let the article through, but it would also let real attacks through on other pages. Instead, your firewall will take thresholds per route, and combine its two scores, so neither one decides alone unless it's high.

### The Decisions Policy

The policy can now have a `decisions` object:

```json
{
  "decisions": {
    "default": {"ml_high": 90, "ml_medium": 60, "heuristic_high": 5, "heuristic_medium": 3},
    "routes": [
      {"path": "/articles", "ml_high": 95, "ml_medium": 80, "heuristic_high": 8, "heuristic_medium": 5}
    ]
  }
}
```

`decisions` works together with `classifier`, since it needs the detector's score.

To pick the thresholds, compare the request's path to each route's `path`. Split off the query first, then percent-decode the path once, without normalizing it. Use the first route that matches exactly, including case. If none matches, use `default`. So `/articles` matches the route above, but `/articles/extra` and `/Articles` don't.

### Deciding

When the detector answered, these checks replace the heuristic and classifier thresholds from earlier stages. When it failed, decide as in earlier stages, with the denylist and the heuristic `threshold`.

A denylist rule still decides first. Otherwise, go through the selected thresholds in this order, and use the first check that passes. Every comparison uses ≥.

| Check | Rule |
| --- | --- |
| heuristic score ≥ `heuristic_high` | `heuristic_high` |
| detector score ≥ `ml_high` | `ml_high` |
| detector score ≥ `ml_medium` and heuristic score ≥ `heuristic_medium` | `combined_scores` |
| none of the above | allowed |

So either score can block a request on its own when it's high enough. In the middle range, it takes both. For example, with the policy above:

| Path | Heuristic score | Detector score | Result |
| --- | --- | --- | --- |
| `/search` | 3 | 65 | `combined_scores` |
| `/articles` | 3 | 65 | allowed |
| `/search` | 6 | 10 | `heuristic_high` |
| `/articles` | 0 | 96 | `ml_high` |
| `/articles` | 5 | 69 | allowed |

These rules follow the policy's `mode`, like any other rule.

### Tests

The tester will write a policy file and execute your program like this:

```bash
$ ./your_program.sh --origin http://127.0.0.1:8081 --detector http://127.0.0.1:8082 --policy /tmp/policy.json
```

It will publish the security article with the real detector:

```bash
$ curl -i http://127.0.0.1:8080/articles -H 'Content-Type: text/plain' --data-binary 'A security article explaining or 1=1 --'
```

First, the policy has `decisions` without the `/articles` route, so the default `heuristic_high` of `5` should block the article with `heuristic_high`. Then the tester restarts your program with the route above. Your firewall should now publish the article: the origin's `200 OK` response, with `X-WAF-Score: 5` and `X-WAF-ML-Score: 69`. The origin should receive the article unchanged, and a product search, `GET /search?q=tea+cup`, should still work.

It will also use a fake detector to check routes that almost match, scores right at a threshold, the order of the checks, and a denylist rule on `/articles`.

The tester will verify that:

- The first route whose path matches exactly picks the thresholds, otherwise `default`
- The checks run in the order above, and every comparison uses ≥
- A denylist rule still decides first, on every route
- Allowed requests reach the origin unchanged

### Notes

- Raising a threshold can let attacks through too. Under the default thresholds above, the `1' AND SLEEP(5)#` probe from earlier stages scores `0` and `77`, so it's now allowed. When you tune a policy, check attacks as well as harmless traffic.
- These thresholds are teaching examples, not recommendations for a real application.
