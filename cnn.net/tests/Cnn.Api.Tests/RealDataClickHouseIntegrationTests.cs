using System.Text.Json;
using Cnn.Api.Services.Stats;
using Microsoft.Extensions.Configuration;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class RealDataClickHouseIntegrationTests
{
    [Fact]
    public async Task QueryRowsAsync_ClickHouseAvailable_ShouldReturnSingleRow()
    {
        var config = ResolveClickHouseConfig();
        if (config == null)
        {
            return;
        }

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(config, "SELECT 1", CancellationToken.None);
        Assert.NotNull(rows);
        Assert.Single(rows!);
        Assert.Equal("1", rows[0].Trim());
    }

    private static ClickHouseHttpConfig? ResolveClickHouseConfig()
    {
        var dsn = Environment.GetEnvironmentVariable("ClickHouse__Dsn")
                  ?? Environment.GetEnvironmentVariable("ClickHouse__HttpDsn");
        if (string.IsNullOrWhiteSpace(dsn))
        {
            var path = "/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/appsettings.json";
            using var stream = File.OpenRead(path);
            using var document = JsonDocument.Parse(stream);
            if (document.RootElement.TryGetProperty("ClickHouse", out var clickHouse) &&
                clickHouse.TryGetProperty("Dsn", out var value))
            {
                dsn = value.GetString();
            }
        }

        if (string.IsNullOrWhiteSpace(dsn))
        {
            return null;
        }

        var configuration = new ConfigurationBuilder()
            .AddInMemoryCollection(new Dictionary<string, string?> { ["ClickHouse:Dsn"] = dsn })
            .Build();
        return ClickHouseHttpHelper.ResolveConfig(configuration);
    }
}
