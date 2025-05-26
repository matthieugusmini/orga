package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions"
	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	m := model{
		workflows: map[string]*v1alpha1.Workflow{},
	}
	p := tea.NewProgram(m, tea.WithAltScreen())

	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String(
			"kubeconfig",
			filepath.Join(home, ".kube", "config"),
			"(optional) absolute path to the kubeconfig file",
		)
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	argoClient, _ := versioned.NewForConfig(config)

	factory := externalversions.NewSharedInformerFactoryWithOptions(
		argoClient,
		30*time.Second,
		externalversions.WithNamespace("argo"),
	)
	workflowInformer := factory.Argoproj().V1alpha1().Workflows().Informer()

	workflowInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			wf := obj.(*v1alpha1.Workflow)
			p.Send(workflowEventMsg{
				workflow:  wf,
				eventType: watch.Added,
			})
		},

		UpdateFunc: func(oldObj, newObj any) {
			wf := newObj.(*v1alpha1.Workflow)
			p.Send(workflowEventMsg{
				workflow:  wf,
				eventType: watch.Modified,
			})
		},

		DeleteFunc: func(obj any) {
			wf := obj.(*v1alpha1.Workflow)
			p.Send(workflowEventMsg{
				workflow:  wf,
				eventType: watch.Deleted,
			})
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run the program: %v", err)
	}
}
