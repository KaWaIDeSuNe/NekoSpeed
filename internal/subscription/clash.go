package subscription

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"nekospeed/internal/model"
)

// nonProxyTypes 是 Clash 配置里不代表代理节点的 type 值，解析时跳过。
var nonProxyTypes = map[string]bool{
	"direct": true, "reject": true, "rejectdrop": true,
	"dns": true, "compatible": true, "pass": true,
}

// parseClash 尝试把 data 按 Clash 配置解析。
// ok 为 false 表示内容不是含顶层 proxies: 键的 YAML（应继续尝试其他格式）；
// ok 为 true 时返回解析出的节点（err 非空表示 YAML 合法但 proxies 结构损坏）。
func parseClash(data []byte) (nodes []model.Node, ok bool, err error) {
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		return nil, false, nil // 不是 YAML，交由其他格式处理
	}
	raw, exists := top["proxies"]
	if !exists {
		return nil, false, nil
	}
	list, isList := raw.([]any)
	if !isList {
		return nil, true, fmt.Errorf("Clash 配置的 proxies 字段不是列表")
	}

	for _, item := range list {
		m, isMap := item.(map[string]any)
		if !isMap {
			continue
		}
		node, keep := clashNode(m, len(nodes)+1)
		if keep {
			nodes = append(nodes, node)
		}
	}
	return nodes, true, nil
}

// clashNode 把 Clash 单节点 map 转为 model.Node；keep 为 false 表示该条应跳过（DIRECT/REJECT 等）。
func clashNode(m map[string]any, index int) (model.Node, bool) {
	typ, _ := m["type"].(string)
	typ = strings.ToLower(strings.TrimSpace(typ))
	if typ == "" || nonProxyTypes[typ] {
		return model.Node{}, false
	}
	name, _ := m["name"].(string)
	server, _ := m["server"].(string)
	if name == "" {
		name = server
	}
	// Raw 直接复用原 map，字段原样交给 mihomo adapter.ParseProxy
	return model.Node{
		Index:  index,
		Name:   name,
		Type:   displayType(typ),
		Server: server,
		Raw:    m,
	}, true
}

// displayType 把 clash 协议 type 转成展示名。
func displayType(typ string) string {
	if d, ok := typeDisplayNames[typ]; ok {
		return d
	}
	return typ
}
