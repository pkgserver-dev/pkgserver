# PackageInstall Controller

Watches PackageRevisions
- only non catalog packages or WET
- WET/Repository Deployment == True -> used for tracking APIService installations 

## Catalog Packages (use DRY approved packages)

// version is irrelavent here since we assume all versions of a given group / kind or resource are within a package
// group/kind -> resource, pkg (not revision), versions
// group/resource -> kind
// -> for an apiService we store apiService in resource

// rbac lookup
// rbac -> group + resources -> version is not relevant, group can be "" or "*" -> means empty group (core)
// regular cr lookup
// group/kind -> resource
// if not found lookup in resource with apiservice resource
// pvar -> leeds to package directly

1. for each package we list the API(s) it supports -> extended apis are defined by CRD/APiService
gr -> kind
gk -> resource, pkg, versions []pkgRev, type

Inventory of api-services
- gk1 -> core
- gk2 -> resource, pkg, versions, type (type packages could also be derived from the fact that there is no pkg referenced)

2. for each package

per cr we lookup the gk and find the package reference or core ref
per rbac role/clusterrole -> we lookup gr, get the kind and than lookup gk and find the relevant packages or core ref
per pvar -> we reference the upstream package

-> the lookup to apiservice is a bit special since the kind is not available, lookup via apiService

pkg -> dependent packagess [gk] -> pkg

## Installed Packages (use WET approved packages)

Inventory of installed packages
- cluster/NS/pkgName -> packageRevision

# Dependency controller 

For each package in the pkgRev for WET packages we have to validate the following things.

1. install the software? works out the dag
req:
- grpc service APIService GET (per cr check apiservices)
answer:
- core -> all good (nothing to be done)
- pkgrev1, pkgrev2, etc -> we need to select a package (maybe latest or installed ocnce)

2. Is the package already installed or not?
req:
- grpc service PackageRevision GET

3. if the software is not installed -> create a packageVariant.
Upstream 
-> use the latest revision from the list
Downstream 
-> repo: same as the one from the original package
-> package: clusterPrefix + packageName

WHERE DO WE RETAIN THE PACKAGE DEPENDENCIES ?


How do we upgrade a package ?


Package Design Rules:
- owner of a package is a single role
- the package that offers a service should be contained
- configuration of the services should remain seperate


## maps

gk 
-> resource
-> version -> []pkgRevs // we need to check if all pkgRev belong to the same package

gr
-> kind

pkg
-> []pkgRevs -> Dependencies

install pkgRev -> is this version installed or not?
check the dependencies of the pkgRev


api := a/b
versions
[v1][pkgArev1, pkgArev2]
[v2][pkgArev2, pkgArev3]

pkgA
[pkgArev1][]dependencies
[pkgArev2][]dependencies
[pkgArev3][]dependencies
[pkgArev4][]dependencies


TODO:
- validate delete of a pkgRev -> check if in use
- validate upgrade of a pkgRev