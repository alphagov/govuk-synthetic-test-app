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
