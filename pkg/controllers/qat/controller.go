// Copyright 2020 Intel Corporation. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package qat contains QAT specific reconciliation logic.
package qat

import (
	"context"
	"reflect"
	"strconv"
	"strings"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	devicepluginv1 "github.com/intel/intel-device-plugins-for-kubernetes/pkg/apis/deviceplugin/v1"
	"github.com/intel/intel-device-plugins-for-kubernetes/pkg/controllers"
	"github.com/pkg/errors"
)

const (
	ownerKey = ".metadata.controller.qat"
	appLabel = "intel-qat-plugin"
)

// +kubebuilder:rbac:groups=deviceplugin.intel.com,resources=qatdeviceplugins,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=deviceplugin.intel.com,resources=qatdeviceplugins/status,verbs=get;update;patch

// SetupReconciler creates a new reconciler for QatDevicePlugin objects.
func SetupReconciler(mgr ctrl.Manager) error {
	c := &controller{scheme: mgr.GetScheme()}
	return controllers.SetupWithManager(mgr, c, devicepluginv1.GroupVersion.String(), "QatDevicePlugin", ownerKey)
}

type controller struct {
	scheme *runtime.Scheme
}

func (c *controller) CreateEmptyObject() runtime.Object {
	return &devicepluginv1.QatDevicePlugin{}
}

func (c *controller) GetTotalObjectCount(ctx context.Context, clnt client.Client) (int, error) {
	var list devicepluginv1.QatDevicePluginList
	if err := clnt.List(ctx, &list); err != nil {
		return 0, err
	}

	return len(list.Items), nil
}

func (c *controller) NewDaemonSet(rawObj runtime.Object) *apps.DaemonSet {
	devicePlugin := rawObj.(*devicepluginv1.QatDevicePlugin)
	yes := true
	return &apps.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    devicePlugin.Namespace,
			GenerateName: devicePlugin.Name + "-",
			Labels: map[string]string{
				"app": appLabel,
			},
		},
		Spec: apps.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": appLabel,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": appLabel,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            appLabel,
							Args:            getPodArgs(devicePlugin),
							Image:           devicePlugin.Spec.Image,
							ImagePullPolicy: "IfNotPresent",
							SecurityContext: &v1.SecurityContext{
								ReadOnlyRootFilesystem: &yes,
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "devdir",
									MountPath: "/dev/vfio",
									ReadOnly:  true,
								},
								{
									Name:      "pcidir",
									MountPath: "/sys/bus/pci",
								},
								{
									Name:      "kubeletsockets",
									MountPath: "/var/lib/kubelet/device-plugins",
								},
							},
						},
					},
					NodeSelector: devicePlugin.Spec.NodeSelector,
					Volumes: []v1.Volume{
						{
							Name: "devdir",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/dev/vfio",
								},
							},
						},
						{
							Name: "pcidir",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/sys/bus/pci",
								},
							},
						},
						{
							Name: "kubeletsockets",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/device-plugins",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (c *controller) UpdateDaemonSet(rawObj runtime.Object, ds *apps.DaemonSet) (updated bool) {
	dp := rawObj.(*devicepluginv1.QatDevicePlugin)

	if ds.Spec.Template.Spec.Containers[0].Image != dp.Spec.Image {
		ds.Spec.Template.Spec.Containers[0].Image = dp.Spec.Image
		updated = true
	}

	if !reflect.DeepEqual(ds.Spec.Template.Spec.NodeSelector, dp.Spec.NodeSelector) {
		ds.Spec.Template.Spec.NodeSelector = dp.Spec.NodeSelector
		updated = true
	}

	newargs := getPodArgs(dp)
	if strings.Join(ds.Spec.Template.Spec.Containers[0].Args, " ") != strings.Join(newargs, " ") {
		ds.Spec.Template.Spec.Containers[0].Args = newargs
		updated = true
	}

	return updated
}

func (c *controller) UpdateStatus(rawObj runtime.Object, ds *apps.DaemonSet, nodeNames []string) (updated bool, err error) {
	dp := rawObj.(*devicepluginv1.QatDevicePlugin)

	dsRef, err := reference.GetReference(c.scheme, ds)
	if err != nil {
		return false, errors.Wrap(err, "unable to make reference to controlled daemon set")
	}

	if dp.Status.ControlledDaemonSet.UID != dsRef.UID {
		dp.Status.ControlledDaemonSet = *dsRef
		updated = true
	}

	if dp.Status.DesiredNumberScheduled != ds.Status.DesiredNumberScheduled {
		dp.Status.DesiredNumberScheduled = ds.Status.DesiredNumberScheduled
		updated = true
	}

	if dp.Status.NumberReady != ds.Status.NumberReady {
		dp.Status.NumberReady = ds.Status.NumberReady
		updated = true
	}

	if strings.Join(dp.Status.NodeNames, ",") != strings.Join(nodeNames, ",") {
		dp.Status.NodeNames = nodeNames
		updated = true
	}

	return updated, nil
}

func getPodArgs(qdp *devicepluginv1.QatDevicePlugin) []string {
	args := make([]string, 0, 8)
	args = append(args, "-v", strconv.Itoa(qdp.Spec.LogLevel))

	if qdp.Spec.DpdkDriver != "" {
		args = append(args, "-dpdk-driver", qdp.Spec.DpdkDriver)
	} else {
		args = append(args, "-dpdk-driver", "vfio-pci")
	}

	if len(qdp.Spec.KernelVfDrivers) > 0 {
		drvs := make([]string, len(qdp.Spec.KernelVfDrivers))
		for i, v := range qdp.Spec.KernelVfDrivers {
			drvs[i] = string(v)
		}
		args = append(args, "-kernel-vf-drivers", strings.Join(drvs, ","))
	} else {
		args = append(args, "-kernel-vf-drivers", "dh895xccvf,c6xxvf,c3xxxvf,d15xxvf")
	}

	if qdp.Spec.MaxNumDevices > 0 {
		args = append(args, "-max-num-devices", strconv.Itoa(qdp.Spec.MaxNumDevices))
	} else {
		args = append(args, "-max-num-devices", "32")
	}

	return args
}
