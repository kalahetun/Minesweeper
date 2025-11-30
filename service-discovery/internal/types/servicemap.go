package types

// Package types 定义 Service Discovery 的核心数据类型

import (
	"encoding/json"
	"time"
)

// ServiceMap 服务地图 - 顶层结构
// 包含服务列表、API 端点和服务级调用拓扑
type ServiceMap struct {
	// Timestamp 发现时间戳 (RFC3339 格式)
	Timestamp time.Time `json:"timestamp"`

	// Services 服务列表及其 API 端点
	// Key: 服务名 (如 "productpage", "reviews")
	Services map[string]ServiceInfo `json:"services"`

	// Topology 服务级调用拓扑 (有向边列表)
	Topology []ServiceEdge `json:"topology"`

	// Metadata 元数据
	Metadata MapMetadata `json:"metadata"`
}

// ServiceInfo 单个服务的信息
type ServiceInfo struct {
	// Name 服务名称
	Name string `json:"name"`

	// Namespace 服务所在命名空间
	Namespace string `json:"namespace"`

	// APIs API 端点列表 (从 VirtualService 提取)
	APIs []APIEndpoint `json:"apis"`

	// Source 数据来源: "virtualservice", "openapi", "merged"
	Source string `json:"source"`
}

// APIEndpoint 单个 API 端点
type APIEndpoint struct {
	// Method HTTP 方法 (GET, POST, PUT, DELETE, PATCH, *)
	// * 表示所有方法
	Method string `json:"method"`

	// Path 路径模式
	Path string `json:"path"`

	// MatchType 匹配类型: "exact", "prefix", "regex"
	MatchType string `json:"match_type"`
}

// ServiceEdge 服务间调用关系
type ServiceEdge struct {
	// Source 调用方服务名
	Source string `json:"source"`

	// Target 被调用方服务名
	Target string `json:"target"`

	// CallCount 统计周期内的调用次数
	CallCount int `json:"call_count"`
}

// MapMetadata 服务地图元数据
type MapMetadata struct {
	// DiscoveryInterval 发现周期配置
	DiscoveryInterval string `json:"discovery_interval"`

	// JaegerLookback Jaeger 查询时间范围
	JaegerLookback string `json:"jaeger_lookback"`

	// Stale 数据是否过期（来自缓存）
	Stale bool `json:"stale"`

	// Errors 发现过程中的错误（如有）
	Errors []string `json:"errors,omitempty"`
}

// MatchType 常量
const (
	MatchTypeExact  = "exact"
	MatchTypePrefix = "prefix"
	MatchTypeRegex  = "regex"
)

// Source 常量
const (
	SourceVirtualService = "virtualservice"
	SourceOpenAPI        = "openapi"
	SourceMerged         = "merged"
)

// HTTP 方法常量
const (
	MethodAll     = "*"
	MethodGet     = "GET"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodDelete  = "DELETE"
	MethodPatch   = "PATCH"
	MethodHead    = "HEAD"
	MethodOptions = "OPTIONS"
)

// NewServiceMap 创建一个新的空 ServiceMap
func NewServiceMap() *ServiceMap {
	return &ServiceMap{
		Timestamp: time.Now(),
		Services:  make(map[string]ServiceInfo),
		Topology:  []ServiceEdge{},
		Metadata:  MapMetadata{},
	}
}

// AddService 添加服务信息
func (sm *ServiceMap) AddService(info ServiceInfo) {
	sm.Services[info.Name] = info
}

// AddEdge 添加调用关系边
func (sm *ServiceMap) AddEdge(edge ServiceEdge) {
	sm.Topology = append(sm.Topology, edge)
}

// AddError 添加错误信息
func (sm *ServiceMap) AddError(err string) {
	sm.Metadata.Errors = append(sm.Metadata.Errors, err)
}

// SetStale 标记数据为过期
func (sm *ServiceMap) SetStale(stale bool) {
	sm.Metadata.Stale = stale
}

// ToJSON 序列化为 JSON
func (sm *ServiceMap) ToJSON() ([]byte, error) {
	return json.Marshal(sm)
}

// ToJSONIndent 序列化为格式化的 JSON
func (sm *ServiceMap) ToJSONIndent() ([]byte, error) {
	return json.MarshalIndent(sm, "", "  ")
}

// FromJSON 从 JSON 反序列化
func FromJSON(data []byte) (*ServiceMap, error) {
	var sm ServiceMap
	if err := json.Unmarshal(data, &sm); err != nil {
		return nil, err
	}
	return &sm, nil
}

// ServiceCount 返回服务数量
func (sm *ServiceMap) ServiceCount() int {
	return len(sm.Services)
}

// EdgeCount 返回拓扑边数量
func (sm *ServiceMap) EdgeCount() int {
	return len(sm.Topology)
}

// HasErrors 检查是否有错误
func (sm *ServiceMap) HasErrors() bool {
	return len(sm.Metadata.Errors) > 0
}
