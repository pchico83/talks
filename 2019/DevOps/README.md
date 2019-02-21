# Full Stack App Development with React, Node.js and Okteto

## Preparing your cluster

Get your Kubernetes credentials from https://k8spin.cloud/

## Deploy the Movies React App

Clone this repository and go to the react-multi-kubectl folder

```console
git clone https://github.com/pchico83/talks
cd 2019/DevOps
```

Run the Movies App by executing:

```console
kubectl apply -f manifests
```

Wait for one or two minutes until the application is running.

The Movies App is available on movies.europe.okteto.net.

## Develop as a Cloud Native Developer

Let's start working on the front-end service first. In order to activate your Cloud Native Development on it, got to your terminal and execute:

```console
cd movies-frontend
okteto up
```

The `okteto up` command will start a remote development environment that automatically synchronizes and applies your code changes without rebuilding containers (eliminating the **docker build/push/pull/redeploy** cycle). 

Since you are already working in your cluster, the API service **will be available at all times**. No need to mock the service nor use any kind of redirection.

Our React example also uses [Parcel](https://parceljs.org/) to bundle the application and enable [Hot Module Replacement](https://parceljs.org/hmr.html) to automatically update modules in the browser at runtime without needing a whole page refresh.

With Okteto we can bring this development experience to the cluster with the same flow you would have in local.

In your IDE you can now edit the file `movies-frontend/src/App.jsx` and change the `Movies` text in line 55 to `Netflix`. Save your changes. 

Go back to the browser, and cool! Your changes are automatically live with no need to refresh your browser!

# Team working on multiple services

Imagine now that you are working on a new feature that also needs some API work from another member of your team.

Let's keep the front-end service in *development mode* and enable it for the API service too. On a different terminal screen execute the command below:

```console
cd movies-api
okteto up
```

The API service will be now ready for development. `okteto up` launched the *express* server directly. You can configure a default command to run in the remote container in your [`okteto.yml`](movies-api/okteto.yml) file. 

In your IDE edit the file `movies-api/data/shows.json` and remove some of the values in the results list. Save your changes. 

Go back to your browser where you were working on the front-end and refresh. The changes to the API were automatically applied and consumed by the front-end. No docker nor kubectl to see your teammate changes on the API service.

## Cleanup

Cancel the `okteto up` command by pressing `CTRL + C` and run the following command to deactivate the cloud native environment in both front-end and api directories:

```console
okteto down
``` 

Run the following command to remove the resources created by this guide: 

```console
kubectl delete -f manifests
```