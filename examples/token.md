## setup environment

GITHUB_USERNAME=henderiw
GITHUB_TOKEN=

## create github personal access token

kubectl create secret generic git-pat \
  --from-literal=username=${GITHUB_USERNAME} \
  --from-literal=password=${GITHUB_TOKEN} \
  --type=kubernetes.io/basic-auth

kubectl create secret generic git-pat \
  --namespace="config-management-system" \
  --from-literal=username=${GITHUB_USERNAME} \
  --from-literal=token=${GITHUB_TOKEN} 