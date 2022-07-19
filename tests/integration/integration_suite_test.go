package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	versionedclient "istio.io/client-go/pkg/clientset/versioned"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = Describe("Post deployment check", func() {
	Context("VirtualService merge", func() {
		// setup
		BeforeEach(func() {
			_, filename, _, _ := runtime.Caller(0)
			pwd := filepath.Dir(filename)

			cmd := exec.Command("kubectl", "apply", "-f", pwd+"/../data/vs.yaml", "-n", "istio-virtualservice-merger")
			_, err := cmd.Output()
			if err != nil {
				panic(err)
			}
			cmd = exec.Command("kubectl", "apply", "-f", pwd+"/../data/vs-merge-1.yaml", "-n", "istio-virtualservice-merger")
			_, err = cmd.Output()
			if err != nil {
				panic(err)
			}
		})

		// check mege is working as expected
		It("VirtualService will have updated path specs", func() {
			// deploy vs
			config := GetK8sConfig()
			ic, err := versionedclient.NewForConfig(config)
			if err != nil {
				panic(err)
			}
			if vs, err := ic.NetworkingV1beta1().VirtualServices("istio-virtualservice-merger").Get(context.TODO(), "integration-test", v1.GetOptions{}); err != nil {
				panic("test virtualservic deployment failed. Error" + err.Error())
			} else {
				httpRoutes := vs.Spec.Http

				paths := make([]string, 0)
				for _, i := range httpRoutes {
					if i.Match != nil {
						for _, j := range i.Match {
							paths = append(paths, j.Uri.GetPrefix())
						}
					}
				}

				Expect(paths).Should(ContainElement("/reviews"))
				Expect(paths).Should(ContainElement("/products"))
				Expect(len(httpRoutes)).To(Equal(3))
				Expect(vs).ToNot(BeNil())
			}
		})

		// cleanup
		AfterEach(func() {
			_, filename, _, _ := runtime.Caller(0)
			pwd := filepath.Dir(filename)

			cmd := exec.Command("kubectl", "delete", "-f", pwd+"/../data/vs-merge-1.yaml", "-n", "istio-virtualservice-merger")
			_, err := cmd.Output()
			if err != nil {
				panic(err)
			}
			cmd = exec.Command("kubectl", "delete", "-f", pwd+"/../data/vs.yaml", "-n", "istio-virtualservice-merger")
			_, err = cmd.Output()
			if err != nil {
				panic(err)
			}
		})
	})
})

// ========================================================
func Contains[T comparable](s []T, e T) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// connect to cluster
func GetK8sConfig() *rest.Config {
	// Location of the kubeconfig file
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	// Create the client config
	ccmd := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: kubeconfig,
		},
		&clientcmd.ConfigOverrides{
			CurrentContext: "",
		},
	)

	rawConfig, err := ccmd.RawConfig()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	config, err := ccmd.ClientConfig()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Printf("\nRunning post deployment check on cluster: %s\n", rawConfig.CurrentContext)
	return config
}
