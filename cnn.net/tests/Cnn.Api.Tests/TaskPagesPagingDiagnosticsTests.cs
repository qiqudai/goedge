using System.Net;
using System.Text;
using Bunit;
using Cnn.Api.Services;
using Cnn.Api.Services.Auth;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.JSInterop;
using Xunit;
using SystemTasksPage = Cnn.Api.Pages.System.Tasks;
using WebsitePurgePage = Cnn.Api.Pages.Website.Purge;

namespace Cnn.Api.Tests;

public sealed class TaskPagesPagingDiagnosticsTests : TestContext
{
    public TaskPagesPagingDiagnosticsTests()
    {
        JSInterop.Mode = JSRuntimeMode.Loose;
    }

    [Fact]
    public async Task SystemTasks_PagingDiagnostics_DefaultOff_AndHideable()
    {
        RegisterPageServices(new TaskPagesApiHandler());

        var cut = RenderComponent<SystemTasksPage>();
        cut.WaitForAssertion(() => Assert.Contains("显示分页诊断", cut.Markup));
        Assert.DoesNotContain("隐藏分页诊断", cut.Markup);
        Assert.DoesNotContain("stateKey: <code>system:tasks:table</code>", cut.Markup);

        await cut.InvokeAsync(() => cut.FindAll("button").Single(b => b.TextContent.Contains("显示分页诊断", StringComparison.Ordinal)).Click());
        cut.WaitForAssertion(() => Assert.Contains("隐藏分页诊断", cut.Markup));
        Assert.Contains("stateKey: <code>system:tasks:table</code>", cut.Markup);

        await cut.InvokeAsync(() => cut.FindAll("button").Single(b => b.TextContent.Contains("隐藏分页诊断", StringComparison.Ordinal)).Click());
        cut.WaitForAssertion(() => Assert.Contains("显示分页诊断", cut.Markup));
        Assert.DoesNotContain("stateKey: <code>system:tasks:table</code>", cut.Markup);
    }

    [Fact]
    public async Task WebsitePurge_ListPagingDiagnostics_DefaultOff_AndHideable()
    {
        RegisterPageServices(new TaskPagesApiHandler());

        var cut = RenderComponent<WebsitePurgePage>();

        await cut.InvokeAsync(() => cut.FindAll("button").Single(b => b.TextContent.Trim() == "操作记录").Click());
        cut.WaitForAssertion(() => Assert.Contains("显示分页诊断", cut.Markup));
        Assert.DoesNotContain("隐藏分页诊断", cut.Markup);
        Assert.DoesNotContain("stateKey: <code>website:purge:list:table</code>", cut.Markup);

        await cut.InvokeAsync(() => cut.FindAll("button").Single(b => b.TextContent.Contains("显示分页诊断", StringComparison.Ordinal)).Click());
        cut.WaitForAssertion(() => Assert.Contains("隐藏分页诊断", cut.Markup));
        Assert.Contains("stateKey: <code>website:purge:list:table</code>", cut.Markup);

        await cut.InvokeAsync(() => cut.FindAll("button").Single(b => b.TextContent.Contains("隐藏分页诊断", StringComparison.Ordinal)).Click());
        cut.WaitForAssertion(() => Assert.Contains("显示分页诊断", cut.Markup));
        Assert.DoesNotContain("stateKey: <code>website:purge:list:table</code>", cut.Markup);
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
    }

    private sealed class TaskPagesApiHandler : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
        {
            var path = request.RequestUri?.AbsolutePath ?? string.Empty;
            var data = path switch
            {
                "/api/v1/admin/tasks/usage" => "{\"limits\":{\"refresh_url\":100,\"refresh_dir\":100,\"preheat\":100},\"used\":{\"date\":\"2026-04-20\",\"refresh_url\":0,\"refresh_dir\":0,\"preheat\":0},\"remaining\":{\"refresh_url\":100,\"refresh_dir\":100,\"preheat\":100}}",
                "/api/v1/admin/tasks" => "{\"list\":[],\"total\":0,\"page\":1}",
                _ => "null"
            };

            var body = $"{{\"code\":200,\"message\":\"ok\",\"data\":{data},\"trace_id\":\"test\"}}";
            return Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(body, Encoding.UTF8, "application/json")
            });
        }
    }
}
