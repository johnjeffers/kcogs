package cluster

import (
	"context"
	"fmt"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cogs"
)

// Client provides access to a Kubernetes cluster
type Client struct {
	name      string
	context   string
	clientset kubernetes.Interface
}

// NewClientFromConfig creates a new cluster client from a parsed kubeconfig
func NewClientFromConfig(name string, config *api.Config, contextName string) (*Client, error) {
	clientConfig := clientcmd.NewNonInteractiveClientConfig(*config, contextName, &clientcmd.ConfigOverrides{}, nil)
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("building config for context %s: %w", contextName, err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating clientset: %w", err)
	}

	return &Client{
		name:      name,
		context:   contextName,
		clientset: clientset,
	}, nil
}

// Name returns the cluster name
func (c *Client) Name() string {
	return c.name
}

// Context returns the kubeconfig context name
func (c *Client) Context() string {
	return c.context
}

// IsHealthy checks cluster connectivity
func (c *Client) IsHealthy(ctx context.Context) error {
	_, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	return err
}

// GetNodes returns all nodes in the cluster
func (c *Client) GetNodes(ctx context.Context) ([]cogs.NodeInfo, error) {
	nodeList, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	nodes := make([]cogs.NodeInfo, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		instanceType := node.Labels["node.kubernetes.io/instance-type"]
		if instanceType == "" {
			instanceType = node.Labels["beta.kubernetes.io/instance-type"]
		}

		region := node.Labels["topology.kubernetes.io/region"]
		if region == "" {
			region = node.Labels["failure-domain.beta.kubernetes.io/region"]
		}

		// Detect Fargate nodes
		isFargate := node.Labels["eks.amazonaws.com/compute-type"] == "fargate"

		cpuCapacity := node.Status.Capacity.Cpu().MilliValue()
		memCapacity := node.Status.Capacity.Memory().Value()

		nodes = append(nodes, cogs.NodeInfo{
			Name:         node.Name,
			ClusterName:  c.name,
			InstanceType: instanceType,
			Region:       region,
			CPUCapacity:  cpuCapacity,
			MemCapacity:  memCapacity,
			Labels:       node.Labels,
			HourlyCost:   0, // Will be set by pricing provider
			IsFargate:    isFargate,
		})
	}

	return nodes, nil
}

// GetPods returns all pods in the cluster
func (c *Client) GetPods(ctx context.Context, namespaces []string) ([]cogs.PodInfo, error) {
	var allPods []cogs.PodInfo

	if len(namespaces) == 0 {
		namespaces = []string{""} // Empty string means all namespaces
	}

	for _, ns := range namespaces {
		podList, err := c.clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
			FieldSelector: "status.phase=Running",
		})
		if err != nil {
			return nil, fmt.Errorf("listing pods in namespace %s: %w", ns, err)
		}

		for _, pod := range podList.Items {
			if pod.Status.Phase != corev1.PodRunning {
				continue
			}

			var cpuRequest, memRequest int64
			for _, container := range pod.Spec.Containers {
				cpuRequest += container.Resources.Requests.Cpu().MilliValue()
				memRequest += container.Resources.Requests.Memory().Value()
			}

			workloadName, workloadKind := getWorkloadInfo(&pod)

			allPods = append(allPods, cogs.PodInfo{
				Name:          pod.Name,
				Namespace:     pod.Namespace,
				ClusterName:   c.name,
				NodeName:      pod.Spec.NodeName,
				WorkloadName:  workloadName,
				WorkloadKind:  workloadKind,
				CPURequest:    cpuRequest,
				MemoryRequest: memRequest,
				Labels:        pod.Labels,
			})
		}
	}

	return allPods, nil
}

// getWorkloadInfo extracts the owning workload name and kind from a pod
func getWorkloadInfo(pod *corev1.Pod) (string, string) {
	for _, owner := range pod.OwnerReferences {
		if owner.Controller != nil && *owner.Controller {
			// ReplicaSets are owned by Deployments - resolve to the Deployment name
			// ReplicaSet names follow the pattern: <deployment-name>-<pod-template-hash>
			if owner.Kind == "ReplicaSet" {
				deploymentName := resolveDeploymentName(owner.Name)
				return deploymentName, "Deployment"
			}
			return owner.Name, owner.Kind
		}
	}
	return pod.Name, "Pod"
}

// resolveDeploymentName extracts the Deployment name from a ReplicaSet name
// ReplicaSet names are formatted as: <deployment-name>-<pod-template-hash>
func resolveDeploymentName(replicaSetName string) string {
	// Find the last hyphen and strip the hash suffix
	lastHyphen := strings.LastIndex(replicaSetName, "-")
	if lastHyphen > 0 {
		return replicaSetName[:lastHyphen]
	}
	return replicaSetName
}

// Manager manages multiple cluster connections
type Manager struct {
	mu         sync.RWMutex
	clients    map[string]*Client
	kubeconfig *api.Config
	filename   string
}

// NewManager creates a new cluster manager
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*Client),
	}
}

// SetKubeconfigContent sets the kubeconfig from file content (replaces existing)
func (m *Manager) SetKubeconfigContent(content []byte, filename string) error {
	config, err := clientcmd.Load(content)
	if err != nil {
		return fmt.Errorf("parsing kubeconfig: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.kubeconfig = config
	m.filename = filename
	// Clear existing connections when kubeconfig changes
	m.clients = make(map[string]*Client)
	return nil
}

// AddKubeconfigContent merges a kubeconfig into the existing one
func (m *Manager) AddKubeconfigContent(content []byte, filename string) error {
	config, err := clientcmd.Load(content)
	if err != nil {
		return fmt.Errorf("parsing kubeconfig: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.kubeconfig == nil {
		m.kubeconfig = config
		m.filename = filename
		return nil
	}

	// Merge contexts, clusters, and users from new config into existing
	for name, ctx := range config.Contexts {
		m.kubeconfig.Contexts[name] = ctx
	}
	for name, cluster := range config.Clusters {
		m.kubeconfig.Clusters[name] = cluster
	}
	for name, user := range config.AuthInfos {
		m.kubeconfig.AuthInfos[name] = user
	}

	// Append filename
	if m.filename != "" {
		m.filename = m.filename + ", " + filename
	} else {
		m.filename = filename
	}

	return nil
}

// GetKubeconfigFilename returns the filename of the loaded kubeconfig
func (m *Manager) GetKubeconfigFilename() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.filename
}

// HasKubeconfig returns true if a kubeconfig is loaded
func (m *Manager) HasKubeconfig() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.kubeconfig != nil
}

// ListContexts returns all contexts from the kubeconfig
func (m *Manager) ListContexts() ([]ContextInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.kubeconfig == nil {
		return nil, nil
	}

	contexts := make([]ContextInfo, 0, len(m.kubeconfig.Contexts))
	for name, ctx := range m.kubeconfig.Contexts {
		contexts = append(contexts, ContextInfo{
			Name:      name,
			Cluster:   ctx.Cluster,
			IsCurrent: name == m.kubeconfig.CurrentContext,
		})
	}

	return contexts, nil
}

// ContextInfo holds information about a kubeconfig context
type ContextInfo struct {
	Name      string `json:"name"`
	Cluster   string `json:"cluster"`
	IsCurrent bool   `json:"isCurrent"`
}

// AddCluster adds a cluster connection using a kubeconfig context
func (m *Manager) AddCluster(ctx context.Context, name, contextName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.kubeconfig == nil {
		return fmt.Errorf("kubeconfig not loaded")
	}

	client, err := NewClientFromConfig(name, m.kubeconfig, contextName)
	if err != nil {
		return err
	}

	// Verify connectivity
	if err := client.IsHealthy(ctx); err != nil {
		return fmt.Errorf("cluster not healthy: %w", err)
	}

	m.clients[name] = client
	return nil
}

// RemoveCluster removes a cluster connection
func (m *Manager) RemoveCluster(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, name)
}

// RemoveContext removes a context from the kubeconfig and disconnects if connected
func (m *Manager) RemoveContext(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove from connected clients
	delete(m.clients, name)

	// Remove from kubeconfig
	if m.kubeconfig != nil {
		if ctx, ok := m.kubeconfig.Contexts[name]; ok {
			// Remove the context
			delete(m.kubeconfig.Contexts, name)
			// Also remove the associated cluster and user if not used by other contexts
			clusterInUse := false
			userInUse := false
			for _, otherCtx := range m.kubeconfig.Contexts {
				if otherCtx.Cluster == ctx.Cluster {
					clusterInUse = true
				}
				if otherCtx.AuthInfo == ctx.AuthInfo {
					userInUse = true
				}
			}
			if !clusterInUse {
				delete(m.kubeconfig.Clusters, ctx.Cluster)
			}
			if !userInUse {
				delete(m.kubeconfig.AuthInfos, ctx.AuthInfo)
			}
		}
	}
}

// GetCluster returns a specific cluster client
func (m *Manager) GetCluster(name string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[name]
	return client, ok
}

// ListClusters returns info about all connected clusters
func (m *Manager) ListClusters() []ClusterInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]ClusterInfo, 0, len(m.clients))
	for name, client := range m.clients {
		infos = append(infos, ClusterInfo{
			Name:    name,
			Context: client.Context(),
			Status:  "connected",
		})
	}
	return infos
}

// ClusterInfo holds information about a connected cluster
type ClusterInfo struct {
	Name    string `json:"name"`
	Context string `json:"context"`
	Status  string `json:"status"`
}

// GetAllNodes returns nodes from all connected clusters
func (m *Manager) GetAllNodes(ctx context.Context) ([]cogs.NodeInfo, error) {
	m.mu.RLock()
	clients := make([]*Client, 0, len(m.clients))
	for _, c := range m.clients {
		clients = append(clients, c)
	}
	m.mu.RUnlock()

	var allNodes []cogs.NodeInfo
	for _, client := range clients {
		nodes, err := client.GetNodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("getting nodes from %s: %w", client.Name(), err)
		}
		allNodes = append(allNodes, nodes...)
	}
	return allNodes, nil
}

// GetAllPods returns pods from all connected clusters
func (m *Manager) GetAllPods(ctx context.Context, namespaces []string) ([]cogs.PodInfo, error) {
	m.mu.RLock()
	clients := make([]*Client, 0, len(m.clients))
	for _, c := range m.clients {
		clients = append(clients, c)
	}
	m.mu.RUnlock()

	var allPods []cogs.PodInfo
	for _, client := range clients {
		pods, err := client.GetPods(ctx, namespaces)
		if err != nil {
			return nil, fmt.Errorf("getting pods from %s: %w", client.Name(), err)
		}
		allPods = append(allPods, pods...)
	}
	return allPods, nil
}
