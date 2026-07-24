# NekoSpeed 问题排查记录

## 案例：anytls 订阅本地测试全部失败（丢包 100%、速度为 0），VPS 正常

### 现象

- 订阅：`https://ding.81dlg.com/dongligang/...`（动力港，42 个 anytls 节点）
- **VPS（境外）**：42/42 成功，速度/延迟/出口全部正常
- **本地 Mac（中国移动网络）**：所有节点 `packetLoss=1`、`avgHTTPS=0`、速度全 0、`outbound` 为空，但显示"成功"
- 同一时刻本地 Clash Verge 使用同一订阅也不通（502）

### 根因

**机场使用 DDNS 高频轮换服务器 IP 抗封锁，IP 池中部分旧 IP 已被 GFW 封死。本地测试成败取决于"解析那一刻轮到的 IP 是否还活着"。**

证据链：

1. 节点配置正常：`type: anytls`、`sni: jd.com`、`skip-cert-verify: true`，mihomo **v1.19.27+** 已支持 anytls（v1.19.29 验证可用）
2. 失败时临时加调试日志可见真实错误：
   ```
   failed to create session: dial tcp 168.107.10.200:51401: operation was canceled
   ```
3. `nc`/`ping` 验证该 IP **全端口 + ICMP 均不可达**；`traceroute` 死在中国移动国际出口（`223.120.x`）—— 典型 GFW 国际出口封 IP
4. 域名是 DDNS 链：`dlg-hk.dlg365.com → trans8888-all.trans8888.com → ddnsXXX.woaitutu520.com → A 记录`，**TTL 仅 5 秒**，IP 在 Oracle Cloud 池内不断更换
5. Go 解析器复现：多次查询返回死/活 IP 混合列表，mihomo `ResolveIPv4` 从列表中**随机选取**（`randv2.IntN`），选中死 IP 即全挂
6. IP 轮换到活 IP（如 `168.107.53.107`）后立即重测 → 延迟 172ms、速度 15.6 MB/s、出口香港，**完全正常**
7. VPS 从不失败：境外无封锁，池中所有 IP 均可达

### 关键排查手法

- `inbound`（入口 IP）**只是本地 DNS 解析 + 直连查询**，不经过代理，有值不代表代理工作；判断代理是否真通看 **`outbound`（出口 IP）是否为空**（`internal/geo/geoip.go:184-200`，outbound 失败静默返回 nil）
- NekoSpeed 对拨号错误完全静默：`latency.go` 只计数丢包、`speed.go` 直接吞错、`geoip.go` outbound 失败无输出。排查时需临时在 `internal/nodepool/pool.go` 的 `proxyDialer.DialContext` 加 `fmt.Fprintf(os.Stderr, ...)` 打印真实错误
- 本机若开着 Clash Verge 等 TUN 代理先排除干扰（本次 TUN 实际未接管路由，utun 无地址无路由表项，`route -n get <ip>` 确认走物理网卡）

### 解决方案

| 方案 | 说明 |
|---|---|
| **直接重跑** | 最简单。TTL 5 秒，重跑时大概率换到活 IP（本次就是这么通的） |
| 临时钉 hosts | `sudo sh -c 'echo "168.107.53.107 dlg-hk.dlg365.com" >> /etc/hosts'`，用完即删（IP 随时失效） |
| 用 VPS 测 | 境外网络无封锁，结果稳定，推荐长期使用 |
| 向机场反馈 | IP 池中被封 IP 占比高（本例 Oracle 韩国段被移动方向封锁较多） |

### 代码层面待改进（可选）

1. **错误透传**：把 ping/download 的真实错误记入 `NodeResult.Err` 或日志（如首 N 条），避免"成功 1/1 但数据全 0"的误导
2. **代理可用性标记**：outbound 查询失败时在 JSON/图中标记 `proxyAlive: false`
3. **anytls:// 分享链接解析**：`internal/subscription/sharelink.go` 的 `shareSchemes` 未包含 anytls，目前仅支持 Clash YAML 形式的 anytls 节点
4. `typeDisplayNames` 缺 anytls 映射，节点类型列显示为小写 `anytls`（仅美观问题）
