using System.Collections.Concurrent;
using System.Net;
using System.Text;
using Bunit;
using Cnn.Api.Pages.System;
using Cnn.Api.Services;
using Cnn.Api.Services.Auth;
using Cnn.Api.Shared;
using Microsoft.AspNetCore.WebUtilities;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.JSInterop;
using MudBlazor;
using MudBlazor.Services;
using Xunit;
using FinanceOrdersPage = Cnn.Api.Pages.Finance.Orders;
using ForwardListPage = Cnn.Api.Pages.Forward.List;
using AccountBillsPage = Cnn.Api.Pages.Account.Bills;
using AccountLogsPage = Cnn.Api.Pages.Account.Logs;
using AccountMessagesPage = Cnn.Api.Pages.Account.Messages;
using NodeGroupsPage = Cnn.Api.Pages.Node.Groups;
using NodeListPage = Cnn.Api.Pages.Node.List;
using SystemAnnouncementsPage = Cnn.Api.Pages.System.Announcements;
using SystemLoginLogsPage = Cnn.Api.Pages.System.LoginLogs;
using SystemLogsPage = Cnn.Api.Pages.System.Logs;
using SystemMessagesPage = Cnn.Api.Pages.System.Messages;
using SystemOpLogsPage = Cnn.Api.Pages.System.OpLogs;
using SystemUsersPage = Cnn.Api.Pages.System.Users;
using WebsiteAccessLogsPage = Cnn.Api.Pages.Website.AccessLogs;
using WebsiteBlockLogsPage = Cnn.Api.Pages.Website.BlockLogs;
using WebsiteCertsPage = Cnn.Api.Pages.Website.Certs;
using WebsiteDnsApiTab = Cnn.Api.Pages.Website.DnsApiTab;
using WebsiteGroupsPage = Cnn.Api.Pages.Website.Groups;
using WebsiteListPage = Cnn.Api.Pages.Website.List;
using WebsitePurgePage = Cnn.Api.Pages.Website.Purge;
using WebsiteSiteResolvePanel = Cnn.Api.Pages.Website.SiteResolvePanel;

namespace Cnn.Api.Tests;

public sealed class PaginationPageRegressionTests : TestContext
{
    public PaginationPageRegressionTests()
    {
        JSInterop.Mode = JSRuntimeMode.Loose;
    }

    [Fact]
    public async Task WebsiteList_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/sites");
        RegisterPageServices(handler);
        SetupPagerState("website:list:table", 3, 50);

        var cut = RenderComponent<WebsiteListPage>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/sites", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/sites")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/sites", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/sites")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/sites", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/sites")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/sites").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/sites").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/sites");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task WebsiteGroups_Pagination_Regression_WithAdminUserSelection()
    {
        var handler = CreateHandler("/api/v1/admin/site_groups");
        RegisterPageServices(handler);
        SetupPagerState("website:groups:table", 3, 50);

        RenderComponent<MudPopoverProvider>();
        var cut = RenderComponent<WebsiteGroupsPage>();
        Assert.Empty(handler.ForPath("/api/v1/admin/site_groups"));

        SetPrivateField(cut.Instance, "_selectedUserId", 1001L);
        await cut.InvokeAsync(() => InvokePrivateAsync(cut.Instance, "OnSelectedUserChangedAsync"));
        WaitForEndpoint(cut, handler, "/api/v1/admin/site_groups", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/site_groups")[0], page: 1, pageSize: 50);
        Assert.Equal("1001", handler.ForPath("/api/v1/admin/site_groups")[0].Query["user_id"]);

        var pager = cut.FindComponent<TablePager>().Instance;
        await cut.InvokeAsync(() => InvokePrivateAsync(pager, "OnPageChangedAsync", 3));
        WaitForEndpoint(cut, handler, "/api/v1/admin/site_groups", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/site_groups")[1], page: 3, pageSize: 50);
        Assert.Equal("1001", handler.ForPath("/api/v1/admin/site_groups")[1].Query["user_id"]);

        await cut.InvokeAsync(() => pager.RefreshAsync());
        WaitForEndpoint(cut, handler, "/api/v1/admin/site_groups", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/site_groups")[2], page: 3, pageSize: 50);
        Assert.Equal("1001", handler.ForPath("/api/v1/admin/site_groups")[2].Query["user_id"]);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/site_groups", 4);
        AssertQuery(handler.ForPath("/api/v1/admin/site_groups")[3], page: 1, pageSize: 50);
        Assert.Equal("1001", handler.ForPath("/api/v1/admin/site_groups")[3].Query["user_id"]);

        var baseline = handler.ForPath("/api/v1/admin/site_groups").Count;
        await cut.InvokeAsync(() => Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync()));
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/site_groups").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/site_groups");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
        Assert.Equal("1001", afterRapid[^1].Query["user_id"]);
    }

    [Fact]
    public async Task WebsiteDnsApiTab_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/dnsapi");
        RegisterPageServices(handler);
        SetupPagerState("website:dnsapi:table", 3, 50);

        RenderComponent<MudPopoverProvider>();
        var cut = RenderComponent<WebsiteDnsApiTab>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/dnsapi", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/dnsapi")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/dnsapi", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/dnsapi")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/dnsapi", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/dnsapi")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/dnsapi").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/dnsapi").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/dnsapi");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task WebsiteResolvePanel_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/sites");
        RegisterPageServices(handler);
        SetupPagerState("website:resolve:table", 3, 50);

        var cut = RenderComponent<WebsiteSiteResolvePanel>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/sites", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/sites")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/sites", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/sites")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/sites", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/sites")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/sites").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/sites").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/sites");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task ForwardList_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/forwards");
        RegisterPageServices(handler);
        SetupPagerState("forward:list:table", 3, 50);

        var cut = RenderComponent<ForwardListPage>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/forwards", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/forwards")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/forwards", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/forwards")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/forwards", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/forwards")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/forwards").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/forwards").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/forwards");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task NodeList_AndMonitorDialog_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/nodes", "/api/v1/admin/nodes/101/monitor_logs");
        RegisterPageServices(handler);
        SetupPagerState("node:list:table", 3, 50);
        SetupPagerState("node:list:monitor:table", 3, 50);

        var cut = RenderComponent<NodeListPage>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/nodes", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/nodes")[0], page: 3, pageSize: 50);

        var nodePager = cut.FindComponents<TablePager>()[0].Instance;
        await nodePager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/nodes", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/nodes")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/nodes", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/nodes")[2], page: 1, pageSize: 50);

        var nodeBaseline = handler.ForPath("/api/v1/admin/nodes").Count;
        await Task.WhenAll(nodePager.RefreshAsync(), nodePager.SearchAsync(), nodePager.RefreshAsync(), nodePager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/nodes").Count >= nodeBaseline + 1));
        var nodeAfterRapid = handler.ForPath("/api/v1/admin/nodes");
        Assert.True(nodeAfterRapid.Count <= nodeBaseline + 2);
        AssertQuery(nodeAfterRapid[^1], page: 1, pageSize: 50);

        cut.WaitForAssertion(() => Assert.Contains(cut.FindAll("button"), b => b.TextContent.Trim() == "日志"));
        await cut.InvokeAsync(() => cut.FindAll("button").Single(b => b.TextContent.Trim() == "日志").Click());
        WaitForEndpoint(cut, handler, "/api/v1/admin/nodes/101/monitor_logs", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/nodes/101/monitor_logs")[0], page: 3, pageSize: 50);

        var monitorPager = cut.FindComponents<TablePager>()[1].Instance;
        await monitorPager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/nodes/101/monitor_logs", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/nodes/101/monitor_logs")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Where(b => b.TextContent.Trim() == "搜索").Last().Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/nodes/101/monitor_logs", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/nodes/101/monitor_logs")[2], page: 1, pageSize: 50);

        var monitorBaseline = handler.ForPath("/api/v1/admin/nodes/101/monitor_logs").Count;
        await Task.WhenAll(monitorPager.RefreshAsync(), monitorPager.SearchAsync(), monitorPager.RefreshAsync(), monitorPager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/nodes/101/monitor_logs").Count >= monitorBaseline + 1));
        var monitorAfterRapid = handler.ForPath("/api/v1/admin/nodes/101/monitor_logs");
        Assert.True(monitorAfterRapid.Count <= monitorBaseline + 2);
        AssertQuery(monitorAfterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task SystemTasks_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/tasks");
        RegisterPageServices(handler);
        SetupPagerState("system:tasks:table", 3, 50);

        var cut = RenderComponent<Tasks>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/tasks")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/tasks")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/tasks")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/tasks").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/tasks").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/tasks");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task SystemMessages_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/messages");
        RegisterPageServices(handler);
        SetupPagerState("system:messages:table", 3, 50);

        var cut = RenderComponent<SystemMessagesPage>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/messages", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/messages")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/messages", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/messages")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/messages", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/messages")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/messages").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/messages").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/messages");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task SystemAnnouncements_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/announcements");
        RegisterPageServices(handler);
        SetupPagerState("system:announcements:table", 3, 50);

        var cut = RenderComponent<SystemAnnouncementsPage>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/announcements", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/announcements")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/announcements", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/announcements")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/announcements", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/announcements")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/announcements").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/announcements").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/announcements");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task AccountLogs_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/user/logs/operation");
        RegisterPageServices(handler);
        SetupPagerState("account:logs:table", 3, 50);

        var cut = RenderComponent<AccountLogsPage>();

        WaitForEndpoint(cut, handler, "/api/v1/user/logs/operation", 1);
        AssertQuery(handler.ForPath("/api/v1/user/logs/operation")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/user/logs/operation", 2);
        AssertQuery(handler.ForPath("/api/v1/user/logs/operation")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/user/logs/operation", 3);
        AssertQuery(handler.ForPath("/api/v1/user/logs/operation")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/user/logs/operation").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/user/logs/operation").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/user/logs/operation");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task AccountMessages_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/user/messages");
        RegisterPageServices(handler);
        SetupPagerState("account:messages:table", 3, 50);

        var cut = RenderComponent<AccountMessagesPage>();

        WaitForEndpoint(cut, handler, "/api/v1/user/messages", 1);
        AssertQuery(handler.ForPath("/api/v1/user/messages")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/user/messages", 2);
        AssertQuery(handler.ForPath("/api/v1/user/messages")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/user/messages", 3);
        AssertQuery(handler.ForPath("/api/v1/user/messages")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/user/messages").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/user/messages").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/user/messages");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task AccountBills_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/user/orders");
        RegisterPageServices(handler);
        SetupPagerState("account:bills:table", 3, 50);

        var cut = RenderComponent<AccountBillsPage>();

        WaitForEndpoint(cut, handler, "/api/v1/user/orders", 1);
        AssertQuery(handler.ForPath("/api/v1/user/orders")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/user/orders", 2);
        AssertQuery(handler.ForPath("/api/v1/user/orders")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/user/orders", 3);
        AssertQuery(handler.ForPath("/api/v1/user/orders")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/user/orders").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/user/orders").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/user/orders");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task WebsiteAccessLogs_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/logs/access");
        RegisterPageServices(handler);
        SetupPagerState("website:access_logs:query:table", 3, 50);

        var cut = RenderComponent<WebsiteAccessLogsPage>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/access")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/access")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/access")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/logs/access").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/logs/access").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/logs/access");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task WebsiteAccessLogs_History_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/logs/access/downloads");
        RegisterPageServices(handler);
        SetupPagerState("website:access_logs:history:table", 3, 50);

        var cut = RenderComponent<WebsiteAccessLogsPage>();
        cut.FindAll("button").Single(b => b.TextContent.Trim() == "申请记录").Click();

        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access/downloads", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/access/downloads")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access/downloads", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/access/downloads")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/access/downloads", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/access/downloads")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/logs/access/downloads").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/logs/access/downloads").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/logs/access/downloads");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task WebsiteBlockLogs_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/logs/block/current");
        RegisterPageServices(handler);
        SetupPagerState("website:block_logs:current:table", 3, 50);

        var cut = RenderComponent<WebsiteBlockLogsPage>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/current", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/block/current")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/current", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/block/current")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/current", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/block/current")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/logs/block/current").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/logs/block/current").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/logs/block/current");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task WebsiteBlockLogs_Stats_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/logs/block/stats");
        RegisterPageServices(handler);
        SetupPagerState("website:block_logs:stats:table", 3, 50);

        var cut = RenderComponent<WebsiteBlockLogsPage>();
        cut.FindAll("button").Single(b => b.TextContent.Trim() == "统计").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/stats", 1);
        await Task.Delay(120);

        var pager = cut.FindComponent<TablePager>().Instance;
        var baseline = handler.ForPath("/api/v1/admin/logs/block/stats").Count;
        await cut.InvokeAsync(() => pager.RefreshAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/logs/block/stats").Count >= baseline + 1));
        AssertQuery(handler.ForPath("/api/v1/admin/logs/block/stats")[^1], page: 3, pageSize: 50);
        await Task.Delay(120);

        baseline = handler.ForPath("/api/v1/admin/logs/block/stats").Count;
        await cut.InvokeAsync(() => pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/logs/block/stats").Count >= baseline + 1));
        AssertQuery(handler.ForPath("/api/v1/admin/logs/block/stats")[^1], page: 1, pageSize: 50);
        await Task.Delay(120);

        baseline = handler.ForPath("/api/v1/admin/logs/block/stats").Count;
        await cut.InvokeAsync(() => Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync()));
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/logs/block/stats").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/logs/block/stats");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task WebsiteBlockLogs_History_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/logs/block/history");
        RegisterPageServices(handler);
        SetupPagerState("website:block_logs:history:table", 3, 50);

        var cut = RenderComponent<WebsiteBlockLogsPage>();
        cut.FindAll("button").Single(b => b.TextContent.Trim() == "历史记录").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/history", 1);
        await Task.Delay(120);

        var pager = cut.FindComponent<TablePager>().Instance;
        var baseline = handler.ForPath("/api/v1/admin/logs/block/history").Count;
        await cut.InvokeAsync(() => pager.RefreshAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/logs/block/history").Count >= baseline + 1));
        AssertQuery(handler.ForPath("/api/v1/admin/logs/block/history")[^1], page: 3, pageSize: 50);
        await Task.Delay(120);

        baseline = handler.ForPath("/api/v1/admin/logs/block/history").Count;
        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/block/history", baseline + 1);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/block/history")[^1], page: 1, pageSize: 50);
        await Task.Delay(120);

        baseline = handler.ForPath("/api/v1/admin/logs/block/history").Count;
        await cut.InvokeAsync(() => Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync()));
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/logs/block/history").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/logs/block/history");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task WebsitePurgeList_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/tasks");
        RegisterPageServices(handler);
        SetupPagerState("website:purge:list:table", 3, 50);

        var cut = RenderComponent<WebsitePurgePage>();
        cut.FindAll("button").Single(b => b.TextContent.Trim() == "操作记录").Click();

        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/tasks")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/tasks")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/tasks", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/tasks")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/tasks").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/tasks").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/tasks");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task WebsiteCerts_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/certs");
        RegisterPageServices(handler);
        SetupPagerState("website:certs:table", 3, 50);

        var cut = RenderComponent<WebsiteCertsPage>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/certs", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/certs")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/certs", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/certs")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/certs", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/certs")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/certs").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/certs").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/certs");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task SystemLoginLogs_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/logs/login");
        RegisterPageServices(handler);
        SetupPagerState("system:login_logs:table", 3, 50);

        var cut = RenderComponent<SystemLoginLogsPage>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/login", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/login")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/login", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/login")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/login", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/login")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/logs/login").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/logs/login").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/logs/login");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task SystemOperationLogs_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/logs/operation");
        RegisterPageServices(handler);
        SetupPagerState("system:operation_logs:table", 3, 50);

        var cut = RenderComponent<SystemOpLogsPage>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/operation", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/operation")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/operation", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/operation")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/operation", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/operation")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/logs/operation").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/logs/operation").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/logs/operation");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task FinanceOrders_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/orders");
        RegisterPageServices(handler);
        SetupPagerState("finance:orders:table", 3, 50);

        var cut = RenderComponent<FinanceOrdersPage>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/orders", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/orders")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/orders", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/orders")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/orders", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/orders")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/orders").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/orders").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/orders");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task SystemLogs_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/logs/login", "/api/v1/admin/logs/operation");
        RegisterPageServices(handler);
        SetupPagerState("system:logs:table", 3, 50);

        var cut = RenderComponent<SystemLogsPage>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/login", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/login")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/login", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/login")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/login", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/login")[2], page: 1, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "操作日志").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/logs/operation", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/logs/operation")[0], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/logs/operation").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/logs/operation").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/logs/operation");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    [Fact]
    public async Task NodeGroups_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/node-groups");
        RegisterPageServices(handler);
        SetupPagerState("node:groups:table", 3, 50);

        var cut = RenderComponent<NodeGroupsPage>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/node-groups", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/node-groups")[0], page: 3, pageSize: 50, pageSizeKey: "limit");

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/node-groups", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/node-groups")[1], page: 3, pageSize: 50, pageSizeKey: "limit");

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/node-groups", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/node-groups")[2], page: 1, pageSize: 50, pageSizeKey: "limit");

        var baseline = handler.ForPath("/api/v1/admin/node-groups").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/node-groups").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/node-groups");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50, pageSizeKey: "limit");
    }

    [Fact]
    public async Task SystemUsers_Pagination_Regression()
    {
        var handler = CreateHandler("/api/v1/admin/users");
        RegisterPageServices(handler);
        SetupPagerState("system:users:table", 3, 50);

        var cut = RenderComponent<SystemUsersPage>();

        WaitForEndpoint(cut, handler, "/api/v1/admin/users", 1);
        AssertQuery(handler.ForPath("/api/v1/admin/users")[0], page: 3, pageSize: 50);

        var pager = cut.FindComponent<TablePager>().Instance;
        await pager.RefreshAsync();
        WaitForEndpoint(cut, handler, "/api/v1/admin/users", 2);
        AssertQuery(handler.ForPath("/api/v1/admin/users")[1], page: 3, pageSize: 50);

        cut.FindAll("button").Single(b => b.TextContent.Trim() == "搜索").Click();
        WaitForEndpoint(cut, handler, "/api/v1/admin/users", 3);
        AssertQuery(handler.ForPath("/api/v1/admin/users")[2], page: 1, pageSize: 50);

        var baseline = handler.ForPath("/api/v1/admin/users").Count;
        await Task.WhenAll(pager.RefreshAsync(), pager.SearchAsync(), pager.RefreshAsync(), pager.SearchAsync());
        cut.WaitForAssertion(() => Assert.True(handler.ForPath("/api/v1/admin/users").Count >= baseline + 1));

        var afterRapid = handler.ForPath("/api/v1/admin/users");
        Assert.True(afterRapid.Count <= baseline + 2);
        AssertQuery(afterRapid[^1], page: 1, pageSize: 50);
    }

    private void RegisterPageServices(RecordingApiHandler handler)
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

    private void SetupPagerState(string key, int page, int pageSize)
    {
        JSInterop.Setup<string?>("cnn.storage.get", invocation =>
            invocation.Arguments.Count == 1 &&
            string.Equals(invocation.Arguments[0]?.ToString(), key, StringComparison.Ordinal))
            .SetResult($"{{\"PagingEnabled\":true,\"Page\":{page},\"PageSize\":{pageSize}}}");
    }

    private static void WaitForEndpoint(IRenderedFragment cut, RecordingApiHandler handler, string path, int count)
    {
        cut.WaitForAssertion(() => Assert.True(handler.ForPath(path).Count >= count));
    }

    private static void AssertQuery(RecordedRequest request, int page, int pageSize, string pageSizeKey = "pageSize")
    {
        Assert.Equal(page.ToString(), request.Query["page"]);
        Assert.Equal(pageSize.ToString(), request.Query[pageSizeKey]);
    }

    private static RecordingApiHandler CreateHandler(params string[] delayedEndpoints)
    {
        return new RecordingApiHandler(TimeSpan.FromMilliseconds(80), delayedEndpoints);
    }

    private static async Task InvokePrivateAsync(object target, string methodName, params object[] args)
    {
        var method = target.GetType().GetMethod(
            methodName,
            System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic);
        Assert.NotNull(method);

        var result = method!.Invoke(target, args);
        if (result is Task task)
        {
            await task;
        }
    }

    private static void SetPrivateField<T>(object target, string fieldName, T value)
    {
        var field = target.GetType().GetField(
            fieldName,
            System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic);
        Assert.NotNull(field);
        field!.SetValue(target, value);
    }

    private sealed class RecordingApiHandler : HttpMessageHandler
    {
        private readonly TimeSpan _delay;
        private readonly HashSet<string> _delayedPaths;
        private readonly ConcurrentQueue<RecordedRequest> _requests = new();

        public RecordingApiHandler(TimeSpan delay, IReadOnlyCollection<string> delayedPaths)
        {
            _delay = delay;
            _delayedPaths = delayedPaths.ToHashSet(StringComparer.Ordinal);
        }

        public IReadOnlyList<RecordedRequest> ForPath(string path)
        {
            return _requests.Where(r => string.Equals(r.Path, path, StringComparison.Ordinal)).ToList();
        }

        protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
        {
            var uri = request.RequestUri ?? new Uri("http://localhost/");
            var query = QueryHelpers.ParseQuery(uri.Query)
                .ToDictionary(static pair => pair.Key, static pair => pair.Value.ToString(), StringComparer.OrdinalIgnoreCase);

            var recorded = new RecordedRequest(uri.AbsolutePath, query);
            _requests.Enqueue(recorded);

            if (_delayedPaths.Contains(uri.AbsolutePath) && _delay > TimeSpan.Zero)
            {
                await Task.Delay(_delay, cancellationToken);
            }

            var data = ResolveData(uri.AbsolutePath);
            var body = $"{{\"code\":200,\"message\":\"ok\",\"data\":{data},\"trace_id\":\"test\"}}";
            return new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(body, Encoding.UTF8, "application/json")
            };
        }

        private static string ResolveData(string path)
        {
            return path switch
            {
                "/api/v1/admin/nodes" => "{\"list\":[{\"id\":101,\"name\":\"node-101\",\"ip\":\"10.0.0.1\",\"line_count\":0,\"online\":true,\"enable\":true,\"anti_blocking\":true}],\"total\":500}",
                "/api/v1/admin/nodes/101/monitor_logs" => "{\"list\":[],\"total\":500}",
                "/api/v1/admin/tasks" => "{\"list\":[],\"total\":500,\"page\":1}",
                "/api/v1/admin/messages" => "{\"list\":[],\"total\":500}",
                "/api/v1/admin/announcements" => "{\"list\":[],\"total\":500}",
                "/api/v1/admin/sites" => "{\"list\":[],\"total\":500}",
                "/api/v1/admin/certs" => "{\"list\":[],\"total\":500}",
                "/api/v1/admin/dnsapi" => "{\"list\":[],\"total\":0}",
                "/api/v1/admin/forwards" => "{\"list\":[],\"total\":500}",
                "/api/v1/admin/forward_groups" => "{\"list\":[],\"total\":0}",
                "/api/v1/admin/user_packages" => "{\"list\":[],\"total\":0}",
                "/api/v1/admin/logs/access" => "{\"list\":[],\"total\":500}",
                "/api/v1/admin/logs/access/downloads" => "{\"list\":[],\"total\":0}",
                "/api/v1/admin/logs/block/current" => "{\"list\":[],\"total\":500}",
                "/api/v1/admin/logs/block/stats" => "{\"list\":[],\"total\":0}",
                "/api/v1/admin/logs/block/history" => "{\"list\":[],\"total\":0}",
                "/api/v1/admin/logs/login" => "{\"list\":[],\"total\":500}",
                "/api/v1/admin/logs/operation" => "{\"list\":[],\"total\":500}",
                "/api/v1/admin/orders" => "{\"list\":[],\"total\":500}",
                "/api/v1/admin/node-groups" => "{\"list\":[],\"total\":500}",
                "/api/v1/admin/users" => "{\"list\":[{\"id\":1001,\"name\":\"alice\",\"email\":\"alice@example.com\"}],\"total\":1}",
                "/api/v1/user/logs/operation" => "{\"list\":[],\"total\":500}",
                "/api/v1/user/messages" => "{\"list\":[],\"total\":500}",
                "/api/v1/user/orders" => "{\"list\":[],\"total\":500}",
                "/api/v1/admin/regions" => "{\"list\":[],\"total\":0}",
                "/api/v1/admin/domains" => "{\"list\":[],\"total\":0}",
                _ => "null"
            };
        }
    }

    private sealed record RecordedRequest(string Path, IReadOnlyDictionary<string, string> Query);
}
