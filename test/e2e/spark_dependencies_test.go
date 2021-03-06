package e2e

import (
	"context"
	"fmt"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func SparkDependenciesElasticsearch(t *testing.T) {
	testCtx := prepare(t)
	defer testCtx.Cleanup()
	storage := v1alpha1.JaegerStorageSpec{
		Type: "elasticsearch",
		Options: v1alpha1.NewOptions(map[string]interface{}{
			"es.server-urls": "http://elasticsearch.default.svc:9200",
		}),
	}
	if err := sparkTest(t, framework.Global, testCtx, storage); err != nil {
		t.Fatal(err)
	}
}

func SparkDependenciesCassandra(t *testing.T) {
	testCtx := prepare(t)
	defer testCtx.Cleanup()

	storage := v1alpha1.JaegerStorageSpec{
		Type: "cassandra",
		Options: v1alpha1.NewOptions(map[string]interface{}{"cassandra.servers": "cassandra.default.svc", "cassandra.keyspace": "jaeger_v1_datacenter1"}),
		CassandraCreateSchema:v1alpha1.JaegerCassandraCreateSchemaSpec{Datacenter:"datacenter1", Mode: "prod"},
	}
	if err := sparkTest(t, framework.Global, testCtx, storage); err != nil {
		t.Fatal(err)
	}
}

func sparkTest(t *testing.T, f *framework.Framework, testCtx *framework.TestCtx, storage v1alpha1.JaegerStorageSpec) error {
	namespace, err := testCtx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	storage.SparkDependencies = v1alpha1.JaegerDependenciesSpec{
		// run immediately
		Schedule: "*/1 * * * *",
	}

	name := "test-spark-deps"
	j := &v1alpha1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "io.jaegertracing/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.JaegerSpec{
			Strategy: "allInOne",
			AllInOne: v1alpha1.JaegerAllInOneSpec{},
			Storage:  storage,
		},
	}

	err = f.Client.Create(context.Background(), j, &framework.CleanupOptions{TestContext: testCtx, Timeout: timeout, RetryInterval: retryInterval})
	if err != nil {
		return err
	}

	err = WaitForCronJob(t, f.KubeClient, namespace, fmt.Sprintf("%s-spark-dependencies", name), retryInterval, timeout)
	if err != nil {
		return err
	}

	err = WaitForJobOfAnOwner(t, f.KubeClient, namespace, fmt.Sprintf("%s-spark-dependencies", name), retryInterval, timeout)
	if err != nil {
		return err
	}

	return e2eutil.WaitForDeployment(t, f.KubeClient, namespace, name, 1, retryInterval, timeout)
}
