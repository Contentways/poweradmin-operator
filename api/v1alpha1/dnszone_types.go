package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// DNSZoneSpec defines the desired state of a DNSZone.
type DNSZoneSpec struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// +kubebuilder:validation:Enum=NATIVE;MASTER;SLAVE
	// +kubebuilder:default=NATIVE
	Type string `json:"type,omitempty"`

	Masters string `json:"masters,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=2
	// +listType=set
	Nameservers []string `json:"nameservers"`
}

// DNSZoneStatus defines the observed state of a DNSZone.
type DNSZoneStatus struct {
	ZoneId int `json:"zoneId,omitempty"`

	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// DNSZone is the Schema for the dnszones API.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Zone",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="ID",type=integer,JSONPath=`.status.zoneId`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
type DNSZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DNSZoneSpec   `json:"spec"`
	Status            DNSZoneStatus `json:"status,omitempty"`
}

// DNSZoneList contains a list of DNSZone resources.
// +kubebuilder:object:root=true
type DNSZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DNSZone `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DNSZone{}, &DNSZoneList{})
}
