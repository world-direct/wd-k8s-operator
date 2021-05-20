/*
Copyright 2021.

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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	loggingv1alpha1 "github.com/world-direct/wd-k8s-operator/api/v1alpha1"
	graylog "github.com/world-direct/wd-k8s-operator/provisioners/graylog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LoggingSetupReconciler reconciles a LoggingSetup object
type LoggingSetupReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const (
	CONDIIONTYPE_USER     = "UserProvisioned"
	CONDIIONTYPE_INDEXSET = "IndexSetProvisioned"
	CONDIIONTYPE_STREAM   = "StreamProvisioned"
)

//+kubebuilder:rbac:groups=logging.world-direct.at,resources=loggingsetups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=logging.world-direct.at,resources=loggingsetups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=logging.world-direct.at,resources=loggingsetups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LoggingSetup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *LoggingSetupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("loggingsetup", req.NamespacedName)

	// Fetch the LoggingSetup instance
	obj := &loggingv1alpha1.LoggingSetup{}
	err := r.Get(ctx, req.NamespacedName, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("LoggingSetup resource not found. Ignoring since object must be deleted")

			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get LoggingSetup")
		return ctrl.Result{}, err
	}

	// check for the created LoggingSetup
	log.Info("Reconcile created instance")

	// collect data for provisioning
	data := &graylog.GraylogProvisioningData{
		Name: obj.Namespace,
	}

	data.User.InitialPassword = obj.Spec.InitialUserPassword
	data.User.Roles = []string{"Reader", "Dashboard Creator"}
	data.IndexSet.TemplateName = "wd-logging-operator-template"
	data.Stream.RuleFieldName = "kubernetes_namespace_name"

	if true || !meta.IsStatusConditionTrue(obj.Status.Conditions, CONDIIONTYPE_USER) {

		err = graylog.ProvisionUser(ctx, r.Log, data)

		if err != nil {
			meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
				Type:    CONDIIONTYPE_USER,
				Status:  metav1.ConditionFalse,
				Reason:  "Failed",
				Message: err.Error(),
			})

			log.Error(err, "Failed to provision User")

		} else {
			meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
				Type:   CONDIIONTYPE_USER,
				Status: metav1.ConditionTrue,
				Reason: "Done",
			})

			obj.Status.GraylogStatus.UserID = data.User.ID
		}
	}

	if true || !meta.IsStatusConditionTrue(obj.Status.Conditions, CONDIIONTYPE_INDEXSET) {

		err = graylog.ProvisionIndexSet(ctx, r.Log, data)

		if err != nil {
			meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
				Type:    CONDIIONTYPE_INDEXSET,
				Status:  metav1.ConditionFalse,
				Reason:  "Failed",
				Message: err.Error(),
			})

			log.Error(err, "Failed to provision IndexSet")

		} else {
			meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
				Type:   CONDIIONTYPE_INDEXSET,
				Status: metav1.ConditionTrue,
				Reason: "Done",
			})

			obj.Status.GraylogStatus.IndexSetID = data.IndexSet.ID

		}
	}

	if true || !meta.IsStatusConditionTrue(obj.Status.Conditions, CONDIIONTYPE_STREAM) {

		err = graylog.ProvisionStream(ctx, r.Log, data)

		if err != nil {
			meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
				Type:    CONDIIONTYPE_STREAM,
				Status:  metav1.ConditionFalse,
				Reason:  "Failed",
				Message: err.Error(),
			})

			log.Error(err, "Failed to provision Stream")

		} else {
			meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
				Type:   CONDIIONTYPE_STREAM,
				Status: metav1.ConditionTrue,
				Reason: "Done",
			})

			obj.Status.GraylogStatus.StreamID = data.Stream.ID
		}
	}

	// Update the Status
	////////////////////////////7

	updateErr := r.Status().Update(ctx, obj)
	if updateErr != nil {
		// this error is not updated to the condition, just logged
		log.Error(updateErr, "Failed to update Object")
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *LoggingSetupReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// the Graylog API for configruation issues
	client, err := graylog.CreateClient(r.Log)
	if err != nil {
		r.Log.Error(err, "Unable to create the client")
		return err
	}

	err = client.Test(context.Background())
	if err != nil {
		r.Log.Error(err, "Unable execute test call")
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&loggingv1alpha1.LoggingSetup{}).
		Complete(r)
}
