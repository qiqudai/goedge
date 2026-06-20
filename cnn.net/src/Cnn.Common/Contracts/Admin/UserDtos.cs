using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class UserListQuery
{
    [JsonPropertyName("page")]
    public int? Page { get; set; }

    [JsonPropertyName("pageSize")]
    public int? PageSize { get; set; }

    [JsonPropertyName("size")]
    public int? Size { get; set; }

    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }
}

public sealed record UserListResult(
    [property: JsonPropertyName("list")] IReadOnlyList<UserItemDto> List,
    [property: JsonPropertyName("total")] int Total
);

public sealed class UserItemDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("email")]
    public string? Email { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("des")]
    public string? Des { get; set; }

    [JsonPropertyName("phone")]
    public string? Phone { get; set; }

    [JsonPropertyName("qq")]
    public string? Qq { get; set; }

    [JsonPropertyName("cert_id")]
    public string? CertId { get; set; }

    [JsonPropertyName("cert_name")]
    public string? CertName { get; set; }

    [JsonPropertyName("cert_no")]
    public string? CertNo { get; set; }

    [JsonPropertyName("cert_verified")]
    public bool? CertVerified { get; set; }

    [JsonPropertyName("white_ip")]
    public string? WhiteIp { get; set; }

    [JsonPropertyName("login_captcha")]
    public string? LoginCaptcha { get; set; }

    [JsonPropertyName("balance")]
    public long? Balance { get; set; }

    [JsonPropertyName("freeze")]
    public long? Freeze { get; set; }

    [JsonPropertyName("create_at")]
    public string? CreateAt { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }

    [JsonPropertyName("type")]
    public int? Type { get; set; }

    [JsonPropertyName("company")]
    public string? Company { get; set; }

    [JsonPropertyName("tea_code")]
    public string? TeaCode { get; set; }

    [JsonPropertyName("group_id")]
    public int GroupId { get; set; }

    [JsonPropertyName("secondary_auth")]
    public bool SecondaryAuth { get; set; }

    [JsonPropertyName("secondary_auth_deadline")]
    public string? SecondaryAuthDeadline { get; set; }

    [JsonPropertyName("secondary_auth_action")]
    public string? SecondaryAuthAction { get; set; }

    [JsonPropertyName("secondary_auth_status")]
    public string? SecondaryAuthStatus { get; set; }
}

public sealed class UserStatusUpdateRequest
{
    [JsonPropertyName("status")]
    public int? Status { get; set; }
}

public sealed class UserCreateRequest
{
    [JsonPropertyName("email")]
    public string? Email { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("des")]
    public string? Des { get; set; }

    [JsonPropertyName("phone")]
    public string? Phone { get; set; }

    [JsonPropertyName("qq")]
    public string? Qq { get; set; }

    [JsonPropertyName("password")]
    public string? Password { get; set; }

    [JsonPropertyName("group_id")]
    public int? GroupId { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }

    [JsonPropertyName("cert_name")]
    public string? CertName { get; set; }

    [JsonPropertyName("cert_no")]
    public string? CertNo { get; set; }

    [JsonPropertyName("company")]
    public string? Company { get; set; }

    [JsonPropertyName("tea_code")]
    public string? TeaCode { get; set; }

    [JsonPropertyName("login_captcha")]
    public string? LoginCaptcha { get; set; }

    [JsonPropertyName("white_ip")]
    public string? WhiteIp { get; set; }
}

public sealed class UserUpdateRequest
{
    [JsonPropertyName("email")]
    public string? Email { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("des")]
    public string? Des { get; set; }

    [JsonPropertyName("phone")]
    public string? Phone { get; set; }

    [JsonPropertyName("qq")]
    public string? Qq { get; set; }

    [JsonPropertyName("group_id")]
    public int? GroupId { get; set; }

    [JsonPropertyName("password")]
    public string? Password { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }

    [JsonPropertyName("cert_name")]
    public string? CertName { get; set; }

    [JsonPropertyName("cert_no")]
    public string? CertNo { get; set; }

    [JsonPropertyName("company")]
    public string? Company { get; set; }

    [JsonPropertyName("tea_code")]
    public string? TeaCode { get; set; }

    [JsonPropertyName("secondary_auth")]
    public bool? SecondaryAuth { get; set; }

    [JsonPropertyName("secondary_auth_deadline")]
    public string? SecondaryAuthDeadline { get; set; }

    [JsonPropertyName("secondary_auth_action")]
    public string? SecondaryAuthAction { get; set; }

    [JsonPropertyName("login_captcha")]
    public string? LoginCaptcha { get; set; }

    [JsonPropertyName("white_ip")]
    public string? WhiteIp { get; set; }
}
