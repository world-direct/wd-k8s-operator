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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LoggingSetupSpec defines the desired state of LoggingSetup
type LoggingSetupSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of LoggingSetup. Edit loggingsetup_types.go to remove/update
	Isolation string `json:"isolation,omitempty"`
}

// LoggingSetupStatus defines the observed state of LoggingSetup
type LoggingSetupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	UserName        string `json:"userName,omitempty"`
	InitialPassword string `json:"initialPassword,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LoggingSetup is the Schema for the loggingsetups API
type LoggingSetup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoggingSetupSpec   `json:"spec,omitempty"`
	Status LoggingSetupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LoggingSetupList contains a list of LoggingSetup
type LoggingSetupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LoggingSetup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LoggingSetup{}, &LoggingSetupList{})
}
