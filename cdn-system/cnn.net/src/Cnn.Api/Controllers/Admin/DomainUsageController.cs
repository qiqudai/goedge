using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/domain_usage")]
public sealed class DomainUsageController : BaseApiController
{
    private readonly IDomainUsageService _service;
    public DomainUsageController(IDomainUsageService service, Cnn.Api.Services.IAdminIdentityResolver identityResolver, IMessageLocalizer localizer) : base(identityResolver, localizer)
    {
        _service = service;
        }

    [HttpGet]
    public async Task<IActionResult> GetAsync([FromQuery(Name = "user_id")] long? userId, [FromQuery(Name = "user_package_id")] long? userPackageId, CancellationToken cancellationToken)
    {
        var resolvedUserId = userId.GetValueOrDefault();
        if (resolvedUserId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var resolvedPackageId = userPackageId.GetValueOrDefault();
        if (resolvedPackageId <= 0)
        {
            resolvedPackageId = await _service.FindDefaultUserPackageIdAsync(resolvedUserId);
            if (resolvedPackageId <= 0)
            {
                return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.NotFound, "package_not_found"));
            }
        }

        var result = await _service.GetUsageAsync(resolvedUserId, resolvedPackageId, cancellationToken);
        if (!result.Success)
        {
            return Ok(ApiResponseFactory.Fail<DomainUsageDto>(HttpContext, _localizer, result.ErrorCode, result.MessageKey));
        }

        var usage = ApplyDomainUsageMessage(result.Data);
        return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, usage));
    }

    private DomainUsageDto? ApplyDomainUsageMessage(DomainUsageDto? usage)
    {
        if (usage == null || !usage.Exceeded)
        {
            return usage;
        }

        var lang = LanguageResolver.Resolve(HttpContext, _localizer.DefaultLanguage);
        if (usage.DomainLimit > 0 && usage.TotalDomains > usage.DomainLimit)
        {
            var template = _localizer.Translate("domain_limit_exceeded", lang);
            usage.Message = string.Format(template, usage.TotalDomains);
            return usage;
        }
        if (usage.MainDomainLimit > 0 && usage.TotalMainDomains > usage.MainDomainLimit)
        {
            var template = _localizer.Translate("main_domain_limit_exceeded", lang);
            usage.Message = string.Format(template, usage.TotalMainDomains);
            return usage;
        }

        return usage;
    }


}
