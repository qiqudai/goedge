using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user/messages")]
public sealed class MessagesController : ControllerBase
{
    private readonly IMessageService _service;
    private readonly IAdminIdentityResolver _identityResolver;
    private readonly IMessageLocalizer _localizer;

    public MessagesController(
        IMessageService service,
        IAdminIdentityResolver identityResolver,
        IMessageLocalizer localizer)
    {
        _service = service;
        _identityResolver = identityResolver;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] MessageListQuery query, CancellationToken cancellationToken)
    {
        var language = LanguageResolver.Resolve(HttpContext, _localizer.DefaultLanguage);
        var result = await _service.ListUserAsync(query, ResolveUserId(), language, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("unread")]
    public async Task<IActionResult> UnreadAsync(CancellationToken cancellationToken)
    {
        var language = LanguageResolver.Resolve(HttpContext, _localizer.DefaultLanguage);
        var result = await _service.GetUnreadAsync(ResolveUserId(), language, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("{id:long}/read")]
    public async Task<IActionResult> MarkReadAsync(long id, CancellationToken cancellationToken)
    {
        var result = await _service.MarkReadAsync(ResolveUserId(), id, cancellationToken);
        return ToResponse(result);
    }

    private long? ResolveUserId()
    {
        var identity = _identityResolver.Resolve(User);
        return identity?.UserId;
    }

    private IActionResult ToResponse<T>(ServiceResult<T> result)
    {
        if (result.Success)
        {
            return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result.Data));
        }

        return Ok(ApiResponseFactory.Fail<T>(HttpContext, _localizer, result.ErrorCode, result.MessageKey));
    }
}
