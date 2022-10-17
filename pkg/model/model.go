package model

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utils "k8s.io/apimachinery/pkg/util/intstr"
)

type ModelServing struct {
	Name      string
	ModelURL  string
	Columns   string
	Namespace string
	Version   string
	Replicas  int32
}

func (m *ModelServing) CreateService(ctx context.Context) *corev1.Service {
	labels := map[string]string{"serving": m.Name}

	service := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprint("ms-", m.Name), Namespace: m.Namespace},
		Spec:       corev1.ServiceSpec{Selector: labels, Ports: []corev1.ServicePort{{Port: 4000, TargetPort: utils.FromInt(4000), Name: "http-serving"}}},
		Status:     corev1.ServiceStatus{},
	}

	return service
}

func (m *ModelServing) CreateDeployment(ctx context.Context) *appsv1.StatefulSet {

	labels := map[string]string{"serving": m.Name}

	found := &appsv1.StatefulSet{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: m.Name, Namespace: m.Namespace},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &m.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: fmt.Sprint("plasmashadow/model_serving:", m.Version),
						Name:  "serving",
						Ports: []corev1.ContainerPort{{ContainerPort: 4000, Name: "serving"}},
						Env: []corev1.EnvVar{{
							Name: "MODEL_PATH",
							ValueFrom: &v1.EnvVarSource{
								ConfigMapKeyRef: &v1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprint("cf-", m.Name),
									},
									Key: "MODEL_PATH",
								},
							},
						}, {
							Name: "COLUMNS",
							ValueFrom: &v1.EnvVarSource{
								ConfigMapKeyRef: &v1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprint("cf-", m.Name),
									},
									Key: "COLUMNS",
								},
							},
						}},
						VolumeMounts: []corev1.VolumeMount{{Name: fmt.Sprint("vol-", m.Name), MountPath: "/data"}}}},
					Volumes: []corev1.Volume{{
						Name:         fmt.Sprint("vol-", m.Name),
						VolumeSource: corev1.VolumeSource{},
					}},
				}},
			VolumeClaimTemplates:                 []corev1.PersistentVolumeClaim{},
			ServiceName:                          fmt.Sprint("ms-", m.Name),
			PodManagementPolicy:                  "",
			UpdateStrategy:                       appsv1.StatefulSetUpdateStrategy{},
			RevisionHistoryLimit:                 new(int32),
			MinReadySeconds:                      0,
			PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{},
		},
		Status: appsv1.StatefulSetStatus{},
	}

	return found
}

func (m *ModelServing) CreateConfigMap(ctx context.Context, modelPath string, columns string) *corev1.ConfigMap {

	found := &corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprint("cf-", m.Name), Namespace: m.Namespace},
		Immutable:  new(bool),
		Data: map[string]string{
			"MODEL_PATH": modelPath,
			"COLUMNS":    columns,
		},
		BinaryData: map[string][]byte{},
	}

	return found
}
