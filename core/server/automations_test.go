package server

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	. "github.com/onsi/gomega"
	pb "github.com/weaveworks/weave-gitops/pkg/api/core"
	"github.com/weaveworks/weave-gitops/pkg/kube"
)

func TestListKustomizations(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx := context.Background()

	c, cleanup := makeGRPCServer(k8sEnv.Rest, t)
	defer cleanup()

	_, k, err := kube.NewKubeHTTPClientWithConfig(k8sEnv.Rest, "")
	g.Expect(err).NotTo(HaveOccurred())

	appName := "myapp"
	ns := newNamespace(ctx, k, g)

	kust := &kustomizev1.Kustomization{
		Spec: kustomizev1.KustomizationSpec{
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
			},
		},
	}
	kust.Name = appName
	kust.Namespace = ns.Name

	g.Expect(k.Create(ctx, kust)).To(Succeed())

	res, err := c.ListKustomizations(ctx, &pb.ListKustomizationsRequest{
		Namespace: ns.Name,
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(res.Kustomizations).To(HaveLen(1))
	g.Expect(res.Kustomizations[0].Name).To(Equal(appName))
}

func TestGetKustomization(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx := context.Background()

	c, cleanup := makeGRPCServer(k8sEnv.Rest, t)
	defer cleanup()

	_, k, err := kube.NewKubeHTTPClientWithConfig(k8sEnv.Rest, "")
	g.Expect(err).NotTo(HaveOccurred())

	appName := "myapp"
	ns := newNamespace(ctx, k, g)

	kust := &kustomizev1.Kustomization{
		Spec: kustomizev1.KustomizationSpec{
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
			},
		},
		Status: kustomizev1.KustomizationStatus{
			Inventory: &kustomizev1.ResourceInventory{
				Entries: []kustomizev1.ResourceRef{
					{
						Version: "v1",
						ID:      ns.Name + "_my-deployment_apps_Deployment",
					},
				},
			},
		},
	}
	kust.Name = appName
	kust.Namespace = ns.Name

	g.Expect(k.Create(ctx, kust)).To(Succeed())

	t.Run("gets a kustomization", func(t *testing.T) {
		_, err = c.GetKustomization(ctx, &pb.GetKustomizationRequest{Name: appName, Namespace: ns.Name})
		g.Expect(err).NotTo(HaveOccurred())

		// The kustomization status does not get populated when running in a test environment,
		// probably because there is no kustomization-controller running to populate it.
		// Even though we populate it in the kustomization.Status object above, it still does not populate.
		// Need some help from the flux team on this.
		// g.Expect(len(res.Kustomization.Inventory)).To(Equal(1))
		// g.Expect(res.Kustomization.Inventory[0].Group).To(Equal("apps"))
	})
	t.Run("returns not found", func(t *testing.T) {
		_, err = c.GetKustomization(ctx, &pb.GetKustomizationRequest{Name: "somename", Namespace: ns.Name})
		g.Expect(err).To(HaveOccurred())

		status, ok := status.FromError(err)
		if !ok {
			t.Error("could not get status from error")
		}

		g.Expect(status.Code()).To(Equal(codes.NotFound))
	})
}

func TestListHelmReleases(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx := context.Background()

	c, cleanup := makeGRPCServer(k8sEnv.Rest, t)
	defer cleanup()

	_, k, err := kube.NewKubeHTTPClientWithConfig(k8sEnv.Rest, "")
	g.Expect(err).NotTo(HaveOccurred())

	appName := "myapp"
	ns := newNamespace(ctx, k, g)

	hr := &helmv2.HelmRelease{
		Spec: helmv2.HelmReleaseSpec{
			Chart: helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind: "GitRepository",
						Name: "somesource",
					},
				},
			},
		},
	}
	hr.Name = appName
	hr.Namespace = ns.Name

	g.Expect(k.Create(ctx, hr)).To(Succeed())

	res, err := c.ListHelmReleases(ctx, &pb.ListHelmReleasesRequest{
		Namespace: ns.Name,
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(res.HelmReleases).To(HaveLen(1))
	g.Expect(res.HelmReleases[0].Name).To(Equal(appName))
}

func newNamespace(ctx context.Context, k client.Client, g *GomegaWithT) *corev1.Namespace {
	ns := &corev1.Namespace{}
	ns.Name = "kube-test-" + rand.String(5)

	g.Expect(k.Create(ctx, ns)).To(Succeed())

	return ns
}
