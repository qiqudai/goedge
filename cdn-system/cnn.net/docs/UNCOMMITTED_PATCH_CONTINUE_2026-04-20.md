# 未提交改动清单（继续补充）

日期: 2026-04-20

## 本轮文档改动
- `docs/PAGINATION_FINAL_ACCEPTANCE_CHECKLIST.md`
  - 补充第 3 章说明（未执行压测项保持未勾选）
  - 更新第 8 章执行备注（并行 test 文件锁，串行结果为准）

## 本轮验证命令
- `dotnet test tests/Cnn.Api.Tests/Cnn.Api.Tests.csproj --filter "PaginationPageRegressionTests"`
  - 结果：25/25 通过
- `dotnet test tests/Cnn.Api.Tests/Cnn.Api.Tests.csproj --filter "TablePagerTests|PaginationPageRegressionTests|TaskPagesPagingDiagnosticsTests|PaginationUiConventionsTests"`
  - 结果：36/36 通过

## 本线程累计关键代码改动（未提交）
- `src/Cnn.Common/Contracts/Admin/CertDtos.cs`
- `src/Cnn.Api/Services/Admin/CertService.cs`
- `src/Cnn.Api/Services/Admin/CertService.Batch.cs`
- `src/Cnn.Api/Controllers/Admin/CertsController.cs`
- `src/Cnn.Api/Controllers/User/CertsController.cs`
- `tests/Cnn.Api.Tests/CertServiceBehaviorTests.cs`
- `tests/Cnn.Api.Tests/SiteServiceOwnershipTests.cs`
- `docs/GO_VS_DOTNET_PARITY_REPORT_2026-04-20.md`
- `docs/PAGINATION_FINAL_ACCEPTANCE_CHECKLIST.md`
