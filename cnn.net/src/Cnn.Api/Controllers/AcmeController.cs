using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers;

[ApiController]
[Route(".well-known/acme-challenge")]
public sealed class AcmeController : ControllerBase
{
    private readonly IAcmeTokenStore _store;

    public AcmeController(IAcmeTokenStore store)
    {
        _store = store;
    }

    [HttpGet("{token}")]
    public IActionResult ServeAsync(string token)
    {
        if (string.IsNullOrWhiteSpace(token))
        {
            return NotFound();
        }

        if (_store.TryGet(token.Trim(), out var value) && !string.IsNullOrWhiteSpace(value))
        {
            return Content(value, "text/plain");
        }

        return NotFound();
    }
}


