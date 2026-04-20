using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Stats;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/dashboard")]
public sealed class DashboardController : BaseApiController
{
    private readonly IDashboardService _service;

    public DashboardController(IDashboardService service, Cnn.Api.Services.IAdminIdentityResolver identityResolver, IMessageLocalizer localizer) : base(identityResolver, localizer)
    {
        _service = service;
    }

    [HttpGet]
    public async Task<IActionResult> GetAsync(
        [FromQuery(Name = "overview_range")] string? overviewRange,
        [FromQuery(Name = "chart_range")] string? chartRange,
        [FromQuery(Name = "ops_range")] string? opsRange,
        [FromQuery(Name = "range")] string? range,
        CancellationToken cancellationToken)
    {
        var language = LanguageResolver.Resolve(HttpContext, _localizer.DefaultLanguage);
        var result = await _service.GetAsync(AccessScope.Admin(), overviewRange, chartRange, opsRange, range, language, cancellationToken);
        return ToResponse(result);
    }

}
