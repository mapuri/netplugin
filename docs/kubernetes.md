### Kubernetes Integration

Netplugin code can handle the network and network-policy instantiation via plugins provided by Kubernetes.
The plugin for Kubernetes is always built as a binary and kept in `$GOPATH/bin` as `k8contivnet`

#### A quick tryout

- Start a two node vagrant setup
```
CONTIV_NODES=2 make demo
```

- Link the `k8contivnet` binary from $GOPATH/bin to kubelet-plugins directory on each vagrant node:
```
ssh netplugin-node1
sudo mkdir -p /usr/libexec/kubernetes/kubelet-plugins/net/exec/k8contivnet
sudo ln -s $GOPATH/bin/k8contivnet /usr/libexec/kubernetes/kubelet-plugins/net/exec/k8contivnet/

ssh netplugin-node2
sudo mkdir -p /usr/libexec/kubernetes/kubelet-plugins/net/exec/k8contivnet
sudo ln -s $GOPATH/bin/k8contivnet /usr/libexec/kubernetes/kubelet-plugins/net/exec/k8contivnet/
```

- Install Kubernetes using the [Ubuntu Setup Guides](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/docs/getting-started-guides/ubuntu.md) on each vagrant node
 Note:
 - Kubernetes cluster IPs should use the IP address assigned to the eth1 devices of the vagrant nodes.
 - The setup already has etcd up and running, so it shall not need to be setup separately.
 - You must make sure that kublet is started with `--network_plugin=k8contivnet` option.

- Start netplugin on the two vagrant nodes:
```
ssh netplugin-node1
netplugin -host-label=host1

ssh netplugin-node2
netplugin -host-label=host2
```

- Now post the desired network intent as specified in [late-bindings example](examples/late_bindings/multiple_vxlan_nets.json) from one of the vagrant nodes. Note that the json input doesn't specify the host information, which is automatically when `kubernetes scheduler` picks up a minion for the host. And `Container` in the json schema is really a pod's name instead of the container(s) within pod.
```
ssh netplugin-node1
netdcli -cfg $GOPATH/src/github.com.contiv/netplugin/examples/late_bindings/multiple_vxlan_nets.json
```

- Now Launch two redis-server instances via Kubernetes. The pods shall be created and placed in their own respective networks.
```
# cat > /tmp/redis-net-orange.json <<EOF
{
  "id": "myContainer1",
  "kind": "Pod",
  "apiVersion": "v1beta1",
  "desiredState": {
    "manifest": {
      "version": "v1beta1",
      "id": "myContainer1",
      "containers": [{
        "name": "myContainer1",
        "image": "dockerfile/redis",
        "cpu": 100,
      }]
    }
  },
  "labels": {
    "name": "redis-master"
  }
}
EOF
# kubectl create -f /tmp/redis-net-orange.json

# cat > /tmp/redis-net-purple.json <<EOF
{
  "id": "myContainer2",
  "kind": "Pod",
  "apiVersion": "v1beta1",
  "desiredState": {
    "manifest": {
      "version": "v1beta1",
      "id": "myContainer2",
      "containers": [{
        "name": "myContainer2",
        "image": "dockerfile/redis",
        "cpu": 100,
      }]
    }
  },
  "labels": {
    "name": "redis-master"
  }
}
EOF
# kubectl create -f /tmp/redis-net-purple.json
```

- Add/Delete the networks or endpoints directly via netplugin, usually before adding or after deleting the pod

#### Pending work items
- Fetch the IP information from the netplugin and display it alongside k8's pod information
- Allocate the networks and network policies based on k8 labels

#### Some details for people interesting in hacking some of this

[Kubernetes](https://github.com/GoogleCloudPlatform/kubernetes) infrastructure model is to create an infrastructure container (called pod).
This requires network plugin to create the network plumbing inside an infrastructure container, which is created dynamically.
And the visible names to the application is identified by pod-name or container-name(s) in the pod.

This network plugin has been enhanced to allow specification of the network container to be different from the application-container.
Further, kubernetes require that a plugin be written and kept in a specific directory which gets called when an applicaiton (aka pod) is launched.
This allows for a binary executable to be called with a specific parameters to do the network plumbing outside Kubernetes. 

For that reason, netplugin produces a new binary, called k8contivnet, a small plugin interface that will get called by Kubernetes
upon init of the plugin, and during creation/deletion of the application pod. The syntax of k8contivnet is as follows, which adheres to 
Kubernetes plugin requirements:

```
$ k8contivnet init
$ k8contivnet setup <pod-name> <pod-namespace> <infra-container-uuid>
$ k8contivnet teardown <pod-name> <pod-namespace> <infra-container-uuid>
$ k8contivnet help
```

This plugin would need to be copied in following directory:
`/usr/libexec/kubernetes/kubelet-plugins/net/exec/k8contivnet/k8contivnet`

