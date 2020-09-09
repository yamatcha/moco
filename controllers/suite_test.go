package controllers

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	mocov1alpha1 "github.com/cybozu-go/moco/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

type AccessorMock struct {
}

func (acc *AccessorMock) Get(addr, user, password string) (*sqlx.DB, error) {

	conf := mysql.NewConfig()
	conf.User = "root"
	conf.Passwd = "test-password"
	conf.Net = "tcp"
	conf.Addr = "localhost:3306"
	conf.Timeout = 3
	conf.ReadTimeout = 3
	conf.InterpolateParams = true

	db, err := sqlx.Connect("mysql", conf.FormatDSN())
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (acc *AccessorMock) Remove(cluster *mocov1alpha1.MySQLCluster) {
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	// logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.StacktraceLevel(&zap.AtomicLevel{}, zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	sch := runtime.NewScheme()
	err = clientgoscheme.AddToScheme(sch)
	Expect(err).NotTo(HaveOccurred())

	err = mocov1alpha1.AddToScheme(sch)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		MetricsBindAddress: ":8081",
		Scheme:             sch,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&MySQLClusterReconciler{
		Client:                 mgr.GetClient(),
		Log:                    ctrl.Log.WithName("controllers").WithName("MySQLCluster"),
		Scheme:                 mgr.GetScheme(),
		ConfInitContainerImage: "dummy",
		CurlContainerImage:     "dummy",
		MySQLAccessor:          &AccessorMock{},
	}).SetupWithManager(mgr, time.Second)
	Expect(err).ToNot(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: sch})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
