package kom

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func (k *Kubectl) WithContext(ctx context.Context) *Kubectl {
	tx := k.getInstance()
	tx.Statement.Context = ctx
	return tx
}
func (k *Kubectl) Resource(obj runtime.Object) *Kubectl {
	tx := k.getInstance()
	tx.Statement.ParseFromRuntimeObj(obj)
	return tx
}
func (k *Kubectl) Namespace(ns string) *Kubectl {
	tx := k.getInstance()
	tx.Statement.Namespace = ns
	return tx
}
func (k *Kubectl) ContainerName(c string) *Kubectl {
	tx := k.getInstance()
	tx.Statement.ContainerName = c
	return tx
}
func (k *Kubectl) Name(name string) *Kubectl {
	tx := k.getInstance()
	tx.Statement.Name = name
	return tx
}

func (k *Kubectl) CRD(group string, version string, kind string) *Kubectl {
	return k.GVK(group, version, kind)
}
func (k *Kubectl) GVK(group string, version string, kind string) *Kubectl {
	gvk := schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    kind,
	}
	tx := k.getInstance()
	tx.Statement.useCustomGVK = true
	tx.Statement.ParseGVKs([]schema.GroupVersionKind{
		gvk,
	})

	return tx
}
func (k *Kubectl) Command(command string, args ...string) *Kubectl {
	tx := k.getInstance()
	tx.Statement.Command = command
	tx.Statement.Args = args
	return tx
}
func (k *Kubectl) GetLogs(requestPtr interface{}, opt *v1.PodLogOptions) *Kubectl {
	tx := k.getInstance()
	tx.Statement.PodLogOptions = opt
	tx.Statement.PodLogOptions.Container = tx.Statement.ContainerName
	tx.Statement.Dest = requestPtr
	tx.Error = tx.Callback().Logs().Execute(tx)
	return tx
}

func (k *Kubectl) Get(dest interface{}) *Kubectl {
	tx := k.getInstance()
	tx.Statement.Dest = dest
	tx.Error = tx.Callback().Get().Execute(tx)
	return tx
}
func (k *Kubectl) List(dest interface{}, opt ...metav1.ListOptions) *Kubectl {
	tx := k.getInstance()
	tx.Statement.ListOptions = opt
	tx.Statement.Dest = dest
	tx.Error = tx.Callback().List().Execute(tx)
	return tx
}
func (k *Kubectl) Create(dest interface{}) *Kubectl {
	tx := k.getInstance()
	tx.Statement.Dest = dest
	tx.Error = tx.Callback().Create().Execute(tx)
	return tx
}
func (k *Kubectl) Update(dest interface{}) *Kubectl {
	tx := k.getInstance()
	tx.Statement.Dest = dest
	tx.Error = tx.Callback().Update().Execute(tx)
	return tx
}
func (k *Kubectl) Delete() *Kubectl {
	tx := k.getInstance()
	tx.Error = tx.Callback().Delete().Execute(tx)
	return tx
}
func (k *Kubectl) Patch(dest interface{}, pt types.PatchType, data string) *Kubectl {
	tx := k.getInstance()
	tx.Statement.Dest = dest
	tx.Statement.PatchData = data
	tx.Statement.PatchType = pt
	tx.Error = tx.Callback().Patch().Execute(tx)
	return tx
}
func (k *Kubectl) ExecuteCommand(dest interface{}) *Kubectl {
	tx := k.getInstance()
	tx.Statement.Dest = dest
	tx.Error = tx.Callback().Exec().Execute(tx)
	return tx
}

func (k *Kubectl) fill(m *unstructured.Unstructured) *Kubectl {
	tx := k.getInstance()
	if tx.Statement.Dest == nil {
		tx.Error = fmt.Errorf("请先执行Get()、List()等方法")
	}
	// 确保将数据填充到传入的 m 中
	if dest, ok := tx.Statement.Dest.(*unstructured.Unstructured); ok {
		*m = *dest
	}
	return k
}
