# HyperData Terminal 侧车

**语言：** [English](README.md) | [中文](README.zh-CN.md)

`hyperdata_orderflow` 与 `hyperdata_positioning` 两个市场数据源，读取
[HyperData Terminal](https://github.com/Co-Messi/HyperData-Terminal) —— 一个独立开源项目，
从五家交易所聚合订单流、大户持仓与强平数据。

## 为什么它是进程，而不是库

HyperData Terminal 是一个 Python **应用**（TUI + REST API + 纸交易引擎）。不存在可 import
的 Go 包，因此没有可链接的对象。后端于是**通过 HTTP 消费它的 REST API**。

这条边界是刻意的，也正是「该库需要支持后续更新」这个要求能真正落地的前提：

| 关注点 | 处理方式 |
|---|---|
| 上游发布新版本 | 移动 `upstream.env` 里的版本钉；Go 代码树完全不动 |
| 上游改名或新增字段 | 适配层忽略未知字段、容忍宽松类型，所以新增字段永远安全 |
| 上游改变了字段含义 | `sidecar.sh contract` 断言每个 provider 实际读取的字段 |
| 上游破坏 API 形状 | provider 软失败；提示词里多出一个 `data_coverage` 区块点名它，而不是静默丢掉数据 |
| 侧车没运行 | provider 被跳过，交易继续，缺口被报告出来 |
| 上游主版本变化 | `VersionWarning` 记录到日志，数据块里也会渲染一行警告 |

上游**不发布 git tag**（2026-09-22 核实：tags 接口为空），所以版本钉是 commit SHA。
这是唯一可复现的选择。

## 目录结构

```
upstream.env   钉住的 revision、实测主版本与默认值（唯一真源）
sidecar.sh     install / update / status / contract / run
Dockerfile     按版本钉构建侧车镜像，且拒绝构建偏离 pin 的代码
README.md      本文档
```

这里**刻意没有单独的探测脚本**。活体兼容性检查是一个 Go 测试
（`marketdata/providers/hyperdata_upstream_test.go`），目的是让契约只有一份定义而不是两份：
它复用适配层自己的 HTTP 客户端、同一批录制的 fixture、同一套形状规则。第二个脚本里手工维护的
字段清单会与 fixture 各自漂移，然后「通过」的原因就不再正确了。

`docker-compose.hyperdata.yml`（仓库根目录）把它和后端组在一起。

## 安装

```bash
./deploy/hyperdata/sidecar.sh install
./deploy/hyperdata/sidecar.sh run       # 前台运行；生产环境请用进程管理器
```

代码检出与虚拟环境放在 `$HYPERDATA_DIR`（默认 `~/.nofx/hyperdata`），**永不放进本仓库**，
因此安装或升级都不会弄脏工作区。

让后端指向它，两种方式：

```bash
# 全局，所有策略继承：
HYPERDATA_BASE_URL=http://127.0.0.1:8420
HYPERDATA_API_KEY=<仅当侧车启动时设置了密钥>
```

或者在策略工作室 → 市场数据源 → *启用 HyperData Terminal 侧车* 里按策略配置。
策略级的值优先于环境变量，因此两者可以共存。

## 校验

```bash
./deploy/hyperdata/sidecar.sh contract
```

它运行两半：

1. **录制的契约（离线）**：`marketdata/providers/hyperdata_contract_test.go` 里的 Go fixture
   测试钉住适配层构建时对应的响应形状，包含各种「可容忍漂移」用例。
2. **活体契约**：`TestHyperDataUpstreamHasNotDrifted` 探测运行中的实例，并把每处与 fixture
   的差异分类：

   ```
   MISSING       本适配层要读的字段没了。破坏性。
   TYPE CHANGED  字段还在，但 JSON 类型变了。破坏性。
   ADDED         上游新增了我们未使用的字段。去看 release notes。
   ```

   它由 `HYPERDATA_LIVE_URL` 控制、默认跳过，以保证常规测试套件不依赖外部进程；
   `sidecar.sh contract` 会替你设置这个变量。侧车没运行时，活体这一半会被跳过并给出说明，
   而不是报成失败。

两者都重要，且互不替代。fixture 测试回答的是「我们有没有弄坏自己的解析」，只有 Go 代码变化时
才会失败；漂移报告回答的是「这次升级改了什么」，是唯一能发现真实上游变动的检查。

但它**看不到什么**也要清楚，因为它比对的是**形状**：

- **名字和类型都没变、但含义变了的字段。** `open_interest` 从 USD 改成张数，或者某个比率从
  小数改成百分数，在这里是看不见的，而它会把提示词里的数值悄悄换掉一个量级。只有 release
  notes 能抓到。
- **被折叠掉的容器内的键改名。** 这些键本来就是动态的（venue / symbol / timeframe），所以通常
  无所谓——但只含单个对象的固定包裹 map 也会被折叠，它的名字被改掉就不会有提示。

普通字段改名**是**会被报出来的：旧名字报 `MISSING`，新名字报 `ADDED`。报告缩小了复核范围，
但不替代复核。

### 版本钉靠什么保持诚实

`upstream.env` 不是文档 —— 它被 `TestUpstreamPinIsStillMeaningful` 读取，三处不一致就会让
构建失败：

| 位置 | 内容 |
|---|---|
| `deploy/hyperdata/upstream.env` | `HYPERDATA_REVISION`（必须是完整 40 位 SHA）与 `HYPERDATA_TESTED_MAJOR` |
| `marketdata/providers/hyperdata_client.go` | `hyperDataTestedMajor` |
| `marketdata/providers/hyperdata_contract_test.go` | `fixtureHealth` 里的 `version` |

由于上游不发布 tag，revision 必须是 commit 才能保证构建可复现；测试会拒绝其他形式。
只改一半 —— 移动了 pin，却漏掉常量或 fixture —— 会明确失败，而不会悄悄通过。

## 升级

```bash
./deploy/hyperdata/sidecar.sh update --to main   # 或 tag，或 commit SHA
./deploy/hyperdata/sidecar.sh contract
# 通过后，把新的 SHA 与主版本写入 upstream.env
```

`update` 不会替你改 `upstream.env`：记录「实际验证过什么」应该是一个刻意的动作，
而不是副作用。

如果漂移报告列出了破坏性变更，那么变的是上游，不是适配层。可选处理，按优先级：

1. **适配** —— 把字段加进容错类型或映射，扩展契约测试里的 fixture，然后重跑。
   记住 fixture 正是漂移报告比对的对象，所以刷新 fixture 就是记录新契约的动作。
2. **回退版本钉** —— 让 `HYPERDATA_REVISION` 留在原处；旧版本依然可用，因为其他都没变。
3. **关闭** —— 策略里设 `enable_hyperdata: false`。市场情报层的其余部分不受影响。

## 用 Docker 部署

```bash
# compose 文件里没有重复版本钉，所以要 source 它。
set -a; . deploy/hyperdata/upstream.env; set +a

HYPERDATA_API_KEY=$(openssl rand -hex 24) \
  docker compose -f docker-compose.yml -f docker-compose.hyperdata.yml \
  --profile hyperdata up -d --build
```

然后把 base URL 设为 `http://hyperdata:8420`，并填上这个密钥。

`HYPERDATA_REVISION` 或 `HYPERDATA_API_KEY` 未设置时 `docker compose` 会立刻失败。
这是刻意的：没有版本钉，镜像就会按 `main` 当时的代码构建；没有密钥，上游会拒绝绑定
非 loopback 地址，容器会跑起来但什么都答不上。

该 API 提供基于钱包推断的持仓、强平危险区与实时订单流。上游默认绑定 loopback，且在没有密钥时
拒绝非 loopback 绑定；overlay 提供了密钥，并且**不发布端口**，因此容器网络是唯一入口。
不要为它加 `ports:` 映射——要从宿主机查看，用：

```bash
docker compose exec hyperdata curl -s localhost:8420/v1/health
```

## 数据本身的已知限制

以下是上游自己声明的注意事项，提示词里也写明了，以免模型从这些数字里读出超过其支撑能力的结论：

- **大户持仓是抽样的。** 上游只扫描它已发现、且在每轮预算内的地址。这不是全市场普查。
- **Hyperliquid 的强平是从大额成交推断的**，而 Binance 的强平流在源头被限流到大约每 symbol
  每秒一条。这不是 Coinglass 级别的覆盖。
- **部分地区的 Binance Futures 流被地理封锁。** socket 会连上但不推数据；上游把该 venue 标为
  `partial` 而不是隐藏它，数据块也会告诉模型「单 venue 的 delta 是更弱的证据」。
- **订单流是逐 symbol 的**，上游只拥有其实例正在追踪的 symbol 的数据。未追踪的 symbol 会被
  明确列为未追踪。
