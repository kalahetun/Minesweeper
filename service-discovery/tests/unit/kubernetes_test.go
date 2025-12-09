package unit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1beta1 "istio.io/api/networking/v1beta1"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	istiofake "istio.io/client-go/pkg/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/boifi/service-discovery/internal/discovery"
	"github.com/boifi/service-discovery/internal/types"
)

// createFakeVirtualService 创建测试用的 VirtualService
func createFakeVirtualService(name, namespace string, httpRoutes []*networkingv1beta1.HTTPRoute) *v1beta1.VirtualService {
	return &v1beta1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingv1beta1.VirtualService{
			Hosts:    []string{"*"},
			Gateways: []string{"bookinfo-gateway"},
			Http:     httpRoutes,
		},
	}
}

// createHTTPRoute 创建 HTTP 路由
func createHTTPRoute(matches []*networkingv1beta1.HTTPMatchRequest, destinations []*networkingv1beta1.HTTPRouteDestination) *networkingv1beta1.HTTPRoute {
	return &networkingv1beta1.HTTPRoute{
		Match: matches,
		Route: destinations,
	}
}

// createMatchRequest 创建匹配请求
func createMatchRequest(method string, uriMatch *networkingv1beta1.StringMatch) *networkingv1beta1.HTTPMatchRequest {
	match := &networkingv1beta1.HTTPMatchRequest{
		Uri: uriMatch,
	}
	if method != "" {
		match.Method = &networkingv1beta1.StringMatch{
			MatchType: &networkingv1beta1.StringMatch_Exact{Exact: method},
		}
	}
	return match
}

// createDestination 创建目标
func createDestination(host string, port uint32) *networkingv1beta1.HTTPRouteDestination {
	return &networkingv1beta1.HTTPRouteDestination{
		Destination: &networkingv1beta1.Destination{
			Host: host,
			Port: &networkingv1beta1.PortSelector{
				Number: port,
			},
		},
	}
}

func TestNewKubernetesDiscoverer(t *testing.T) {
	// 使用 fake clientset 创建发现器
	fakeClient := istiofake.NewSimpleClientset()

	discoverer := discovery.NewKubernetesDiscovererWithClient(fakeClient, nil)

	assert.NotNil(t, discoverer)
}

func TestListVirtualServices_Empty(t *testing.T) {
	ctx := context.Background()
	fakeClient := istiofake.NewSimpleClientset()
	discoverer := discovery.NewKubernetesDiscovererWithClient(fakeClient, nil)

	vsList, err := discoverer.ListVirtualServices(ctx, "")

	require.NoError(t, err)
	assert.Empty(t, vsList)
}

func TestListVirtualServices_WithData(t *testing.T) {
	ctx := context.Background()

	// 创建测试数据
	vs1 := createFakeVirtualService("bookinfo", "default", []*networkingv1beta1.HTTPRoute{
		createHTTPRoute(
			[]*networkingv1beta1.HTTPMatchRequest{
				createMatchRequest("", &networkingv1beta1.StringMatch{
					MatchType: &networkingv1beta1.StringMatch_Exact{Exact: "/productpage"},
				}),
			},
			[]*networkingv1beta1.HTTPRouteDestination{
				createDestination("productpage", 9080),
			},
		),
	})

	vs2 := createFakeVirtualService("reviews-vs", "default", []*networkingv1beta1.HTTPRoute{
		createHTTPRoute(
			[]*networkingv1beta1.HTTPMatchRequest{
				createMatchRequest("GET", &networkingv1beta1.StringMatch{
					MatchType: &networkingv1beta1.StringMatch_Prefix{Prefix: "/reviews"},
				}),
			},
			[]*networkingv1beta1.HTTPRouteDestination{
				createDestination("reviews", 9080),
			},
		),
	})

	fakeClient := istiofake.NewSimpleClientset(vs1, vs2)
	discoverer := discovery.NewKubernetesDiscovererWithClient(fakeClient, nil)

	vsList, err := discoverer.ListVirtualServices(ctx, "")

	require.NoError(t, err)
	assert.Len(t, vsList, 2)
}

func TestListVirtualServices_FilterByNamespace(t *testing.T) {
	ctx := context.Background()

	vs1 := createFakeVirtualService("vs1", "default", nil)
	vs2 := createFakeVirtualService("vs2", "production", nil)

	fakeClient := istiofake.NewSimpleClientset(vs1, vs2)
	discoverer := discovery.NewKubernetesDiscovererWithClient(fakeClient, nil)

	// 只获取 default 命名空间
	vsList, err := discoverer.ListVirtualServices(ctx, "default")

	require.NoError(t, err)
	assert.Len(t, vsList, 1)
	assert.Equal(t, "vs1", vsList[0].Name)
}

func TestParseVirtualService_ExactMatch(t *testing.T) {
	vs := createFakeVirtualService("bookinfo", "default", []*networkingv1beta1.HTTPRoute{
		createHTTPRoute(
			[]*networkingv1beta1.HTTPMatchRequest{
				createMatchRequest("GET", &networkingv1beta1.StringMatch{
					MatchType: &networkingv1beta1.StringMatch_Exact{Exact: "/productpage"},
				}),
			},
			[]*networkingv1beta1.HTTPRouteDestination{
				createDestination("productpage", 9080),
			},
		),
	})

	fakeClient := istiofake.NewSimpleClientset()
	discoverer := discovery.NewKubernetesDiscovererWithClient(fakeClient, nil)

	services := discoverer.ParseVirtualService(vs)

	require.Len(t, services, 1)
	svc := services[0]
	assert.Equal(t, "productpage", svc.Name)
	assert.Equal(t, "default", svc.Namespace)
	assert.Equal(t, types.SourceVirtualService, svc.Source)

	require.Len(t, svc.APIs, 1)
	api := svc.APIs[0]
	assert.Equal(t, "GET", api.Method)
	assert.Equal(t, "/productpage", api.Path)
	assert.Equal(t, types.MatchTypeExact, api.MatchType)
}

func TestParseVirtualService_PrefixMatch(t *testing.T) {
	vs := createFakeVirtualService("reviews-vs", "default", []*networkingv1beta1.HTTPRoute{
		createHTTPRoute(
			[]*networkingv1beta1.HTTPMatchRequest{
				createMatchRequest("", &networkingv1beta1.StringMatch{
					MatchType: &networkingv1beta1.StringMatch_Prefix{Prefix: "/api/v1/reviews"},
				}),
			},
			[]*networkingv1beta1.HTTPRouteDestination{
				createDestination("reviews", 9080),
			},
		),
	})

	fakeClient := istiofake.NewSimpleClientset()
	discoverer := discovery.NewKubernetesDiscovererWithClient(fakeClient, nil)

	services := discoverer.ParseVirtualService(vs)

	require.Len(t, services, 1)
	svc := services[0]
	assert.Equal(t, "reviews", svc.Name)

	require.Len(t, svc.APIs, 1)
	api := svc.APIs[0]
	assert.Equal(t, types.MethodAll, api.Method) // 无方法指定，默认为 *
	assert.Equal(t, "/api/v1/reviews", api.Path)
	assert.Equal(t, types.MatchTypePrefix, api.MatchType)
}

func TestParseVirtualService_RegexMatch(t *testing.T) {
	vs := createFakeVirtualService("ratings-vs", "default", []*networkingv1beta1.HTTPRoute{
		createHTTPRoute(
			[]*networkingv1beta1.HTTPMatchRequest{
				createMatchRequest("GET", &networkingv1beta1.StringMatch{
					MatchType: &networkingv1beta1.StringMatch_Regex{Regex: "/ratings/[0-9]+"},
				}),
			},
			[]*networkingv1beta1.HTTPRouteDestination{
				createDestination("ratings", 9080),
			},
		),
	})

	fakeClient := istiofake.NewSimpleClientset()
	discoverer := discovery.NewKubernetesDiscovererWithClient(fakeClient, nil)

	services := discoverer.ParseVirtualService(vs)

	require.Len(t, services, 1)
	require.Len(t, services[0].APIs, 1)
	api := services[0].APIs[0]
	assert.Equal(t, "/ratings/[0-9]+", api.Path)
	assert.Equal(t, types.MatchTypeRegex, api.MatchType)
}

func TestParseVirtualService_MultipleRoutes(t *testing.T) {
	vs := createFakeVirtualService("multi-route", "default", []*networkingv1beta1.HTTPRoute{
		createHTTPRoute(
			[]*networkingv1beta1.HTTPMatchRequest{
				createMatchRequest("GET", &networkingv1beta1.StringMatch{
					MatchType: &networkingv1beta1.StringMatch_Exact{Exact: "/products"},
				}),
			},
			[]*networkingv1beta1.HTTPRouteDestination{
				createDestination("product-service", 8080),
			},
		),
		createHTTPRoute(
			[]*networkingv1beta1.HTTPMatchRequest{
				createMatchRequest("POST", &networkingv1beta1.StringMatch{
					MatchType: &networkingv1beta1.StringMatch_Exact{Exact: "/products"},
				}),
			},
			[]*networkingv1beta1.HTTPRouteDestination{
				createDestination("product-service", 8080),
			},
		),
	})

	fakeClient := istiofake.NewSimpleClientset()
	discoverer := discovery.NewKubernetesDiscovererWithClient(fakeClient, nil)

	services := discoverer.ParseVirtualService(vs)

	// 同一服务应该聚合
	require.Len(t, services, 1)
	svc := services[0]
	assert.Equal(t, "product-service", svc.Name)
	assert.Len(t, svc.APIs, 2) // 两个 API 端点
}

func TestParseVirtualService_NoMatch(t *testing.T) {
	// 没有 match 但有 route 的情况
	vs := createFakeVirtualService("no-match", "default", []*networkingv1beta1.HTTPRoute{
		{
			Route: []*networkingv1beta1.HTTPRouteDestination{
				createDestination("default-service", 8080),
			},
		},
	})

	fakeClient := istiofake.NewSimpleClientset()
	discoverer := discovery.NewKubernetesDiscovererWithClient(fakeClient, nil)

	services := discoverer.ParseVirtualService(vs)

	// 应该仍然能发现服务，API 路径为空或通配符
	require.Len(t, services, 1)
	assert.Equal(t, "default-service", services[0].Name)
}

func TestParseVirtualService_MultipleDestinations(t *testing.T) {
	// 多个目标服务（如金丝雀部署）
	vs := createFakeVirtualService("canary", "default", []*networkingv1beta1.HTTPRoute{
		{
			Match: []*networkingv1beta1.HTTPMatchRequest{
				createMatchRequest("GET", &networkingv1beta1.StringMatch{
					MatchType: &networkingv1beta1.StringMatch_Prefix{Prefix: "/api"},
				}),
			},
			Route: []*networkingv1beta1.HTTPRouteDestination{
				{
					Destination: &networkingv1beta1.Destination{
						Host:   "service-v1",
						Subset: "stable",
					},
					Weight: 90,
				},
				{
					Destination: &networkingv1beta1.Destination{
						Host:   "service-v2",
						Subset: "canary",
					},
					Weight: 10,
				},
			},
		},
	})

	fakeClient := istiofake.NewSimpleClientset()
	discoverer := discovery.NewKubernetesDiscovererWithClient(fakeClient, nil)

	services := discoverer.ParseVirtualService(vs)

	// 应该发现两个服务
	assert.Len(t, services, 2)

	serviceNames := make(map[string]bool)
	for _, svc := range services {
		serviceNames[svc.Name] = true
	}
	assert.True(t, serviceNames["service-v1"])
	assert.True(t, serviceNames["service-v2"])
}

func TestAggregateServices(t *testing.T) {
	ctx := context.Background()

	// 创建多个 VirtualService
	vs1 := createFakeVirtualService("vs1", "default", []*networkingv1beta1.HTTPRoute{
		createHTTPRoute(
			[]*networkingv1beta1.HTTPMatchRequest{
				createMatchRequest("GET", &networkingv1beta1.StringMatch{
					MatchType: &networkingv1beta1.StringMatch_Exact{Exact: "/api/products"},
				}),
			},
			[]*networkingv1beta1.HTTPRouteDestination{
				createDestination("product-service", 8080),
			},
		),
	})

	vs2 := createFakeVirtualService("vs2", "default", []*networkingv1beta1.HTTPRoute{
		createHTTPRoute(
			[]*networkingv1beta1.HTTPMatchRequest{
				createMatchRequest("GET", &networkingv1beta1.StringMatch{
					MatchType: &networkingv1beta1.StringMatch_Exact{Exact: "/api/orders"},
				}),
			},
			[]*networkingv1beta1.HTTPRouteDestination{
				createDestination("order-service", 8080),
			},
		),
	})

	fakeClient := istiofake.NewSimpleClientset(vs1, vs2)
	discoverer := discovery.NewKubernetesDiscovererWithClient(fakeClient, nil)

	services, err := discoverer.AggregateServices(ctx, "")

	require.NoError(t, err)
	assert.Len(t, services, 2)

	// 验证服务名存在
	serviceMap := make(map[string]types.ServiceInfo)
	for _, svc := range services {
		serviceMap[svc.Name] = svc
	}

	assert.Contains(t, serviceMap, "product-service")
	assert.Contains(t, serviceMap, "order-service")
}

func TestAggregateServices_MergeAPIs(t *testing.T) {
	ctx := context.Background()

	// 两个 VirtualService 指向同一个服务
	vs1 := createFakeVirtualService("vs1", "default", []*networkingv1beta1.HTTPRoute{
		createHTTPRoute(
			[]*networkingv1beta1.HTTPMatchRequest{
				createMatchRequest("GET", &networkingv1beta1.StringMatch{
					MatchType: &networkingv1beta1.StringMatch_Exact{Exact: "/api/v1/users"},
				}),
			},
			[]*networkingv1beta1.HTTPRouteDestination{
				createDestination("user-service", 8080),
			},
		),
	})

	vs2 := createFakeVirtualService("vs2", "default", []*networkingv1beta1.HTTPRoute{
		createHTTPRoute(
			[]*networkingv1beta1.HTTPMatchRequest{
				createMatchRequest("POST", &networkingv1beta1.StringMatch{
					MatchType: &networkingv1beta1.StringMatch_Exact{Exact: "/api/v1/users"},
				}),
			},
			[]*networkingv1beta1.HTTPRouteDestination{
				createDestination("user-service", 8080),
			},
		),
	})

	fakeClient := istiofake.NewSimpleClientset(vs1, vs2)
	discoverer := discovery.NewKubernetesDiscovererWithClient(fakeClient, nil)

	services, err := discoverer.AggregateServices(ctx, "")

	require.NoError(t, err)
	// 同一服务应该被合并
	require.Len(t, services, 1)
	assert.Equal(t, "user-service", services[0].Name)
	// API 应该被合并
	assert.Len(t, services[0].APIs, 2)
}

func TestExtractHTTPMethods(t *testing.T) {
	tests := []struct {
		name     string
		method   *networkingv1beta1.StringMatch
		expected string
	}{
		{
			name:     "nil method returns *",
			method:   nil,
			expected: types.MethodAll,
		},
		{
			name: "exact GET",
			method: &networkingv1beta1.StringMatch{
				MatchType: &networkingv1beta1.StringMatch_Exact{Exact: "GET"},
			},
			expected: "GET",
		},
		{
			name: "exact POST",
			method: &networkingv1beta1.StringMatch{
				MatchType: &networkingv1beta1.StringMatch_Exact{Exact: "POST"},
			},
			expected: "POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := discovery.ExtractHTTPMethod(tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseVirtualService_EmptyHTTPRoutes(t *testing.T) {
	vs := createFakeVirtualService("empty", "default", nil)

	fakeClient := istiofake.NewSimpleClientset()
	discoverer := discovery.NewKubernetesDiscovererWithClient(fakeClient, nil)

	services := discoverer.ParseVirtualService(vs)

	assert.Empty(t, services)
}

func TestParseVirtualService_EmptyDestinations(t *testing.T) {
	vs := createFakeVirtualService("no-dest", "default", []*networkingv1beta1.HTTPRoute{
		{
			Match: []*networkingv1beta1.HTTPMatchRequest{
				createMatchRequest("GET", &networkingv1beta1.StringMatch{
					MatchType: &networkingv1beta1.StringMatch_Exact{Exact: "/api"},
				}),
			},
			Route: nil, // 空目标
		},
	})

	fakeClient := istiofake.NewSimpleClientset()
	discoverer := discovery.NewKubernetesDiscovererWithClient(fakeClient, nil)

	services := discoverer.ParseVirtualService(vs)

	assert.Empty(t, services)
}
