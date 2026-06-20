using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user/message_sub")]
public sealed class MessageSubscriptionsController : ControllerBase
{
    private readonly IMessageService _service;
    private readonly IAdminIdentityResolver _identityResolver;
    private readonly IMessageLocalizer _localizer;

    public MessageSubscriptionsController(
        IMessageService service,
        IAdminIdentityResolver identityResolver,
        IMessageLocalizer localizer)
    {
        _service = service;
        _identityResolver = identityResolver;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync(CancellationToken cancellationToken)
    {
        var language = LanguageResolver.Resolve(HttpContext, _localizer.DefaultLanguage);
        var result = await _service.ListSubscriptionsAsync(ResolveUserId(), language, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut]
    public async Task<IActionResult> UpdateAsync([FromBody] MessageSubUpdateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateSubscriptionsAsync(ResolveUserId(), request, cancellationToken);
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
