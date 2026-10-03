# license-rule-checklist

离线核查软件交付清单：依据输入的许可规则，定位每个交付根可达组件的缺失材料与不兼容依赖。
工具只使用 Go 标准库，不访问任何外部许可库；所有结论仅适用于输入文件中给出的规则。

## 运行

```sh
go run ./cmd/licensecheck -manifest examples/manifest.json -rules examples/rules.json -out report.json
```

参数：

- `-manifest`：JSON 组件清单。
- `-rules`：JSON 许可规则。
- `-out`：JSON 报告输出路径。

退出码：

- `0`：核查完成且没有问题。
- `1`：核查完成，但至少一个交付根存在问题（报告已写出）。
- `2`：输入错误或运行失败；此时不写结果文件。

## 清单格式

```json
{
  "roots": ["svc-auth"],
  "components": [
    {
      "id": "svc-auth",
      "license": "MIT",
      "dependencies": ["lib-crypto"],
      "materials": ["license", "notice"]
    }
  ]
}
```

- 每个组件必须有唯一、非空的 `id`。
- `dependencies` 必须引用存在的组件，同一依赖不可重复。
- `roots` 必须引用存在的组件且不可重复；不可达组件不影响任何根的报告。

## 规则格式

```json
{
  "licenses": [
    {"code": "MIT", "required_materials": ["license", "notice"]}
  ],
  "allowed_pairs": [
    {"parent": "MIT", "child": "GPL"}
  ]
}
```

- `licenses` 中每个 `code` 唯一且非空，`required_materials` 是持有该许可的组件必需材料集合。
- `allowed_pairs` 是有向允许对：`parent` 许可的组件允许携带 `child` 许可的依赖。
  方向相反并不自动允许；相同许可之间也不默认兼容，必须显式列出。

## 核查规则

对每个交付根：

1. 从根做 BFS 遍历可达依赖，允许环路；每个组件、每条有向依赖边在该根下只核查一次。
2. 每个可达组件单独列出，共享组件会出现在各自根的报告中。
3. 组件问题定位在组件自身：
   - `unknown_license`：组件许可未在规则中声明；检查继续进行。
   - `missing_materials`：已附材料缺少规则要求项（仅对已知许可检查）。
4. 边问题定位在父组件，并通过 `dependency` 标出子组件：
   - 父或子任一许可未知：`undecidable_dependency`（不同时再记为不兼容）。
   - 双方许可已知但有向对不在允许集合：`incompatible_dependency`。
5. 每个问题都附带从根到所定位组件的最短路径 `path`；存在并列时选择编号序列字典序最小者。

## 输入错误

以下情况退出码为 `2`，且不写结果：组件编号为空或重复、许可规则代号重复、
依赖引用不存在、交付根不存在或重复、JSON 无法解析。

## 排序

- 可达组件按编号字典序排列。
- 问题先按定位组件编号、再按依赖组件编号、最后按问题类型稳定排序。
- 缺失材料按名称字典序排列。

## 测试

```sh
go test ./...
```

测试覆盖许可对方向相反、依赖环路、跨根共享依赖、未知许可导致的无法判断、
最短路径字典序平局以及各类输入错误。
