# GOV.UK Replatforming test app

This is a very simple test app to help GOV.UK developers familiarise themselves with the new Kubernetes-based hosting.

## How to use this app

Navigate to the test app url on your browser and set the status parameter as the status response you want, e.g. <test app url>?status=200

## Run the app locally

The app is intended to run on the GOV.UK Kubernetes clusters, but it is also possible to run it locally.

```sh
docker build -t govuk-replatform-test-app .
```

```sh
docker run --rm -p3000:3000 -p9394:9394 govuk-replatform-test-app
```

You should then be able to browse the web app on http://localhost:3000/ and the Prometheus metrics on http://localhost:9394/metrics.
