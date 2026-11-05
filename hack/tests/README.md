# kind image preparation regression

Run from the repository root (Bash 4+, Python 3):

```sh
python3 -B -m unittest discover -s hack/tests -p test_image_prepare.py
```

The tests run the real `image:prepare` shell function against command stubs.
They verify target-platform selection, platform-specific node grouping, legacy
Docker handling, forced API versions, failure propagation and temporary-file
cleanup. They do not replace testing an actual Docker/containerd image import.

`image:prepare` obtains a temporary kubeconfig from the selected kind cluster
and reads each node's Kubernetes-reported OS/architecture. It supports the
project's Linux amd64/arm64 targets without using the host architecture or
changing the user's current kubeconfig/context. Images are pulled and exported
for one platform at a time and imported only into matching nodes.

Docker client/server API 1.48+ supports platform-filtered `docker image save`.
Older clients/servers using classic image storage retain the ordinary save path:
that store keeps one platform per tag, and each pull/save/load completes before
the next platform is pulled. An older effective API with containerd storage
fails with an upgrade/API-override diagnostic instead of exporting an archive
that references unavailable platforms. `DOCKER_API_VERSION` is included in the
capability check. Docker daemon settings are never modified.
