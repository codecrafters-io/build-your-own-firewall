"""Report observed mistakes on held-out synthetic examples, without training."""
import csv
import json
import re
from pathlib import Path
from model import load_model, score


def evaluate():
    model = load_model()
    with Path(__file__).with_name('heldout.csv').open(newline='') as source:
        rows = list(csv.DictReader(source))
    totals = {name: {'attacks': 0, 'attacks_blocked': 0, 'benign': 0, 'benign_blocked': 0}
              for name in ['rules', 'ml_cutoff', 'combined']}
    observations = []
    for row in rows:
        text = row['text'].casefold()
        ml = score([text], model)
        hard = any(term in text for term in ['union select', '<script'])
        heuristic = 3 * bool(re.search(r'or\s+1\s*=\s*1', text)) + 2 * ('--' in text)
        decisions = {'rules': hard or heuristic >= 5, 'ml_cutoff': ml >= 70,
                     'combined': hard or heuristic >= 5 or ml >= 90 or (ml >= 60 and heuristic >= 3)}
        attack = row['attack'] == '1'
        for name, blocked in decisions.items():
            totals[name]['attacks' if attack else 'benign'] += 1
            totals[name]['attacks_blocked' if attack else 'benign_blocked'] += int(blocked)
        observations.append({'text': row['text'], 'attack': attack, 'score': ml, 'blocked': decisions})
    return {'dataset': 'small synthetic held-out set; not production accuracy',
            'training_sha256': model['training_sha256'], 'totals': totals, 'observations': observations}


if __name__ == '__main__':
    print(json.dumps(evaluate(), indent=2))
