using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/api_key")]
public sealed class ApiKeyController : BaseApiController
{
    private readonly IApiKeyService _service;

    public ApiKeyController(IApiKeyService service, IAdminIdentityResolver identityResolver, IMessageLocalizer localizer) : base(identityResolver, localizer)
    {
        _service = service;
    }

    [HttpGet]
    public async Task<IActionResult> GetAsync(CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<ApiKeyDto>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.GetAsync(userId, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut]
    public async Task<IActionResult> UpdateAsync([FromBody] ApiKeyUpdateRequest request, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<bool>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.UpdateAsync(userId, request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("reset")]
    public async Task<IActionResult> ResetAsync(CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<ApiKeySecretDto>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.ResetSecretAsync(userId, cancellationToken);
        return ToResponse(result);
    }

}
