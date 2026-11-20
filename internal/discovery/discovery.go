package discovery

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/vutratenko/shturval-alert-wall/internal/source"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var vmaGVR = schema.GroupVersionResource{
	Group:    "operator.victoriametrics.com",
	Version:  "v1beta1",
	Resource: "vmalertmanagers",
}

type StaticEndpoint struct {
	Cluster string `json:"cluster" yaml:"cluster"`
	URL     string `json:"url" yaml:"url"`
}

type Discoverer struct {
	Client client.Reader
	Static []StaticEndpoint
	Port   int
}

func (d *Discoverer) Sources(ctx context.Context) ([]source.Source, error) {
	byCluster := map[string]string{}
	for _, s := range d.Static {
		if s.Cluster != "" && s.URL != "" {
			byCluster[s.Cluster] = s.URL
		}
	}
	if d.Client != nil {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{Group: vmaGVR.Group, Version: vmaGVR.Version, Kind: "VMAlertmanagerList"})
		if err := d.Client.List(ctx, list); err == nil {
			for _, item := range list.Items {
				ns := item.GetNamespace()
				name := item.GetName()
				url := alertmanagerServiceURL(name, ns, d.Port)
				byCluster[ns] = url
			}
		}
	}
	out := make([]source.Source, 0, len(byCluster))
	for cluster, url := range byCluster {
		out = append(out, &source.Alertmanager{ClusterName: cluster, BaseURL: url})
	}
	return out, nil
}

func alertmanagerServiceURL(name, ns string, port int) string {
	if port <= 0 {
		port = 9093
	}
	svc := fmt.Sprintf("vmalertmanager-%s", strings.TrimPrefix(name, "vmalertmanager-"))
	if name == svc || strings.HasPrefix(name, "vmalertmanager-") {
		svc = name
	} else if !strings.HasPrefix(name, "vmalertmanager") {
		svc = "vmalertmanager-" + name
	} else {
		svc = name
	}
	return fmt.Sprintf("http://%s.%s.svc:%d", svc, ns, port)
}

type Refreshing struct {
	Discoverer Discoverer
	mu         sync.RWMutex
	sources    []source.Source
}

func (r *Refreshing) Replace(sources []source.Source) {
	r.mu.Lock()
	r.sources = sources
	r.mu.Unlock()
}

func (r *Refreshing) Snapshot() []source.Source {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]source.Source, len(r.sources))
	copy(out, r.sources)
	return out
}

func (r *Refreshing) Refresh(ctx context.Context) error {
	src, err := r.Discoverer.Sources(ctx)
	if err != nil {
		return err
	}
	r.Replace(src)
	return nil
}

// ListVMAlertmanagers is a test helper using typed list.
func ListVMAlertmanagers(ctx context.Context, c client.Reader) ([]StaticEndpoint, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{Group: vmaGVR.Group, Version: vmaGVR.Version, Kind: "VMAlertmanagerList"})
	if err := c.List(ctx, list); err != nil {
		return nil, err
	}
	out := make([]StaticEndpoint, 0, len(list.Items))
	for _, item := range list.Items {
		out = append(out, StaticEndpoint{
			Cluster: item.GetNamespace(),
			URL:     alertmanagerServiceURL(item.GetName(), item.GetNamespace(), 9093),
		})
	}
	return out, nil
}

// NewFakeClientListItem builds unstructured VMAlertmanager for tests.
func NewFakeClientListItem(ns, name string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{Group: vmaGVR.Group, Version: vmaGVR.Version, Kind: "VMAlertmanager"})
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetCreationTimestamp(metav1.Now())
	return u
}