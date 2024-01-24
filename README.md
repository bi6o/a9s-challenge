# a9s-challenge
This document should serve as a step-by-step guide to running the challenge locally! Hopefully it works as expected 😅

## Description
This project was built using the `operator-sdk` cli tool to get the boilerplate code. It creates a new type `Dummy` with a single Spec `Message`. It also sets the message to `I'm just a dummy`.

I used the `operator-sdk` to initialize a new project, with domain `a9s-interview.com`, linked to this repo. I also used the sdk to create a new resource of kind `Dummy` in the `interview` group.

I extended the generated controller and type files to accomodate the requirements. 

I was also using `make generate` and `make manifests` to generate and update the goa dn yaml files managed by the sdk.

- Added scaffolding comments to allow CRUD access to different k8s resources.
- The controller gets the Dummy Resource and sets it's `SpecEcho` to it's Spec `Message`.
- It also checks if there is an already existing `Pod` for the `Dummy` Resource.
- If the pod exists: 
    - the controller sets the `Dummy`'s `PodStatus` to the pod's `Status.Phase`
- If the pod does not exist 
    - The controller creates a new pod with `nginx:latest` image and references the `Dummy` resource
    - It tries to get the `Pod` from k8s with a retry (every 500ms, for 30s)
    - Finally it updates the `Dummy`'s `PodStatus` with the pod's `Status.Phase`

There are some integration test cases added, I leveraged the boilerplate test file `internal/controller/suite_test.go`. To run the tests, simply run `make test`. I would have added more cases in a real life scenario!

## Requirements
- `minikube`
- `kubectl`
- `Docker`
- `Make`

## Getting Started
Here are the steps I took for running and testing the code:
- Run `minikube start` to start the cluster
- Run `make docker-build docker-push` to build the docker image and push it to dockerhub
- Run `make deploy` to deploy to k8s
- Run `kubectl apply -f config/samples/dummy.yaml` to create the Dummy Resource
- Run `kubectl get pods -n a9s-challenge-system` to get the pods and their statuses
    - There should be one instance `a9s-challenge-controller-manager-<pod-hash>` with two ready containers in `Running` status, this is the pod related to the implemented controller 
- Run `kubectl describe pod a9s-challenge-controller-manager-<pod-hash>` (cope from above) to see the logs from the controller 
- Run `kubectl get dummy dummy -n default -o yaml` (yaml formatting is optional) to see the created `Dummy` resource
    - It should have a spec message `I'm just a dummy`
    - It should have a `status.specEcho` of `I'm just a dummy`
    - It should have a `status.podStatus` of `Running`
- Run `kubectl logs dummy-pod -n default` to see logs about the pod our controller created