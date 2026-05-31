/*
Copyright 2026 Patrick Omland.

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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	k8smeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	poweradmin "contentways.dev/contentways/poweradmin-go/poweradmin"
	dnsv1alpha1 "contentways.dev/contentways/poweradmin-operator/api/v1alpha1"
)

var _ = Describe("DNSRecordReconciler", func() {
	const (
		timeout  = 10 * time.Second
		interval = 250 * time.Millisecond
	)

	var (
		mockZone   *MockZoneClient
		mockRecord *MockRecordClient
	)

	BeforeEach(func() {
		mockZone = sharedMockZone
		mockZone.ExpectedCalls = nil
		mockZone.Calls = nil

		mockRecord = sharedMockRecord
		mockRecord.ExpectedCalls = nil
		mockRecord.Calls = nil
	})

	Context("when creating a DNSRecord", func() {
		It("should create the record in Poweradmin and set Ready=True", func() {
			mockZone.On("GetByName", mock.Anything, "contentways.org").Return(&poweradmin.Zone{ID: 4}, nil, nil)
			mockRecord.On("Create", mock.Anything, 4, poweradmin.RecordCreateOpts{
				Name:    "test-operator",
				Type:    "A",
				Content: "1.2.3.4",
				TTL:     3600,
			}).Return(int64(100), nil, nil)
			mockRecord.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&poweradmin.Record{}, nil, nil).Maybe()
			mockRecord.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Maybe()

			record := &dnsv1alpha1.DNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-record-create",
					Namespace: "default",
				},
				Spec: dnsv1alpha1.DNSRecordSpec{
					ZoneName: "contentways.org",
					Name:     "test-operator",
					Type:     "A",
					Content:  "1.2.3.4",
					TTL:      3600,
				},
			}
			Expect(k8sClient.Create(context.Background(), record)).To(Succeed())

			Eventually(func() int {
				updated := &dnsv1alpha1.DNSRecord{}
				_ = k8sClient.Get(context.Background(), types.NamespacedName{Name: "test-record-create", Namespace: "default"}, updated)
				return updated.Status.RecordID
			}, timeout, interval).Should(Equal(100))

			updated := &dnsv1alpha1.DNSRecord{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: "test-record-create", Namespace: "default"}, updated)).To(Succeed())
			cond := k8smeta.FindStatusCondition(updated.Status.Conditions, conditionReady)
			Expect(cond).NotTo(BeNil())
			Expect(string(cond.Status)).To(Equal("True"))

			Expect(k8sClient.Delete(context.Background(), updated)).To(Succeed())
		})
	})

	Context("when deleting a DNSRecord", func() {
		It("should delete the record from Poweradmin and remove the finalizer", func() {
			mockZone.On("GetByName", mock.Anything, "contentways.org").Return(&poweradmin.Zone{ID: 4}, nil, nil)
			mockRecord.On("Create", mock.Anything, 4, mock.Anything).Return(int64(200), nil, nil)
			mockRecord.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&poweradmin.Record{}, nil, nil).Maybe()
			mockRecord.On("Delete", mock.Anything, 4, int64(200)).Return(nil, nil)

			record := &dnsv1alpha1.DNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-record-delete",
					Namespace: "default",
				},
				Spec: dnsv1alpha1.DNSRecordSpec{
					ZoneName: "contentways.org",
					Name:     "test-delete-record",
					Type:     "A",
					Content:  "9.9.9.9",
					TTL:      3600,
				},
			}
			Expect(k8sClient.Create(context.Background(), record)).To(Succeed())

			Eventually(func() int {
				updated := &dnsv1alpha1.DNSRecord{}
				_ = k8sClient.Get(context.Background(), types.NamespacedName{Name: "test-record-delete", Namespace: "default"}, updated)
				return updated.Status.RecordID
			}, timeout, interval).Should(Equal(200))

			updated := &dnsv1alpha1.DNSRecord{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: "test-record-delete", Namespace: "default"}, updated)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), updated)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: "test-record-delete", Namespace: "default"}, &dnsv1alpha1.DNSRecord{})
			}, timeout, interval).Should(MatchError(ContainSubstring("not found")))

			mockRecord.AssertCalled(GinkgoT(), "Delete", mock.Anything, 4, int64(200))
		})
	})
})
