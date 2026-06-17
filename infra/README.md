# CDK Infrastructure

AWS CDK v2 infrastructure for `card-onboarding-workers`.

Context values:

- `env`: deployment environment suffix, defaults to `dev`
- `maxFileSizeBytes`: preprocessor file size limit, defaults to `10485760`
- `onboardServiceBaseUrl`: required by `card-onboarding-worker`, defaults to `http://localhost:8080`
- `onboardServiceTimeout`: worker HTTP timeout, defaults to `5s`

Run synth from the repository root:

```sh
make cdk-synth ENV=dev ONBOARD_SERVICE_BASE_URL=https://example.internal
```
