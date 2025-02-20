# Copyright (c) 2023-2025, Nubificus LTD
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

{{/*
Set the correct containerd conf path depending on the k8s distribution
*/}}
{{- define "containerdConfPath" -}}
{{- if eq .k8sDistribution "rke2" -}}
/var/lib/rancher/rke2/agent/etc/containerd/
{{- else if eq .k8sDistribution "k3s" -}}
 /var/lib/rancher/k3s/agent/etc/containerd/
{{- else if eq .k8sDistribution "k0s" -}}
/etc/k0s/containerd.d/
{{- else -}}
/etc/containerd/
{{- end -}}
{{- end -}}

