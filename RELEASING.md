# Release the inbound WAF course

The old `v1` tester grades the retired AI gateway. Release `v2` introduced the
inbound WAF, and `v3` carries the operability fixes and strengthened grader.
Keeping the same stage IDs does not make the retired tester compatible. A local
pass does not update the hosted tester.

The current ten-stage course removes rate limiting (`ou9`). Publish a matching
tester before updating the hosted course. Response inspection (`as4`) and
virtual patching (`mu9`) keep their slugs and now occupy Stages 9 and 10.

The integrated application exercises also require the matching tester. It owns
the protected application, embeds the frozen model from `model_development`, and
prints direct/application-versus-firewall responses and the final replay.
Learner starters contain only the firewall project; there is no separate lab.

Course CI pins an SDK revision that downloads release archives through GitHub's
asset API. The public release-download route returned repeated `504` responses
during the v3 rollout while the asset API served the same archive successfully.
Keep the pin until that downloader fix is available on SDK main.

1. Put the matching course and `firewall-tester` changes on reviewable branches.
   From this course checkout, run `make test`. In the tester repository, run its
   Test workflow with `course_ref` set to the exact course commit to verify the
   pair. It checks the references, grader regressions, model, and real tester
   CLI.
2. Merge the coordinated changes into both repositories. Keep the hosted WAF
   course rollout disabled while the published tester still grades the old
   curriculum. The SDK's course checks download the latest published tester, so
   those checks cannot validate the WAF until the new release exists.
3. Tag the reviewed tester commit with the next unused `v*` release tag and push
   it. The Publish workflow now requires the complete Test workflow to pass
   against the course's `main` branch before packaging either architecture.
4. Confirm that the new GitHub release is the latest non-prerelease release and
   contains `<tag>_linux_amd64.tar.gz` and `<tag>_linux_arm64.tar.gz`, each with
   `tester` and `test.sh`. These names are used by course-sdk.
5. Rerun the course's hosted SDK checks for Go and Python. Update the hosted
   course to select the new tester release, then submit a health-check solution
   and both final references through the learner interface before enabling the
   rollout. Confirm that the previous-stage implementation fails the new stage.

Publishing a tag and selecting the hosted tester are separate release actions.
The local fixes and checks do not perform either action.
