namespace Cnn.Api.Services.Stats;

public static class AccessLogGeoExpressions
{
    public static string ClientCountryExpr()
    {
        return "if(trim(BOTH ' ' FROM client_country) = '' OR client_country = '-', '-', trim(BOTH ' ' FROM client_country))";
    }

    public static string ClientProvinceExpr()
    {
        return $"if(trim(BOTH ' ' FROM client_province) = '' OR client_province = '-', {ClientCountryExpr()}, trim(BOTH ' ' FROM client_province))";
    }

    public static string FormatLocation(string? country, string? province)
    {
        country = NormalizeLocationPart(country);
        province = NormalizeLocationPart(province);
        if (string.IsNullOrWhiteSpace(country) && string.IsNullOrWhiteSpace(province))
        {
            return "-";
        }

        if (string.IsNullOrWhiteSpace(province))
        {
            return country;
        }

        if (string.IsNullOrWhiteSpace(country))
        {
            return province;
        }

        return country + "-" + province;
    }

    private static string NormalizeLocationPart(string? value)
    {
        value = value?.Trim() ?? string.Empty;
        return value == "-" ? string.Empty : value;
    }
}
