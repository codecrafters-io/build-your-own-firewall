# Stage 4: Defeat evasion

The approved course design and the request to continue with stage 4 establish
this scope. The allocated slug is `wt4` and its difficulty is `medium`.

| Objective                 | Assertions                                                                                                                                                 | New harness                                                                       |
| ------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| Inspect a normalized copy | Disguised denied terms are blocked before reaching the origin. Log matches and clean requests retain their original bytes. Decoding stops after one layer. | Normalization policy fixtures, quoted request diagnostics, reusable policy runner |

## Contract decisions

The new `normalization` policy object enables removal of policy-supplied
zero-width characters, NFKC, Unicode case folding, and a single
character-mapping pass using the supplied homoglyph table. Removal comes first
because an invisible character between a base letter and a combining mark stops
NFKC from composing them, which would hide an accented term such as `café`.
Denylist terms are already normalized. Policies without this object retain
literal matching, preserving the earlier stages' explicit case-sensitive
contract. The concurrent stage-3 update adds text-part extraction, null content,
and fail-closed parsing. Stage 4 preserves that contract and normalizes each
extracted text string independently.

Inspect both the normalized original text and one decoded variant when the
entire original text string is standard base64 containing UTF-8. Decode before
normalizing so case folding cannot corrupt base64. Normalize the decoded text
with the same pipeline. Do not recursively decode or rewrite requests.

The source design leaves the encoded span unspecified. Whole-string base64 is
the bounded interface for this stage. Embedded fragments, URL-safe base64,
recursive decoding, and general confusable detection are outside scope. Any
string the standard decoder accepts is inspected, including one with nonzero
unused bits before the padding. Rejecting such noncanonical encodings would let
an attacker change one character and skip inspection of the same decoded text.
No assertion requires a full Unicode confusables database or all invisible
characters. The policy defines the exact mapping and removal sets.

## Sources and scope

- [Approved course design](https://docs.google.com/document/d/1hmB4UUlX-47w43dQ5K2IPOlStJlqUSdh7LnG6PKbLeA/edit)
- [Unicode normalization forms](https://www.unicode.org/reports/tr15/)
- [Unicode security mechanisms](https://www.unicode.org/reports/tr39/)
- [Base64, RFC 4648](https://www.rfc-editor.org/rfc/rfc4648)
- [Go case folding](https://pkg.go.dev/golang.org/x/text/cases#Fold)
- [Python normalization](https://docs.python.org/3.14/library/unicodedata.html#unicodedata.normalize)

Reviewed the reports' contents and the RFC's encoding sections. Normalization,
case folding, character confusables, and standard base64 are relevant.
Identifier profiles, script restrictions, domain names, other base encodings,
streaming normalization, and Unicode conformance machinery are outside this
course stage. The small supplied table is a teaching policy rather than a claim
of UTS #39 conformance or Cloudflare detector fidelity.

Go uses `golang.org/x/text` pinned to v0.32.0 with Go 1.25. The SDK copies the
first-stage module manifests into every later solution, so the dependency is
declared in the shared Go starter. Compilation propagates only that module
metadata to earlier generated Go artifacts, with their program logic unchanged.
Python 3.14 uses `unicodedata` and `str.casefold`. Fixtures use established
characters whose normalization is stable across the libraries' Unicode versions.
Sharp s ensures plain lowercasing cannot pass as full case folding.

## Verification plan

The four scenario groups cover full-width uppercase text, invisible characters
combined with a Cyrillic look-alike, a bounded base64 sequence, and a log-mode
preservation sequence. Eight SDK requests exercise both matching and nonmatching
traffic, including a zero-width character between `e` and a combining accent and
base64 whose final character has nonzero unused bits. Log mode supplies a Greek
mapping instead of the Cyrillic mapping.

Record focused regressions for every omitted transform, lowercasing in place of
case folding, a hardcoded map, skipping base64, folding before decoding,
skipping decoded-text normalization, recursively decoding, rewriting the
forwarded request, removing zero-width characters after NFKC, and rejecting
noncanonical base64. Reuse origin accounting, error checks, response capture,
and process cleanup from earlier stages. Quote request bytes so invisible
characters remain visible in diagnostics.

Run build, vet, formatting, all regression fixtures, and 100 repetitions of
success and representative failures. Mutation-check the new output normalizer.
Compare fixed normalization examples through Python and the pinned Go library.
Run Go/Python solutions with the platform scripts, verify that prior solutions
fail this stage, regenerate SDK diffs, and manually check HTTP traffic.

The course-wide attack replay and scoreboard remain separate work. This stage
uses deterministic local fixtures and does not claim model detection quality.

## Verification results

- Build, vet, formatting, the full merged regression suite, and the Stage 4
  release launcher pass. Fifteen golden scenarios cover the reference and
  focused broken implementations.
- The earlier six-request reference, bounded-decoding, and request-preservation
  regressions each passed 100 repetitions, including the three earlier stages in
  the success run. After adding the accent-order and unused-base64-bit cases,
  the expanded regression suite and both language solutions passed final checks.
- Go 1.25 and Python 3.14 solutions pass all four implemented stages through the
  platform scripts. Their previous-stage solutions fail the new full-width
  prompt check because a denied request reaches the origin.
- Fixed normalization examples agree across the actual Go helper with pinned
  `x/text` and Python 3.14. The expected normalized words are independently
  checked, including `ignore previous instructions` and `strasse`.
- Manual curl checks verify full-width text, disguised text parts, encoded
  Unicode, preserved log-mode traffic, clean Unicode bytes, bounded decoding,
  non-UTF-8 decoded bytes, and inherited fail-closed parsing.
- The new output-normalization unit test rejects a deliberate mutation in an
  isolated copy. A concurrent Stage 3 crash fixture exposed an allocated client
  port in its traceback. Comparison normalization now removes just that port,
  retaining the traceback and failure verdict.
- SDK regeneration is stable. Earlier Go artifacts receive the shared module
  dependency, while Stage 4 carries the expanded Stage 3 parsing helpers
  verbatim. Generated Python bytecode was removed before regeneration.

All checks use local mocks and dummy credentials. No live model, tester release,
or hosted CI result is claimed.
