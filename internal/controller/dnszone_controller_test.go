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

	poweradmin "contentways.dev/contentways/poweradmin-go/poweradmin"
	dnsv1alpha1 "contentways.dev/contentways/poweradmin-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	k8smeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("DNSZoneReconciler", func() {
	const (
		timeout  = 10 * time.Second
		interval = 250 * time.Millisecond
	)

	var mockZone *MockZoneClient

	BeforeEach(func() {
		mockZone = sharedMockZone
		mockZone.ExpectedCalls = nil
		mockZone.Calls = nil
	})

	BeforeEach(func() {
		mockZone.ExpectedCalls = nil
		mockZone.Calls = nil
	})

	Context("when creating a DNSZone", func() {
		It("should create the zone in Poweradmin and set Ready=True", func() {
			mockZone.On("Create", mock.Anything, poweradmin.ZoneCreateOpts{
				Name: "test-create.example.org",
				Type: poweradmin.ZoneTypeNative,
			}).Return(42, nil, nil)
			mockZone.On("Update", mock.Anything, 42, mock.Anything).Return(&poweradmin.Zone{ID: 42}, nil, nil).Maybe()
			mockZone.On("Delete", mock.Anything, mock.Anything).Return(nil, nil).Maybe()

			zone := &dnsv1alpha1.DNSZone{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-create",
					Namespace: "default",
				},
				Spec: dnsv1alpha1.DNSZoneSpec{
					Name: "test-create.example.org",
					Type: "NATIVE",
				},
			}
			Expect(k8sClient.Create(context.Background(), zone)).To(Succeed())

			Eventually(func() int {
				updated := &dnsv1alpha1.DNSZone{}
				_ = k8sClient.Get(context.Background(), types.NamespacedName{Name: "test-create", Namespace: "default"}, updated)
				return updated.Status.ZoneId
			}, timeout, interval).Should(Equal(42))

			updated := &dnsv1alpha1.DNSZone{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: "test-create", Namespace: "default"}, updated)).To(Succeed())
			cond := k8smeta.FindStatusCondition(updated.Status.Conditions, conditionReady)
			Expect(cond).NotTo(BeNil())
			Expect(string(cond.Status)).To(Equal("True"))

			Expect(k8sClient.Delete(context.Background(), updated)).To(Succeed())
		})
	})

	Context("when deleting a DNSZone", func() {
		It("should delete the zone from Poweradmin and remove the finalizer", func() {
			mockZone.On("Create", mock.Anything, mock.Anything).Return(99, nil, nil)
			mockZone.On("Update", mock.Anything, 99, mock.Anything).Return(&poweradmin.Zone{ID: 99}, nil, nil).Maybe()
			mockZone.On("Delete", mock.Anything, 99).Return(nil, nil)

			zone := &dnsv1alpha1.DNSZone{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-delete",
					Namespace: "default",
				},
				Spec: dnsv1alpha1.DNSZoneSpec{
					Name: "test-delete.example.org",
					Type: "NATIVE",
				},
			}
			Expect(k8sClient.Create(context.Background(), zone)).To(Succeed())

			Eventually(func() int {
				updated := &dnsv1alpha1.DNSZone{}
				_ = k8sClient.Get(context.Background(), types.NamespacedName{Name: "test-delete", Namespace: "default"}, updated)
				return updated.Status.ZoneId
			}, timeout, interval).Should(Equal(99))

			updated := &dnsv1alpha1.DNSZone{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: "test-delete", Namespace: "default"}, updated)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), updated)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: "test-delete", Namespace: "default"}, &dnsv1alpha1.DNSZone{})
			}, timeout, interval).Should(MatchError(ContainSubstring("not found")))

			mockZone.AssertCalled(GinkgoT(), "Delete", mock.Anything, 99)
		})
	})
})
