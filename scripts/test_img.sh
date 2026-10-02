#!/bin/bash

set -e

docker build -t ghcr.io/alphagov/govuk/govuk-synthetic-test-app:manual-pipeline-test .
docker push ghcr.io/alphagov/govuk/govuk-synthetic-test-app:manual-pipeline-test 
kubectl delete -f ./govuk-synthetic-test-app.yaml
kubectl apply -f ./govuk-synthetic-test-app.yaml

