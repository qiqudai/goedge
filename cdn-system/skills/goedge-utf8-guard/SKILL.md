---
name: goedge-utf8-guard
description: 确保 cdn-system 前端/后端源码保持 UTF-8 编码。每次写入前后必须校验，发现问题立即修复并复检。
---

# GoEdge UTF-8 Guard

## 目标

- 前后端源码保持 UTF-8（建议无 BOM）
- 每次修改前后执行编码校验
- 发现问题立即修复，直至复检通过

## 适用范围

- 前端：`web/`（重点 `web/admin/src`）
- 后端：`api/`、`common/`、`agent/`
- 其他源码目录按需补充

## 标准流程

1. 修改前执行校验
2. 进行修改
3. 修改后再次校验
4. 如发现问题，立即修复并复检直到通过（必须）

## 校验命令

使用内置脚本执行校验：
```powershell
powershell -File "E:/cdn/goedge/cdn-system/skills/goedge-utf8-guard/scripts/check-utf8.ps1" `
  -Paths "E:/cdn/goedge/cdn-system/web/admin/src","E:/cdn/goedge/cdn-system/api","E:/cdn/goedge/cdn-system/common","E:/cdn/goedge/cdn-system/agent"
```

## 修复参考

### 将 GB18030 转为 UTF-8（无 BOM）
```powershell
$path = "E:/path/to/file.vue"
$bytes = [IO.File]::ReadAllBytes($path)
$text = [Text.Encoding]::GetEncoding("GB18030").GetString($bytes)
[IO.File]::WriteAllText($path, $text, [Text.UTF8Encoding]::new($false))
```

### 修复 U+FFFD（�）字符
1. 打开文件，手动修正被替换字符的文本/字符串
2. 保存为 UTF-8（无 BOM）
3. 重新运行校验脚本
