using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Agent;

[ApiController]
[Route("api/v1/agent/acme/tokens")]
public sealed class AcmeTokensController : ControllerBase
{
    private readonly IAcmeTokenStore _store;
    private readonly IMessageLocalizer _localizer;

    public AcmeTokensController(IAcmeTokenStore store, IMessageLocalizer localizer)
    {
        _store = store;
        _localizer = localizer;
    }

    [HttpPost]
    public IActionResult PutAsync([FromBody] AcmeTokenRequest request)
    {
        if (string.IsNullOrWhiteSpace(request?.Token))
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.MissingParam, "token_required"));
        }
        if (string.IsNullOrWhiteSpace(request.Value))
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.MissingParam, "value_required"));
        }

        var ttl = TimeSpan.FromMinutes(15);
        if (request.Ttl > 0)
        {
            ttl = TimeSpan.FromSeconds(request.Ttl);
        }
        _store.Put(request.Token.Trim(), request.Value.Trim(), ttl);

        return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, true));
    }

    [HttpDelete("{token}")]
    public IActionResult DeleteAsync(string token)
    {
        if (string.IsNullOrWhiteSpace(token))
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.MissingParam, "token_required"));
        }

        _store.Delete(token.Trim());
        return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, true));
    }

    public sealed class AcmeTokenRequest
    {
        public string? Token { get; set; }
        public string? Value { get; set; }
        public long Ttl { get; set; }
    }
}


