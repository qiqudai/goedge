namespace Cnn.Api.Services.Common;

/// <summary>
/// 统一 API 层的访问范围（权限上下文）。
/// 用于替代散落在各处的 bool isAdmin 和 long userId。
/// </summary>
public sealed record AccessScope(bool IsAdmin, long UserId)
{
    /// <summary>管理员范围。可以可选地指定代理哪个用户操作。</summary>
    public static AccessScope Admin(long userId = 0) => new(true, userId);
    
    /// <summary>普通用户范围。必须指定 UserId。</summary>
    public static AccessScope User(long userId) => new(false, userId);

    /// <summary>是否存在有效的用户 ID。</summary>
    public bool HasUserId => UserId > 0;

    /// <summary>如果是管理员且未指定 UserId，则视为全局管理员范围。</summary>
    public bool IsGlobalAdmin => IsAdmin && UserId <= 0;
}
