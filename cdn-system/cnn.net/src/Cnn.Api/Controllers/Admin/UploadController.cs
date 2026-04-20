using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/upload")]
public sealed class UploadController : ControllerBase
{
    private readonly IUploadService _service;
    private readonly IMessageLocalizer _localizer;

    public UploadController(IUploadService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpPost("image")]
    public async Task<IActionResult> UploadImageAsync([FromForm(Name = "file")] IFormFile? file, CancellationToken cancellationToken)
    {
        var result = await _service.SaveImageAsync(file, cancellationToken);
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
