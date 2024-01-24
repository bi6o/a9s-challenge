package controller

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	interviewv1alpha1 "github.com/bi6o/a9s-challenge/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var dummyReconciler *DummyReconciler

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = interviewv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	dummyReconciler = &DummyReconciler{
		Client: k8sClient,
		Scheme: scheme.Scheme,
	}
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Dummy Controller", func() {

	const (
		DummyName      = "test-dummy"
		DummyNamespace = "default"
		Timeout        = time.Second * 10
		Interval       = time.Millisecond * 250
	)

	Context("When updating Dummy Status", func() {
		It("Should update Dummy's status", func() {
			By("By creating a new Dummy")
			ctx := context.Background()
			dummy := &interviewv1alpha1.Dummy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "interview.a9s-interview.com/v1alpha1",
					Kind:       "Dummy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      DummyName,
					Namespace: DummyNamespace,
				},
				Spec: interviewv1alpha1.DummySpec{
					Message: "I'm just a dummy",
				},
			}
			Expect(k8sClient.Create(ctx, dummy)).Should(Succeed())

			dummyLookupKey := types.NamespacedName{Name: DummyName, Namespace: DummyNamespace}
			createdDummy := &interviewv1alpha1.Dummy{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, dummyLookupKey, createdDummy)
				return err == nil
			}, Timeout, Interval).Should(BeTrue())

			Expect(createdDummy.Spec.Message).Should(Equal("I'm just a dummy"))

			createdDummy.Status.SpecEcho = "I'm just a dummy updated"
			Expect(k8sClient.Status().Update(ctx, createdDummy)).Should(Succeed())

			updatedDummy := &interviewv1alpha1.Dummy{}
			Eventually(func() string {
				k8sClient.Get(ctx, dummyLookupKey, updatedDummy)
				return updatedDummy.Status.SpecEcho
			}, Timeout, Interval).Should(Equal("I'm just a dummy updated"))

			Expect(k8sClient.Delete(ctx, dummy)).Should(Succeed())
		})

		It("Should handle non-existent Dummy", func() {
			By("By trying to reconcile a non-existent Dummy")
			ctx := context.Background()
			_, err := dummyReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent-dummy",
					Namespace: DummyNamespace,
				},
			})

			Expect(err).To(HaveOccurred())
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("Should create a Pod when a new Dummy is created", func() {
			By("By creating a new Dummy")

			ctx := context.Background()
			dummy := &interviewv1alpha1.Dummy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dummy1",
					Namespace: DummyNamespace,
				},
				Spec: interviewv1alpha1.DummySpec{
					Message: "Create a pod dummy",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: "interview.a9s-interview.com/v1alpha1",
					Kind:       "Dummy",
				},
			}
			Expect(k8sClient.Create(ctx, dummy)).Should(Succeed())

			_, err := dummyReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "dummy1",
					Namespace: DummyNamespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			pod := &corev1.Pod{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      "dummy1-pod",
					Namespace: DummyNamespace,
				}, pod)
				return err == nil
			}, Timeout, Interval).Should(BeTrue())

			Expect(pod.Name).To(Equal("dummy1-pod"))

			Expect(k8sClient.Delete(ctx, dummy)).Should(Succeed())
		})
	})
})
