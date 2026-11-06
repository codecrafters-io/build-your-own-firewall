"""Dependency-free inference for the course's small synthetic-text classifier."""
import json
import math
from pathlib import Path


def features(text):
    text = text.casefold()
    return {text[index:index + 3] for index in range(max(0, len(text) - 2))}


def score(inputs, model):
    present = features('\n'.join(inputs))
    logit = model['bias'] + sum(weight for token, weight in model['weights'].items() if token in present)
    probability = 1 / (1 + math.exp(-max(-40, min(40, logit))))
    return round(100 * probability)


def load_model():
    return json.loads(Path(__file__).with_name('model.json').read_text())
