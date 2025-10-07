# GOV.UK Synthetic test app

This is a simple synthetic test app used to test that the GOV.UK CI/CD pipeline is working.

## Run the app locally

The app is intended to run on the GOV.UK Kubernetes clusters, but it is also possible to run it locally.

```sh
docker build -t govuk-synthetic-test-app .
```

```sh
docker run --rm -p3000:3000 govuk-synthetic-test-app
```

You should then be able to browse the web app on http://localhost:3000/.

## Running Ginkgo tests

NOTE - at the moment the tests only work when deployed on to an EKS cluster as it is using the synthetic-test-assumer role.

The following commands target integration, replace integration with staging / production to target those environments.

- To get the tests running you will need to apply this manifest

`gds aws govuk-integration-platformengineer -- kubectl apply -f ./govuk-synthetic-test-app.yaml -n apps`

- To see the results of the tests you can run this kubectl command

`gds aws govuk-integration-platformengineer -- kubectl logs govuk-synthetic-test-app-runner -n apps`

- To delete the pod after the tests have completed run this kubectl command

`gds aws govuk-integration-platformengineer -- kubectl delete -f ./govuk-synthetic-test-app.yaml -n apps`
