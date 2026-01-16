# kcogs

Get visibility into Cost of Goods Sold in Kubernetes clusters.

⚠️ This is a WORK IN PROGRESS. It currently only pulls AWS cost data.

## Localdev

Currently, the only way to use the app is to clone this repo and run it locally.

### Prerequisites

- Go 1.25
- Node 25
- Valid AWS credentials
- Valid kubeconfig file(s)

### Running the App

AWS credentials are required to retrieve cost data. Set `AWS_PROFILE` before starting the app.

```sh
export AWS_PROFILE=profile_name
```

Start the app:

```sh
make install && make dev
```

Open http://localhost:3000
