# nacos-operator

nacos-operator is an operator for deploying nacos.

## Overview
With nacos-operator, you can deploy a standalone nacos or  nacos cluster.

## Usage

### NacosStandalone

You can use `NacosStandalone` CRD to create a standalone nacos-server for developing,testing etc.

#### Simple usage example
```shell
kubectl create -f https://github.com/YunWZ/nacos-operator/blob/main/config/samples/nacos_v1alpha1_nacosstandalone.yaml
```

#### Advanced usage examples
The complete `a`CR example is as follows:
```yaml
apiVersion: nacos.yunweizhan.com.cn/v1alpha1
kind: NacosStandalone
metadata:
  labels:
    app.kubernetes.io/name: nacosstandalone
    app.kubernetes.io/instance: nacosstandalone-sample
    app.kubernetes.io/part-of: nacos-operator
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: nacos-operator
  name: nacosstandalone-sample
  namespace: default
spec:
  image: "docker.io/nacos/nacos-server" #Nacos server image
  imagePullSecrets: #a list of sercrets name
    - name: secretNameA
  service:
    type: ClusterIP # Support type is ClusterIP/NodePort/LoadBalancer. Defaults to ClusterIP.
  pvc: # Reference PersistentVolumeClaimSpec
    storageClassName: "default"
    resources:
      requests:
        storage: 10Gi
  livenessProbe:
    HTTPGet: 
      path: /nacos/v2/console/health/liveness #Nacos server liveness endpoint. Defaults to "/nacos/v2/console/health/liveness"
      port: client-http #The port of Nacos server liveness endpoint. Defaults to  client-http (8848)
      scheme: HTTP
    initialDelaySeconds: 5
    periodSeconds: 5
    timeoutSeconds: 10
    successThreshold: 1
    failureThreshold: 5
  readinessProbe: 
    HTTPGet:
      path: /nacos/v2/console/health/readiness #Nacos server readiness endpoint. Defaults to "/nacos/v2/console/health/readiness"
      port: client-http #The port of Nacos server liveness endpoint. Defaults to  client-http (8848)
      scheme: HTTP
    initialDelaySeconds: 5
    periodSeconds: 5
    timeoutSeconds: 3
    successThreshold: 1
    failureThreshold: 3
  startupProbe:
    HTTPGet: 
      path: /nacos/v2/console/health/readiness #Nacos server startup probe. Defaults to "/nacos/v2/console/health/readiness"
      port: client-http #The port of Nacos server liveness endpoint. Defaults to  client-http (8848)
      scheme: HTTP
    initialDelaySeconds: 10
    periodSeconds: 10
    timeoutSeconds: 5
    successThreshold: 1
    failureThreshold: 50
  database:
    mysql:
      jdbcUrl: "jdbc:mysql://%s:%s/%s?characterEncoding=utf8&connectTimeout=1000&socketTimeout=3000&autoReconnect=true&useUnicode=true&useSSL=false&serverTimezone=UTC" #The template will be formatted with dbHost and dbPort (in `dbServer` section).
      dbServer:
        - dbHost: 127.0.0.1
          dbPort: 3306
      dbName: nacos
      secret:
        name: db-user-pw #Sercret name of Mysql database. The secret must contain two keys: `user` and `password`.
  jvmOptions: "-Xmx3072m -Xmn1024m" #Nacos server jvm options.
  resources: # cpu,memory resources of nacos server
    requests:
      cpu: 100m
      memory: 4Gi
    limits:
      cpu: 100m
      memory: 4Gi
  applicationConfig: # ConfigMap name. The ConfigMap will be projected as nacos server conf.
    name: application-config-map
```
### NacosCluster

You can use `NacosCluster` CRD to create a nacos-server cluster for developing,testing,production etc.

Simple usage example:
```shell
kubectl create -f https://github.com/YunWZ/nacos-operator/blob/main/config/samples/nacos_v1alpha1_nacoscluster.yaml
```

#### Advanced usage examples
The complete `a`CR example is as follows:
```yaml
apiVersion: nacos.yunweizhan.com.cn/v1alpha1
kind: NacosCluster
metadata:
  labels:
    app.kubernetes.io/name: nacoscluster
    app.kubernetes.io/instance: nacoscluster-sample
    app.kubernetes.io/part-of: nacos-operator
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: nacos-operator
  name: nacoscluster-sample
spec:
  image: "docker.io/nacos/nacos-server" #Nacos server image
  imagePullSecrets: #a list of sercrets name
    - name: secretNameA
  service:
    type: ClusterIP # Support type is ClusterIP/NodePort/LoadBalancer. Defaults to ClusterIP.
  pvc: # Reference PersistentVolumeClaimSpec
    storageClassName: "default"
    resources:
      requests:
        storage: 10Gi
  livenessProbe:
    HTTPGet:
      path: /nacos/v2/console/health/liveness #Nacos server liveness endpoint. Defaults to "/nacos/v2/console/health/liveness"
      port: client-http #The port of Nacos server liveness endpoint. Defaults to  client-http (8848)
      scheme: HTTP
    initialDelaySeconds: 5
    periodSeconds: 5
    timeoutSeconds: 10
    successThreshold: 1
    failureThreshold: 5
  readinessProbe:
    HTTPGet:
      path: /nacos/v2/console/health/readiness #Nacos server readiness endpoint. Defaults to "/nacos/v2/console/health/readiness"
      port: client-http #The port of Nacos server liveness endpoint. Defaults to  client-http (8848)
      scheme: HTTP
    initialDelaySeconds: 5
    periodSeconds: 5
    timeoutSeconds: 3
    successThreshold: 1
    failureThreshold: 3
  startupProbe:
    HTTPGet:
      path: /nacos/v2/console/health/readiness #Nacos server startup probe. Defaults to "/nacos/v2/console/health/readiness"
      port: client-http #The port of Nacos server liveness endpoint. Defaults to  client-http (8848)
      scheme: HTTP
    initialDelaySeconds: 10
    periodSeconds: 10
    timeoutSeconds: 5
    successThreshold: 1
    failureThreshold: 50
  database:
    mysql:
      jdbcUrl: "jdbc:mysql://%s:%s/%s?characterEncoding=utf8&connectTimeout=1000&socketTimeout=3000&autoReconnect=true&useUnicode=true&useSSL=false&serverTimezone=UTC" #The template will be formatted with dbHost and dbPort (in `dbServer` section).
      dbServer:
        - dbHost: 127.0.0.1
          dbPort: 3306
      dbName: nacos
      secret:
        name: db-user-pw #Sercret name of Mysql database. The secret must contain two keys: `user` and `password`.
  jvmOptions: "-Xmx3072m -Xmn1024m" #Nacos server jvm options.
  resources: # cpu,memory resources of nacos server
    requests:
      cpu: 100m
      memory: 4Gi
    limits:
      cpu: 100m
      memory: 4Gi
  applicationConfig: # ConfigMap name. The ConfigMap will be projected as nacos server conf.
    name: application-config-map
  affinity: {} #Reference https://kubernetes.io/docs/tasks/configure-pod-container/assign-pods-nodes-using-node-affinity/
  tolerations: [] #Reference https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/
  domain: "cluster.local"
```

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

