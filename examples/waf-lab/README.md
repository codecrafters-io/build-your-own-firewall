# Local WAF lab

This lab lets you exercise your WAF against a real HTTP application and a small
supplied ML model. Use the final reference solution to explore, or your own
solution as you finish each capability. All records and secrets are dummy
fixtures, and both supplied services bind to loopback.

Run the commands from the course repository root. The lab services need Python
3.12 or later and use only its standard library.

## Start the services

In one terminal:

```sh
python3 examples/waf-lab/application.py
```

In another:

```sh
python3 examples/waf-lab/detector.py
```

The application listens on `127.0.0.1:8081`; the detector listens on
`127.0.0.1:8082`. To inspect inference directly:

```sh
curl --include http://127.0.0.1:8082/classify \
  -H 'Content-Type: application/json' \
  --data-binary '{"inputs":["/search","union select password from users"]}'
```

The JSON response contains an integer `score` from `0` to `100`. Higher means
more suspicious under this model. This score is not a calibrated probability of
an attack.

## Start the WAF

In a third terminal, choose your Go or Python solution and use the absolute path
to `examples/waf-lab/policy.json`. For example, from the repository root:

```sh
./solutions/python/10-mu9/code/your_program.sh \
  --origin http://127.0.0.1:8081 \
  --detector http://127.0.0.1:8082 \
  --policy "$PWD/examples/waf-lab/policy.json"
```

Use the Go solution's launcher instead to try the same contract in Go. The
shipped launchers require the language tools described by the starter
repository.

## Compare the vulnerable endpoint and its virtual patch

The application trusts the client's requested fields. A direct request exposes
the dummy private field:

```sh
curl --include http://127.0.0.1:8081/api/export \
  -H 'Content-Type: application/json' \
  --data-binary '{"fields":["password_hash"]}'
```

Through the WAF, the same request should return `403` with
`error.rule: "export-fields"`:

```sh
curl --include http://127.0.0.1:8080/api/export \
  -H 'Content-Type: application/json' \
  --data-binary '{"fields":["password_hash"]}'
```

A public-field request through the same WAF must succeed:

```sh
curl --include http://127.0.0.1:8080/api/export \
  -H 'Content-Type: application/json' \
  --data-binary '{"fields":["customer_id","name"]}'
```

Also try `/debug`: the direct application response contains a dummy internal
key, while the WAF should return `403` with rule `response_leak`.

Stop each service with Ctrl-C when finished. Keep this intentionally unvalidated
lab application on loopback.

## Understand and evaluate the supplied ML model

`model.json` is a frozen logistic-regression model. `train.py` learns weights
for binary character-trigram features from 52 labeled synthetic examples in
`training.csv`. The training file's SHA-256 is recorded in the model; the
training script never reads `heldout.csv`. Learners need not train the model to
complete the course.

Rebuild it and evaluate the separate 24-example held-out set:

```sh
python3 examples/waf-lab/train.py
python3 examples/waf-lab/evaluate.py
python3 -m unittest discover -s examples/waf-lab -p 'test_*.py'
```

The checked-in `evaluation.json` records the observed results:

| Decision strategy                         | Attacks blocked | Legitimate examples blocked |
| ----------------------------------------- | --------------- | --------------------------- |
| Sample signatures and heuristic cutoff    | 3/12            | 2/12                        |
| ML cutoff at 70                           | 10/12           | 0/12                        |
| Sample combined thresholds and signatures | 6/12            | 2/12                        |

The combined policy performs worse than ML alone on this set. That is a useful
tuning result: adding conditions and signatures does not guarantee improvement.
Inspect the per-example scores before changing thresholds. Do not tune against
this held-out set and continue treating its results as independent validation;
use a new set for that purpose.

This tiny, authored synthetic dataset is a teaching fixture. It is not
representative production traffic, an adversarial robustness study, or a vendor
benchmark. The classifier misses the private-field export, which the
endpoint-specific patch is designed to catch.
