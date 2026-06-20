using Cnn.Common.Contracts.Agent;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Agent;

[ApiController]
[Route("api/v1/agent/certs")]
public sealed class CertsController : ControllerBase
{
    private readonly ICertService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly INodeRateLimitService _rateLimitService;

    public CertsController(ICertService service, IMessageLocalizer localizer, INodeRateLimitService rateLimitService)
    {
        _service = service;
        _localizer = localizer;
        _rateLimitService = rateLimitService;
    }

    [HttpPost("issued")]
    public async Task<IActionResult> ReceiveIssuedAsync([FromBody] AgentIssuedCertRequest request, CancellationToken cancellationToken)
    {
        if (request.RateLimited && HttpContext.Items.TryGetValue("node_id", out var nodeIdRaw) && nodeIdRaw != null)
        {
            var cooldown = request.RateCooldown > 0 ? TimeSpan.FromSeconds(request.RateCooldown) : TimeSpan.FromMinutes(10);
            if (long.TryParse(nodeIdRaw.ToString(), out var nodeId) && nodeId > 0)
            {
                _rateLimitService.MarkLimited(nodeId, cooldown);
            }
        }

        var result = await _service.UpdateIssuedCertAsync(request, cancellationToken);
        if (result.Success)
        {
            return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, true));
        }

        return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, result.ErrorCode, result.MessageKey));
    }
}


