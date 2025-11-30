// Package discovery 实现服务发现功能
package discovery

import (
	"context"
	"fmt"

	networkingv1beta1 "istio.io/api/networking/v1beta1"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	versionedclient "istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/boifi/service-discovery/internal/types"
	"github.com/boifi/service-discovery/pkg/logger"
)

// KubernetesDiscoverer 通过 Kubernetes API 发现服务
type KubernetesDiscoverer struct {
	istioClient versionedclient.Interface
	log         logger.Logger
}

// NewKubernetesDiscoverer 创建新的 Kubernetes 发现器
// kubeconfigPath 为空时使用 in-cluster 配置
func NewKubernetesDiscoverer(kubeconfigPath string, log logger.Logger) (*KubernetesDiscoverer, error) {
	var config *rest.Config
	var err error

	if kubeconfigPath == "" {
		// 使用 in-cluster 配置
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	} else {
		// 使用 kubeconfig 文件
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}

	istioClient, err := versionedclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create istio client: %w", err)
	}

	if log == nil {
		log = logger.NewDefault()
	}

	return &KubernetesDiscoverer{
		istioClient: istioClient,
		log:         log,
	}, nil
}

// NewKubernetesDiscovererWithClient 使用提供的客户端创建发现器（用于测试）
func NewKubernetesDiscovererWithClient(istioClient versionedclient.Interface, log logger.Logger) *KubernetesDiscoverer {
	if log == nil {
		log = logger.NewDefault()
	}
	return &KubernetesDiscoverer{
		istioClient: istioClient,
		log:         log,
	}
}

// ListVirtualServices 列出指定命名空间（或所有命名空间）的 VirtualService
// namespace 为空时列出所有命名空间
func (k *KubernetesDiscoverer) ListVirtualServices(ctx context.Context, namespace string) ([]*v1beta1.VirtualService, error) {
	k.log.Debug("listing virtual services", "namespace", namespace)

	vsList, err := k.istioClient.NetworkingV1beta1().VirtualServices(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		k.log.Error("failed to list virtual services", "error", err.Error(), "namespace", namespace)
		return nil, fmt.Errorf("failed to list virtual services: %w", err)
	}

	result := make([]*v1beta1.VirtualService, 0, len(vsList.Items))
	for i := range vsList.Items {
		vs := vsList.Items[i]
		result = append(result, vs)
	}

	k.log.Debug("found virtual services", "count", len(result), "namespace", namespace)
	return result, nil
}

// ParseVirtualService 解析单个 VirtualService，提取服务和 API 端点信息
func (k *KubernetesDiscoverer) ParseVirtualService(vs *v1beta1.VirtualService) []types.ServiceInfo {
	if vs == nil || vs.Spec.Http == nil {
		return nil
	}

	// 使用 map 聚合同一服务的 API
	serviceAPIs := make(map[string]*types.ServiceInfo)

	for _, httpRoute := range vs.Spec.Http {
		// 获取目标服务
		destinations := httpRoute.Route
		if len(destinations) == 0 {
			continue
		}

		// 获取 API 端点
		apis := k.extractAPIsFromRoute(httpRoute)

		// 为每个目标服务添加 API
		for _, dest := range destinations {
			if dest.Destination == nil || dest.Destination.Host == "" {
				continue
			}

			serviceName := dest.Destination.Host

			// 获取或创建服务信息
			svcInfo, exists := serviceAPIs[serviceName]
			if !exists {
				svcInfo = &types.ServiceInfo{
					Name:      serviceName,
					Namespace: vs.Namespace,
					APIs:      []types.APIEndpoint{},
					Source:    types.SourceVirtualService,
				}
				serviceAPIs[serviceName] = svcInfo
			}

			// 添加 API 端点
			svcInfo.APIs = append(svcInfo.APIs, apis...)
		}
	}

	// 转换为切片
	result := make([]types.ServiceInfo, 0, len(serviceAPIs))
	for _, svc := range serviceAPIs {
		result = append(result, *svc)
	}

	k.log.Debug("parsed virtual service",
		"name", vs.Name,
		"namespace", vs.Namespace,
		"services_found", len(result),
	)

	return result
}

// extractAPIsFromRoute 从 HTTP 路由中提取 API 端点
func (k *KubernetesDiscoverer) extractAPIsFromRoute(httpRoute *networkingv1beta1.HTTPRoute) []types.APIEndpoint {
	var apis []types.APIEndpoint

	// 如果没有 match，创建一个通配符 API
	if len(httpRoute.Match) == 0 {
		apis = append(apis, types.APIEndpoint{
			Method:    types.MethodAll,
			Path:      "/*",
			MatchType: types.MatchTypePrefix,
		})
		return apis
	}

	for _, match := range httpRoute.Match {
		api := types.APIEndpoint{
			Method: ExtractHTTPMethod(match.Method),
		}

		// 提取 URI 匹配
		if match.Uri != nil {
			switch m := match.Uri.MatchType.(type) {
			case *networkingv1beta1.StringMatch_Exact:
				api.Path = m.Exact
				api.MatchType = types.MatchTypeExact
			case *networkingv1beta1.StringMatch_Prefix:
				api.Path = m.Prefix
				api.MatchType = types.MatchTypePrefix
			case *networkingv1beta1.StringMatch_Regex:
				api.Path = m.Regex
				api.MatchType = types.MatchTypeRegex
			default:
				// 未知匹配类型，跳过
				continue
			}
		} else {
			// 没有 URI 匹配，使用通配符
			api.Path = "/*"
			api.MatchType = types.MatchTypePrefix
		}

		apis = append(apis, api)
	}

	return apis
}

// ExtractHTTPMethod 从 StringMatch 中提取 HTTP 方法
// 如果未指定方法，返回 "*"
func ExtractHTTPMethod(method *networkingv1beta1.StringMatch) string {
	if method == nil {
		return types.MethodAll
	}

	switch m := method.MatchType.(type) {
	case *networkingv1beta1.StringMatch_Exact:
		return m.Exact
	case *networkingv1beta1.StringMatch_Prefix:
		return m.Prefix
	case *networkingv1beta1.StringMatch_Regex:
		return m.Regex
	default:
		return types.MethodAll
	}
}

// AggregateServices 聚合所有 VirtualService 中的服务信息
// 相同服务的 API 端点会被合并
func (k *KubernetesDiscoverer) AggregateServices(ctx context.Context, namespace string) ([]types.ServiceInfo, error) {
	vsList, err := k.ListVirtualServices(ctx, namespace)
	if err != nil {
		return nil, err
	}

	// 使用 map 聚合同一服务的信息
	serviceMap := make(map[string]*types.ServiceInfo)

	for _, vs := range vsList {
		services := k.ParseVirtualService(vs)
		for _, svc := range services {
			existing, exists := serviceMap[svc.Name]
			if !exists {
				// 新服务
				svcCopy := svc
				serviceMap[svc.Name] = &svcCopy
			} else {
				// 合并 API
				existing.APIs = mergeAPIs(existing.APIs, svc.APIs)
			}
		}
	}

	// 转换为切片
	result := make([]types.ServiceInfo, 0, len(serviceMap))
	for _, svc := range serviceMap {
		result = append(result, *svc)
	}

	k.log.Info("aggregated services from virtual services",
		"namespace", namespace,
		"total_services", len(result),
		"total_virtualservices", len(vsList),
	)

	return result, nil
}

// mergeAPIs 合并两个 API 列表，去除重复项
func mergeAPIs(existing, new []types.APIEndpoint) []types.APIEndpoint {
	// 使用 map 去重
	apiSet := make(map[string]types.APIEndpoint)

	// 添加现有 API
	for _, api := range existing {
		key := fmt.Sprintf("%s:%s:%s", api.Method, api.Path, api.MatchType)
		apiSet[key] = api
	}

	// 添加新 API
	for _, api := range new {
		key := fmt.Sprintf("%s:%s:%s", api.Method, api.Path, api.MatchType)
		apiSet[key] = api
	}

	// 转换为切片
	result := make([]types.APIEndpoint, 0, len(apiSet))
	for _, api := range apiSet {
		result = append(result, api)
	}

	return result
}

// Discover 执行完整的服务发现流程
// 返回发现的所有服务信息
func (k *KubernetesDiscoverer) Discover(ctx context.Context, namespace string) ([]types.ServiceInfo, error) {
	k.log.Info("starting kubernetes service discovery", "namespace", namespace)

	services, err := k.AggregateServices(ctx, namespace)
	if err != nil {
		k.log.Error("kubernetes discovery failed", "error", err.Error())
		return nil, err
	}

	k.log.Info("kubernetes discovery completed",
		"services_discovered", len(services),
	)

	return services, nil
}
