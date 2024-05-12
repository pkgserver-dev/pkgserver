# PackageScheduler Controller

Watches PackageRevisions
- only non catalog packages or WET packages

Leverages the catalogstore info:
- pkgrev -> other pkg dependencies
- check if the resolution was successfull

Walk through all the packages from the catalogstore
1. check dependencies
2. check completness of dependencies
3. build a dag

Execute the installation as per dag
- start with the root an go down
- select an installed package or latest revision
- create a pvar:
    - upstream -> dependency; revision is installed or latests
    - downstream -> target, repository is the samme; realm/package comes from the selected package
    - packageContext:
        - readinessGates: none
        - labels: none
        - annotations: approval
        - inputs: none
- report the status in the packageRevision Condition

where do we keep track of the dependencies
- installed pkg:
    -> which revision
    -> pkg revisions dependent on this

## what does isInstalled mean



[configsync usage](https://github.com/nephio-project/nephio/issues/210)

```yaml
apiVersion: configsync.gke.io/v1beta1
kind: RootSync
metadata:
  name: nephio-workload-cluster-sync
  namespace: config-management-system
spec:
  sourceFormat: unstructured
  git:
    repo: https://github.com/nephio-test/test-edge-01
    branch: main
    dir: /deployed-packages
    auth: ...some auth stuff not shown here...
```

```yaml
apiVersion: configsync.gke.io/v1beta1
kind: RootSync
metadata:
  name: nephio-workload-cluster-sync
  namespace: config-management-system
spec:
  sourceFormat: unstructured
  git:
    repo: https://github.com/nephio-test/test-edge-01
    branch: main
    dir: /free5gc-upf
    revision: free5gc-upf/v6
    auth: ...some auth stuff not shown here...
```

To be discussed:
- how do we create input context ?