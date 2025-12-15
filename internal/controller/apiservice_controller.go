/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	myservicev1 "k8s-opreator/api/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// ApiserviceReconciler reconciles a Apiservice object
type ApiserviceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=myservice.cyk.io,resources=apiservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=myservice.cyk.io,resources=apiservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=myservice.cyk.io,resources=apiservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Apiservice object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *ApiserviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// TODO(user): your logic here
	// 1. 获取Apiservice实例
	apiservice := &myservicev1.Apiservice{}
	if err := r.Get(ctx, req.NamespacedName, apiservice); err != nil {
		if errors.IsNotFound(err) {
			logf.FromContext(ctx).Info("Apiservice resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logf.FromContext(ctx).Error(err, "Failed to get Apiservice")
		return ctrl.Result{}, err
	}

	// 2. 检查Deployment是否存在，不存在则创建
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      apiservice.Name,
		Namespace: apiservice.Namespace,
	}, deployment)

	if err != nil && errors.IsNotFound(err) {
		// 创建新的Deployment
		dep := r.deploymentForApiservice(apiservice)
		logf.Log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err := r.Create(ctx, dep); err != nil {
			logf.Log.Error(err, "Failed to create new Deployment")
			return ctrl.Result{}, err
		}
		// 等待下一次协调
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logf.Log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// 3. 确保Deployment副本数与期望值一致
	size := apiservice.Spec.Replicas
	if *deployment.Spec.Replicas != size {
		deployment.Spec.Replicas = &size
		if err := r.Update(ctx, deployment); err != nil {
			logf.Log.Error(err, "Failed to update Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return ctrl.Result{}, err
		}
	}

	// 4. 管理Service
	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      apiservice.Name,
		Namespace: apiservice.Namespace,
	}, service)

	if err != nil && errors.IsNotFound(err) {
		// 创建Service
		svc := r.serviceForApiservice(apiservice)
		logf.Log.Info("Creating a new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		if err := r.Create(ctx, svc); err != nil {
			logf.Log.Error(err, "Failed to create new Service")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		logf.Log.Error(err, "Failed to get Service")
		return ctrl.Result{}, err
	}

	// 5. 更新状态
	if err := r.updateStatus(ctx, apiservice, deployment); err != nil {
		logf.Log.Error(err, "Failed to update apiservice status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// deploymentForWebApp 创建Deployment对象
func (r *ApiserviceReconciler) deploymentForApiservice(apiservice *myservicev1.Apiservice) *appsv1.Deployment {
	labels := map[string]string{
		"app":        apiservice.Name,
		"controller": apiservice.Name,
	}

	// 构建环境变量
	var envVars []corev1.EnvVar
	for _, env := range apiservice.Spec.Env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  env.Name,
			Value: env.Value,
		})
	}

	// 构建资源请求
	resources := corev1.ResourceRequirements{}
	if apiservice.Spec.Resources.Requests.CPU != "" || apiservice.Spec.Resources.Requests.Memory != "" {
		resources.Requests = corev1.ResourceList{}
		if apiservice.Spec.Resources.Requests.CPU != "" {
			resources.Requests[corev1.ResourceCPU] = resource.MustParse(apiservice.Spec.Resources.Requests.CPU)
		}
		if apiservice.Spec.Resources.Requests.Memory != "" {
			resources.Requests[corev1.ResourceMemory] = resource.MustParse(apiservice.Spec.Resources.Requests.Memory)
		}
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiservice.Name,
			Namespace: apiservice.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &apiservice.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:     apiservice.Spec.Image,
						Name:      apiservice.Name,
						Ports:     []corev1.ContainerPort{{ContainerPort: apiservice.Spec.Port}},
						Env:       envVars,
						Resources: resources,
					}},
				},
			},
		},
	}

	// 设置WebApp为Deployment的Owner
	controllerutil.SetControllerReference(apiservice, dep, r.Scheme)
	return dep
}

// serviceForWebApp 创建Service对象
func (r *ApiserviceReconciler) serviceForApiservice(apiservice *myservicev1.Apiservice) *corev1.Service {
	labels := map[string]string{
		"app":        apiservice.Name,
		"controller": apiservice.Name,
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiservice.Name,
			Namespace: apiservice.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       apiservice.Spec.Port,
				TargetPort: intstr.FromInt(int(apiservice.Spec.Port)),
			}},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	controllerutil.SetControllerReference(apiservice, svc, r.Scheme)
	return svc
}

// updateStatus 更新WebApp状态
func (r *ApiserviceReconciler) updateStatus(ctx context.Context, apiservice *myservicev1.Apiservice, deployment *appsv1.Deployment) error {
	// 获取最新的WebApp
	latestApiservice := &myservicev1.Apiservice{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      apiservice.Name,
		Namespace: apiservice.Namespace,
	}, latestApiservice); err != nil {
		return err
	}

	// 更新状态
	latestApiservice.Status.AvailableReplicas = deployment.Status.AvailableReplicas
	latestApiservice.Status.ServiceEndpoint = fmt.Sprintf("%s:%d", apiservice.Name, apiservice.Spec.Port)

	// 添加条件
	condition := metav1.Condition{
		Type:               "Available",
		Status:             metav1.ConditionTrue,
		Reason:             "DeploymentReady",
		Message:            fmt.Sprintf("Deployment has %d available replicas", deployment.Status.AvailableReplicas),
		LastTransitionTime: metav1.Now(),
	}

	latestApiservice.Status.Conditions = append(latestApiservice.Status.Conditions, condition)

	// 更新状态
	return r.Status().Update(ctx, latestApiservice)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApiserviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myservicev1.Apiservice{}).
		Named("apiservice").
		Complete(r)
}
