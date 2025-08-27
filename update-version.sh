#!/bin/bash

set -eux

git config --global user.email "govuk-ci@users.noreply.github.com"
git config --global user.name "govuk-ci"

git clone https://${GH_TOKEN}@github.com/alphagov/govuk-synthetic-test-app.git test-app

cd test-app

branch=add-synthetic-test-cronjob

git checkout -b "${branch}"
git pull origin "${branch}"

echo ${IMAGE_TAG} > ".version"

git add ".version"

git commit -m "Update version to to ${IMAGE_TAG}"
git push --set-upstream origin "${branch}"
