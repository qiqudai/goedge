using System.Collections.Concurrent;
using System.Net;
using System.Text;
using System.Text.Json;
using Bunit;
using Cnn.Api.Services;
using Cnn.Api.Services.Auth;
using Microsoft.AspNetCore.WebUtilities;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.JSInterop;
using MudBlazor.Services;
using Xunit;
using WebsiteAccessLogsPage = Cnn.Api.Pages.Website.AccessLogs;
using WebsiteBlockLogsPage = Cnn.Api.Pages.Website.BlockLogs;
using WebsiteDnsApiTab = Cnn.Api.Pages.Website.DnsApiTab;
using WebsiteGroupsPage = Cnn.Api.Pages.Website.Groups;
using WebsitePurgePage = Cnn.Api.Pages.Website.Purge;

namespace Cnn.Api.Tests;

public sealed class RemainingWebsiteComponentsInteractionTests : TestContext
{
    public RemainingWebsiteComponentsInteractionTests()
    {
        JSInterop.Mode = JSRuntimeMode.Loose;
    }

    [Fact]
    public async Task WebsiteDnsApiTab_Save_PreservesSelectedUserAndRendersDynamicFields()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        RenderComponent<MudBlazor.MudPopoverProvider>();

        var cut = RenderComponent<WebsiteDnsApiTab>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/dnsapi", 1);

        SetPrivateField(cut.Instance, "_selectedUserId", 1001L);
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OnSelectedUserChangedAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/dnsapi", 2);
        Assert.Equal("1001", handler.ForPath("/api/v1/admin/dnsapi")[^1].Query["user_id"]);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OpenCreate"));
        SetEditor(cut.Instance, "_editing", new Dictionary<string, object?>
        {
            ["UserId"] = 1001L,
            ["Name"] = "cf-main",
            ["Remark"] = "test-remark",
            ["Type"] = "cloudflare",
            ["Credentials"] = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
            {
                ["email"] = "ops@example.com",
                ["api_key"] = "secret"
            }
        });
        cut.Render();

        cut.WaitForAssertion(() =>
        {
            Assert.Contains("邮箱", cut.Markup);
            Assert.Contains("API Key", cut.Markup);
        });

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SaveAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/dnsapi", 4);

        var createRequest = Assert.Single(handler.ForMethodAndPath(HttpMethod.Post, "/api/v1/admin/dnsapi"));
        Assert.Contains("\"user_id\":1001", createRequest.Body, StringComparison.OrdinalIgnoreCase);
        Assert.Contains("\"type\":\"cloudflare\"", createRequest.Body, StringComparison.OrdinalIgnoreCase);
        Assert.Contains("ops@example.com", createRequest.Body, StringComparison.Ordinal);
        Assert.Equal("1001", handler.ForPath("/api/v1/admin/dnsapi")[^1].Query["user_id"]);
        Assert.Contains("cf-main", cut.Markup);
    }

    [Fact]
    public async Task WebsiteDnsApiTab_SearchAndBatchDelete_TrackKeywordAndSelectedRows()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        JSInterop.Setup<bool>("confirm", _ => true).SetResult(true);
        RenderComponent<MudBlazor.MudPopoverProvider>();

        var cut = RenderComponent<WebsiteDnsApiTab>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/dnsapi", 1);

        SetPrivateField(cut.Instance, "_keyword", "  cloudflare  ");
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SearchAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/dnsapi", 2);
        Assert.Equal("cloudflare", handler.ForPath("/api/v1/admin/dnsapi")[^1].Query["keyword"]);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ToggleAllChecked", true));
        Assert.Equal(1, GetHashSetCount(cut.Instance, "_selectedIds"));

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "DeleteBatchAsync"));
        Assert.Single(handler.ForMethodAndPath(HttpMethod.Delete, "/api/v1/admin/dnsapi/900"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/dnsapi", 3);
        Assert.Equal(0, GetHashSetCount(cut.Instance, "_selectedIds"));
    }

    [Fact]
    public async Task WebsiteDnsApiTab_EditAndSingleDelete_CoverRowActions()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        JSInterop.Setup<bool>("confirm", _ => true).SetResult(true);
        RenderComponent<MudBlazor.MudPopoverProvider>();

        var cut = RenderComponent<WebsiteDnsApiTab>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/dnsapi", 1);

        var item = GetFirstListItem(GetPrivateField<object>(cut.Instance, "_items"));
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OpenEdit", item));

        Assert.Equal("cf-main", GetPropertyFromPrivateField<string>(cut.Instance, "_editing", "Name"));
        Assert.Equal("cloudflare", GetPropertyFromPrivateField<string>(cut.Instance, "_editing", "Type"));
        Assert.Equal("ops@example.com", GetDictionaryValueFromPrivateField(cut.Instance, "_editing", "Credentials", "email"));
        Assert.Equal("secret", GetDictionaryValueFromPrivateField(cut.Instance, "_editing", "Credentials", "api_key"));

        SetEditor(cut.Instance, "_editing", new Dictionary<string, object?>
        {
            ["Id"] = 900L,
            ["UserId"] = 1001L,
            ["Name"] = "cf-main-updated",
            ["Remark"] = "remark-updated",
            ["Type"] = "cloudflare",
            ["Credentials"] = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
            {
                ["email"] = "ops@example.com",
                ["api_key"] = "secret-2"
            }
        });
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SaveAsync"));
        Assert.Single(handler.ForMethodAndPath(HttpMethod.Put, "/api/v1/admin/dnsapi/900"));

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "DeleteAsync", item));
        Assert.Single(handler.ForMethodAndPath(HttpMethod.Delete, "/api/v1/admin/dnsapi/900"));
    }

    [Fact]
    public async Task WebsiteDnsApiTab_UserSearchAndTypeValidation_CoverEditorControls()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        RenderComponent<MudBlazor.MudPopoverProvider>();

        var cut = RenderComponent<WebsiteDnsApiTab>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/dnsapi", 1);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OpenCreate"));
        SetPrivateField(cut.Instance, "_editorUserKeyword", "alice@example.com");
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SearchEditorUsersAsync"));
        Assert.Contains(handler.ForPath("/api/v1/admin/users"), r => r.Query.TryGetValue("keyword", out var value) && value == "alice@example.com");
        Assert.Equal(1, GetCollectionCount(cut.Instance, "_editorUsers"));

        SetEditor(cut.Instance, "_editing", new Dictionary<string, object?>
        {
            ["UserId"] = 1001L,
            ["Name"] = "missing-type",
            ["Type"] = ""
        });
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SaveAsync"));
        Assert.Equal("请选择DNS类型", GetPrivateField<string?>(cut.Instance, "_message"));

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OnTypeChangedAsync", "aliyun"));
        Assert.Equal("aliyun", GetPropertyFromPrivateField<string>(cut.Instance, "_editing", "Type"));
        Assert.Equal(0, GetDictionaryCountFromPrivateField(cut.Instance, "_editing", "Credentials"));
    }

    [Fact]
    public async Task WebsiteDnsApiTab_RefreshAndPagingDiagnostics_CoverToolbarControls()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        RenderComponent<MudBlazor.MudPopoverProvider>();

        var cut = RenderComponent<WebsiteDnsApiTab>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/dnsapi", 1);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "RefreshAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/dnsapi", 2);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "TogglePagingDiagnostics"));
        cut.Render();
        Assert.Contains("website:dnsapi:table", cut.Markup);
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "TogglePagingDiagnostics"));
        cut.Render();
        Assert.DoesNotContain("website:dnsapi:table", cut.Markup);
    }

    [Fact]
    public async Task WebsiteDnsApiTab_NameValidation_ShowsExpectedMessage()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        RenderComponent<MudBlazor.MudPopoverProvider>();

        var cut = RenderComponent<WebsiteDnsApiTab>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/dnsapi", 1);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OpenCreate"));
        SetEditor(cut.Instance, "_editing", new Dictionary<string, object?>
        {
            ["UserId"] = 1001L,
            ["Name"] = "",
            ["Type"] = "cloudflare"
        });
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SaveAsync"));
        Assert.Equal("请填写名称", GetPrivateField<string?>(cut.Instance, "_message"));
    }

    [Fact]
    public async Task WebsiteGroups_Save_UsesSelectedUserAndRefreshesFilteredList()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        RenderComponent<MudBlazor.MudPopoverProvider>();

        var cut = RenderComponent<WebsiteGroupsPage>();
        Assert.Empty(handler.ForPath("/api/v1/admin/site_groups"));

        SetPrivateField(cut.Instance, "_selectedUserId", 1001L);
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OnSelectedUserChangedAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/site_groups", 1);
        Assert.Equal("1001", handler.ForPath("/api/v1/admin/site_groups")[0].Query["user_id"]);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OpenCreate"));
        SetEditor(cut.Instance, "_editing", new Dictionary<string, object?>
        {
            ["UserId"] = 1001L,
            ["Name"] = "group-alpha",
            ["Remark"] = "primary"
        });

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SaveAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/site_groups", 3);

        var createRequest = Assert.Single(handler.ForMethodAndPath(HttpMethod.Post, "/api/v1/admin/site_groups"));
        Assert.Contains("\"user_id\":1001", createRequest.Body, StringComparison.OrdinalIgnoreCase);
        Assert.Contains("\"name\":\"group-alpha\"", createRequest.Body, StringComparison.OrdinalIgnoreCase);
        Assert.Equal("1001", handler.ForPath("/api/v1/admin/site_groups")[^1].Query["user_id"]);
        Assert.Contains("group-alpha", cut.Markup);
    }

    [Fact]
    public async Task WebsiteGroups_SearchAndValidation_CoverToolbarControls()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        RenderComponent<MudBlazor.MudPopoverProvider>();

        var cut = RenderComponent<WebsiteGroupsPage>();

        SetPrivateField(cut.Instance, "_selectedUserId", 1001L);
        SetPropertyOnPrivateField(cut.Instance, "_query", "Keyword", "video");
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SearchAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/site_groups", 1);
        Assert.Equal("1001", handler.ForPath("/api/v1/admin/site_groups")[0].Query["user_id"]);
        Assert.Equal("video", handler.ForPath("/api/v1/admin/site_groups")[0].Query["keyword"]);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OpenCreate"));
        SetEditor(cut.Instance, "_editing", new Dictionary<string, object?>
        {
            ["UserId"] = 0L,
            ["Name"] = ""
        });
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SaveAsync"));
        Assert.Equal("名称不能为空", GetPrivateField<string?>(cut.Instance, "_message"));

        SetEditor(cut.Instance, "_editing", new Dictionary<string, object?>
        {
            ["UserId"] = 0L,
            ["Name"] = "group-beta"
        });
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SaveAsync"));
        Assert.Equal("请选择用户", GetPrivateField<string?>(cut.Instance, "_message"));
    }

    [Fact]
    public async Task WebsiteGroups_EditAndSingleDelete_WorkInUserScope()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        Services.GetRequiredService<ClientSession>().Set(null, "user", "demo");
        RenderComponent<MudBlazor.MudPopoverProvider>();

        var cut = RenderComponent<WebsiteGroupsPage>();
        WaitForEndpoint(cut, handler, "/api/v1/user/site_groups", 1);

        var item = GetFirstListItem(GetPrivateField<object>(cut.Instance, "_items"));
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OpenEdit", item));
        Assert.Equal("group-alpha", GetPropertyFromPrivateField<string>(cut.Instance, "_editing", "Name"));
        Assert.Equal("primary", GetPropertyFromPrivateField<string>(cut.Instance, "_editing", "Remark"));

        SetEditor(cut.Instance, "_editing", new Dictionary<string, object?>
        {
            ["Id"] = 700L,
            ["UserId"] = 1001L,
            ["Name"] = "group-updated",
            ["Remark"] = "remark-updated"
        });
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SaveAsync"));
        Assert.Single(handler.ForMethodAndPath(HttpMethod.Put, "/api/v1/user/site_groups/700"));

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "DeleteAsync", item));
        Assert.Single(handler.ForMethodAndPath(HttpMethod.Delete, "/api/v1/user/site_groups/700"));
    }

    [Fact]
    public async Task WebsiteGroups_BatchDeleteAndPagingDiagnostics_CoverSelectionControls()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        JSInterop.Setup<bool>("confirm", _ => true).SetResult(true);
        RenderComponent<MudBlazor.MudPopoverProvider>();

        var cut = RenderComponent<WebsiteGroupsPage>();
        SetPrivateField(cut.Instance, "_selectedUserId", 1001L);
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OnSelectedUserChangedAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/site_groups", 1);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ToggleAllChecked", true));
        Assert.Equal(1, GetHashSetCount(cut.Instance, "_selectedIds"));
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ToggleAllChecked", false));
        Assert.Equal(0, GetHashSetCount(cut.Instance, "_selectedIds"));

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ToggleAllChecked", true));
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "BatchDeleteAsync"));
        Assert.Single(handler.ForPath("/api/v1/admin/site_groups/700/delete_preview"));
        Assert.Contains(handler.Requests(), r =>
            r.Method == HttpMethod.Post &&
            r.Path.EndsWith("/delete_request", StringComparison.Ordinal));

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "TogglePagingDiagnostics"));
        cut.Render();
        Assert.Contains("website:groups:table", cut.Markup);
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "TogglePagingDiagnostics"));
        cut.Render();
        Assert.DoesNotContain("website:groups:table", cut.Markup);
    }

    [Fact]
    public async Task WebsiteGroups_AdminSingleDelete_UsesDeleteWorkflow()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        JSInterop.Setup<bool>("confirm", _ => true).SetResult(true);
        RenderComponent<MudBlazor.MudPopoverProvider>();

        var cut = RenderComponent<WebsiteGroupsPage>();
        SetPrivateField(cut.Instance, "_selectedUserId", 1001L);
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OnSelectedUserChangedAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/site_groups", 1);

        var item = GetFirstListItem(GetPrivateField<object>(cut.Instance, "_items"));
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "DeleteAsync", item));
        Assert.Single(handler.ForPath("/api/v1/admin/site_groups/700/delete_preview"));
        Assert.Single(handler.ForMethodAndPath(HttpMethod.Post, "/api/v1/admin/site_groups/700/delete_request"));
    }

    [Fact]
    public async Task WebsiteAccessLogs_SearchAndReset_PropagatesAdvancedFilters()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);

        var cut = RenderComponent<WebsiteAccessLogsPage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access", 1);

        SetPropertyOnPrivateField(cut.Instance, "_filters", "Domain", "cdn.example.com");
        SetPropertyOnPrivateField(cut.Instance, "_filters", "Keyword", "/video");
        SetPropertyOnPrivateField(cut.Instance, "_filters", "Method", "POST");
        SetPropertyOnPrivateField(cut.Instance, "_filters", "ClientIp", "203.0.113.10");
        SetPropertyOnPrivateField(cut.Instance, "_filters", "CacheStatus", "HIT");
        SetPropertyOnPrivateField(cut.Instance, "_filters", "StartTime", new DateTime(2026, 4, 21, 8, 0, 0));
        SetPropertyOnPrivateField(cut.Instance, "_filters", "EndTime", new DateTime(2026, 4, 21, 9, 0, 0));

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SearchAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access", 2);

        var filtered = handler.ForPath("/api/v1/admin/logs/access")[^1];
        Assert.Equal("cdn.example.com", filtered.Query["domain"]);
        Assert.Equal("/video", filtered.Query["keyword"]);
        Assert.Equal("POST", filtered.Query["method"]);
        Assert.Equal("203.0.113.10", filtered.Query["client_ip"]);
        Assert.Equal("HIT", filtered.Query["cache_status"]);
        Assert.Equal("2026-04-21 08:00:00", filtered.Query["start_time"]);
        Assert.Equal("2026-04-21 09:00:00", filtered.Query["end_time"]);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ResetFiltersAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access", 3);

        var reset = handler.ForPath("/api/v1/admin/logs/access")[^1];
        Assert.False(reset.Query.ContainsKey("domain"));
        Assert.False(reset.Query.ContainsKey("keyword"));
        Assert.False(reset.Query.ContainsKey("method"));
        Assert.False(reset.Query.ContainsKey("client_ip"));
        Assert.False(reset.Query.ContainsKey("cache_status"));
        Assert.False(reset.Query.ContainsKey("start_time"));
        Assert.False(reset.Query.ContainsKey("end_time"));
    }

    [Fact]
    public async Task WebsiteAccessLogs_HistorySearchAndDownload_RequestExpectedApis()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);

        var cut = RenderComponent<WebsiteAccessLogsPage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/domains", 1);
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access", 1);

        await cut.InvokeAsync(() => cut.FindAll("button").Single(b => b.TextContent.Trim() == "申请记录").Click());
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access/downloads", 1);

        SetPropertyOnPrivateField(cut.Instance, "_downloadQuery", "Keyword", "access-20260421");
        SetPropertyOnPrivateField(cut.Instance, "_downloadQuery", "State", "done");
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SearchDownloadsAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access/downloads", 2);
        var historyQuery = handler.ForPath("/api/v1/admin/logs/access/downloads")[^1].Query;
        Assert.Equal("access-20260421", historyQuery["keyword"]);
        Assert.Equal("done", historyQuery["state"]);

        await cut.InvokeAsync(() => cut.FindAll("button").Single(b => b.TextContent.Trim() == "日志查询").Click());
        SetPropertyOnPrivateField(cut.Instance, "_filters", "Domain", "cdn.example.com");
        SetPropertyOnPrivateField(cut.Instance, "_filters", "Keyword", "/video");
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "DownloadAsync"));

        Assert.Single(handler.ForMethodAndPath(HttpMethod.Post, "/api/v1/admin/logs/access/downloads"));
        Assert.NotEmpty(handler.ForMethodAndPath(HttpMethod.Put, "/api/v1/admin/logs/access/downloads/501"));
        Assert.Contains(handler.ForPath("/api/v1/admin/logs/access"), r => r.Query.TryGetValue("pageSize", out var value) && value == "200");
    }

    [Fact]
    public async Task WebsiteAccessLogs_FilterChipAndAdvancedToggle_ClearExpectedFields()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);

        var cut = RenderComponent<WebsiteAccessLogsPage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access", 1);

        SetPrivateField(cut.Instance, "_advancedVisible", true);
        cut.Render();
        Assert.Contains("时间范围", cut.Markup);
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "HideAdvanced"));
        cut.Render();
        Assert.DoesNotContain("时间范围", cut.Markup);

        SetPropertyOnPrivateField(cut.Instance, "_filters", "Domain", "cdn.example.com");
        SetPropertyOnPrivateField(cut.Instance, "_filters", "Keyword", "/vod");
        SetPropertyOnPrivateField(cut.Instance, "_filters", "StartTime", new DateTime(2026, 4, 21, 8, 0, 0));
        SetPropertyOnPrivateField(cut.Instance, "_filters", "EndTime", new DateTime(2026, 4, 21, 9, 0, 0));

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ClearDomain"));
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ClearKeyword"));
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ClearTimeRange"));

        Assert.Equal(string.Empty, GetPropertyFromPrivateField<string>(cut.Instance, "_filters", "Domain"));
        Assert.Equal(string.Empty, GetPropertyFromPrivateField<string>(cut.Instance, "_filters", "Keyword"));
        Assert.Null(GetPropertyFromPrivateField<DateTime?>(cut.Instance, "_filters", "StartTime"));
        Assert.Null(GetPropertyFromPrivateField<DateTime?>(cut.Instance, "_filters", "EndTime"));
    }

    [Fact]
    public async Task WebsiteAccessLogs_PagingDiagnostics_ToggleAcrossTabs()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);

        var cut = RenderComponent<WebsiteAccessLogsPage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access", 1);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "TogglePagingDiagnostics"));
        cut.Render();
        Assert.Contains("website:access_logs:query:table", cut.Markup);

        await cut.InvokeAsync(() => cut.FindAll("button").Single(b => b.TextContent.Trim() == "申请记录").Click());
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access/downloads", 1);
        cut.Render();
        Assert.Contains("website:access_logs:history:table", cut.Markup);
    }

    [Fact]
    public async Task WebsiteAccessLogs_DomainOptionsAndClearButton_CoverFilterHelpers()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);

        var cut = RenderComponent<WebsiteAccessLogsPage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/domains", 1);
        Assert.Equal(1, GetCollectionCount(cut.Instance, "_domainOptions"));

        SetPropertyOnPrivateField(cut.Instance, "_filters", "Domain", "cdn.example.com");
        SetPropertyOnPrivateField(cut.Instance, "_filters", "Keyword", "/clear-me");
        SetPropertyOnPrivateField(cut.Instance, "_filters", "StartTime", new DateTime(2026, 4, 21, 8, 0, 0));
        SetPropertyOnPrivateField(cut.Instance, "_filters", "EndTime", new DateTime(2026, 4, 21, 9, 0, 0));
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ResetFiltersAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access", 2);

        var query = handler.ForPath("/api/v1/admin/logs/access")[^1].Query;
        Assert.False(query.ContainsKey("domain"));
        Assert.False(query.ContainsKey("keyword"));
        Assert.False(query.ContainsKey("start_time"));
        Assert.False(query.ContainsKey("end_time"));
    }

    [Fact]
    public async Task WebsiteBlockLogs_SelectAllAndHistoryTimeRange_Work()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);

        var cut = RenderComponent<WebsiteBlockLogsPage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/current", 1);
        cut.WaitForAssertion(() => Assert.Contains("203.0.113.11", cut.Markup));

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ToggleCurrentSelectAll", new Microsoft.AspNetCore.Components.ChangeEventArgs
        {
            Value = "on"
        }));
        Assert.Equal(2, GetHashSetCount(cut.Instance, "_currentSelected"));

        await cut.InvokeAsync(() => cut.FindAll("button").Single(b => b.TextContent.Trim() == "历史记录").Click());
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/history", 1);

        SetPropertyOnPrivateField(cut.Instance, "_historyFilter", "Type", "time_range");
        SetPropertyOnPrivateField(cut.Instance, "_historyFilter", "StartTime", new DateTime(2026, 4, 21, 10, 0, 0));
        SetPropertyOnPrivateField(cut.Instance, "_historyFilter", "EndTime", new DateTime(2026, 4, 21, 11, 0, 0));

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SearchHistoryAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/history", 2);

        var history = handler.ForPath("/api/v1/admin/logs/block/history")[^1];
        Assert.Equal("time_range", history.Query["type"]);
        Assert.Equal("2026-04-21 10:00:00", history.Query["start_time"]);
        Assert.Equal("2026-04-21 11:00:00", history.Query["end_time"]);
    }

    [Fact]
    public async Task WebsiteBlockLogs_UnblockSiteAndExport_UseExpectedControls()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        JSInterop.Setup<bool>("confirm", _ => true).SetResult(true);

        var cut = RenderComponent<WebsiteBlockLogsPage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/current", 1);

        SetPropertyOnPrivateField(cut.Instance, "_currentFilter", "Type", "site_id");
        SetPropertyOnPrivateField(cut.Instance, "_currentFilter", "Keyword", "11");
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "UnblockSiteAsync"));

        Assert.Single(handler.ForMethodAndPath(HttpMethod.Get, "/api/v1/admin/sites/11"));
        var updateSite = Assert.Single(handler.ForMethodAndPath(HttpMethod.Put, "/api/v1/admin/sites/11"));
        Assert.Contains("\"blacklist\":[]", updateSite.Body, StringComparison.OrdinalIgnoreCase);
        Assert.Contains("\"whitelist\":[\"198.51.100.2\"]", updateSite.Body, StringComparison.OrdinalIgnoreCase);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ExportCurrentAsync"));
        Assert.Contains(handler.ForPath("/api/v1/admin/logs/block/current"), r => r.Query.TryGetValue("pageSize", out var value) && value == "200");

        await cut.InvokeAsync(() => cut.FindAll("button").Single(b => b.TextContent.Trim() == "历史记录").Click());
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/history", 1);
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ExportHistoryAsync"));
        Assert.Contains(handler.ForPath("/api/v1/admin/logs/block/history"), r => r.Query.TryGetValue("pageSize", out var value) && value == "200");
    }

    [Fact]
    public async Task WebsiteBlockLogs_SingleUnblockAndHistoryFilterReset_Work()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        JSInterop.Setup<bool>("confirm", _ => true).SetResult(true);

        var cut = RenderComponent<WebsiteBlockLogsPage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/current", 1);

        var item = GetFirstListItem(GetPrivateField<object>(cut.Instance, "_currentItems"));
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "UnblockAsync", item));
        Assert.Single(handler.ForMethodAndPath(HttpMethod.Get, "/api/v1/admin/sites/11"));
        Assert.Single(handler.ForMethodAndPath(HttpMethod.Put, "/api/v1/admin/sites/11"));

        SetPropertyOnPrivateField(cut.Instance, "_historyFilter", "Keyword", "11");
        SetPropertyOnPrivateField(cut.Instance, "_historyFilter", "StartTime", new DateTime(2026, 4, 21, 1, 0, 0));
        SetPropertyOnPrivateField(cut.Instance, "_historyFilter", "EndTime", new DateTime(2026, 4, 21, 2, 0, 0));
        SetPropertyOnPrivateField(cut.Instance, "_historyFilter", "Type", "time_range");
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "HistoryFilterChanged"));
        Assert.Equal(string.Empty, GetPropertyFromPrivateField<string>(cut.Instance, "_historyFilter", "Keyword"));
        Assert.Null(GetPropertyFromPrivateField<DateTime?>(cut.Instance, "_historyFilter", "StartTime"));
        Assert.Null(GetPropertyFromPrivateField<DateTime?>(cut.Instance, "_historyFilter", "EndTime"));
    }

    [Fact]
    public async Task WebsiteBlockLogs_BatchUnblockAndStatsDiagnostics_CoverRemainingControls()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        JSInterop.Setup<bool>("confirm", _ => true).SetResult(true);

        var cut = RenderComponent<WebsiteBlockLogsPage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/current", 1);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ToggleCurrentSelectAll", new Microsoft.AspNetCore.Components.ChangeEventArgs
        {
            Value = "on"
        }));
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "UnblockBatchAsync"));
        Assert.Single(handler.ForMethodAndPath(HttpMethod.Get, "/api/v1/admin/sites/11"));
        Assert.Single(handler.ForMethodAndPath(HttpMethod.Get, "/api/v1/admin/sites/12"));
        Assert.Single(handler.ForMethodAndPath(HttpMethod.Put, "/api/v1/admin/sites/11"));
        Assert.Single(handler.ForMethodAndPath(HttpMethod.Put, "/api/v1/admin/sites/12"));

        await cut.InvokeAsync(() => cut.FindAll("button").Single(b => b.TextContent.Trim() == "统计").Click());
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/stats", 1);
        Assert.Contains("11", cut.Markup);
        Assert.Contains("3", cut.Markup);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "TogglePagingDiagnostics"));
        cut.Render();
        Assert.Contains("website:block_logs:stats:table", cut.Markup);
    }

    [Fact]
    public async Task WebsiteBlockLogs_CurrentSearchTypeSwitch_TracksQueryType()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);

        var cut = RenderComponent<WebsiteBlockLogsPage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/current", 1);

        SetPropertyOnPrivateField(cut.Instance, "_currentFilter", "Type", "site_id");
        SetPropertyOnPrivateField(cut.Instance, "_currentFilter", "Keyword", "11");
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SearchCurrentAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/current", 2);
        var currentQuery = handler.ForPath("/api/v1/admin/logs/block/current")[^1].Query;
        Assert.Equal("site_id", currentQuery["type"]);
        Assert.Equal("11", currentQuery["keyword"]);
    }

    [Fact]
    public async Task WebsitePurge_SubmitAndBatchResubmit_Workflow_Works()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        JSInterop.Setup<bool>("confirm", _ => true).SetResult(true);

        var cut = RenderComponent<WebsitePurgePage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks/usage", 1);

        SetPrivateField(cut.Instance, "_taskType", "preheat");
        SetPrivateField(cut.Instance, "_urls", "https://cdn.example.com/a.js\nhttps://cdn.example.com/b.js");
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SubmitAsync"));

        Assert.Equal("list", GetPrivateField<string>(cut.Instance, "_activeTab"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks/usage", 2);
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OnListPageQueryChangedAsync", new Cnn.Api.Shared.TablePageQuery
        {
            PagingEnabled = true,
            Page = 1,
            PageSize = 10
        }));
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks", 2);
        Assert.Equal(2, GetCollectionCount(cut.Instance, "_items"));

        var submitRequest = Assert.Single(handler.ForMethodAndPath(HttpMethod.Post, "/api/v1/admin/tasks"));
        Assert.Contains("\"type\":\"preheat\"", submitRequest.Body, StringComparison.OrdinalIgnoreCase);
        Assert.Contains("cdn.example.com/a.js", submitRequest.Body, StringComparison.Ordinal);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ToggleSelectAll", new Microsoft.AspNetCore.Components.ChangeEventArgs
        {
            Value = true
        }));
        Assert.Equal(2, GetHashSetCount(cut.Instance, "_selectedIds"));

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ResubmitBatchAsync"));
        Assert.Equal(2, handler.ForMethodAndPath(HttpMethod.Post, "/api/v1/admin/tasks/101/resubmit").Count
            + handler.ForMethodAndPath(HttpMethod.Post, "/api/v1/admin/tasks/102/resubmit").Count);
        Assert.Equal("重新提交成功", GetPrivateField<string>(cut.Instance, "_message"));
    }

    [Fact]
    public async Task WebsitePurge_TaskTypeSwitchSearchAndSingleResubmit_CoverControls()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);
        JSInterop.Setup<bool>("confirm", _ => true).SetResult(true);

        var cut = RenderComponent<WebsitePurgePage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks/usage", 1);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OnTaskTypeChangedAsync", "refresh_dir"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks/usage", 2);
        Assert.Equal("refresh_dir", GetPrivateField<string>(cut.Instance, "_taskType"));

        SetPrivateField(cut.Instance, "_activeTab", "list");
        SetPrivateField(cut.Instance, "_keyword", "asset");
        SetPrivateField(cut.Instance, "_listType", "preheat");
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OnListPageQueryChangedAsync", new Cnn.Api.Shared.TablePageQuery
        {
            PagingEnabled = true,
            Page = 1,
            PageSize = 10
        }));
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks", 1);
        var listQuery = handler.ForPath("/api/v1/admin/tasks")[^1].Query;
        Assert.Equal("asset", listQuery["keyword"]);
        Assert.Equal("preheat", listQuery["type"]);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ResubmitAsync", 101L));
        Assert.Single(handler.ForMethodAndPath(HttpMethod.Post, "/api/v1/admin/tasks/101/resubmit"));
    }

    [Fact]
    public async Task WebsitePurge_PagingDiagnosticsAndStatusBadge_CoverListControls()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);

        var cut = RenderComponent<WebsitePurgePage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks/usage", 1);

        Assert.Contains("每日限额100次，今日剩余90次", cut.Markup);
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "RefreshUsageAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks/usage", 2);

        SetPrivateField(cut.Instance, "_activeTab", "list");
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OnListPageQueryChangedAsync", new Cnn.Api.Shared.TablePageQuery
        {
            PagingEnabled = true,
            Page = 1,
            PageSize = 10
        }));
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks", 1);
        cut.Render();
        Assert.Contains("等待", cut.Markup);
        Assert.Contains("完成", cut.Markup);
        Assert.Contains("text-bg-secondary", cut.Markup);
        Assert.Contains("text-bg-success", cut.Markup);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "TogglePagingDiagnostics"));
        cut.Render();
        Assert.Contains("website:purge:list:table", cut.Markup);
    }

    [Fact]
    public async Task WebsitePurge_EnterSearchAndToggleSelectedRow_CoverListInteractions()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);

        var cut = RenderComponent<WebsitePurgePage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks/usage", 1);

        SetPrivateField(cut.Instance, "_activeTab", "list");
        SetPrivateField(cut.Instance, "_keyword", "enter-search");
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OnKeywordKeyDown", new Microsoft.AspNetCore.Components.Web.KeyboardEventArgs
        {
            Key = "Enter"
        }));
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks", 1);
        Assert.Equal("enter-search", handler.ForPath("/api/v1/admin/tasks")[^1].Query["keyword"]);

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ToggleSelectAll", new Microsoft.AspNetCore.Components.ChangeEventArgs
        {
            Value = true
        }));
        Assert.Equal(2, GetHashSetCount(cut.Instance, "_selectedIds"));

        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "ToggleSelected", 101L, new Microsoft.AspNetCore.Components.ChangeEventArgs
        {
            Value = false
        }));
        Assert.Equal(1, GetHashSetCount(cut.Instance, "_selectedIds"));
    }

    [Fact]
    public async Task WebsitePurge_EmptySubmit_ShowsValidationMessage()
    {
        var handler = new WebsiteInteractionApiHandler();
        RegisterPageServices(handler);

        var cut = RenderComponent<WebsitePurgePage>();
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks/usage", 1);

        SetPrivateField(cut.Instance, "_urls", "   ");
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "SubmitAsync"));
        Assert.Equal("请输入URL", GetPrivateField<string>(cut.Instance, "_message"));
    }

    private void RegisterPageServices(HttpMessageHandler handler)
    {
        Services.AddSingleton<IConfiguration>(_ => new ConfigurationBuilder().Build());
        Services.AddSingleton<IAuthTokenService, AuthTokenService>();
        Services.AddSingleton<ClientSession>();
        Services.AddSingleton<LocalStorageService>();
        Services.AddSingleton<HttpClient>(_ => new HttpClient(handler)
        {
            BaseAddress = new Uri("http://localhost")
        });
        Services.AddSingleton<ApiClient>();
        Services.AddMudServices();
    }

    private static void WaitForEndpoint(IRenderedFragment cut, WebsiteInteractionApiHandler handler, string path, int count)
    {
        cut.WaitForAssertion(() => Assert.True(handler.ForPath(path).Count >= count));
    }

    private static async Task InvokePrivateAsync(object target, string methodName, params object[] args)
    {
        var method = target.GetType().GetMethod(methodName, System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic);
        Assert.NotNull(method);
        var result = method!.Invoke(target, args);
        if (result is Task task)
        {
            await task;
        }
    }

    private static void SetPrivateField<T>(object target, string fieldName, T value)
    {
        var field = target.GetType().GetField(fieldName, System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic);
        Assert.NotNull(field);
        field!.SetValue(target, value);
    }

    private static void SetPropertyOnPrivateField(object target, string fieldName, string propertyName, object? value)
    {
        var field = target.GetType().GetField(fieldName, System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic);
        Assert.NotNull(field);
        var owner = field!.GetValue(target);
        Assert.NotNull(owner);
        var property = owner!.GetType().GetProperty(propertyName);
        Assert.NotNull(property);
        property!.SetValue(owner, value);
    }

    private static int GetHashSetCount(object target, string fieldName)
    {
        var field = target.GetType().GetField(fieldName, System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic);
        Assert.NotNull(field);
        var value = field!.GetValue(target);
        Assert.NotNull(value);
        var countProperty = value!.GetType().GetProperty("Count");
        Assert.NotNull(countProperty);
        return (int)(countProperty!.GetValue(value) ?? 0);
    }

    private static int GetCollectionCount(object target, string fieldName)
    {
        var field = target.GetType().GetField(fieldName, System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic);
        Assert.NotNull(field);
        var value = field!.GetValue(target);
        Assert.NotNull(value);
        var countProperty = value!.GetType().GetProperty("Count");
        Assert.NotNull(countProperty);
        return (int)(countProperty!.GetValue(value) ?? 0);
    }

    private static T GetPropertyFromPrivateField<T>(object target, string fieldName, string propertyName)
    {
        var field = target.GetType().GetField(fieldName, System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic);
        Assert.NotNull(field);
        var owner = field!.GetValue(target);
        Assert.NotNull(owner);
        var property = owner!.GetType().GetProperty(propertyName);
        Assert.NotNull(property);
        return (T)property!.GetValue(owner)!;
    }

    private static T GetPrivateField<T>(object target, string fieldName)
    {
        var field = target.GetType().GetField(fieldName, System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic);
        Assert.NotNull(field);
        return (T)field!.GetValue(target)!;
    }

    private static string GetDictionaryValueFromPrivateField(object target, string fieldName, string propertyName, string key)
    {
        var field = target.GetType().GetField(fieldName, System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic);
        Assert.NotNull(field);
        var owner = field!.GetValue(target);
        Assert.NotNull(owner);
        var property = owner!.GetType().GetProperty(propertyName);
        Assert.NotNull(property);
        var dictionary = property!.GetValue(owner);
        Assert.NotNull(dictionary);
        var tryGetValue = dictionary!.GetType().GetMethod("TryGetValue");
        Assert.NotNull(tryGetValue);
        var args = new object?[] { key, null };
        var ok = (bool)(tryGetValue!.Invoke(dictionary, args) ?? false);
        Assert.True(ok);
        return (string)args[1]!;
    }

    private static int GetDictionaryCountFromPrivateField(object target, string fieldName, string propertyName)
    {
        var field = target.GetType().GetField(fieldName, System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic);
        Assert.NotNull(field);
        var owner = field!.GetValue(target);
        Assert.NotNull(owner);
        var property = owner!.GetType().GetProperty(propertyName);
        Assert.NotNull(property);
        var dictionary = property!.GetValue(owner);
        Assert.NotNull(dictionary);
        var countProperty = dictionary!.GetType().GetProperty("Count");
        Assert.NotNull(countProperty);
        return (int)(countProperty!.GetValue(dictionary) ?? 0);
    }

    private static object GetFirstListItem(object list)
    {
        var enumerable = (System.Collections.IEnumerable)list;
        foreach (var item in enumerable)
        {
            return item!;
        }

        throw new InvalidOperationException("Expected at least one item.");
    }

    private static void SetEditor(object target, string fieldName, IReadOnlyDictionary<string, object?> values)
    {
        var field = target.GetType().GetField(fieldName, System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic);
        Assert.NotNull(field);
        var editor = field!.GetValue(target);
        Assert.NotNull(editor);

        foreach (var pair in values)
        {
            var property = editor!.GetType().GetProperty(pair.Key);
            Assert.NotNull(property);
            property!.SetValue(editor, pair.Value);
        }
    }

    private sealed class WebsiteInteractionApiHandler : HttpMessageHandler
    {
        private readonly ConcurrentQueue<RecordedRequest> _requests = new();
        private int _dnsApiId = 900;
        private int _groupId = 700;

        public IReadOnlyList<RecordedRequest> ForPath(string path)
        {
            return _requests.Where(x => string.Equals(x.Path, path, StringComparison.Ordinal)).ToList();
        }

        public IReadOnlyList<RecordedRequest> ForMethodAndPath(HttpMethod method, string path)
        {
            return _requests.Where(x => x.Method == method && string.Equals(x.Path, path, StringComparison.Ordinal)).ToList();
        }

        public IReadOnlyList<RecordedRequest> Requests()
        {
            return _requests.ToList();
        }

        protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
        {
            var uri = request.RequestUri ?? new Uri("http://localhost/");
            var query = QueryHelpers.ParseQuery(uri.Query)
                .Where(static pair => !string.IsNullOrWhiteSpace(pair.Value.ToString()))
                .ToDictionary(static pair => pair.Key, static pair => pair.Value.ToString(), StringComparer.OrdinalIgnoreCase);
            var body = request.Content == null
                ? string.Empty
                : await request.Content.ReadAsStringAsync(cancellationToken);

            _requests.Enqueue(new RecordedRequest(request.Method, uri.AbsolutePath, query, body));

            var data = ResolveData(request.Method, uri.AbsolutePath, query);
            var message = ResolveMessage(request.Method, uri.AbsolutePath);
            var responseBody = $"{{\"code\":200,\"message\":\"{message}\",\"data\":{data},\"trace_id\":\"test\"}}";
            return new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(responseBody, Encoding.UTF8, "application/json")
            };
        }

        private string ResolveData(HttpMethod method, string path, IReadOnlyDictionary<string, string> query)
        {
            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/users", StringComparison.Ordinal))
            {
                return "{\"list\":[{\"id\":1001,\"name\":\"alice\",\"email\":\"alice@example.com\"}],\"total\":1}";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/dns/providers/types", StringComparison.Ordinal))
            {
                return "{\"types\":[{\"type\":\"cloudflare\",\"name\":\"Cloudflare\"},{\"type\":\"aliyun\",\"name\":\"Aliyun\"}]}";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/dnsapi/types", StringComparison.Ordinal))
            {
                return "{\"types\":[{\"type\":\"cloudflare\",\"name\":\"Cloudflare\",\"fields\":[\"email\",\"api_key\"]},{\"type\":\"aliyun\",\"name\":\"Aliyun\",\"fields\":[\"access_key_id\",\"access_key_secret\"]}]}";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/dnsapi", StringComparison.Ordinal))
            {
                var userId = query.TryGetValue("user_id", out var selectedUser) ? selectedUser : "0";
                return $$"""
{"list":[{"id":{{_dnsApiId}},"uid":{{userId}},"name":"cf-main","type":"cloudflare","remark":"test-remark","auth":"{\"email\":\"ops@example.com\",\"api_key\":\"secret\"}"}],"total":1}
""";
            }

            if (method == HttpMethod.Post && string.Equals(path, "/api/v1/admin/dnsapi", StringComparison.Ordinal))
            {
                _dnsApiId++;
                return "true";
            }

            if (method == HttpMethod.Put && path.StartsWith("/api/v1/admin/dnsapi/", StringComparison.Ordinal))
            {
                return "true";
            }

            if (method == HttpMethod.Delete && path.StartsWith("/api/v1/admin/dnsapi/", StringComparison.Ordinal))
            {
                return "true";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/site_groups", StringComparison.Ordinal))
            {
                var userId = query.TryGetValue("user_id", out var selectedUser) ? selectedUser : "0";
                return $$"""
{"list":[{"id":{{_groupId}},"user_id":{{userId}},"name":"group-alpha","remark":"primary"}],"total":1}
""";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/user/site_groups", StringComparison.Ordinal))
            {
                return $$"""
{"list":[{"id":{{_groupId}},"user_id":1001,"name":"group-alpha","remark":"primary"}],"total":1}
""";
            }

            if (method == HttpMethod.Post && string.Equals(path, "/api/v1/admin/site_groups", StringComparison.Ordinal))
            {
                _groupId++;
                return "true";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/site_groups/700/delete_preview", StringComparison.Ordinal))
            {
                return "{\"canDelete\":true,\"message\":\"ok\",\"references\":[]}";
            }

            if (method == HttpMethod.Post && string.Equals(path, "/api/v1/admin/site_groups/700/delete_request", StringComparison.Ordinal))
            {
                return "{\"queued\":true,\"message\":\"queued\",\"task\":{\"taskId\":7700,\"taskNo\":\"task-7700\",\"state\":\"waiting\"},\"references\":[]}";
            }

            if (method == HttpMethod.Put && string.Equals(path, "/api/v1/user/site_groups/700", StringComparison.Ordinal))
            {
                return "true";
            }

            if (method == HttpMethod.Delete && string.Equals(path, "/api/v1/user/site_groups/700", StringComparison.Ordinal))
            {
                return "true";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/logs/access", StringComparison.Ordinal))
            {
                return "{\"list\":[{\"timestamp\":\"2026-04-21T08:30:00Z\",\"host\":\"cdn.example.com\",\"scheme\":\"https\",\"method\":\"GET\",\"uri\":\"/video.m3u8\",\"status\":200,\"remote_addr\":\"203.0.113.10\",\"bytes\":120,\"request_time\":0.03,\"upstream_response_time\":0.02,\"upstream_addr\":\"10.0.0.2:80\",\"upstream_cache_status\":\"HIT\",\"http_referer\":\"-\",\"http_user_agent\":\"curl\",\"node_id\":\"n-1\",\"node_ip\":\"10.0.0.1\",\"ssl_protocol\":\"TLSv1.3\",\"ssl_cipher\":\"TLS_AES_128_GCM_SHA256\"}],\"total\":1}";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/logs/access/downloads", StringComparison.Ordinal))
            {
                return "{\"list\":[{\"id\":501,\"file_name\":\"access-20260421.csv\",\"state\":\"done\",\"rows\":2,\"error\":\"\",\"requester_user_id\":1001,\"scope\":\"admin\",\"created_at\":\"2026-04-21T08:40:00Z\",\"finished_at\":\"2026-04-21T08:41:00Z\"}],\"total\":1}";
            }

            if (method == HttpMethod.Post && string.Equals(path, "/api/v1/admin/logs/access/downloads", StringComparison.Ordinal))
            {
                return "{\"id\":501,\"state\":\"pending\"}";
            }

            if (method == HttpMethod.Put && string.Equals(path, "/api/v1/admin/logs/access/downloads/501", StringComparison.Ordinal))
            {
                return "true";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/domains", StringComparison.Ordinal))
            {
                return "{\"list\":[{\"id\":1,\"user_id\":1001,\"name\":\"cdn.example.com\",\"cname\":\"edge.example.com\",\"status\":1,\"origins\":[],\"created_at\":\"2026-04-21T08:00:00Z\",\"updated_at\":\"2026-04-21T08:00:00Z\"}],\"total\":1}";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/logs/block/current", StringComparison.Ordinal))
            {
                return "{\"list\":[{\"site_id\":11,\"domain\":\"a.example.com\",\"ip\":\"203.0.113.11\",\"location\":\"CN\",\"filter\":\"cc-a\",\"block_time\":\"2026-04-21T09:00:00Z\",\"release_time\":\"2026-04-21T10:00:00Z\"},{\"site_id\":12,\"domain\":\"b.example.com\",\"ip\":\"203.0.113.12\",\"location\":\"US\",\"filter\":\"cc-b\",\"block_time\":\"2026-04-21T09:05:00Z\",\"release_time\":\"2026-04-21T10:05:00Z\"}],\"total\":2}";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/logs/block/history", StringComparison.Ordinal))
            {
                return "{\"list\":[{\"site_id\":11,\"domain\":\"a.example.com\",\"ip\":\"203.0.113.11\",\"location\":\"CN\",\"filter\":\"cc-a\",\"block_time\":\"2026-04-21T09:00:00Z\",\"is_manual\":true}],\"total\":1}";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/logs/block/stats", StringComparison.Ordinal))
            {
                return "{\"list\":[{\"site_id\":11,\"domain\":\"a.example.com\",\"count\":3}],\"total\":1}";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/sites/11", StringComparison.Ordinal))
            {
                return "{\"id\":11,\"settings\":{\"security\":{\"blacklist\":[\"203.0.113.11\",\"203.0.113.15\"],\"whitelist\":[\"198.51.100.2\"]}}}";
            }

            if (method == HttpMethod.Put && string.Equals(path, "/api/v1/admin/sites/11", StringComparison.Ordinal))
            {
                return "{\"id\":11,\"settings\":{\"security\":{\"blacklist\":[],\"whitelist\":[\"198.51.100.2\"]}}}";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/sites/12", StringComparison.Ordinal))
            {
                return "{\"id\":12,\"settings\":{\"security\":{\"blacklist\":[\"203.0.113.12\"],\"whitelist\":[]}}}";
            }

            if (method == HttpMethod.Put && string.Equals(path, "/api/v1/admin/sites/12", StringComparison.Ordinal))
            {
                return "{\"id\":12,\"settings\":{\"security\":{\"blacklist\":[],\"whitelist\":[\"203.0.113.12\"]}}}";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/tasks/usage", StringComparison.Ordinal))
            {
                return "{\"limits\":{\"refresh_url\":100,\"refresh_dir\":50,\"preheat\":20},\"remaining\":{\"refresh_url\":90,\"refresh_dir\":40,\"preheat\":10},\"used\":{\"date\":\"2026-04-21\",\"refresh_url\":10,\"refresh_dir\":10,\"preheat\":10}}";
            }

            if (method == HttpMethod.Get && string.Equals(path, "/api/v1/admin/tasks", StringComparison.Ordinal))
            {
                return "{\"list\":[{\"id\":101,\"type\":\"preheat\",\"name\":\"task-101\",\"data\":\"https://cdn.example.com/a.js\",\"state\":\"waiting\",\"create_at\":\"2026-04-21T09:30:00Z\"},{\"id\":102,\"type\":\"refresh_url\",\"name\":\"task-102\",\"data\":\"https://cdn.example.com/b.js\",\"state\":\"done\",\"create_at\":\"2026-04-21T09:40:00Z\"}],\"total\":2,\"page\":1}";
            }

            if (method == HttpMethod.Post && string.Equals(path, "/api/v1/admin/tasks", StringComparison.Ordinal))
            {
                return "true";
            }

            if (method == HttpMethod.Post && path.StartsWith("/api/v1/admin/tasks/", StringComparison.Ordinal) && path.EndsWith("/resubmit", StringComparison.Ordinal))
            {
                return "true";
            }

            return "null";
        }

        private static string ResolveMessage(HttpMethod method, string path)
        {
            return (method.Method, path) switch
            {
                ("POST", "/api/v1/admin/dnsapi") => "dns-created",
                ("POST", "/api/v1/admin/site_groups") => "group-created",
                ("POST", "/api/v1/admin/tasks") => "task-created",
                _ => "ok"
            };
        }
    }

    private sealed record RecordedRequest(HttpMethod Method, string Path, IReadOnlyDictionary<string, string> Query, string Body);
}
