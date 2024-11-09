package example

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/weibaohui/kom/kom"
	"github.com/weibaohui/kom/kom_starter"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
)

// TestMain 是测试的入口函数
func TestMain(m *testing.M) {
	// 初始化操作
	fmt.Println("Initializing test environment...")
	// 在这里可以设置数据库连接、启动服务、创建临时文件等
	kom_starter.Init()

	// 创建测试必须得Pod，后面不会再创建了
	InitTestDeploy()
	// 调用 m.Run() 运行所有测试
	exitCode := m.Run()

	// 清理操作
	fmt.Println("Cleaning up test environment...")
	// 在这里可以关闭数据库连接、删除临时文件等

	// 退出程序
	os.Exit(exitCode)
}

// 创建一个通道用于控制停止信号
var stopCh = make(chan struct{})

func InitTestDeploy() {
	yaml := `apiVersion: v1
kind: Pod
metadata:
  name: random
  namespace: default
spec:
  containers:
  - args:
    - |
      mkdir -p /var/log;
      while true; do
        random_char="A$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | head -c 1)";
        echo $random_char | tee -a /var/log/random_a.log;
        sleep 5;
      done
    command:
    - /bin/sh
    - -c
    image: alpine
    name: random
`
	result := kom.DefaultCluster().Applier().Apply(yaml)
	for _, s := range result {
		fmt.Printf("%s\n", s)
	}

	// 定义检查的周期
	period := 5 * time.Second

	// 启动一个 goroutine 来执行 Until
	go wait.Until(checkCondition, period, stopCh)
	// 使用一个超时通道，在 10 秒后自动关闭 stopCh
	time.AfterFunc(60*time.Second, func() {
		fmt.Println("Timeout reached, stopping monitoring.")
		close(stopCh)
	})
	// 防止主程序过早退出
	<-stopCh

	fmt.Println("Stopped checking condition.")

}

// 定义一个函数，用于检查条件
func checkCondition() {
	fmt.Println("Checking condition at", time.Now())

	var pod corev1.Pod
	err := kom.DefaultCluster().Resource(&pod).Namespace("default").
		Name("random").Get(&pod).Error
	if err != nil {
		return
	}
	if pod.Status.Phase == "Running" {
		fmt.Println("pod is running at", time.Now())
		// 停止检查
		close(stopCh)
	}
}

func TestYamlApplyDelete(t *testing.T) {
	yaml := `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config
  namespace: default
data:
  key: value
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-deployment
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: example
  template:
    metadata:
      labels:
        app: example
    spec:
      containers:
        - name: example-container
          image: nginx
`

	// Apply the YAML
	t.Run("Apply Resources", func(t *testing.T) {
		result := kom.DefaultCluster().Applier().Apply(yaml)
		for _, r := range result {
			fmt.Println(r)
		}
	})

	// Delete the resources
	t.Run("Delete Resources", func(t *testing.T) {
		result := kom.DefaultCluster().Applier().Delete(yaml)
		for _, r := range result {
			fmt.Println(r)
		}
	})
}

func TestCrdExample(t *testing.T) {
	yaml := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: crontabs.stable.example.com
spec:
  group: stable.example.com
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                cronSpec:
                  type: string
                image:
                  type: string
                replicas:
                  type: integer
  scope: Namespaced
  names:
    plural: crontabs
    singular: crontab
    kind: CronTab
`

	t.Run("Apply CRD", func(t *testing.T) {
		result := kom.DefaultCluster().Applier().Apply(yaml)
		for _, str := range result {
			fmt.Println(str)
		}
	})

	t.Run("Create CR", func(t *testing.T) {
		time.Sleep(10 * time.Second)
		var crontab = unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "stable.example.com/v1",
				"kind":       "CronTab",
				"metadata": map[string]interface{}{
					"name":      "test-crontab",
					"namespace": "default",
				},
				"spec": map[string]interface{}{
					"cronSpec": "* * * * */8",
					"image":    "test-crontab-image",
				},
			},
		}

		err := kom.DefaultCluster().
			WithContext(context.TODO()).
			CRD("stable.example.com", "v1", "CronTab").
			Name(crontab.GetName()).
			Namespace(crontab.GetNamespace()).
			Create(&crontab).Error
		if err != nil {
			fmt.Printf("CRD Create error: %v\n", err)
		}
	})

	t.Run("Get CR", func(t *testing.T) {
		var crontab unstructured.Unstructured
		err := kom.DefaultCluster().
			WithContext(context.TODO()).
			CRD("stable.example.com", "v1", "CronTab").
			Name("test-crontab").
			Namespace("default").
			Get(&crontab).Error
		if err != nil {
			fmt.Printf("CRD Get error: %v\n", err)
		}
	})

	t.Run("List CR", func(t *testing.T) {
		var crontabList []unstructured.Unstructured
		err := kom.DefaultCluster().
			WithContext(context.TODO()).
			CRD("stable.example.com", "v1", "CronTab").
			Namespace("default").
			List(&crontabList).Error
		if err != nil {
			fmt.Printf("CRD List error: %v\n", err)
		}
		fmt.Printf("CRD List count %d\n", len(crontabList))
	})

	t.Run("Delete CR", func(t *testing.T) {
		err := kom.DefaultCluster().
			WithContext(context.TODO()).
			CRD("stable.example.com", "v1", "CronTab").
			Name("test-crontab").
			Namespace("default").
			Delete().Error
		if err != nil {
			fmt.Printf("CRD Delete error: %v\n", err)
		}
	})
}

func TestBuiltInExample(t *testing.T) {
	yaml := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
`

	t.Run("Apply Built-in Resources", func(t *testing.T) {
		result := kom.DefaultCluster().Applier().Apply(yaml)
		for _, str := range result {
			fmt.Println(str)
		}
	})

	t.Run("Get Deployment", func(t *testing.T) {
		item := v1.Deployment{}
		err := kom.DefaultCluster().
			WithContext(context.TODO()).
			Resource(&item).
			Namespace("default").
			Name("nginx").
			Get(&item).Error
		if err != nil {
			t.Fatalf("Deployment Get error: %v", err)
		}
		fmt.Printf("Get Item %s\n", item.Spec.Template.Spec.Containers[0].Image)
	})

	t.Run("Delete Built-in Resources", func(t *testing.T) {
		result := kom.DefaultCluster().Applier().Delete(yaml)
		for _, str := range result {
			fmt.Println(str)
		}
	})
}

func TestPodLogs(t *testing.T) {
	yaml := `apiVersion: v1
kind: Pod
metadata:
  name: random-char-pod-1
  namespace: default
spec:
  containers:
  - args:
    - |
      mkdir -p /var/log;
      while true; do
        random_char="A$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | head -c 1)";
        echo $random_char | tee -a /var/log/random_a.log;
        sleep 5;
      done
    command:
    - /bin/sh
    - -c
    image: alpine
    name: container-b
`

	t.Run("Apply Pod", func(t *testing.T) {
		result := kom.DefaultCluster().Applier().Apply(yaml)
		for _, str := range result {
			fmt.Println(str)
			if strings.Contains(str, "err") {
				t.Fatalf("Apply Pod error: %v", str)
			}
		}
	})

	t.Run("Get Pod Logs", func(t *testing.T) {
		time.Sleep(10 * time.Second)
		// 进行后续的测试逻辑
		t.Log("Waited for 5 seconds")

		var stream io.ReadCloser
		err := kom.DefaultCluster().
			Namespace("default").
			Name("random-char-pod").
			ContainerName("container").
			GetLogs(&stream, &corev1.PodLogOptions{}).Error
		if err != nil {
			fmt.Printf("Error getting pod logs:%v\n", err)
		}
		if stream == nil {
			return
		}
		reader := bufio.NewReader(stream)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Fatalf("Error reading stream: %v", err)
			}
			if !strings.Contains(line, "A") {
				t.Fatalf("日志读取测试失败,应该包含A。%s", line)
			}
		}
	})

	t.Run("Cleanup Pod", func(t *testing.T) {
		result := kom.DefaultCluster().Applier().Delete(yaml)
		for _, str := range result {
			fmt.Println(str)
			if strings.Contains(str, "err") {
				t.Fatalf("Cleanup error: %v", str)
			}
		}
	})
}
