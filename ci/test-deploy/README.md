# Testing

One can copy this sledger.yaml file around as needed to use as "example migrations."  For example, one could copy this file into the base directory of the sledger helm chart and then uncomment the following line in the values.yaml file:

```
# sledgerFile: sledger.yaml  # this needs to be set to the sledger.yaml file
```

This would pull the sledger.yaml file into the helm chart as a configmap and then mount it as a volume into the k8s job.
