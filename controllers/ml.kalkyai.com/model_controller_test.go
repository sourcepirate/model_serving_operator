package mlkalkyaicom

import (
	"context"
	"time"

	mlv1alpha1 "github.com/kalkyai/model-serving-operator/apis/ml.kalkyai.com/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Model Controller", func() {

	Context("Model Controller test", func() {

		ctx := context.Background()

		namespace := &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
		}
		modelObject := &mlv1alpha1.Model{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
			Spec: mlv1alpha1.ModelSpec{
				Location:  "iris.sav",
				Replicas:  1,
				Endpoint:  "https://sgp1.digitaloceanspaces.com",
				Accesskey: "",
				SecretKey: "",
				Columns:   "sepal.length,sepal.width,petal.length,petal.width",
				Version:   "0.6",
				Bucket:    "test",
			},
		}

		typeNamespaceName := types.NamespacedName{Name: "test", Namespace: "test"}

		BeforeEach(func() {
			By("Creating a namespace to perform the tests")
			err := k8sClient.Create(ctx, namespace)
			Expect(err).To(Not(HaveOccurred()))
		})

		AfterEach(func() {
			// TODO(user): Attention if you improve this code by adding other context test you MUST
			// be aware of the current delete namespace limitations. More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("Deleting the Namespace to perform the tests")
			_ = k8sClient.Delete(ctx, namespace)

		})

		It("should successfully reconcile a model service object", func() {

			err := k8sClient.Create(ctx, modelObject)

			Expect(err).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")

			Eventually(func() error {
				found := &mlv1alpha1.Model{}
				return k8sClient.Get(ctx, typeNamespaceName, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			modelReconciler := &ModelReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = modelReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if Statefulset was successfully created in the reconciliation")
			Eventually(func() error {
				found := &appsv1.StatefulSet{}
				return k8sClient.Get(ctx, typeNamespaceName, found)
			}, time.Minute, time.Second).Should(Succeed())
		})

	})

})
