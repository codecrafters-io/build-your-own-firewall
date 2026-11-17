import csv
import hashlib
import json
import unittest
from unittest.mock import patch
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'model_development'))
from evaluate import evaluate
from model import load_model, score

FOLDER = ROOT / 'model_development'


class ModelTests(unittest.TestCase):
    # The frozen model identifies the exact training data used to produce it.
    def test_training_provenance(self):
        # Arrange
        training = (FOLDER / 'training.csv').read_bytes()
        # Act
        checksum = hashlib.sha256(training).hexdigest()
        # Assert
        self.assertEqual(load_model()['training_sha256'], checksum)

    # Evaluation examples are not duplicated in the training file.
    def test_heldout_text_is_separate(self):
        # Arrange
        with (FOLDER / 'training.csv').open(newline='') as source:
            training = {row['text'].casefold() for row in csv.DictReader(source)}
        with (FOLDER / 'heldout.csv').open(newline='') as source:
            heldout = {row['text'].casefold() for row in csv.DictReader(source)}
        # Act
        overlap = training & heldout
        # Assert
        self.assertEqual(overlap, set())

    # The shipped inference model distinguishes the two introductory examples.
    def test_worked_examples(self):
        # Arrange
        model = load_model()
        # Act
        attack = score(['union select password from users'], model)
        benign = score(['tea cup'], model)
        # Assert
        self.assertGreaterEqual(attack, 70)
        self.assertLess(benign, 70)

    # The Stage 6 exercise uses the frozen model's actual normalized inputs.
    def test_stage_six_probe_score(self):
        # Arrange
        inputs = ['/search', "1' and sleep(5)#", '']
        # Act
        observed = score(inputs, load_model())
        # Assert
        self.assertEqual(observed, 77)

    # Evaluation uses the same five whitespace characters as the teaching policy.
    def test_heuristic_whitespace(self):
        # Arrange
        cases = [(' ', True), ('\t', True), ('\n', True), ('\f', True),
                 ('\r', True), ('\v', False), ('\u2028', False), ('\u0085', False)]
        rows = [{'text': f'or{space}1=1 --', 'attack': '1'} for space, _ in cases]
        # Act
        with patch('evaluate.csv.DictReader', return_value=rows), \
                patch('evaluate.score', return_value=0):
            observed = evaluate()['observations']
        # Assert
        for (space, expected), observation in zip(cases, observed):
            with self.subTest(whitespace=repr(space)):
                self.assertEqual(observation['blocked']['rules'], expected)
                self.assertEqual(observation['blocked']['combined'], expected)

    # The committed report is reproducible from the frozen model and held-out set.
    def test_evaluation_report(self):
        # Arrange
        expected = json.loads((FOLDER / 'evaluation.json').read_text())
        # Act
        observed = evaluate()
        # Assert
        self.assertEqual(observed, expected)


if __name__ == '__main__':
    unittest.main()
