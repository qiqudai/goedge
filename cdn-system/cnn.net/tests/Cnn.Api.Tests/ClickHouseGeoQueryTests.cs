using System.Net;
using System.Text;
using Cnn.Api.Controllers.Admin;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Stats;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Microsoft.AspNetCore.Http;
using Microsoft.AspNetCore.Mvc;
using Microsoft.Extensions.Configuration;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class ClickHouseGeoQueryTests
{
    [Fact]
    public async Task QueryRegionRankingAsync_ShouldQueryUploadedCountryField()
    {
        using var server = new SequentialHttpServer(
            """
            {"item":"中国","request_count":12,"out_traffic":345,"origin_traffic":120}
            """);

        var service = new RankingService(BuildConfiguration(server.BaseUrl));
        var filter = new HostFilter();
        filter.Exact.Add("demo.test");

        var result = await service.QueryRegionRankingAsync(
            "country",
            new DateTime(2026, 4, 20, 0, 0, 0),
            new DateTime(2026, 4, 20, 23, 59, 59),
            filter,
            null,
            10,
            CancellationToken.None);

        Assert.NotEmpty(server.CapturedQueries);
        var query = server.CapturedQueries[0];
        Assert.Contains("client_country", query);
        Assert.DoesNotContain("remote_addr AS ip", query);

        var item = Assert.Single(result);
        Assert.Equal("中国", item.Item);
        Assert.Equal((ulong)12, item.RequestCount);
    }

    [Fact]
    public async Task ListCurrentAsync_ShouldUseUploadedGeoFieldsForLocation()
    {
        using var server = new SequentialHttpServer(
            """
            {"total":1}
            """,
            """
            {"host":"demo.test","remote_addr":"1.2.3.4","agg_client_country":"中国","agg_client_province":"广东省","block_time":"2026-04-20 10:00:00","status":403}
            """);

        var service = new BlockLogService(
            BuildConfiguration(server.BaseUrl),
            new StubSiteHostIndexService("demo.test"));

        var result = await service.ListCurrentAsync(
            new BlockLogQuery { Page = 1, PageSize = 10, Type = "ip", Keyword = string.Empty },
            new DateTime(2026, 4, 20, 0, 0, 0),
            new DateTime(2026, 4, 20, 23, 59, 59),
            null,
            true,
            CancellationToken.None);

        Assert.True(result.Success);
        Assert.True(server.CapturedQueries.Count >= 2);
        Assert.Contains("argMax(", server.CapturedQueries[1]);
        Assert.Contains("client_country", server.CapturedQueries[1]);
        Assert.Contains("client_province", server.CapturedQueries[1]);

        var item = Assert.Single(result.Data!.List);
        Assert.Equal("中国-广东省", item.Location);
        Assert.Equal("demo.test", item.Domain);
        Assert.Equal("1.2.3.4", item.Ip);
    }

    [Fact]
    public async Task StatsService_GetRankingAsync_ShouldExposeCountryItemToFrontendDto()
    {
        using var server = new SequentialHttpServer(
            """
            {"item":"中国","request_count":12,"out_traffic":1048576,"origin_traffic":524288}
            """);

        var rankingService = new RankingService(BuildConfiguration(server.BaseUrl));
        var statsService = new StatsService(
            new StubAccessStatsService(),
            rankingService,
            new StubHostFilterResolver("demo.test"),
            new StubSystemConfigService(),
            BuildConfiguration(server.BaseUrl));

        var result = await statsService.GetRankingAsync(
            "country",
            null,
            new StatsRange(
                new DateTime(2026, 4, 20, 0, 0, 0),
                new DateTime(2026, 4, 20, 23, 59, 59),
                TimeSpan.FromDays(1),
                "MM-dd"),
            AccessScope.Admin(),
            CancellationToken.None);

        Assert.True(result.Success);
        var item = Assert.Single(result.Data!.List);
        Assert.Equal(1, item.Rank);
        Assert.Equal("中国", item.Item);
        Assert.Equal(12, item.RequestCount);
        Assert.Equal("1.00 MB", item.OutTraffic);
        Assert.Equal("512.00 KB", item.OriginTraffic);
    }

    [Fact]
    public async Task StatsController_RankingAsync_ShouldReturnCountryRankingResponse()
    {
        using var server = new SequentialHttpServer(
            """
            {"item":"中国","request_count":12,"out_traffic":1048576,"origin_traffic":524288}
            """);

        var statsService = new StatsService(
            new StubAccessStatsService(),
            new RankingService(BuildConfiguration(server.BaseUrl)),
            new StubHostFilterResolver("demo.test"),
            new StubSystemConfigService(),
            BuildConfiguration(server.BaseUrl));

        var controller = new StatsController(statsService, new AdminIdentityResolver(), new StubMessageLocalizer())
        {
            ControllerContext = new ControllerContext
            {
                HttpContext = BuildHttpContext("?time_range=7d")
            }
        };

        var action = await controller.RankingAsync("country", null, CancellationToken.None);
        var ok = Assert.IsType<OkObjectResult>(action);
        var response = Assert.IsType<ApiResponse<StatRankingResultDto>>(ok.Value);

        Assert.Equal(200, response.Code);
        var item = Assert.Single(response.Data!.List);
        Assert.Equal("中国", item.Item);
        Assert.Equal(12, item.RequestCount);
    }

    [Fact]
    public async Task BlockLogsController_CurrentAsync_ShouldReturnLocationFromGeoFields()
    {
        using var server = new SequentialHttpServer(
            """
            {"total":1}
            """,
            """
            {"host":"demo.test","remote_addr":"1.2.3.4","agg_client_country":"中国","agg_client_province":"广东省","block_time":"2026-04-20 10:00:00","status":403}
            """);

        var controller = new BlockLogsController(
            new BlockLogService(BuildConfiguration(server.BaseUrl), new StubSiteHostIndexService("demo.test")),
            new StubMessageLocalizer())
        {
            ControllerContext = new ControllerContext
            {
                HttpContext = BuildHttpContext()
            }
        };

        var action = await controller.CurrentAsync(
            new BlockLogQuery { Page = 1, PageSize = 10, Type = "ip", Keyword = string.Empty },
            CancellationToken.None);

        var ok = Assert.IsType<OkObjectResult>(action);
        var response = Assert.IsType<ApiResponse<BlockCurrentListResult>>(ok.Value);

        Assert.Equal(200, response.Code);
        var item = Assert.Single(response.Data!.List);
        Assert.Equal("中国-广东省", item.Location);
        Assert.Equal("demo.test", item.Domain);
    }

    private static IConfiguration BuildConfiguration(string baseUrl)
    {
        return new ConfigurationBuilder()
            .AddInMemoryCollection(new Dictionary<string, string?>
            {
                ["ClickHouse:Dsn"] = $"{baseUrl}default"
            })
            .Build();
    }

    private static DefaultHttpContext BuildHttpContext(string queryString = "")
    {
        var context = new DefaultHttpContext();
        context.TraceIdentifier = "test-trace";
        if (!string.IsNullOrWhiteSpace(queryString))
        {
            context.Request.QueryString = new QueryString(queryString);
        }

        return context;
    }

    private sealed class StubSiteHostIndexService : ISiteHostIndexService
    {
        private readonly SiteHostIndex _index = new();

        public StubSiteHostIndexService(string host)
        {
            _index.Filter.Exact.Add(host);
            _index.SiteFilters[1] = new HostFilter();
            _index.SiteFilters[1].Exact.Add(host);
        }

        public Task<SiteHostIndex> LoadAsync(long userId, CancellationToken cancellationToken)
        {
            return Task.FromResult(_index);
        }
    }

    private sealed class StubHostFilterResolver : IHostFilterResolver
    {
        private readonly HostFilter _filter = new();

        public StubHostFilterResolver(string host)
        {
            _filter.Exact.Add(host);
        }

        public Task<HostFilter> ResolveAsync(AccessScope scope, CancellationToken cancellationToken)
        {
            return Task.FromResult(_filter);
        }
    }

    private sealed class StubSystemConfigService : ISystemConfigService
    {
        public Task<Dictionary<string, string>> LoadSystemConfigAsync(CancellationToken cancellationToken)
        {
            return Task.FromResult(new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
            {
                ["res_rank_size"] = "10"
            });
        }

        public bool ParseBoolFlag(string? raw)
        {
            return false;
        }
    }

    private sealed class StubAccessStatsService : IAccessStatsService
    {
        public Task<IReadOnlyList<AccessBucket>> QueryBucketsAsync(StatsRange range, HostFilter hostFilter, CancellationToken cancellationToken)
        {
            return Task.FromResult<IReadOnlyList<AccessBucket>>(Array.Empty<AccessBucket>());
        }

        public Task<AccessTotals> QueryTotalsAsync(DateTime start, DateTime end, HostFilter hostFilter, CancellationToken cancellationToken)
        {
            return Task.FromResult(new AccessTotals(0, 0, 0));
        }

        public BucketSeries BuildSeries(StatsRange range, IReadOnlyList<AccessBucket> buckets)
        {
            return new BucketSeries();
        }

        public IReadOnlyList<int> BlockedStatusCodes => Array.Empty<int>();
    }

    private sealed class StubMessageLocalizer : IMessageLocalizer
    {
        public string DefaultLanguage => "zh-CN";

        public string Translate(string key, string? language)
        {
            return key;
        }
    }

    private sealed class SequentialHttpServer : IDisposable
    {
        private readonly HttpListener _listener = new();
        private readonly Task _serverTask;
        private readonly Queue<string> _responses;

        public SequentialHttpServer(params string[] responses)
        {
            var port = GetFreePort();
            BaseUrl = $"http://127.0.0.1:{port}/";
            _listener.Prefixes.Add(BaseUrl);
            _responses = new Queue<string>(responses);
            _listener.Start();

            _serverTask = Task.Run(async () =>
            {
                while (_listener.IsListening && _responses.Count > 0)
                {
                    var context = await _listener.GetContextAsync();
                    CapturedQueries.Add(Uri.UnescapeDataString(context.Request.Url?.Query ?? string.Empty));
                    var body = _responses.Dequeue();
                    var bytes = Encoding.UTF8.GetBytes(body + "\n");
                    context.Response.StatusCode = 200;
                    context.Response.ContentType = "text/plain; charset=utf-8";
                    await context.Response.OutputStream.WriteAsync(bytes);
                    context.Response.Close();
                }
            });
        }

        public string BaseUrl { get; }

        public List<string> CapturedQueries { get; } = new();

        public void Dispose()
        {
            _listener.Stop();
            _listener.Close();
            try
            {
                _serverTask.Wait(TimeSpan.FromSeconds(2));
            }
            catch
            {
                // ignore shutdown exceptions from listener cancellation
            }
        }

        private static int GetFreePort()
        {
            var listener = new System.Net.Sockets.TcpListener(IPAddress.Loopback, 0);
            listener.Start();
            var port = ((System.Net.IPEndPoint)listener.LocalEndpoint).Port;
            listener.Stop();
            return port;
        }
    }
}
