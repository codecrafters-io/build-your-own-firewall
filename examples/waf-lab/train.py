"""Rebuild the frozen teaching model from training.csv; never reads heldout.csv."""
import csv
import hashlib
import json
import math
from pathlib import Path
from model import features

folder = Path(__file__).resolve().parent
raw = (folder / 'training.csv').read_bytes()
with (folder / 'training.csv').open(newline='') as source:
    rows = list(csv.DictReader(source))
vocabulary = sorted(set().union(*(features(row['text']) for row in rows)))
indices = {token: index for index, token in enumerate(vocabulary)}
samples = [([indices[token] for token in sorted(features(row['text']))], int(row['attack'])) for row in rows]
weights = [0.0] * len(vocabulary)
bias = 0.0
for epoch in range(1200):
    gradient = [0.0] * len(weights)
    bias_gradient = 0.0
    for active, label in samples:
        logit = bias + sum(weights[index] for index in active)
        error = 1 / (1 + math.exp(-max(-40, min(40, logit)))) - label
        bias_gradient += error
        for index in active:
            gradient[index] += error
    bias -= 0.4 * bias_gradient / len(samples)
    weights = [weight - 0.4 * (value / len(samples) + 0.005 * weight)
               for weight, value in zip(weights, gradient)]
model = {'kind': 'binary-character-trigram-logistic-regression',
         'training_sha256': hashlib.sha256(raw).hexdigest(), 'training_rows': len(rows),
         'bias': round(bias, 8), 'weights': {token: round(weight, 8) for token, weight in zip(vocabulary, weights)}}
(folder / 'model.json').write_text(json.dumps(model, sort_keys=True, indent=2) + '\n')
print(f'Trained on {len(rows)} synthetic examples; {len(weights)} features. Held-out data was not read.')
