#!/bin/bash

export GITLOG="$(git log --pretty=format:"* %an: <https://github.com/touchbistro/$GITHUB_REPO_NAME/commit/%H|%s>" $LAST_DEPLOYED_SHA..$REVISION)"

generate_slack_post_data()
{
  cat <<EOF
{
  "text":"$ICON $DEPLOY_MESSAGE (<$SHIPIT_LINK|logs>|<$DIFF_LINK|diff>)\n$GITLOG"
}
EOF
}
