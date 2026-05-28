package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// DNSRecordSpec defines the desired state of a DNSRecord.
type DNSRecordSpec struct {
	// +kubebuilder:validation:Required
	ZoneName string `json:"zoneName"`

	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// +kubebuilder:validation:Enum=A;AAAA;CNAME;MX;TXT;NS;SRV;CAA;PTR;SOA
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// +kubebuilder:validation:Required
	Content string `json:"content"`

	// +kubebuilder:default=3600
	TTL int `json:"ttl,omitempty"`

	// +kubebuilder:default=0
	Priority int `json:"priority,omitempty"`

	// +kubebuilder:default=false
	Disabled bool `json:"disabled,omitempty"`
}

// DNSRecordStatus defines the observed state of a DNSRecord.
type DNSRecordStatus struct {
	RecordId int `json:"recordId,omitempty"`
	ZoneId   int `json:"zoneId,omitempty"`

	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// DNSRecord is the Schema for the dnsrecords API.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Zone",type=string,JSONPath=`.spec.zoneName`
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Content",type=string,JSONPath=`.spec.content`
// +kubebuilder:printcolumn:name="TTL",type=integer,JSONPath=`.spec.ttl`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
type DNSRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DNSRecordSpec   `json:"spec"`
	Status            DNSRecordStatus `json:"status,omitempty"`
}

// DNSRecordList contains a list of DNSRecord resources.
// +kubebuilder:object:root=true
type DNSRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DNSRecord `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DNSRecord{}, &DNSRecordList{})
}
