using System.Net;
using System.Text;
using System.Text.Json;
using Cnn.Api.Services.Agent;
using Microsoft.Extensions.Configuration;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class AgentLogGeoWriteTests
{
    [Fact]
    public async Task InsertAccessLogsAsync_ShouldWriteCountryAndProvinceFields()
    {
        using var listener = new HttpListener();
        var port = GetFreePort();
        var prefix = $"http://127.0.0.1:{port}/";
        listener.Prefixes.Add(prefix);
        listener.Start();

        string? capturedBody = null;
        string? capturedQuery = null;
        var serverTask = Task.Run(async () =>
        {
            var context = await listener.GetContextAsync();
            capturedQuery = context.Request.Url?.Query;
            using var reader = new StreamReader(context.Request.InputStream, Encoding.UTF8);
            capturedBody = await reader.ReadToEndAsync();
            context.Response.StatusCode = 200;
            context.Response.Close();
        });

        var configuration = new ConfigurationBuilder()
            .AddInMemoryCollection(new Dictionary<string, string?>
            {
                ["ClickHouse:Dsn"] = $"http://127.0.0.1:{port}/default"
            })
            .Build();

        var service = new AgentLogService(configuration);
        var lines = new[]
        {
            """
            {"time_iso8601":"2026-04-20T14:20:00+07:00","remote_addr":"1.2.3.4","client_country":"中国","client_province":"广东省","host":"demo.test","request":"GET /hello HTTP/1.1","status":200,"body_bytes_sent":123,"request_time":0.12,"upstream_addr":"127.0.0.1:8080","upstream_response_time":"0.09","upstream_cache_status":"MISS","http_referer":"-","http_user_agent":"curl/8.0","scheme":"https","ssl_protocol":"TLSv1.3","ssl_cipher":"TLS_AES_128_GCM_SHA256"}
            """
        };

        var written = await service.InsertAccessLogsAsync("node-1", "10.0.0.1", lines, CancellationToken.None);
        await serverTask;

        Assert.Equal(1, written);
        Assert.NotNull(capturedQuery);
        Assert.Contains("INSERT%20INTO%20node_access_logs%20FORMAT%20JSONEachRow", capturedQuery);
        Assert.NotNull(capturedBody);
        using var document = JsonDocument.Parse(capturedBody);
        var root = document.RootElement;
        Assert.Equal("中国", root.GetProperty("client_country").GetString());
        Assert.Equal("广东省", root.GetProperty("client_province").GetString());
        Assert.Equal("1.2.3.4", root.GetProperty("remote_addr").GetString());
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
