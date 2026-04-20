using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/messages")]
public sealed class MessagesController : ControllerBase
{
    private readonly IMessageService _service;
    private readonly IMessageLocalizer _localizer;

    public MessagesController(IMessageService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] MessageListQuery query, CancellationToken cancellationToken)
    {
        var language = LanguageResolver.Resolve(HttpContext, _localizer.DefaultLanguage);
        var result = await _service.ListAdminAsync(query, language, cancellationToken);
        return ToResponse(result);
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
