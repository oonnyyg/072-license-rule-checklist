# license-rule-checklist

离线软件交付清单核查工具。它只依据输入 JSON 中给出的许可规则工作，
不访问任何外部许可库，因此结论仅适用于输入规则本身。

## 运行

```sh
go run ./cmd/licensecheck -input manifest.json -output report.json
```

退出码：

- `0`：所有交付根均未发现问题。
- `1`：核查完成但报告中存在问题（结果文件已写出）。
- `2`：输入错误（解析失败、引用缺失、编号/规则重复等），不写结果文件。

## 输入格式

```json
{
  "components": [
    {
      "id": "app",
      "license": "MIT",
      "dependsOn": ["lib"],
      "materials": ["LICENSE"]
    }
  ],
  "rules": [
    {
      "license": "MIT",
      "required": ["LICENSE", "NOTICE"],
      "allowedPairs": [["MIT", "GPL"]]
    }
  ],
  "roots": ["app"]
}
```

- `components[].id`：唯一且非空的组件编号。
- `license`：许可代号。
- `dependsOn`：直接依赖编号列表，允许成环。
- `materials`：已随附的材料名称。
- `rules[].required`：该许可必需的材料集合。
- `allowedPairs`：允许的有向许可对 `[父许可, 子许可]`，表示携带该方向
  的依赖是兼容的。同许可**不**默认兼容，必须显式列出（如 `["X","X"]`）。
- `roots`：交付根编号列表，可为多个；重复的根编号视为输入错误。

输入错误包括：组件编号为空或重复、许可规则重复、依赖引用不存在、
交付根不存在、JSON 语法错误或对象内出现重复键。

## 核查规则

对每个交付根：

1. 沿 `dependsOn` 做有向遍历得到可达集合；依赖图允许环，每个组件和
   每条有向边在该根下只核查一次。
2. 逐组件比较必需材料与已附材料，缺失的按规则声明顺序报告为
   `missing-materials`，问题定位在组件自身。
3. 逐边按父 -> 子方向核对许可对：
   - 规则中存在该有向对：通过；
   - 两端许可均已知但无此有向对：`incompatible-license`；
   - 父或子许可未知：`unknown-license-edge`（无法判断，不重复记为
     不兼容）。
4. 许可代号在规则中找不到时，该组件报 `unknown-license`，其余检查
   继续进行。
5. 共享组件在不同根的报告中各自列出；不可达组件及其出边不影响该根。

每条问题都附 `path`：从根到问题定位组件的最短路径（按编号序列的
字典序选择最小者）。边问题定位到父组件。

## 输出格式

输出为按根声明顺序排列的报告数组；`reachable` 按编号排序，`problems`
按定位组件编号、问题类型、依赖边稳定排序。

```json
[
  {
    "root": "app",
    "reachable": ["app", "lib"],
    "problems": [
      {
        "kind": "missing-materials",
        "component": "lib",
        "path": ["app", "lib"],
        "materials": ["LICENSE"]
      },
      {
        "kind": "incompatible-license",
        "component": "app",
        "path": ["app"],
        "edge": {"parent": "app", "child": "lib"}
      }
    ]
  }
]
```

问题类型：`missing-materials`、`unknown-license`、
`unknown-license-edge`、`incompatible-license`。

## 测试

```sh
go test ./...
```

测试覆盖：许可对方向相反不兼容、同许可不默认兼容、依赖环路只核查
一次、共享依赖在多根报告中分别出现、不可达组件被忽略、未知许可边
标记为无法判断、最短路径字典序选择，以及各退出码和输入错误场景。
