using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Deletion;
using Cnn.Api.Services.Tasks.Workflow;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user/rules/cc/matchers")]
public sealed class CcMatchersController : ControllerBase
{
    private readonly ICcMatcherService _service;
    private readonly IUserPackagePermissionService _permissionService;
    private readonly IAdminIdentityResolver _identityResolver;
    private readonly IMessageLocalizer _localizer;
    private readonly IDeletionPreviewService _deletionPreviewService;
    private readonly IResourceDeleteRequestService _resourceDeleteRequestService;

    public CcMatchersController(
        ICcMatcherService service,
        IUserPackagePermissionService permissionService,
        IAdminIdentityResolver identityResolver,
        IMessageLocalizer localizer,
        IDeletionPreviewService deletionPreviewService,
        IResourceDeleteRequestService resourceDeleteRequestService)
    {
        _service = service;
        _permissionService = permissionService;
        _identityResolver = identityResolver;
        _localizer = localizer;
        _deletionPreviewService = deletionPreviewService;
        _resourceDeleteRequestService = resourceDeleteRequestService;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] CcListQuery query, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var result = await _service.ListAsync(query, userId, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("{id:long}")]
    public async Task<IActionResult> GetAsync(long id, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var result = await _service.GetAsync(id, cancellationToken);
        if (result.Success && result.Data != null)
        {
            if (result.Data.UserId != 0 && result.Data.UserId != userId)
            {
                return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.PermissionDenied));
            }
        }

        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] CcMatcherUpsertRequest request, CancellationToken cancellationToken)
    {
        var (userId, error) = await EnsureCustomCcRuleAllowedAsync(cancellationToken);
        if (error != null)
        {
            return error;
        }

        var result = await _service.CreateAsync(request, userId, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{id:long}")]
    public async Task<IActionResult> UpdateAsync(long id, [FromBody] CcMatcherUpsertRequest request, CancellationToken cancellationToken)
    {
        var (userId, error) = await EnsureCustomCcRuleAllowedAsync(cancellationToken);
        if (error != null)
        {
            return error;
        }

        var result = await _service.UpdateAsync(id, request, userId, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpDelete("{id:long}")]
    public async Task<IActionResult> DeleteAsync(long id, CancellationToken cancellationToken)
    {
        var (userId, error) = await EnsureCustomCcRuleAllowedAsync(cancellationToken);
        if (error != null)
        {
            return error;
        }

        return await RequestDeleteAuthorizedAsync(id, userId, cancellationToken);
    }

    [HttpGet("{id:long}/delete_preview")]
    public async Task<IActionResult> DeletePreviewAsync(long id, CancellationToken cancellationToken)
    {
        var (userId, error) = await EnsureCustomCcRuleAllowedAsync(cancellationToken);
        if (error != null)
        {
            return error;
        }

        var authResult = await EnsureRuleOwnerAsync(id, userId, cancellationToken);
        if (authResult != null)
        {
            return authResult;
        }

        var result = await _deletionPreviewService.PreviewAsync(ResourceTypes.CcMatcher, id, cancellationToken);
        return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result));
    }

    [HttpPost("{id:long}/delete_request")]
    public async Task<IActionResult> RequestDeleteAsync(long id, CancellationToken cancellationToken)
    {
        var (userId, error) = await EnsureCustomCcRuleAllowedAsync(cancellationToken);
        if (error != null)
        {
            return error;
        }

        return await RequestDeleteAuthorizedAsync(id, userId, cancellationToken);
    }

    private async Task<(long UserId, IActionResult? Error)> EnsureCustomCcRuleAllowedAsync(CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return (0, Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required")));
        }

        var permission = await _permissionService.UserHasCustomCcRuleAsync(userId, cancellationToken);
        if (!permission.Success)
        {
            var messageKey = permission.MessageKey ?? "custom_cc_rule_check_failed";
            return (0, Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InternalError, messageKey)));
        }

        if (!permission.Data)
        {
            return (0, Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.PermissionDenied, "custom_cc_rule_not_enabled")));
        }

        return (userId, null);
    }

    private long ResolveUserId()
    {
        var identity = _identityResolver.Resolve(User);
        return identity?.UserId ?? 0;
    }

    private async Task<IActionResult?> EnsureRuleOwnerAsync(long id, long userId, CancellationToken cancellationToken)
    {
        var result = await _service.GetAsync(id, cancellationToken);
        if (!result.Success || result.Data == null)
        {
            return ToResponse(result);
        }

        if (result.Data.UserId != userId)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.PermissionDenied));
        }

        return null;
    }

    private async Task<IActionResult> RequestDeleteAuthorizedAsync(long id, long userId, CancellationToken cancellationToken)
    {
        var authResult = await EnsureRuleOwnerAsync(id, userId, cancellationToken);
        if (authResult != null)
        {
            return authResult;
        }

        var result = await _resourceDeleteRequestService.RequestDeleteAsync(
            DeleteRequestCommandFactory.Create(ResourceTypes.CcMatcher, id, userId, userId),
            cancellationToken);
        if (result.Success)
        {
            return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result.Data));
        }

        return Ok(ApiResponseFactory.Fail<DeleteRequestResult>(
            HttpContext,
            _localizer,
            result.ErrorCode,
            result.MessageKey,
            data: result.Data));
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
