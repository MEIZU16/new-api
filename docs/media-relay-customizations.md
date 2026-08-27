# 媒体模型中转与 OpenAI Images 兼容层

本文记录 `saki-media-docs` 分支为图片和视频模型增加的中转、参数约束、计费、重试和用户文档行为。它面向维护者和 API 集成方，用于说明本分支相对通用中转逻辑的定制内容。

## 目标与范围

目标生产部署将 New API 作为统一的媒体 API 网关，对用户提供注册、鉴权、额度、充值、限流、日志以及图片和视频生成能力。

目标生产部署只公开以下 8 个模型：

```text
gemini-3-pro-image
gemini-3-pro-image-2k
gemini-3-pro-image-4k
gemini-3.1-flash-image
gemini-3.1-flash-image-2k
gemini-3.1-flash-image-4k
omni-flash
gpt-image-2
```

上述 8 模型白名单和“不提供 Chat”属于**部署配置要求**，由渠道模型列表、路由、模型权限和公开模型元数据共同落实，不是仓库源码对所有部署强制执行的全局不变量。通用 New API 路由仍包含其他 API 类型；运维人员必须在目标环境中只启用上述媒体能力。

## 公共 API 契约

### Gemini 图片模型

6 个 Gemini 图片 SKU 均使用 OpenAI Images 风格接口：

```text
POST /v1/images/generations
POST /v1/images/edits
```

用户只需要使用 OpenAI Images 请求格式。New API 会在渠道中转阶段完成上游图片协议转换；上游专用请求结构不是公共 API 的一部分，也不应出现在用户示例中。

### GPT Image 2

`gpt-image-2` 使用同样的公共路径：

```text
POST /v1/images/generations
POST /v1/images/edits
```

其 generation 请求只公开 `model` 和 `prompt`。edit 请求使用 `multipart/form-data`，公开 `model`、`prompt` 和 `image`。

该模型通过 OpenAI Images 兼容渠道直接中转，不经过 Gemini 图片转换器。

### 视频模型

`omni-flash` 使用服务端任务接口：

```text
POST /v1/videos
GET  /v1/videos/{task_id}
GET  /v1/videos/{task_id}/content
```

创建请求接受 `application/json` 和 `multipart/form-data` 两种形态；上传参考图必须使用 `multipart/form-data`。`seconds` 只接受 `4`、`6`、`8`、`10`；`size` 只接受 `1280x720` 和 `720x1280`，省略时按 `720x1280` 计费和生成。

视频内容必须通过 New API 的 `/content` 接口流式代理。创建和轮询响应通过公共响应类型重建，只保留任务 ID、公开模型名、状态、进度、时间、时长、尺寸和错误等公共字段；上游下载地址、签名地址、内部任务 ID 及供应商元数据不得透传给客户端。

### 视频生成模式与参考图

`omni-flash` 支持四种生成模式。模式由参考图数量推断，也可以用 `mode`（`t2v` / `i2v` / `r2v`）或 `operation`（`text_to_video` / `image_to_video` / `start_end_frame` / `reference_to_video`）显式指定。两者只能选其一，同时提供返回 HTTP 400。

| 模式 | 参考图数量 | 显式取值 |
| --- | --- | --- |
| 文生视频 | 0 | `t2v` / `text_to_video` |
| 首帧生视频 | 1 | `i2v` / `image_to_video` |
| 首尾帧生视频 | 2 | `start_end_frame` |
| 多参考生视频 | 1～7 | `r2v` / `reference_to_video` |

两张参考图默认按首尾帧解释；要把两张图当作参考图集合，必须显式指定 `r2v`。

#### 文生视频（t2v）

不带参考图，只发送 `prompt`。没有参考图时模式即推断为文生视频，`mode` 可以省略：

```bash
curl https://example.com/v1/videos \
  -H "Authorization: Bearer $NEW_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "omni-flash",
    "prompt": "A paper boat sailing through a neon city in the rain.",
    "seconds": "6",
    "size": "1280x720"
  }'
```

#### 图生视频（i2v）

用 `multipart/form-data` 上传一张 `input_reference`，视频以该画面作为首帧：

```bash
curl https://example.com/v1/videos \
  -H "Authorization: Bearer $NEW_API_KEY" \
  -F "model=omni-flash" \
  -F "prompt=Animate the reference image with a slow push-in as rain starts." \
  -F "seconds=6" \
  -F "size=1280x720" \
  -F "mode=i2v" \
  -F "input_reference=@reference.png"
```

#### 多参考生视频（r2v）

参考图可以用重复的 `input_reference` 字段上传，顺序即上游读取顺序：

```bash
curl https://example.com/v1/videos \
  -H "Authorization: Bearer $NEW_API_KEY" \
  -F "model=omni-flash" \
  -F "prompt=Combine the referenced subjects into one scene." \
  -F "seconds=4" \
  -F "size=1280x720" \
  -F "mode=r2v" \
  -F "input_reference=@reference-1.png" \
  -F "input_reference=@reference-2.png" \
  -F "input_reference=@reference-3.png"
```

也可以传 URL。`input_reference` 接受裸 URL 字符串或 `{"image_url": ...}` 对象，`reference_images` 接受两种形态组成的数组：

```bash
curl https://example.com/v1/videos \
  -H "Authorization: Bearer $NEW_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "omni-flash",
    "prompt": "Combine the referenced subjects into one scene.",
    "seconds": "4",
    "size": "1280x720",
    "reference_images": [
      {"image_url": "https://example.com/a.png"},
      {"image_url": "https://example.com/b.png"},
      "https://example.com/c.png"
    ]
  }'
```

首尾帧另有一种 metadata 写法，`first_frame_url` 与 `last_frame_url` 必须成对出现，缺一返回 HTTP 400：

```bash
-F 'metadata={"first_frame_url":"https://example.com/a.png","last_frame_url":"https://example.com/b.png"}'
```

参考图字节限制为单张 32 MiB、上传合计 64 MiB。参考图数量与模式不匹配（例如 `i2v` 传两张图）在上游调用阶段返回 HTTP 400，不会静默丢弃多余的参考图。使用日志中的任务动作按实际模式记录，上传参考图的请求不再记成文生视频。

## 图片生成请求

### Gemini 图片生成

```bash
curl https://example.com/v1/images/generations \
  -H "Authorization: Bearer $NEW_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-3-pro-image-2k",
    "prompt": "A calm lake surrounded by pine trees at sunrise.",
    "extra_fields": {
      "aspect_ratio": "16:9"
    }
  }'
```

用户可以使用 `extra_fields.aspect_ratio` 指定画幅。用户不能通过 `size`、`quality` 或其他字段改变公开 SKU 锁定的分辨率档位。

### GPT Image 2 图片生成

```bash
curl https://example.com/v1/images/generations \
  -H "Authorization: Bearer $NEW_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2",
    "prompt": "A serene koi pond at sunset, rendered as a woodblock print."
  }'
```

`gpt-image-2` generation 文档不公开 `size`、`quality`、`n`、`response_format` 或 Gemini 图片参数。

## 参考图编辑

参考图编辑必须使用 `multipart/form-data`。

### 单张参考图

```bash
curl https://example.com/v1/images/edits \
  -H "Authorization: Bearer $NEW_API_KEY" \
  -F "model=gemini-3.1-flash-image" \
  -F "prompt=Repaint the reference as a watercolor illustration." \
  -F "image=@reference.png"
```

### 多张参考图

推荐通过重复的 `image` 字段上传多张图片：

```bash
curl https://example.com/v1/images/edits \
  -H "Authorization: Bearer $NEW_API_KEY" \
  -F "model=gemini-3.1-flash-image" \
  -F "prompt=Combine the composition and visual elements from all references." \
  -F 'extra_fields={"aspect_ratio":"1:1"}' \
  -F "image=@reference-1.png" \
  -F "image=@reference-2.png" \
  -F "image=@reference-3.png" \
  -F "image=@reference-4.png" \
  -F "image=@reference-5.png"
```

转换器同时接受 `image[]` 和索引形式的图片字段，但新客户端应使用重复的 `image`。重复字段能保持同名文件的上传顺序；索引字段当前按字段名排序，`image[10]` 与 `image[2]` 等名称不能保证数值顺序，也不应与其他命名形式混用来表达有序参考图。

### 参考图数量策略

New API 当前**没有写死的参考图张数上限**。参考图数量仍会受到 multipart 解析、HTTP 请求和部署平台资源限制，因此这不等于无限制上传。

为约束单请求内存和传输体积，仍执行以下字节限制：

```text
单张参考图最大：16 MiB
所有参考图合计最大：32 MiB
```

曾经加入的 `maxGeminiReferenceImages = 4` 是网关侧人为限制，不是 Gemini 协议限制，现已删除。输入参考图数量与输出图片数量 `n` 是两个独立概念。

### 文件类型验证

Gemini 图片转换器只接受：

```text
PNG
JPEG
WebP
```

文件类型根据实际文件字节检测，不信任客户端提供的 `Content-Type` 或文件扩展名。伪造类型会返回客户端错误且不发送备用渠道。

`mask` 当前不属于 Gemini 图片编辑公共契约；包含 `mask` 的请求会被拒绝。

## 图片输出数量

图片生成和编辑的输出数量规则为：

```text
n 省略或为 0：按 1 张处理
n 为 1～4：接受
n 大于 4：HTTP 400
```

`n > 4` 在调用上游前被拒绝，不产生图片费用。

此处的 4 张只约束**输出图片数量**，不约束输入参考图数量。

Gemini 图片转换路径会记录实际上游返回的图片数，并在固定价格模式下按该数量调整结算。OpenAI 兼容直通路径也会在固定价格模式下按响应 `data` 数量调整价格倍率，但其使用日志当前仍可能回退到请求中的 `n`；比例计费路径不强制改为实际图片数。

## 分辨率锁定

Gemini 图片分辨率由公开模型名决定：

| 公开模型后缀 | 锁定档位 |
| --- | --- |
| 无分辨率后缀 | 1K |
| `-2k` | 2K |
| `-4k` | 4K |

分辨率锁定依据用户请求中的原始公开模型名执行。渠道内部模型映射不能丢失该信息，客户端参数也不能升级或降低计费档位。

## 画幅比例

### Gemini Pro 图片模型

支持：

```text
1:1、9:16、16:9、3:4、4:3、3:2、2:3、5:4、4:5、21:9
```

### Gemini Flash 图片模型

支持：

```text
1:1、9:16、16:9、3:4、4:3、3:2、2:3、5:4、4:5、21:9、4:1、1:4、8:1、1:8
```

部分图片渠道只原生支持较小的比例集合。当渠道返回明确标记为可重试的“不支持该画幅”错误时，New API 的常规重试流程可能尝试下一个候选渠道，但渠道选择当前不会依据画幅能力做预筛选。后端转换器验证比例语法，目标部署仍需通过渠道配置和文档约束实际可用比例。

## 响应格式

6 个 Gemini 图片 SKU 的成功响应统一为 OpenAI Images 结构：

```json
{
  "created": 1234567890,
  "data": [
    {
      "b64_json": "..."
    }
  ]
}
```

Gemini 图片转换路径稳定使用 `data[].b64_json`。显式请求其他 `response_format` 的 Gemini 图片请求会被拒绝，避免用户代码在渠道回退后得到不同形态的响应。

`gpt-image-2` 使用 OpenAI 兼容直通响应，New API 当前不会把上游 `url` 响应转换为 Base64。目标生产渠道已经验收 `data[].b64_json`，但运维人员仍必须确保所选上游持续提供该响应契约，不能把 Gemini 转换器的强制保证直接套用于直通渠道。

调用方可以按模型文档示例解码 Base64：

```python
from base64 import b64decode
from pathlib import Path

Path("generated-image.jpg").write_bytes(
    b64decode(response["data"][0]["b64_json"])
)
```

`gpt-image-2` 编辑示例使用 PNG 文件名；Gemini 实际返回格式应根据响应字节判断，不能只依赖上传文件扩展名。

## 计费与日志

- 固定价格图片路径可以按响应中的实际图片数调整价格倍率。
- Gemini 图片转换路径会把实际响应图片数写入请求上下文，因此其使用日志中的“生成数量”使用实际值。
- OpenAI 兼容直通路径当前不会写入该日志上下文；其日志数量在缺少实际值时回退到请求中的 `n`。
- 比例计费路径保留原有计费语义，不应仅凭本功能文档假定其一定按响应图片数重算。
- 非法参数、参考图类型错误和超出上传字节限制的请求在调用上游前失败，不应扣除生成费用。
- 视频按秒计费时，模型广场、详情、价格表和使用日志应统一显示按秒单位。
- 固定价格和按秒价格均继续使用 New API 的 quota 结算与审计日志。

## 重试策略

以下本地错误显式携带 skip-retry 标记，不应发送到备用渠道：

- 安全策略或内容审核拦截；
- New API 本地参数验证错误；
- 非法参考图类型；
- 参考图单张或合计字节超限；
- 本地转换器拒绝的不支持请求格式。

上游错误是否重试由错误标记、错误码和部署中的状态码范围配置共同决定。网络错误、鉴权失效、限流或其他上游响应不能仅凭错误描述推断是否回退；例如某些 4xx 状态在特定配置中仍可能重试。

当上游成功响应但没有可用图片时，Gemini 响应处理器返回 `bad_response_body`。该错误码当前位于全局不可重试列表，因此不会回退。上游明确表示提示词被安全策略拦截时同样不可重试，避免相同内容被继续发送到备用渠道。

## multipart 内存保护

`ParseMultipartFormReusable()` 从可重放的请求体存储创建 Reader，再交给 Go multipart 解析器处理。请求体已经落盘时，不会为了重新解析而把整个缓存一次性复制回内存。

参考图转换过程中同时检查 multipart 声明大小和实际读取字节：

1. 声明大小超过单张或合计限制时立即拒绝；
2. 实际读取使用有界 Reader；
3. 实际内容超过单张限制时返回 HTTP 413；
4. 实际累计字节超过合计限制时返回 HTTP 413；
5. Base64 转换只在文件通过大小和类型验证后执行。

## Advanced Custom 渠道配置

Gemini 图片渠道使用专用请求转换器：

```text
openai_images_to_gemini_generate_content
```

该转换器只应用于 Gemini 图片 generation/edit 路由。`gpt-image-2` 的 OpenAI Images 路由使用 `converter=none`，避免错误套用 Gemini 参数或响应转换。

渠道应分别配置 generation 和 edit 路由。参考配置结构：

```json
{
  "advanced_routes": [
    {
      "incoming_path": "/v1/images/generations",
      "converter": "openai_images_to_gemini_generate_content"
    },
    {
      "incoming_path": "/v1/images/edits",
      "converter": "openai_images_to_gemini_generate_content"
    }
  ]
}
```

实际 `upstream_path`、鉴权和模型映射由部署环境配置，不应硬编码到客户端文档。

## 模型广场文档行为

图片模型详情页根据模型能力展示不同示例：

- Gemini 图片：generation JSON 示例以及 edit multipart 示例；
- `gpt-image-2` generation：只展示 `model`、`prompt`；
- `gpt-image-2` edit：展示 `model`、`prompt`、`image`；
- edit 示例覆盖 cURL、Python、TypeScript 和 JavaScript；
- `gpt-image-2` 非 cURL 编辑示例保存为 `edited-image.png`；
- Gemini 非 cURL 编辑示例按当前文档约定保存为 JPEG；
- 不向用户展示上游专用请求字段。

`omni-flash` 详情页按生成模式分成三段示例，每段都覆盖 cURL、Python、TypeScript 和 JavaScript，并且都是从创建、轮询到 `/content` 下载的完整流程：

- 文生视频：JSON 创建请求，不含 `mode` 和 `input_reference`；
- 图生视频：`multipart/form-data` 创建请求，`mode=i2v` 加一张 `input_reference`；
- 多参考生视频：`multipart/form-data` 创建请求，`mode=r2v` 加重复的 `input_reference` 字段。

参考图示例必须使用重复字段而不是覆盖式赋值（cURL 重复 `-F`，Python 重复元组，浏览器端使用 `FormData.append`），否则复制出去的代码会把参考图集合压缩成一张图。参考图创建示例不携带 `application/json` 内容类型。

`mode` 和 `input_reference` 只在支持参考图的视频 SKU 的参数表中展示，其它视频模型的参数表保持 `prompt`、`seconds`、`size` 三项。

相关前端实现位于：

```text
web/src/features/pricing/components/model-details-api.tsx
web/src/features/pricing/lib/media-code-samples.ts
web/src/features/pricing/lib/media-docs.ts
web/src/features/pricing/lib/mock-stats.ts
```

## 主要实现文件

### OpenAI Images 请求与 Gemini 图片转换

```text
relay/channel/advancedcustom/adaptor.go
relay/channel/advancedcustom/gemini_image.go
relaykit/dto/openai_image.go
relaykit/relayconvert/openai_image_gemini.go
relaykit/relayconvert/request_registry.go
```

### 图片响应、计费和日志

```text
relay/channel/gemini/relay_generate_content_image.go
relay/channel/openai/relay_image.go
relay/image_handler.go
constant/context_key.go
```

### 重试状态判定

```text
controller/relay.go
setting/operation_setting/status_code_ranges.go
```

### multipart 请求体复用

```text
common/gin.go
common/gin_multipart_test.go
```

### 视频任务、轮询白名单与内容代理

```text
controller/video_proxy.go
router/video-router.go
relay/relay_task.go
relay/channel/task/sora/adaptor.go
service/task_polling.go
```

## 回归测试

关键测试覆盖：

- OpenAI Images generation 转换；
- multipart edit 转换；
- 多于 4 张参考图仍可通过；
- 单张 16 MiB 和合计 32 MiB 边界；
- 声明大小与实际读取大小不一致；
- 伪造 MIME 类型；
- 输出 `b64_json` 转换；
- 固定价格路径的实际输出数量倍率；
- Gemini 实际输出数量上下文；
- 安全拦截不重试；
- `n > 4` 在上游调用前返回 HTTP 400；
- Sora 视频轮询的公共任务 ID、公开模型名和上游字段剥离；
- `omni-flash` 三种生成模式的示例代码：图生视频只带一张参考图、多参考生视频带齐三张参考图、默认示例不含 `mode` 和 `input_reference`；
- 参考图示例在四种语言中都使用重复字段而不是覆盖式赋值，且不携带 `application/json` 内容类型；
- `mode` 和 `input_reference` 只出现在支持参考图的视频模型参数表中。

`/v1/videos/{task_id}/content` 的鉴权、所有权校验、响应头和流式传输当前主要由实现审查与生产验收保证；本节不宣称已有完整的内容代理自动化测试覆盖。

后端验证命令：

```bash
go test ./...

cd relaykit
GOWORK=off go test ./...
```

前端验证命令：

```bash
cd web
NODE_OPTIONS=--no-experimental-webstorage bun run test -- --maxWorkers=4
bun run typecheck
bun run build
```

## 已验证行为

多参考图生产验收使用公开 OpenAI multipart 格式重复提交 5 个 `image` 字段：

```text
红色圆形
绿色三角形
蓝色正方形
黄色五角星
紫色菱形
```

模型成功返回一张 1024×1024 图片，并按要求保留五种形状、颜色和排列顺序。这证明当前网关不会在第 5 张参考图处触发人为数量限制，同时仍执行单张和合计字节保护。

视频三种模式在本地网关对接 Omni 上游完成验收，公开模型名为 `omni-flash`：

| 模式 | 请求形态 | 记录动作 | 上游 operation | 结果 |
| --- | --- | --- | --- | --- |
| 文生视频 | JSON，无参考图 | `textGenerate` | `text_to_video` | 1280×720，4 秒 |
| 图生视频 | multipart，1 张 `input_reference` | `generate` | `image_to_video` | 1280×720，4 秒 |
| 多参考生视频 | multipart，`mode=r2v` 加 3 张 `input_reference` | `referenceGenerate` | `reference_to_video` | 1280×720，4 秒 |

三个任务都通过 `/v1/videos/{task_id}/content` 取回视频；多参考生视频的成片同时包含三张参考图中的全部图形，说明重复的 `input_reference` 字段没有被丢弃。文档中的 `mode=i2v` 加 `seconds=6` 形态另行验收，产出 6 秒 1280×720 视频，按 6 秒计费。

计费按秒线性：4 秒任务与 6 秒任务的 quota 之比等于时长之比，使用日志按实际动作而不是统一按文生视频记录。`mode` 与 `operation` 同时提供返回 HTTP 400，声明 `i2v` 却上传两张参考图由上游返回 HTTP 400，两种情况都不扣费。

## 变更记录

| 提交 | 说明 |
| --- | --- |
| `11a49f93` | 增加 OpenAI Images 到 Gemini 图片请求/响应兼容层 |
| `8c7507a0` | 增加参考图字节、MIME、multipart 内存和 Gemini 实际输出数量保护 |
| `15fab62a` | 为 `gpt-image-2` 启用并记录参考图编辑文档 |
| `63600237` | 删除人为添加的 Gemini 参考图 4 张固定上限 |
| `0342b9a2` | 把多参考视频请求透传到 Omni 上游 |
| 当前变更 | 增加维护文档，将 Sora 轮询响应改为公共字段白名单，并为 `omni-flash` 补齐三种生成模式的用户文档 |

`8c7507a0` 中曾加入的参考图数量上限已经由 `63600237` 删除；单张 16 MiB、合计 32 MiB、真实字节 MIME 检测和不可重试错误策略继续保留。

## 非目标

- 不在公共请求中暴露上游专用图片协议；
- 不允许客户端通过可选字段改变 SKU 锁定的分辨率和价格；
- 不通过 `gpt-image-2` 文档暗示未经验证的上游参数；
- 不将上游视频签名地址直接返回给客户端；
- 不在本功能中修改图片生产服务本身，协议适配、限制、计费和用户文档均由 New API 完成。
