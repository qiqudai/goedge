using Cnn.Api.Services.Stats;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class AccessLogGeoExpressionsTests
{
    [Fact]
    public void ClientCountryExpr_ShouldUseUploadedCountryField()
    {
        var expr = AccessLogGeoExpressions.ClientCountryExpr();

        Assert.Contains("client_country", expr);
        Assert.DoesNotContain("remote_addr", expr);
    }

    [Fact]
    public void ClientProvinceExpr_ShouldFallbackToCountry()
    {
        var expr = AccessLogGeoExpressions.ClientProvinceExpr();

        Assert.Contains("client_province", expr);
        Assert.Contains("client_country", expr);
    }

    [Theory]
    [InlineData("", "", "-")]
    [InlineData("中国", "", "中国")]
    [InlineData("", "广东省", "广东省")]
    [InlineData("中国", "广东省", "中国-广东省")]
    [InlineData("-", "-", "-")]
    public void FormatLocation_ShouldMatchGoBehavior(string country, string province, string expected)
    {
        Assert.Equal(expected, AccessLogGeoExpressions.FormatLocation(country, province));
    }
}
